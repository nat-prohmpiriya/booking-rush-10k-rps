package repository

import (
	"context"

	"github.com/prohmpiriya/booking-rush-10k-rps/apps/auth-service/internal/domain"
)

// UserRepository defines the interface for user data access
type UserRepository interface {
	// Create creates a new user
	Create(ctx context.Context, user *domain.User) error
	// GetByID retrieves a user by ID
	GetByID(ctx context.Context, id string) (*domain.User, error)
	// GetByEmail retrieves a user by email
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	// Update updates a user
	Update(ctx context.Context, user *domain.User) error
	// Delete deletes a user
	Delete(ctx context.Context, id string) error
	// ExistsByEmail checks if a user exists with the given email
	ExistsByEmail(ctx context.Context, email string) (bool, error)
}

// SessionRepository defines the interface for session data access
type SessionRepository interface {
	// Create creates a new session
	Create(ctx context.Context, session *domain.Session) error
	// GetByID retrieves a session by ID
	GetByID(ctx context.Context, id string) (*domain.Session, error)
	// GetByRefreshToken retrieves a session by refresh token
	GetByRefreshToken(ctx context.Context, token string) (*domain.Session, error)
	// GetByUserID retrieves all sessions for a user
	GetByUserID(ctx context.Context, userID string) ([]*domain.Session, error)
	// Delete deletes a session
	Delete(ctx context.Context, id string) error
	// DeleteByUserID deletes all sessions for a user
	DeleteByUserID(ctx context.Context, userID string) error
	// DeleteExpired deletes all expired sessions
	DeleteExpired(ctx context.Context) error
}
