package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRequestID_GeneratesNew(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.Use(RequestID())
	r.GET("/test", func(c *gin.Context) {
		requestID := GetRequestID(c)
		c.String(http.StatusOK, requestID)
	})

	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, c.Request)

	// Check response header
	headerID := w.Header().Get(RequestIDHeader)
	if headerID == "" {
		t.Error("Expected X-Request-ID header to be set")
	}

	// Check response body (request ID returned)
	bodyID := w.Body.String()
	if bodyID == "" {
		t.Error("Expected request ID in body")
	}

	// Header and body should match
	if headerID != bodyID {
		t.Errorf("Header ID (%s) should match body ID (%s)", headerID, bodyID)
	}
}

func TestRequestID_UsesExisting(t *testing.T) {
	existingID := "existing-request-id-123"

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.Use(RequestID())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, GetRequestID(c))
	})

	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Request.Header.Set(RequestIDHeader, existingID)
	r.ServeHTTP(w, c.Request)

	// Should use existing ID
	if w.Body.String() != existingID {
		t.Errorf("Expected existing ID %s, got %s", existingID, w.Body.String())
	}
}

func TestCORS_Headers(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.Use(CORS())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Request.Header.Set("Origin", "http://example.com")
	r.ServeHTTP(w, c.Request)

	// Check CORS headers
	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("Expected Access-Control-Allow-Origin header")
	}
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("Expected Access-Control-Allow-Methods header")
	}
}

func TestCORS_Preflight(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.Use(CORS())
	r.OPTIONS("/test", func(c *gin.Context) {
		// This shouldn't be reached, CORS middleware handles OPTIONS
	})

	c.Request = httptest.NewRequest(http.MethodOptions, "/test", nil)
	c.Request.Header.Set("Origin", "http://example.com")
	r.ServeHTTP(w, c.Request)

	// Preflight should return 204
	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status %d for preflight, got %d", http.StatusNoContent, w.Code)
	}
}
