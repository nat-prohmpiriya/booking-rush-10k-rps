package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	pkgmiddleware "github.com/prohmpiriya/booking-rush-10k-rps/pkg/middleware"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// ServiceConfig holds configuration for a backend service
type ServiceConfig struct {
	Name    string
	BaseURL string
	Timeout time.Duration
}

// RouteConfig holds configuration for a route
type RouteConfig struct {
	// PathPrefix is the prefix that triggers this route (e.g., "/api/v1/auth")
	PathPrefix string
	// StripPrefix removes the prefix before forwarding (e.g., strip "/api/v1" from path)
	StripPrefix string
	// Service is the target backend service
	Service ServiceConfig
	// RequireAuth indicates if JWT authentication is required
	RequireAuth bool
	// AllowedMethods restricts which HTTP methods are allowed (empty = all)
	AllowedMethods []string
}

// ProxyConfig holds the overall proxy configuration
type ProxyConfig struct {
	Routes        []RouteConfig
	DefaultTimeout time.Duration
	JWTSecret     string
}

// ReverseProxy manages routing to backend services
type ReverseProxy struct {
	config   ProxyConfig
	proxies  map[string]*httputil.ReverseProxy
	mu       sync.RWMutex
	client   *http.Client
}

// NewReverseProxy creates a new reverse proxy instance
func NewReverseProxy(config ProxyConfig) *ReverseProxy {
	if config.DefaultTimeout == 0 {
		config.DefaultTimeout = 30 * time.Second
	}

	// Create optimized HTTP transport for high performance
	// MaxIdleConns/MaxIdleConnsPerHost set to 15000 to handle 10K+ SSE connections at scale
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          15000,
		MaxIdleConnsPerHost:   15000,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableCompression:    false,
		ForceAttemptHTTP2:     true,
	}

	rp := &ReverseProxy{
		config:  config,
		proxies: make(map[string]*httputil.ReverseProxy),
		client: &http.Client{
			Transport: transport,
			Timeout:   config.DefaultTimeout,
		},
	}

	// Initialize proxies for each unique service
	for _, route := range config.Routes {
		if _, exists := rp.proxies[route.Service.Name]; !exists {
			rp.initProxy(route.Service)
		}
	}

	return rp
}

// initProxy initializes a reverse proxy for a service
func (rp *ReverseProxy) initProxy(service ServiceConfig) {
	targetURL, err := url.Parse(service.BaseURL)
	if err != nil {
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.Transport = rp.client.Transport

	// Custom director to modify requests before forwarding
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = targetURL.Host
	}

	// Custom error handler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		w.Header().Set("Content-Type", "application/json")
		if isTimeoutError(err) {
			w.WriteHeader(http.StatusGatewayTimeout)
			io.WriteString(w, `{"success":false,"error":{"code":"GATEWAY_TIMEOUT","message":"Backend service timed out"}}`)
		} else if isConnectionError(err) {
			w.WriteHeader(http.StatusBadGateway)
			io.WriteString(w, `{"success":false,"error":{"code":"BAD_GATEWAY","message":"Backend service unavailable"}}`)
		} else {
			w.WriteHeader(http.StatusBadGateway)
			io.WriteString(w, `{"success":false,"error":{"code":"BAD_GATEWAY","message":"Backend service error"}}`)
		}
	}

	// Custom response modifier
	proxy.ModifyResponse = func(resp *http.Response) error {
		// Add gateway headers
		resp.Header.Set("X-Proxied-By", "api-gateway")
		return nil
	}

	rp.mu.Lock()
	rp.proxies[service.Name] = proxy
	rp.mu.Unlock()
}

// findRoute finds the matching route for a request
func (rp *ReverseProxy) findRoute(path, method string) *RouteConfig {
	for _, route := range rp.config.Routes {
		if strings.HasPrefix(path, route.PathPrefix) {
			// Check method if restricted
			if len(route.AllowedMethods) > 0 {
				allowed := false
				for _, m := range route.AllowedMethods {
					if strings.EqualFold(m, method) {
						allowed = true
						break
					}
				}
				if !allowed {
					continue
				}
			}
			return &route
		}
	}
	return nil
}

// Handler returns a Gin handler for proxying requests
func (rp *ReverseProxy) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, span := telemetry.StartSpan(c.Request.Context(), "gateway.proxy")
		defer span.End()
		c.Request = c.Request.WithContext(ctx)

		span.SetAttributes(
			attribute.String("http.method", c.Request.Method),
			attribute.String("http.path", c.Request.URL.Path),
		)

		route := rp.findRoute(c.Request.URL.Path, c.Request.Method)
		if route == nil {
			span.SetStatus(codes.Error, "No route configured for this path")
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "ROUTE_NOT_FOUND",
					"message": "No route configured for this path",
				},
			})
			c.Abort()
			return
		}

		span.SetAttributes(attribute.String("target.service", route.Service.Name))

		// Get proxy for this service
		rp.mu.RLock()
		proxy, exists := rp.proxies[route.Service.Name]
		rp.mu.RUnlock()

		if !exists {
			span.SetStatus(codes.Error, "Backend service not configured")
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "SERVICE_NOT_CONFIGURED",
					"message": "Backend service not configured",
				},
			})
			c.Abort()
			return
		}

		// Strip prefix if configured
		if route.StripPrefix != "" {
			c.Request.URL.Path = strings.TrimPrefix(c.Request.URL.Path, route.StripPrefix)
			if c.Request.URL.Path == "" {
				c.Request.URL.Path = "/"
			}
		}

		// Add user context headers if authenticated
		if userID, exists := c.Get(pkgmiddleware.ContextKeyUserID); exists {
			c.Request.Header.Set("X-User-ID", userID.(string))
		}
		if email, exists := c.Get(pkgmiddleware.ContextKeyEmail); exists {
			c.Request.Header.Set("X-User-Email", email.(string))
		}
		if role, exists := c.Get(pkgmiddleware.ContextKeyRole); exists {
			c.Request.Header.Set("X-User-Role", role.(string))
		}
		if tenantID, exists := c.Get(pkgmiddleware.ContextKeyTenantID); exists {
			c.Request.Header.Set("X-Tenant-ID", tenantID.(string))
		}

		// Add request ID for tracing
		if requestID := c.GetHeader("X-Request-ID"); requestID != "" {
			c.Request.Header.Set("X-Request-ID", requestID)
		}

		// Set timeout context
		timeout := route.Service.Timeout
		if timeout == 0 {
			timeout = rp.config.DefaultTimeout
		}
		timeoutCtx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()
		c.Request = c.Request.WithContext(timeoutCtx)

		span.SetStatus(codes.Ok, "")

		// Proxy the request with panic recovery
		func() {
			defer func() {
				if r := recover(); r != nil {
					span.SetStatus(codes.Error, fmt.Sprintf("panic: %v", r))
					span.RecordError(fmt.Errorf("panic: %v", r))
					// Can't use c.JSON as writer may be partially written
					// Just log the panic for now
				}
			}()
			proxy.ServeHTTP(c.Writer, c.Request)
		}()
	}
}

// isTimeoutError checks if error is a timeout
func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return true
	}
	if err == context.DeadlineExceeded {
		return true
	}
	return strings.Contains(err.Error(), "timeout")
}

// isConnectionError checks if error is a connection error
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}
	if _, ok := err.(*net.OpError); ok {
		return true
	}
	return strings.Contains(err.Error(), "connection refused") ||
		strings.Contains(err.Error(), "no such host")
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// DefaultConfig returns default proxy configuration (reads from environment variables)
func DefaultConfig() ProxyConfig {
	authURL := getEnvOrDefault("AUTH_SERVICE_URL", "http://localhost:8081")
	ticketURL := getEnvOrDefault("TICKET_SERVICE_URL", "http://localhost:8082")
	bookingURL := getEnvOrDefault("BOOKING_SERVICE_URL", "http://localhost:8083")

	return ProxyConfig{
		DefaultTimeout: 30 * time.Second,
		Routes: []RouteConfig{
			// Auth Service routes (public)
			{
				PathPrefix:  "/api/v1/auth",
				StripPrefix: "",
				Service: ServiceConfig{
					Name:    "auth-service",
					BaseURL: authURL,
					Timeout: 10 * time.Second,
				},
				RequireAuth: false,
			},
			// Ticket/Events Service routes (public read, protected write)
			{
				PathPrefix:  "/api/v1/events",
				StripPrefix: "",
				Service: ServiceConfig{
					Name:    "ticket-service",
					BaseURL: ticketURL,
					Timeout: 15 * time.Second,
				},
				RequireAuth:    false, // GET is public, POST/PUT/DELETE will have JWT middleware
				AllowedMethods: []string{"GET"},
			},
			{
				PathPrefix:  "/api/v1/events",
				StripPrefix: "",
				Service: ServiceConfig{
					Name:    "ticket-service",
					BaseURL: ticketURL,
					Timeout: 15 * time.Second,
				},
				RequireAuth:    true,
				AllowedMethods: []string{"POST", "PUT", "DELETE", "PATCH"},
			},
			// Booking Service routes (protected)
			{
				PathPrefix:  "/api/v1/bookings",
				StripPrefix: "",
				Service: ServiceConfig{
					Name:    "booking-service",
					BaseURL: bookingURL,
					Timeout: 30 * time.Second, // Longer timeout for booking operations
				},
				RequireAuth: true,
			},
			// User profile routes (protected)
			{
				PathPrefix:  "/api/v1/users",
				StripPrefix: "",
				Service: ServiceConfig{
					Name:    "auth-service",
					BaseURL: authURL,
					Timeout: 10 * time.Second,
				},
				RequireAuth: true,
			},
		},
	}
}

// ConfigFromEnv creates proxy config from environment variables
func ConfigFromEnv(authURL, ticketURL, bookingURL, paymentURL, jwtSecret string) ProxyConfig {
	if authURL == "" {
		authURL = "http://localhost:8081"
	}
	if ticketURL == "" {
		ticketURL = "http://localhost:8082"
	}
	if bookingURL == "" {
		bookingURL = "http://localhost:8083"
	}
	if paymentURL == "" {
		paymentURL = "http://localhost:8084"
	}

	return ProxyConfig{
		DefaultTimeout: 30 * time.Second,
		JWTSecret:      jwtSecret,
		Routes: []RouteConfig{
			// Auth Service routes
			{
				PathPrefix:  "/api/v1/auth",
				StripPrefix: "",
				Service: ServiceConfig{
					Name:    "auth-service",
					BaseURL: authURL,
					Timeout: 10 * time.Second,
				},
				RequireAuth: false,
			},
			// Events/Tickets - GET is public
			{
				PathPrefix:  "/api/v1/events",
				StripPrefix: "",
				Service: ServiceConfig{
					Name:    "ticket-service",
					BaseURL: ticketURL,
					Timeout: 15 * time.Second,
				},
				RequireAuth:    false,
				AllowedMethods: []string{"GET"},
			},
			// Events/Tickets - Write operations need auth
			{
				PathPrefix:  "/api/v1/events",
				StripPrefix: "",
				Service: ServiceConfig{
					Name:    "ticket-service",
					BaseURL: ticketURL,
					Timeout: 15 * time.Second,
				},
				RequireAuth:    true,
				AllowedMethods: []string{"POST", "PUT", "DELETE", "PATCH"},
			},
			// Shows - public GET
			{
				PathPrefix:  "/api/v1/shows",
				StripPrefix: "",
				Service: ServiceConfig{
					Name:    "ticket-service",
					BaseURL: ticketURL,
					Timeout: 15 * time.Second,
				},
				RequireAuth:    false,
				AllowedMethods: []string{"GET"},
			},
			// Shows - protected writes
			{
				PathPrefix:  "/api/v1/shows",
				StripPrefix: "",
				Service: ServiceConfig{
					Name:    "ticket-service",
					BaseURL: ticketURL,
					Timeout: 15 * time.Second,
				},
				RequireAuth:    true,
				AllowedMethods: []string{"POST", "PUT", "DELETE", "PATCH"},
			},
			// Zones - public GET
			{
				PathPrefix:  "/api/v1/zones",
				StripPrefix: "",
				Service: ServiceConfig{
					Name:    "ticket-service",
					BaseURL: ticketURL,
					Timeout: 15 * time.Second,
				},
				RequireAuth:    false,
				AllowedMethods: []string{"GET"},
			},
			// Zones - protected writes
			{
				PathPrefix:  "/api/v1/zones",
				StripPrefix: "",
				Service: ServiceConfig{
					Name:    "ticket-service",
					BaseURL: ticketURL,
					Timeout: 15 * time.Second,
				},
				RequireAuth:    true,
				AllowedMethods: []string{"POST", "PUT", "DELETE", "PATCH"},
			},
			// Bookings - all protected
			{
				PathPrefix:  "/api/v1/bookings",
				StripPrefix: "",
				Service: ServiceConfig{
					Name:    "booking-service",
					BaseURL: bookingURL,
					Timeout: 30 * time.Second,
				},
				RequireAuth: true,
			},
			// Queue - all protected (SSE needs 5 minutes timeout)
			{
				PathPrefix:  "/api/v1/queue",
				StripPrefix: "",
				Service: ServiceConfig{
					Name:    "booking-service",
					BaseURL: bookingURL,
					Timeout: 5 * time.Minute, // SSE streaming requires longer timeout
				},
				RequireAuth: true,
			},
			// Admin - booking service admin endpoints (protected)
			{
				PathPrefix:  "/api/v1/admin",
				StripPrefix: "",
				Service: ServiceConfig{
					Name:    "booking-service",
					BaseURL: bookingURL,
					Timeout: 60 * time.Second, // Sync may take longer
				},
				RequireAuth: true,
			},
			// Payments - all protected
			{
				PathPrefix:  "/api/v1/payments",
				StripPrefix: "",
				Service: ServiceConfig{
					Name:    "payment-service",
					BaseURL: paymentURL,
					Timeout: 60 * time.Second, // Payment needs longer timeout
				},
				RequireAuth: true,
			},
			// Stripe Webhooks - public (uses Stripe signature verification)
			{
				PathPrefix:  "/api/v1/webhooks",
				StripPrefix: "",
				Service: ServiceConfig{
					Name:    "payment-service",
					BaseURL: paymentURL,
					Timeout: 30 * time.Second,
				},
				RequireAuth:    false,
				AllowedMethods: []string{"POST"},
			},
			// User profile - protected
			{
				PathPrefix:  "/api/v1/users",
				StripPrefix: "",
				Service: ServiceConfig{
					Name:    "auth-service",
					BaseURL: authURL,
					Timeout: 10 * time.Second,
				},
				RequireAuth: true,
			},
		},
	}
}

// GetRequireAuthRoutes returns routes that require authentication
func (rp *ReverseProxy) GetRequireAuthRoutes() []RouteConfig {
	var routes []RouteConfig
	for _, r := range rp.config.Routes {
		if r.RequireAuth {
			routes = append(routes, r)
		}
	}
	return routes
}

// GetPublicRoutes returns routes that don't require authentication
func (rp *ReverseProxy) GetPublicRoutes() []RouteConfig {
	var routes []RouteConfig
	for _, r := range rp.config.Routes {
		if !r.RequireAuth {
			routes = append(routes, r)
		}
	}
	return routes
}

// HealthCheck checks if all backend services are reachable
func (rp *ReverseProxy) HealthCheck(ctx context.Context) map[string]bool {
	results := make(map[string]bool)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Get unique services
	services := make(map[string]ServiceConfig)
	for _, route := range rp.config.Routes {
		services[route.Service.Name] = route.Service
	}

	for name, service := range services {
		wg.Add(1)
		go func(name string, service ServiceConfig) {
			defer wg.Done()

			healthURL := fmt.Sprintf("%s/health", service.BaseURL)
			req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
			if err != nil {
				mu.Lock()
				results[name] = false
				mu.Unlock()
				return
			}

			resp, err := rp.client.Do(req)
			if err != nil {
				mu.Lock()
				results[name] = false
				mu.Unlock()
				return
			}
			defer resp.Body.Close()

			mu.Lock()
			results[name] = resp.StatusCode == http.StatusOK
			mu.Unlock()
		}(name, service)
	}

	wg.Wait()
	return results
}
