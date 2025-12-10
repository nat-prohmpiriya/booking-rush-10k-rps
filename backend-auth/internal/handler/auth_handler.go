package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-auth-service/internal/dto"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-auth-service/internal/service"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/response"
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
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest(err.Error()))
		return
	}

	// Validate email format
	if valid, msg := req.ValidateEmail(); !valid {
		c.JSON(http.StatusBadRequest, response.Error("INVALID_EMAIL", msg))
		return
	}

	// Validate password strength
	if valid, msg := req.ValidatePassword(); !valid {
		c.JSON(http.StatusBadRequest, response.Error("WEAK_PASSWORD", msg))
		return
	}

	result, err := h.authService.Register(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, service.ErrUserAlreadyExists) {
			c.JSON(http.StatusConflict, response.Error("USER_EXISTS", "User with this email already exists"))
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalError(err.Error()))
		return
	}

	c.JSON(http.StatusCreated, response.Success(result))
}

// Login handles user login
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest(err.Error()))
		return
	}

	userAgent := c.GetHeader("User-Agent")
	ip := c.ClientIP()

	result, err := h.authService.Login(c.Request.Context(), &req, userAgent, ip)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, response.Error("INVALID_CREDENTIALS", "Invalid email or password"))
			return
		}
		if errors.Is(err, service.ErrUserInactive) {
			c.JSON(http.StatusForbidden, response.Error("USER_INACTIVE", "User account is inactive"))
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.Success(result))
}

// RefreshToken handles token refresh
// POST /api/v1/auth/refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest(err.Error()))
		return
	}

	result, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, service.ErrSessionNotFound) {
			c.JSON(http.StatusUnauthorized, response.Error("INVALID_TOKEN", "Invalid or expired refresh token"))
			return
		}
		if errors.Is(err, service.ErrTokenExpired) {
			c.JSON(http.StatusUnauthorized, response.Error("TOKEN_EXPIRED", "Refresh token has expired"))
			return
		}
		if errors.Is(err, service.ErrUserInactive) {
			c.JSON(http.StatusForbidden, response.Error("USER_INACTIVE", "User account is inactive"))
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.Success(result))
}

// Logout handles user logout
// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest(err.Error()))
		return
	}

	if err := h.authService.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.Success(gin.H{"message": "Logged out successfully"}))
}

// LogoutAll handles logging out all sessions
// POST /api/v1/auth/logout-all
func (h *AuthHandler) LogoutAll(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("User not authenticated"))
		return
	}

	if err := h.authService.LogoutAll(c.Request.Context(), userID.(string)); err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.Success(gin.H{"message": "All sessions logged out successfully"}))
}

// Me returns current user info
// GET /api/v1/auth/me
func (h *AuthHandler) Me(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("User not authenticated"))
		return
	}

	user, err := h.authService.GetUser(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalError(err.Error()))
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, response.NotFound("User not found"))
		return
	}

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
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("User not authenticated"))
		return
	}

	var req dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest(err.Error()))
		return
	}

	// Validate update request
	if valid, msg := req.Validate(); !valid {
		c.JSON(http.StatusBadRequest, response.Error("VALIDATION_ERROR", msg))
		return
	}

	user, err := h.authService.UpdateProfile(c.Request.Context(), userID.(string), &req)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, response.NotFound("User not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalError(err.Error()))
		return
	}

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
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, response.Error("MISSING_TOKEN", "Authorization header is required"))
		return
	}

	// Extract token from "Bearer <token>"
	const bearerPrefix = "Bearer "
	if len(authHeader) <= len(bearerPrefix) {
		c.JSON(http.StatusUnauthorized, response.Error("INVALID_TOKEN", "Invalid authorization header format"))
		return
	}
	token := authHeader[len(bearerPrefix):]

	claims, err := h.authService.ValidateToken(c.Request.Context(), token)
	if err != nil {
		if errors.Is(err, service.ErrTokenExpired) {
			c.JSON(http.StatusUnauthorized, response.Error("TOKEN_EXPIRED", "Access token has expired"))
			return
		}
		c.JSON(http.StatusUnauthorized, response.Error("INVALID_TOKEN", "Invalid access token"))
		return
	}

	c.JSON(http.StatusOK, response.Success(gin.H{
		"user_id":   claims.UserID,
		"email":     claims.Email,
		"role":      claims.Role,
		"tenant_id": claims.TenantID,
	}))
}
