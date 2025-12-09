package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/prohmpiriya/booking-rush-10k-rps/apps/auth-service/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/apps/auth-service/internal/dto"
	"github.com/prohmpiriya/booking-rush-10k-rps/apps/auth-service/internal/repository"
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
	JWTSecret           string
	AccessTokenExpiry   time.Duration
	RefreshTokenExpiry  time.Duration
	BcryptCost          int
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
	// Check if user already exists
	exists, err := s.userRepo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUserAlreadyExists
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), s.config.BcryptCost)
	if err != nil {
		return nil, err
	}

	// Create user
	now := time.Now()
	user := &domain.User{
		ID:           uuid.New().String(),
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Name:         req.Name,
		Role:         domain.RoleUser,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	// Generate tokens
	tokenPair, err := s.generateTokenPair(user)
	if err != nil {
		return nil, err
	}

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
	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	// Check if user is active
	if !user.IsActive {
		return nil, ErrUserInactive
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Generate tokens
	tokenPair, err := s.generateTokenPair(user)
	if err != nil {
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
		return nil, err
	}

	return &dto.AuthResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
		User:         s.toUserResponse(user),
	}, nil
}

// RefreshToken refreshes access token using refresh token
func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*dto.AuthResponse, error) {
	// Get session by refresh token
	session, err := s.sessionRepo.GetByRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, ErrSessionNotFound
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		// Delete expired session
		_ = s.sessionRepo.Delete(ctx, session.ID)
		return nil, ErrTokenExpired
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, session.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	if !user.IsActive {
		return nil, ErrUserInactive
	}

	// Generate new token pair
	tokenPair, err := s.generateTokenPair(user)
	if err != nil {
		return nil, err
	}

	// Update session with new refresh token
	session.RefreshToken = tokenPair.RefreshToken
	session.ExpiresAt = time.Now().Add(s.config.RefreshTokenExpiry)

	// Delete old session and create new one (atomic operation not needed for this use case)
	_ = s.sessionRepo.Delete(ctx, session.ID)
	session.ID = uuid.New().String()
	session.CreatedAt = time.Now()
	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, err
	}

	return &dto.AuthResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
		User:         s.toUserResponse(user),
	}, nil
}

// Logout logs out a user (invalidates session)
func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	session, err := s.sessionRepo.GetByRefreshToken(ctx, refreshToken)
	if err != nil {
		return err
	}
	if session == nil {
		return nil // Already logged out
	}
	return s.sessionRepo.Delete(ctx, session.ID)
}

// LogoutAll logs out all sessions for a user
func (s *authService) LogoutAll(ctx context.Context, userID string) error {
	return s.sessionRepo.DeleteByUserID(ctx, userID)
}

// ValidateToken validates an access token and returns claims
func (s *authService) ValidateToken(ctx context.Context, tokenString string) (*domain.Claims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(s.config.JWTSecret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	return &domain.Claims{
		UserID: claims["user_id"].(string),
		Email:  claims["email"].(string),
		Role:   domain.Role(claims["role"].(string)),
	}, nil
}

// GetUser retrieves user by ID
func (s *authService) GetUser(ctx context.Context, id string) (*domain.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

// generateTokenPair generates access and refresh tokens
func (s *authService) generateTokenPair(user *domain.User) (*domain.TokenPair, error) {
	// Generate access token
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"role":    string(user.Role),
		"exp":     time.Now().Add(s.config.AccessTokenExpiry).Unix(),
		"iat":     time.Now().Unix(),
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
