package service

import (
	"context"
	"testing"
	"time"

	"github.com/prohmpiriya/booking-rush-10k-rps/backend-auth/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-auth/internal/dto"
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

func TestAuthService_RefreshTokenRotation(t *testing.T) {
	userRepo := newMockUserRepository()
	sessionRepo := newMockSessionRepository()
	config := &AuthServiceConfig{
		JWTSecret:          "test-secret-key",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		BcryptCost:         10,
	}
	svc := NewAuthService(userRepo, sessionRepo, config)

	// Create user
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("Password1!"), 10)
	testUser := &domain.User{
		ID:           "rotation-user-id",
		Email:        "rotation@example.com",
		PasswordHash: string(hashedPassword),
		Name:         "Rotation Test",
		Role:         domain.RoleUser,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	userRepo.users[testUser.ID] = testUser
	userRepo.emailIndex[testUser.Email] = testUser

	t.Run("old refresh token is invalidated after rotation", func(t *testing.T) {
		// Login to get initial tokens
		loginReq := &dto.LoginRequest{
			Email:    "rotation@example.com",
			Password: "Password1!",
		}
		loginResp, err := svc.Login(context.Background(), loginReq, "Test-Agent", "127.0.0.1")
		if err != nil {
			t.Fatalf("Login() error = %v", err)
		}

		oldRefreshToken := loginResp.RefreshToken

		// Refresh token to get new tokens
		refreshResp, err := svc.RefreshToken(context.Background(), oldRefreshToken)
		if err != nil {
			t.Fatalf("RefreshToken() error = %v", err)
		}

		// Verify new refresh token is different
		if refreshResp.RefreshToken == oldRefreshToken {
			t.Error("RefreshToken() should return a different refresh token")
		}

		// Try to use old refresh token - should fail
		_, err = svc.RefreshToken(context.Background(), oldRefreshToken)
		if err != ErrSessionNotFound {
			t.Errorf("Using old refresh token should fail with ErrSessionNotFound, got %v", err)
		}

		// New refresh token should still work
		_, err = svc.RefreshToken(context.Background(), refreshResp.RefreshToken)
		if err != nil {
			t.Errorf("Using new refresh token should succeed, got error: %v", err)
		}
	})

	t.Run("refresh token generates valid access token", func(t *testing.T) {
		// Login
		loginReq := &dto.LoginRequest{
			Email:    "rotation@example.com",
			Password: "Password1!",
		}
		loginResp, err := svc.Login(context.Background(), loginReq, "Test-Agent", "127.0.0.1")
		if err != nil {
			t.Fatalf("Login() error = %v", err)
		}

		// Refresh
		refreshResp, err := svc.RefreshToken(context.Background(), loginResp.RefreshToken)
		if err != nil {
			t.Fatalf("RefreshToken() error = %v", err)
		}

		// Validate new access token
		claims, err := svc.ValidateToken(context.Background(), refreshResp.AccessToken)
		if err != nil {
			t.Fatalf("ValidateToken() error = %v", err)
		}

		if claims.UserID != testUser.ID {
			t.Errorf("ValidateToken() UserID = %v, want %v", claims.UserID, testUser.ID)
		}
		if claims.Email != testUser.Email {
			t.Errorf("ValidateToken() Email = %v, want %v", claims.Email, testUser.Email)
		}
	})
}

func TestAuthService_RefreshTokenWithInactiveUser(t *testing.T) {
	userRepo := newMockUserRepository()
	sessionRepo := newMockSessionRepository()
	config := &AuthServiceConfig{
		JWTSecret:          "test-secret-key",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		BcryptCost:         10,
	}
	svc := NewAuthService(userRepo, sessionRepo, config)

	// Create active user
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("Password1!"), 10)
	testUser := &domain.User{
		ID:           "deactivate-user-id",
		Email:        "deactivate@example.com",
		PasswordHash: string(hashedPassword),
		Name:         "Deactivate Test",
		Role:         domain.RoleUser,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	userRepo.users[testUser.ID] = testUser
	userRepo.emailIndex[testUser.Email] = testUser

	// Login while user is active
	loginReq := &dto.LoginRequest{
		Email:    "deactivate@example.com",
		Password: "Password1!",
	}
	loginResp, err := svc.Login(context.Background(), loginReq, "Test-Agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	// Deactivate user
	testUser.IsActive = false

	// Try to refresh token - should fail because user is inactive
	_, err = svc.RefreshToken(context.Background(), loginResp.RefreshToken)
	if err != ErrUserInactive {
		t.Errorf("RefreshToken() with inactive user error = %v, want %v", err, ErrUserInactive)
	}
}

func TestAuthService_RefreshTokenExpired(t *testing.T) {
	userRepo := newMockUserRepository()
	sessionRepo := newMockSessionRepository()
	config := &AuthServiceConfig{
		JWTSecret:          "test-secret-key",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		BcryptCost:         10,
	}
	svc := NewAuthService(userRepo, sessionRepo, config)

	// Create user
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("Password1!"), 10)
	testUser := &domain.User{
		ID:           "expired-user-id",
		Email:        "expired@example.com",
		PasswordHash: string(hashedPassword),
		Name:         "Expired Test",
		Role:         domain.RoleUser,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	userRepo.users[testUser.ID] = testUser
	userRepo.emailIndex[testUser.Email] = testUser

	// Create an expired session directly
	expiredSession := &domain.Session{
		ID:           "expired-session-id",
		UserID:       testUser.ID,
		RefreshToken: "expired-refresh-token",
		UserAgent:    "Test-Agent",
		IP:           "127.0.0.1",
		ExpiresAt:    time.Now().Add(-1 * time.Hour), // Expired 1 hour ago
		CreatedAt:    time.Now().Add(-8 * 24 * time.Hour),
	}
	sessionRepo.sessions[expiredSession.ID] = expiredSession
	sessionRepo.refreshTokenIndex[expiredSession.RefreshToken] = expiredSession

	// Try to refresh with expired token
	_, err := svc.RefreshToken(context.Background(), "expired-refresh-token")
	if err != ErrTokenExpired {
		t.Errorf("RefreshToken() with expired session error = %v, want %v", err, ErrTokenExpired)
	}

	// Verify session was deleted
	if _, exists := sessionRepo.sessions[expiredSession.ID]; exists {
		t.Error("Expired session should be deleted after refresh attempt")
	}
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

	t.Run("successful logout", func(t *testing.T) {
		// Login to get fresh token
		loginReq := &dto.LoginRequest{
			Email:    "logout@example.com",
			Password: "Password1!",
		}
		loginResp, err := svc.Login(context.Background(), loginReq, "Test-Agent", "127.0.0.1")
		if err != nil {
			t.Fatalf("Login() error = %v", err)
		}

		err = svc.Logout(context.Background(), loginResp.RefreshToken)
		if err != nil {
			t.Fatalf("Logout() error = %v", err)
		}

		// Try to refresh with the old token - should fail
		_, err = svc.RefreshToken(context.Background(), loginResp.RefreshToken)
		if err != ErrSessionNotFound {
			t.Errorf("After logout, RefreshToken() error = %v, want %v", err, ErrSessionNotFound)
		}
	})

	t.Run("logout with invalid token does not error", func(t *testing.T) {
		err := svc.Logout(context.Background(), "non-existent-refresh-token")
		if err != nil {
			t.Errorf("Logout() with invalid token should not error, got %v", err)
		}
	})

	t.Run("logout twice does not error", func(t *testing.T) {
		// Login to get fresh token
		loginReq := &dto.LoginRequest{
			Email:    "logout@example.com",
			Password: "Password1!",
		}
		loginResp, err := svc.Login(context.Background(), loginReq, "Test-Agent", "127.0.0.1")
		if err != nil {
			t.Fatalf("Login() error = %v", err)
		}

		// First logout
		err = svc.Logout(context.Background(), loginResp.RefreshToken)
		if err != nil {
			t.Fatalf("First Logout() error = %v", err)
		}

		// Second logout with same token should not error
		err = svc.Logout(context.Background(), loginResp.RefreshToken)
		if err != nil {
			t.Errorf("Second Logout() should not error, got %v", err)
		}
	})
}

func TestAuthService_LogoutAll(t *testing.T) {
	userRepo := newMockUserRepository()
	sessionRepo := newMockSessionRepository()
	config := &AuthServiceConfig{
		JWTSecret:          "test-secret-key",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		BcryptCost:         10,
	}
	svc := NewAuthService(userRepo, sessionRepo, config)

	// Create user
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("Password1!"), 10)
	testUser := &domain.User{
		ID:           "logoutall-user-id",
		Email:        "logoutall@example.com",
		PasswordHash: string(hashedPassword),
		Name:         "LogoutAll Test",
		Role:         domain.RoleUser,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	userRepo.users[testUser.ID] = testUser
	userRepo.emailIndex[testUser.Email] = testUser

	t.Run("logout all sessions", func(t *testing.T) {
		loginReq := &dto.LoginRequest{
			Email:    "logoutall@example.com",
			Password: "Password1!",
		}

		// Create multiple sessions
		session1, err := svc.Login(context.Background(), loginReq, "Chrome", "192.168.1.1")
		if err != nil {
			t.Fatalf("Login 1 error = %v", err)
		}
		session2, err := svc.Login(context.Background(), loginReq, "Firefox", "192.168.1.2")
		if err != nil {
			t.Fatalf("Login 2 error = %v", err)
		}
		session3, err := svc.Login(context.Background(), loginReq, "Safari", "192.168.1.3")
		if err != nil {
			t.Fatalf("Login 3 error = %v", err)
		}

		// Verify all sessions exist
		sessions, _ := sessionRepo.GetByUserID(context.Background(), testUser.ID)
		if len(sessions) != 3 {
			t.Fatalf("Expected 3 sessions, got %d", len(sessions))
		}

		// LogoutAll
		err = svc.LogoutAll(context.Background(), testUser.ID)
		if err != nil {
			t.Fatalf("LogoutAll() error = %v", err)
		}

		// Verify all sessions are deleted
		sessions, _ = sessionRepo.GetByUserID(context.Background(), testUser.ID)
		if len(sessions) != 0 {
			t.Errorf("Expected 0 sessions after LogoutAll, got %d", len(sessions))
		}

		// All refresh tokens should be invalid
		_, err = svc.RefreshToken(context.Background(), session1.RefreshToken)
		if err != ErrSessionNotFound {
			t.Errorf("Session 1 RefreshToken() error = %v, want %v", err, ErrSessionNotFound)
		}
		_, err = svc.RefreshToken(context.Background(), session2.RefreshToken)
		if err != ErrSessionNotFound {
			t.Errorf("Session 2 RefreshToken() error = %v, want %v", err, ErrSessionNotFound)
		}
		_, err = svc.RefreshToken(context.Background(), session3.RefreshToken)
		if err != ErrSessionNotFound {
			t.Errorf("Session 3 RefreshToken() error = %v, want %v", err, ErrSessionNotFound)
		}
	})

	t.Run("logout all with no sessions does not error", func(t *testing.T) {
		err := svc.LogoutAll(context.Background(), "non-existent-user-id")
		if err != nil {
			t.Errorf("LogoutAll() with no sessions should not error, got %v", err)
		}
	})
}

func TestJWTClaimsContainTenantID(t *testing.T) {
	userRepo := newMockUserRepository()
	sessionRepo := newMockSessionRepository()
	config := &AuthServiceConfig{
		JWTSecret:          "test-secret-key",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		BcryptCost:         10,
	}
	svc := NewAuthService(userRepo, sessionRepo, config)

	// Create user with tenant_id
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("Password1!"), 10)
	testUser := &domain.User{
		ID:           "tenant-user-id",
		Email:        "tenant@example.com",
		PasswordHash: string(hashedPassword),
		Name:         "Tenant Test",
		Role:         domain.RoleUser,
		TenantID:     "tenant-123",
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	userRepo.users[testUser.ID] = testUser
	userRepo.emailIndex[testUser.Email] = testUser

	// Login to get token with tenant_id
	loginReq := &dto.LoginRequest{
		Email:    "tenant@example.com",
		Password: "Password1!",
	}
	loginResp, err := svc.Login(context.Background(), loginReq, "Test-Agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	// Validate token and check tenant_id
	claims, err := svc.ValidateToken(context.Background(), loginResp.AccessToken)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}

	if claims.TenantID != "tenant-123" {
		t.Errorf("ValidateToken() TenantID = %v, want tenant-123", claims.TenantID)
	}
	if claims.UserID != testUser.ID {
		t.Errorf("ValidateToken() UserID = %v, want %v", claims.UserID, testUser.ID)
	}
	if claims.Email != testUser.Email {
		t.Errorf("ValidateToken() Email = %v, want %v", claims.Email, testUser.Email)
	}
	if claims.Role != domain.RoleUser {
		t.Errorf("ValidateToken() Role = %v, want %v", claims.Role, domain.RoleUser)
	}
}

func TestTokenExpiry(t *testing.T) {
	userRepo := newMockUserRepository()
	sessionRepo := newMockSessionRepository()
	config := &AuthServiceConfig{
		JWTSecret:          "test-secret-key",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		BcryptCost:         10,
	}
	svc := NewAuthService(userRepo, sessionRepo, config)

	t.Run("access token expires in 15 minutes", func(t *testing.T) {
		regReq := &dto.RegisterRequest{
			Email:    "expiry@example.com",
			Password: "Password1!",
			Name:     "Expiry Test",
		}
		resp, err := svc.Register(context.Background(), regReq)
		if err != nil {
			t.Fatalf("Register() error = %v", err)
		}

		// ExpiresIn should be around 15 minutes (900 seconds)
		expectedExpiry := int64(15 * 60) // 900 seconds
		if resp.ExpiresIn != expectedExpiry {
			t.Errorf("ExpiresIn = %d, want %d", resp.ExpiresIn, expectedExpiry)
		}
	})

	t.Run("refresh token expires in 7 days", func(t *testing.T) {
		// Create user and login to get session
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("Password1!"), 10)
		testUser := &domain.User{
			ID:           "expiry-user-id",
			Email:        "expiry2@example.com",
			PasswordHash: string(hashedPassword),
			Name:         "Expiry Test 2",
			Role:         domain.RoleUser,
			IsActive:     true,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		userRepo.users[testUser.ID] = testUser
		userRepo.emailIndex[testUser.Email] = testUser

		loginReq := &dto.LoginRequest{
			Email:    "expiry2@example.com",
			Password: "Password1!",
		}
		_, err := svc.Login(context.Background(), loginReq, "Test-Agent", "127.0.0.1")
		if err != nil {
			t.Fatalf("Login() error = %v", err)
		}

		// Check session expiry is around 7 days from now
		sessions := sessionRepo.userSessions[testUser.ID]
		if len(sessions) == 0 {
			t.Fatal("No sessions created")
		}

		session := sessions[0]
		expectedExpiry := time.Now().Add(7 * 24 * time.Hour)
		diff := session.ExpiresAt.Sub(expectedExpiry)
		if diff < -time.Second || diff > time.Second {
			t.Errorf("Session ExpiresAt = %v, expected around %v", session.ExpiresAt, expectedExpiry)
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

func TestAuthService_GetUser(t *testing.T) {
	userRepo := newMockUserRepository()
	sessionRepo := newMockSessionRepository()
	config := &AuthServiceConfig{
		JWTSecret:          "test-secret-key",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		BcryptCost:         10,
	}
	svc := NewAuthService(userRepo, sessionRepo, config)

	// Create user
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("Password1!"), 10)
	testUser := &domain.User{
		ID:           "getuser-test-id",
		Email:        "getuser@example.com",
		PasswordHash: string(hashedPassword),
		Name:         "GetUser Test",
		Role:         domain.RoleUser,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	userRepo.users[testUser.ID] = testUser
	userRepo.emailIndex[testUser.Email] = testUser

	t.Run("get existing user", func(t *testing.T) {
		user, err := svc.GetUser(context.Background(), testUser.ID)
		if err != nil {
			t.Fatalf("GetUser() error = %v", err)
		}
		if user == nil {
			t.Fatal("GetUser() returned nil user")
		}
		if user.ID != testUser.ID {
			t.Errorf("GetUser() ID = %v, want %v", user.ID, testUser.ID)
		}
		if user.Email != testUser.Email {
			t.Errorf("GetUser() Email = %v, want %v", user.Email, testUser.Email)
		}
		if user.Name != testUser.Name {
			t.Errorf("GetUser() Name = %v, want %v", user.Name, testUser.Name)
		}
	})

	t.Run("get non-existent user", func(t *testing.T) {
		user, err := svc.GetUser(context.Background(), "non-existent-id")
		if err != nil {
			t.Fatalf("GetUser() error = %v", err)
		}
		if user != nil {
			t.Error("GetUser() should return nil for non-existent user")
		}
	})
}

func TestAuthService_UpdateProfile(t *testing.T) {
	userRepo := newMockUserRepository()
	sessionRepo := newMockSessionRepository()
	config := &AuthServiceConfig{
		JWTSecret:          "test-secret-key",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		BcryptCost:         10,
	}
	svc := NewAuthService(userRepo, sessionRepo, config)

	// Create user
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("Password1!"), 10)
	testUser := &domain.User{
		ID:           "updateprofile-test-id",
		Email:        "updateprofile@example.com",
		PasswordHash: string(hashedPassword),
		Name:         "Original Name",
		Role:         domain.RoleUser,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	userRepo.users[testUser.ID] = testUser
	userRepo.emailIndex[testUser.Email] = testUser

	t.Run("update name successfully", func(t *testing.T) {
		req := &dto.UpdateProfileRequest{
			Name: "Updated Name",
		}
		user, err := svc.UpdateProfile(context.Background(), testUser.ID, req)
		if err != nil {
			t.Fatalf("UpdateProfile() error = %v", err)
		}
		if user.Name != "Updated Name" {
			t.Errorf("UpdateProfile() Name = %v, want 'Updated Name'", user.Name)
		}
		// Verify the change persisted in repository
		if userRepo.users[testUser.ID].Name != "Updated Name" {
			t.Error("UpdateProfile() did not persist name change")
		}
	})

	t.Run("update non-existent user", func(t *testing.T) {
		req := &dto.UpdateProfileRequest{
			Name: "New Name",
		}
		_, err := svc.UpdateProfile(context.Background(), "non-existent-id", req)
		if err != ErrUserNotFound {
			t.Errorf("UpdateProfile() error = %v, want %v", err, ErrUserNotFound)
		}
	})

	t.Run("empty name does not change existing name", func(t *testing.T) {
		// Reset user name
		userRepo.users[testUser.ID].Name = "Keep This Name"

		req := &dto.UpdateProfileRequest{
			Name: "",
		}
		user, err := svc.UpdateProfile(context.Background(), testUser.ID, req)
		if err != nil {
			t.Fatalf("UpdateProfile() error = %v", err)
		}
		if user.Name != "Keep This Name" {
			t.Errorf("UpdateProfile() should not change name when empty, got %v", user.Name)
		}
	})
}
