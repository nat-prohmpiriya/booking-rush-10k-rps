package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-auth/internal/dto"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-auth/internal/service"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/response"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// AuthHandler handles authentication HTTP requests
type AuthHandler struct {
	authService service.AuthService
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Register handles user registration
// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.auth.register")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid request body")
		c.JSON(http.StatusBadRequest, response.BadRequest(err.Error()))
		return
	}

	span.SetAttributes(attribute.String("email", req.Email))

	// Validate email format
	if valid, msg := req.ValidateEmail(); !valid {
		span.SetStatus(codes.Error, "invalid email")
		c.JSON(http.StatusBadRequest, response.Error("INVALID_EMAIL", msg))
		return
	}

	// Validate password strength
	if valid, msg := req.ValidatePassword(); !valid {
		span.SetStatus(codes.Error, "weak password")
		c.JSON(http.StatusBadRequest, response.Error("WEAK_PASSWORD", msg))
		return
	}

	result, err := h.authService.Register(ctx, &req)
	if err != nil {
		span.RecordError(err)
		if errors.Is(err, service.ErrUserAlreadyExists) {
			span.SetStatus(codes.Error, "user already exists")
			c.JSON(http.StatusConflict, response.Error("USER_EXISTS", "User with this email already exists"))
			return
		}
		span.SetStatus(codes.Error, err.Error())
		c.JSON(http.StatusInternalServerError, response.InternalError(err.Error()))
		return
	}

	span.SetAttributes(attribute.String("user_id", result.User.ID))
	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusCreated, response.Success(result))
}

// Login handles user login
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.auth.login")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid request body")
		c.JSON(http.StatusBadRequest, response.BadRequest(err.Error()))
		return
	}

	span.SetAttributes(attribute.String("email", req.Email))

	userAgent := c.GetHeader("User-Agent")
	ip := c.ClientIP()

	result, err := h.authService.Login(ctx, &req, userAgent, ip)
	if err != nil {
		span.RecordError(err)
		if errors.Is(err, service.ErrInvalidCredentials) {
			span.SetStatus(codes.Error, "invalid credentials")
			c.JSON(http.StatusUnauthorized, response.Error("INVALID_CREDENTIALS", "Invalid email or password"))
			return
		}
		if errors.Is(err, service.ErrUserInactive) {
			span.SetStatus(codes.Error, "user inactive")
			c.JSON(http.StatusForbidden, response.Error("USER_INACTIVE", "User account is inactive"))
			return
		}
		span.SetStatus(codes.Error, err.Error())
		c.JSON(http.StatusInternalServerError, response.InternalError(err.Error()))
		return
	}

	span.SetAttributes(attribute.String("user_id", result.User.ID))
	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusOK, response.Success(result))
}

// RefreshToken handles token refresh
// POST /api/v1/auth/refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.auth.refresh_token")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid request body")
		c.JSON(http.StatusBadRequest, response.BadRequest(err.Error()))
		return
	}

	result, err := h.authService.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		span.RecordError(err)
		if errors.Is(err, service.ErrSessionNotFound) {
			span.SetStatus(codes.Error, "session not found")
			c.JSON(http.StatusUnauthorized, response.Error("INVALID_TOKEN", "Invalid or expired refresh token"))
			return
		}
		if errors.Is(err, service.ErrTokenExpired) {
			span.SetStatus(codes.Error, "token expired")
			c.JSON(http.StatusUnauthorized, response.Error("TOKEN_EXPIRED", "Refresh token has expired"))
			return
		}
		if errors.Is(err, service.ErrUserInactive) {
			span.SetStatus(codes.Error, "user inactive")
			c.JSON(http.StatusForbidden, response.Error("USER_INACTIVE", "User account is inactive"))
			return
		}
		span.SetStatus(codes.Error, err.Error())
		c.JSON(http.StatusInternalServerError, response.InternalError(err.Error()))
		return
	}

	span.SetAttributes(attribute.String("user_id", result.User.ID))
	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusOK, response.Success(result))
}

// Logout handles user logout
// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.auth.logout")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid request body")
		c.JSON(http.StatusBadRequest, response.BadRequest(err.Error()))
		return
	}

	if err := h.authService.Logout(ctx, req.RefreshToken); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		c.JSON(http.StatusInternalServerError, response.InternalError(err.Error()))
		return
	}

	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusOK, response.Success(gin.H{"message": "Logged out successfully"}))
}

// LogoutAll handles logging out all sessions
// POST /api/v1/auth/logout-all
func (h *AuthHandler) LogoutAll(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.auth.logout_all")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	userID, exists := c.Get("user_id")
	if !exists {
		span.SetStatus(codes.Error, "user not authenticated")
		c.JSON(http.StatusUnauthorized, response.Unauthorized("User not authenticated"))
		return
	}

	span.SetAttributes(attribute.String("user_id", userID.(string)))

	if err := h.authService.LogoutAll(ctx, userID.(string)); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		c.JSON(http.StatusInternalServerError, response.InternalError(err.Error()))
		return
	}

	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusOK, response.Success(gin.H{"message": "All sessions logged out successfully"}))
}

// Me returns current user info
// GET /api/v1/auth/me
func (h *AuthHandler) Me(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.auth.me")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	userID, exists := c.Get("user_id")
	if !exists {
		span.SetStatus(codes.Error, "user not authenticated")
		c.JSON(http.StatusUnauthorized, response.Unauthorized("User not authenticated"))
		return
	}

	span.SetAttributes(attribute.String("user_id", userID.(string)))

	user, err := h.authService.GetUser(ctx, userID.(string))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		c.JSON(http.StatusInternalServerError, response.InternalError(err.Error()))
		return
	}
	if user == nil {
		span.SetStatus(codes.Error, "user not found")
		c.JSON(http.StatusNotFound, response.NotFound("User not found"))
		return
	}

	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusOK, response.Success(dto.UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Role:      string(user.Role),
		CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}))
}

// UpdateMe updates current user profile
// PUT /api/v1/auth/me
func (h *AuthHandler) UpdateMe(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.auth.update_me")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	userID, exists := c.Get("user_id")
	if !exists {
		span.SetStatus(codes.Error, "user not authenticated")
		c.JSON(http.StatusUnauthorized, response.Unauthorized("User not authenticated"))
		return
	}

	span.SetAttributes(attribute.String("user_id", userID.(string)))

	var req dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid request body")
		c.JSON(http.StatusBadRequest, response.BadRequest(err.Error()))
		return
	}

	// Validate update request
	if valid, msg := req.Validate(); !valid {
		span.SetStatus(codes.Error, "validation error")
		c.JSON(http.StatusBadRequest, response.Error("VALIDATION_ERROR", msg))
		return
	}

	user, err := h.authService.UpdateProfile(ctx, userID.(string), &req)
	if err != nil {
		span.RecordError(err)
		if errors.Is(err, service.ErrUserNotFound) {
			span.SetStatus(codes.Error, "user not found")
			c.JSON(http.StatusNotFound, response.NotFound("User not found"))
			return
		}
		span.SetStatus(codes.Error, err.Error())
		c.JSON(http.StatusInternalServerError, response.InternalError(err.Error()))
		return
	}

	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusOK, response.Success(dto.UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Role:      string(user.Role),
		CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}))
}

// ValidateToken validates a token (internal endpoint for other services)
// POST /api/v1/auth/validate
func (h *AuthHandler) ValidateToken(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.auth.validate_token")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		span.SetStatus(codes.Error, "missing token")
		c.JSON(http.StatusUnauthorized, response.Error("MISSING_TOKEN", "Authorization header is required"))
		return
	}

	// Extract token from "Bearer <token>"
	const bearerPrefix = "Bearer "
	if len(authHeader) <= len(bearerPrefix) {
		span.SetStatus(codes.Error, "invalid auth header format")
		c.JSON(http.StatusUnauthorized, response.Error("INVALID_TOKEN", "Invalid authorization header format"))
		return
	}
	token := authHeader[len(bearerPrefix):]

	claims, err := h.authService.ValidateToken(ctx, token)
	if err != nil {
		span.RecordError(err)
		if errors.Is(err, service.ErrTokenExpired) {
			span.SetStatus(codes.Error, "token expired")
			c.JSON(http.StatusUnauthorized, response.Error("TOKEN_EXPIRED", "Access token has expired"))
			return
		}
		span.SetStatus(codes.Error, "invalid token")
		c.JSON(http.StatusUnauthorized, response.Error("INVALID_TOKEN", "Invalid access token"))
		return
	}

	span.SetAttributes(attribute.String("user_id", claims.UserID))
	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusOK, response.Success(gin.H{
		"user_id":   claims.UserID,
		"email":     claims.Email,
		"role":      claims.Role,
		"tenant_id": claims.TenantID,
	}))
}

// GetStripeCustomerID returns the Stripe Customer ID for a user
// GET /api/v1/auth/users/:id/stripe-customer
func (h *AuthHandler) GetStripeCustomerID(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.auth.get_stripe_customer_id")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	userID := c.Param("id")
	if userID == "" {
		span.SetStatus(codes.Error, "user_id required")
		c.JSON(http.StatusBadRequest, response.BadRequest("user_id is required"))
		return
	}

	span.SetAttributes(attribute.String("user_id", userID))

	stripeCustomerID, err := h.authService.GetStripeCustomerID(ctx, userID)
	if err != nil {
		span.RecordError(err)
		if errors.Is(err, service.ErrUserNotFound) {
			span.SetStatus(codes.Error, "user not found")
			c.JSON(http.StatusNotFound, response.NotFound("User not found"))
			return
		}
		span.SetStatus(codes.Error, err.Error())
		c.JSON(http.StatusInternalServerError, response.InternalError(err.Error()))
		return
	}

	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusOK, response.Success(gin.H{
		"user_id":            userID,
		"stripe_customer_id": stripeCustomerID,
	}))
}

// UpdateStripeCustomerID updates the Stripe Customer ID for a user
// PUT /api/v1/auth/users/:id/stripe-customer
func (h *AuthHandler) UpdateStripeCustomerID(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.auth.update_stripe_customer_id")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	userID := c.Param("id")
	if userID == "" {
		span.SetStatus(codes.Error, "user_id required")
		c.JSON(http.StatusBadRequest, response.BadRequest("user_id is required"))
		return
	}

	span.SetAttributes(attribute.String("user_id", userID))

	var req struct {
		StripeCustomerID string `json:"stripe_customer_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid request body")
		c.JSON(http.StatusBadRequest, response.BadRequest(err.Error()))
		return
	}

	if err := h.authService.UpdateStripeCustomerID(ctx, userID, req.StripeCustomerID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		c.JSON(http.StatusInternalServerError, response.InternalError(err.Error()))
		return
	}

	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusOK, response.Success(gin.H{
		"user_id":            userID,
		"stripe_customer_id": req.StripeCustomerID,
	}))
}
