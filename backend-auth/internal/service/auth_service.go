package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-auth/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-auth/internal/dto"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-auth/internal/repository"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserInactive       = errors.New("user is inactive")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token expired")
	ErrSessionNotFound    = errors.New("session not found")
)

// AuthServiceConfig holds configuration for AuthService
type AuthServiceConfig struct {
	JWTSecret          string
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
	BcryptCost         int
}

// AuthService defines the interface for authentication operations
type AuthService interface {
	// Register registers a new user
	Register(ctx context.Context, req *dto.RegisterRequest) (*dto.AuthResponse, error)
	// Login authenticates a user
	Login(ctx context.Context, req *dto.LoginRequest, userAgent, ip string) (*dto.AuthResponse, error)
	// RefreshToken refreshes access token using refresh token
	RefreshToken(ctx context.Context, refreshToken string) (*dto.AuthResponse, error)
	// Logout logs out a user (invalidates session)
	Logout(ctx context.Context, refreshToken string) error
	// LogoutAll logs out all sessions for a user
	LogoutAll(ctx context.Context, userID string) error
	// ValidateToken validates an access token and returns claims
	ValidateToken(ctx context.Context, token string) (*domain.Claims, error)
	// GetUser retrieves user by ID
	GetUser(ctx context.Context, id string) (*domain.User, error)
	// UpdateProfile updates user profile
	UpdateProfile(ctx context.Context, userID string, req *dto.UpdateProfileRequest) (*domain.User, error)
	// GetStripeCustomerID retrieves the Stripe Customer ID for a user
	GetStripeCustomerID(ctx context.Context, userID string) (string, error)
	// UpdateStripeCustomerID updates the Stripe Customer ID for a user
	UpdateStripeCustomerID(ctx context.Context, userID, stripeCustomerID string) error
}

// authService implements AuthService
type authService struct {
	userRepo    repository.UserRepository
	sessionRepo repository.SessionRepository
	config      *AuthServiceConfig
}

// NewAuthService creates a new AuthService
func NewAuthService(
	userRepo repository.UserRepository,
	sessionRepo repository.SessionRepository,
	config *AuthServiceConfig,
) AuthService {
	if config.BcryptCost == 0 {
		config.BcryptCost = bcrypt.DefaultCost
	}
	if config.AccessTokenExpiry == 0 {
		config.AccessTokenExpiry = 15 * time.Minute
	}
	if config.RefreshTokenExpiry == 0 {
		config.RefreshTokenExpiry = 7 * 24 * time.Hour
	}
	return &authService{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		config:      config,
	}
}

// Register registers a new user
func (s *authService) Register(ctx context.Context, req *dto.RegisterRequest) (*dto.AuthResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "service.auth.register")
	defer span.End()

	span.SetAttributes(attribute.String("email", req.Email))

	// Check if user already exists
	exists, err := s.userRepo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if exists {
		span.SetStatus(codes.Error, "user already exists")
		return nil, ErrUserAlreadyExists
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), s.config.BcryptCost)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	// Create user
	now := time.Now()
	user := &domain.User{
		ID:           uuid.New().String(),
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Name:         req.Name,
		Role:         domain.RoleCustomer,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	// Generate tokens
	tokenPair, err := s.generateTokenPair(user)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(attribute.String("user_id", user.ID))
	span.SetStatus(codes.Ok, "")

	// Create session (not storing for registration - user needs to login)
	return &dto.AuthResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
		User:         s.toUserResponse(user),
	}, nil
}

// Login authenticates a user
func (s *authService) Login(ctx context.Context, req *dto.LoginRequest, userAgent, ip string) (*dto.AuthResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "service.auth.login")
	defer span.End()

	span.SetAttributes(attribute.String("email", req.Email))

	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if user == nil {
		span.SetStatus(codes.Error, "invalid credentials")
		return nil, ErrInvalidCredentials
	}

	// Check if user is active
	if !user.IsActive {
		span.SetStatus(codes.Error, "user inactive")
		return nil, ErrUserInactive
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		span.SetStatus(codes.Error, "invalid credentials")
		return nil, ErrInvalidCredentials
	}

	// Generate tokens
	tokenPair, err := s.generateTokenPair(user)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	// Create session
	session := &domain.Session{
		ID:           uuid.New().String(),
		UserID:       user.ID,
		RefreshToken: tokenPair.RefreshToken,
		UserAgent:    userAgent,
		IP:           ip,
		ExpiresAt:    time.Now().Add(s.config.RefreshTokenExpiry),
		CreatedAt:    time.Now(),
	}

	if err := s.sessionRepo.Create(ctx, session); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(attribute.String("user_id", user.ID))
	span.SetStatus(codes.Ok, "")

	return &dto.AuthResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
		User:         s.toUserResponse(user),
	}, nil
}

// RefreshToken refreshes access token using refresh token
func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*dto.AuthResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "service.auth.refresh_token")
	defer span.End()

	// Get session by refresh token
	session, err := s.sessionRepo.GetByRefreshToken(ctx, refreshToken)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if session == nil {
		span.SetStatus(codes.Error, "session not found")
		return nil, ErrSessionNotFound
	}

	span.SetAttributes(attribute.String("user_id", session.UserID))

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		// Delete expired session
		_ = s.sessionRepo.Delete(ctx, session.ID)
		span.SetStatus(codes.Error, "token expired")
		return nil, ErrTokenExpired
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, session.UserID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if user == nil {
		span.SetStatus(codes.Error, "user not found")
		return nil, ErrUserNotFound
	}
	if !user.IsActive {
		span.SetStatus(codes.Error, "user inactive")
		return nil, ErrUserInactive
	}

	// Generate new token pair
	tokenPair, err := s.generateTokenPair(user)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	// Delete old session first (before modifying the session object)
	_ = s.sessionRepo.Delete(ctx, session.ID)

	// Update session with new refresh token and create new session
	session.ID = uuid.New().String()
	session.RefreshToken = tokenPair.RefreshToken
	session.ExpiresAt = time.Now().Add(s.config.RefreshTokenExpiry)
	session.CreatedAt = time.Now()
	if err := s.sessionRepo.Create(ctx, session); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return &dto.AuthResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
		User:         s.toUserResponse(user),
	}, nil
}

// Logout logs out a user (invalidates session)
func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	ctx, span := telemetry.StartSpan(ctx, "service.auth.logout")
	defer span.End()

	session, err := s.sessionRepo.GetByRefreshToken(ctx, refreshToken)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	if session == nil {
		span.SetStatus(codes.Ok, "already logged out")
		return nil // Already logged out
	}

	span.SetAttributes(attribute.String("user_id", session.UserID))

	if err := s.sessionRepo.Delete(ctx, session.ID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// LogoutAll logs out all sessions for a user
func (s *authService) LogoutAll(ctx context.Context, userID string) error {
	ctx, span := telemetry.StartSpan(ctx, "service.auth.logout_all")
	defer span.End()

	span.SetAttributes(attribute.String("user_id", userID))

	if err := s.sessionRepo.DeleteByUserID(ctx, userID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// ValidateToken validates an access token and returns claims
func (s *authService) ValidateToken(ctx context.Context, tokenString string) (*domain.Claims, error) {
	ctx, span := telemetry.StartSpan(ctx, "service.auth.validate_token")
	defer span.End()

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(s.config.JWTSecret), nil
	})

	if err != nil {
		span.RecordError(err)
		if errors.Is(err, jwt.ErrTokenExpired) {
			span.SetStatus(codes.Error, "token expired")
			return nil, ErrTokenExpired
		}
		span.SetStatus(codes.Error, "invalid token")
		return nil, ErrInvalidToken
	}

	if !token.Valid {
		span.SetStatus(codes.Error, "invalid token")
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		span.SetStatus(codes.Error, "invalid claims")
		return nil, ErrInvalidToken
	}

	// Extract tenant_id safely (might be nil or empty)
	tenantID := ""
	if tid, ok := claims["tenant_id"].(string); ok {
		tenantID = tid
	}

	userID := claims["user_id"].(string)
	span.SetAttributes(attribute.String("user_id", userID))
	span.SetStatus(codes.Ok, "")

	return &domain.Claims{
		UserID:   userID,
		Email:    claims["email"].(string),
		Role:     domain.Role(claims["role"].(string)),
		TenantID: tenantID,
	}, nil
}

// GetUser retrieves user by ID
func (s *authService) GetUser(ctx context.Context, id string) (*domain.User, error) {
	ctx, span := telemetry.StartSpan(ctx, "service.auth.get_user")
	defer span.End()

	span.SetAttributes(attribute.String("user_id", id))

	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return user, nil
}

// UpdateProfile updates user profile
func (s *authService) UpdateProfile(ctx context.Context, userID string, req *dto.UpdateProfileRequest) (*domain.User, error) {
	ctx, span := telemetry.StartSpan(ctx, "service.auth.update_profile")
	defer span.End()

	span.SetAttributes(attribute.String("user_id", userID))

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if user == nil {
		span.SetStatus(codes.Error, "user not found")
		return nil, ErrUserNotFound
	}

	// Update fields
	if req.Name != "" {
		user.Name = req.Name
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return user, nil
}

// generateTokenPair generates access and refresh tokens
func (s *authService) generateTokenPair(user *domain.User) (*domain.TokenPair, error) {
	// Generate access token
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":       user.ID, // Standard JWT subject claim
		"user_id":   user.ID,
		"email":     user.Email,
		"role":      string(user.Role),
		"tenant_id": user.TenantID,
		"exp":       time.Now().Add(s.config.AccessTokenExpiry).Unix(),
		"iat":       time.Now().Unix(),
	})

	accessTokenString, err := accessToken.SignedString([]byte(s.config.JWTSecret))
	if err != nil {
		return nil, err
	}

	// Generate refresh token (random string)
	refreshTokenBytes := make([]byte, 32)
	if _, err := rand.Read(refreshTokenBytes); err != nil {
		return nil, err
	}
	refreshTokenString := base64.URLEncoding.EncodeToString(refreshTokenBytes)

	return &domain.TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresIn:    int64(s.config.AccessTokenExpiry.Seconds()),
	}, nil
}

// toUserResponse converts User to UserResponse
func (s *authService) toUserResponse(user *domain.User) dto.UserResponse {
	return dto.UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Role:      string(user.Role),
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	}
}

// GetStripeCustomerID retrieves the Stripe Customer ID for a user
func (s *authService) GetStripeCustomerID(ctx context.Context, userID string) (string, error) {
	ctx, span := telemetry.StartSpan(ctx, "service.auth.get_stripe_customer_id")
	defer span.End()

	span.SetAttributes(attribute.String("user_id", userID))

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}
	if user == nil {
		span.SetStatus(codes.Error, "user not found")
		return "", ErrUserNotFound
	}

	span.SetStatus(codes.Ok, "")
	return user.StripeCustomerID, nil
}

// UpdateStripeCustomerID updates the Stripe Customer ID for a user
func (s *authService) UpdateStripeCustomerID(ctx context.Context, userID, stripeCustomerID string) error {
	ctx, span := telemetry.StartSpan(ctx, "service.auth.update_stripe_customer_id")
	defer span.End()

	span.SetAttributes(attribute.String("user_id", userID))

	if err := s.userRepo.UpdateStripeCustomerID(ctx, userID, stripeCustomerID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}
