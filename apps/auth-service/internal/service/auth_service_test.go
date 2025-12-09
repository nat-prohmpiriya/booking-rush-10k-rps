package service

import (
	"context"
	"testing"
	"time"

	"github.com/prohmpiriya/booking-rush-10k-rps/apps/auth-service/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/apps/auth-service/internal/dto"
	"golang.org/x/crypto/bcrypt"
)

// mockUserRepository is a mock implementation of UserRepository
type mockUserRepository struct {
	users       map[string]*domain.User
	emailIndex  map[string]*domain.User
	createError error
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		users:      make(map[string]*domain.User),
		emailIndex: make(map[string]*domain.User),
	}
}

func (r *mockUserRepository) Create(ctx context.Context, user *domain.User) error {
	if r.createError != nil {
		return r.createError
	}
	r.users[user.ID] = user
	r.emailIndex[user.Email] = user
	return nil
}

func (r *mockUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return r.users[id], nil
}

func (r *mockUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return r.emailIndex[email], nil
}

func (r *mockUserRepository) Update(ctx context.Context, user *domain.User) error {
	r.users[user.ID] = user
	r.emailIndex[user.Email] = user
	return nil
}

func (r *mockUserRepository) Delete(ctx context.Context, id string) error {
	user := r.users[id]
	if user != nil {
		delete(r.emailIndex, user.Email)
		delete(r.users, id)
	}
	return nil
}

func (r *mockUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	_, exists := r.emailIndex[email]
	return exists, nil
}

// mockSessionRepository is a mock implementation of SessionRepository
type mockSessionRepository struct {
	sessions          map[string]*domain.Session
	refreshTokenIndex map[string]*domain.Session
	userSessions      map[string][]*domain.Session
}

func newMockSessionRepository() *mockSessionRepository {
	return &mockSessionRepository{
		sessions:          make(map[string]*domain.Session),
		refreshTokenIndex: make(map[string]*domain.Session),
		userSessions:      make(map[string][]*domain.Session),
	}
}

func (r *mockSessionRepository) Create(ctx context.Context, session *domain.Session) error {
	r.sessions[session.ID] = session
	r.refreshTokenIndex[session.RefreshToken] = session
	r.userSessions[session.UserID] = append(r.userSessions[session.UserID], session)
	return nil
}

func (r *mockSessionRepository) GetByID(ctx context.Context, id string) (*domain.Session, error) {
	return r.sessions[id], nil
}

func (r *mockSessionRepository) GetByRefreshToken(ctx context.Context, token string) (*domain.Session, error) {
	return r.refreshTokenIndex[token], nil
}

func (r *mockSessionRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.Session, error) {
	return r.userSessions[userID], nil
}

func (r *mockSessionRepository) Delete(ctx context.Context, id string) error {
	session := r.sessions[id]
	if session != nil {
		delete(r.refreshTokenIndex, session.RefreshToken)
		delete(r.sessions, id)
	}
	return nil
}

func (r *mockSessionRepository) DeleteByUserID(ctx context.Context, userID string) error {
	for _, session := range r.userSessions[userID] {
		delete(r.refreshTokenIndex, session.RefreshToken)
		delete(r.sessions, session.ID)
	}
	delete(r.userSessions, userID)
	return nil
}

func (r *mockSessionRepository) DeleteExpired(ctx context.Context) error {
	for id, session := range r.sessions {
		if time.Now().After(session.ExpiresAt) {
			delete(r.refreshTokenIndex, session.RefreshToken)
			delete(r.sessions, id)
		}
	}
	return nil
}

func TestAuthService_Register(t *testing.T) {
	userRepo := newMockUserRepository()
	sessionRepo := newMockSessionRepository()
	config := &AuthServiceConfig{
		JWTSecret:          "test-secret-key",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		BcryptCost:         12,
	}
	svc := NewAuthService(userRepo, sessionRepo, config)

	t.Run("successful registration", func(t *testing.T) {
		req := &dto.RegisterRequest{
			Email:    "test@example.com",
			Password: "Password1!",
			Name:     "Test User",
		}

		resp, err := svc.Register(context.Background(), req)
		if err != nil {
			t.Fatalf("Register() error = %v", err)
		}

		if resp.AccessToken == "" {
			t.Error("Register() AccessToken is empty")
		}
		if resp.RefreshToken == "" {
			t.Error("Register() RefreshToken is empty")
		}
		if resp.User.Email != req.Email {
			t.Errorf("Register() User.Email = %v, want %v", resp.User.Email, req.Email)
		}
		if resp.User.Name != req.Name {
			t.Errorf("Register() User.Name = %v, want %v", resp.User.Name, req.Name)
		}
		if resp.User.Role != "user" {
			t.Errorf("Register() User.Role = %v, want user", resp.User.Role)
		}
	})

	t.Run("duplicate email", func(t *testing.T) {
		req := &dto.RegisterRequest{
			Email:    "test@example.com", // Same email as previous test
			Password: "Password2!",
			Name:     "Another User",
		}

		_, err := svc.Register(context.Background(), req)
		if err != ErrUserAlreadyExists {
			t.Errorf("Register() error = %v, want %v", err, ErrUserAlreadyExists)
		}
	})
}

func TestAuthService_Login(t *testing.T) {
	userRepo := newMockUserRepository()
	sessionRepo := newMockSessionRepository()
	config := &AuthServiceConfig{
		JWTSecret:          "test-secret-key",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		BcryptCost:         10, // Lower cost for faster tests
	}
	svc := NewAuthService(userRepo, sessionRepo, config)

	// Create a test user
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("Password1!"), 10)
	testUser := &domain.User{
		ID:           "test-user-id",
		Email:        "login@example.com",
		PasswordHash: string(hashedPassword),
		Name:         "Login Test",
		Role:         domain.RoleUser,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	userRepo.users[testUser.ID] = testUser
	userRepo.emailIndex[testUser.Email] = testUser

	t.Run("successful login", func(t *testing.T) {
		req := &dto.LoginRequest{
			Email:    "login@example.com",
			Password: "Password1!",
		}

		resp, err := svc.Login(context.Background(), req, "Test-Agent", "127.0.0.1")
		if err != nil {
			t.Fatalf("Login() error = %v", err)
		}

		if resp.AccessToken == "" {
			t.Error("Login() AccessToken is empty")
		}
		if resp.RefreshToken == "" {
			t.Error("Login() RefreshToken is empty")
		}
		if resp.User.Email != req.Email {
			t.Errorf("Login() User.Email = %v, want %v", resp.User.Email, req.Email)
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		req := &dto.LoginRequest{
			Email:    "login@example.com",
			Password: "WrongPassword1!",
		}

		_, err := svc.Login(context.Background(), req, "Test-Agent", "127.0.0.1")
		if err != ErrInvalidCredentials {
			t.Errorf("Login() error = %v, want %v", err, ErrInvalidCredentials)
		}
	})

	t.Run("non-existent user", func(t *testing.T) {
		req := &dto.LoginRequest{
			Email:    "nonexistent@example.com",
			Password: "Password1!",
		}

		_, err := svc.Login(context.Background(), req, "Test-Agent", "127.0.0.1")
		if err != ErrInvalidCredentials {
			t.Errorf("Login() error = %v, want %v", err, ErrInvalidCredentials)
		}
	})

	t.Run("inactive user", func(t *testing.T) {
		// Create inactive user
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("Password1!"), 10)
		inactiveUser := &domain.User{
			ID:           "inactive-user-id",
			Email:        "inactive@example.com",
			PasswordHash: string(hashedPassword),
			Name:         "Inactive User",
			Role:         domain.RoleUser,
			IsActive:     false,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		userRepo.users[inactiveUser.ID] = inactiveUser
		userRepo.emailIndex[inactiveUser.Email] = inactiveUser

		req := &dto.LoginRequest{
			Email:    "inactive@example.com",
			Password: "Password1!",
		}

		_, err := svc.Login(context.Background(), req, "Test-Agent", "127.0.0.1")
		if err != ErrUserInactive {
			t.Errorf("Login() error = %v, want %v", err, ErrUserInactive)
		}
	})
}

func TestAuthService_ValidateToken(t *testing.T) {
	userRepo := newMockUserRepository()
	sessionRepo := newMockSessionRepository()
	config := &AuthServiceConfig{
		JWTSecret:          "test-secret-key",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		BcryptCost:         10,
	}
	svc := NewAuthService(userRepo, sessionRepo, config)

	// Create a test user and register to get a valid token
	regReq := &dto.RegisterRequest{
		Email:    "validate@example.com",
		Password: "Password1!",
		Name:     "Validate Test",
	}
	regResp, err := svc.Register(context.Background(), regReq)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	t.Run("valid token", func(t *testing.T) {
		claims, err := svc.ValidateToken(context.Background(), regResp.AccessToken)
		if err != nil {
			t.Fatalf("ValidateToken() error = %v", err)
		}

		if claims.Email != regReq.Email {
			t.Errorf("ValidateToken() Email = %v, want %v", claims.Email, regReq.Email)
		}
		if claims.Role != domain.RoleUser {
			t.Errorf("ValidateToken() Role = %v, want %v", claims.Role, domain.RoleUser)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		_, err := svc.ValidateToken(context.Background(), "invalid-token")
		if err != ErrInvalidToken {
			t.Errorf("ValidateToken() error = %v, want %v", err, ErrInvalidToken)
		}
	})

	t.Run("tampered token", func(t *testing.T) {
		// Modify a character in the token
		tamperedToken := regResp.AccessToken[:len(regResp.AccessToken)-1] + "X"
		_, err := svc.ValidateToken(context.Background(), tamperedToken)
		if err != ErrInvalidToken {
			t.Errorf("ValidateToken() error = %v, want %v", err, ErrInvalidToken)
		}
	})
}

func TestAuthService_RefreshToken(t *testing.T) {
	userRepo := newMockUserRepository()
	sessionRepo := newMockSessionRepository()
	config := &AuthServiceConfig{
		JWTSecret:          "test-secret-key",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		BcryptCost:         10,
	}
	svc := NewAuthService(userRepo, sessionRepo, config)

	// Create user and login to get refresh token
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("Password1!"), 10)
	testUser := &domain.User{
		ID:           "refresh-user-id",
		Email:        "refresh@example.com",
		PasswordHash: string(hashedPassword),
		Name:         "Refresh Test",
		Role:         domain.RoleUser,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	userRepo.users[testUser.ID] = testUser
	userRepo.emailIndex[testUser.Email] = testUser

	loginReq := &dto.LoginRequest{
		Email:    "refresh@example.com",
		Password: "Password1!",
	}
	loginResp, err := svc.Login(context.Background(), loginReq, "Test-Agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	t.Run("successful refresh", func(t *testing.T) {
		resp, err := svc.RefreshToken(context.Background(), loginResp.RefreshToken)
		if err != nil {
			t.Fatalf("RefreshToken() error = %v", err)
		}

		if resp.AccessToken == "" {
			t.Error("RefreshToken() AccessToken is empty")
		}
		if resp.RefreshToken == "" {
			t.Error("RefreshToken() new RefreshToken is empty")
		}
		if resp.RefreshToken == loginResp.RefreshToken {
			t.Error("RefreshToken() should return a new refresh token")
		}
	})

	t.Run("invalid refresh token", func(t *testing.T) {
		_, err := svc.RefreshToken(context.Background(), "invalid-refresh-token")
		if err != ErrSessionNotFound {
			t.Errorf("RefreshToken() error = %v, want %v", err, ErrSessionNotFound)
		}
	})
}

func TestAuthService_Logout(t *testing.T) {
	userRepo := newMockUserRepository()
	sessionRepo := newMockSessionRepository()
	config := &AuthServiceConfig{
		JWTSecret:          "test-secret-key",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		BcryptCost:         10,
	}
	svc := NewAuthService(userRepo, sessionRepo, config)

	// Create user and login
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("Password1!"), 10)
	testUser := &domain.User{
		ID:           "logout-user-id",
		Email:        "logout@example.com",
		PasswordHash: string(hashedPassword),
		Name:         "Logout Test",
		Role:         domain.RoleUser,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	userRepo.users[testUser.ID] = testUser
	userRepo.emailIndex[testUser.Email] = testUser

	loginReq := &dto.LoginRequest{
		Email:    "logout@example.com",
		Password: "Password1!",
	}
	loginResp, err := svc.Login(context.Background(), loginReq, "Test-Agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	t.Run("successful logout", func(t *testing.T) {
		err := svc.Logout(context.Background(), loginResp.RefreshToken)
		if err != nil {
			t.Fatalf("Logout() error = %v", err)
		}

		// Try to refresh with the old token - should fail
		_, err = svc.RefreshToken(context.Background(), loginResp.RefreshToken)
		if err != ErrSessionNotFound {
			t.Errorf("After logout, RefreshToken() error = %v, want %v", err, ErrSessionNotFound)
		}
	})
}

func TestBcryptCost(t *testing.T) {
	userRepo := newMockUserRepository()
	sessionRepo := newMockSessionRepository()
	config := &AuthServiceConfig{
		JWTSecret:          "test-secret-key",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		BcryptCost:         12, // Verify cost 12 is used
	}
	svc := NewAuthService(userRepo, sessionRepo, config)

	req := &dto.RegisterRequest{
		Email:    "bcrypt@example.com",
		Password: "Password1!",
		Name:     "Bcrypt Test",
	}

	_, err := svc.Register(context.Background(), req)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Get the created user and check bcrypt cost
	user := userRepo.emailIndex[req.Email]
	cost, err := bcrypt.Cost([]byte(user.PasswordHash))
	if err != nil {
		t.Fatalf("bcrypt.Cost() error = %v", err)
	}
	if cost != 12 {
		t.Errorf("bcrypt cost = %d, want 12", cost)
	}
}
