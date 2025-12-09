package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prohmpiriya/booking-rush-10k-rps/apps/auth-service/internal/domain"
)

// PostgresSessionRepository implements SessionRepository using PostgreSQL
type PostgresSessionRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresSessionRepository creates a new PostgresSessionRepository
func NewPostgresSessionRepository(pool *pgxpool.Pool) *PostgresSessionRepository {
	return &PostgresSessionRepository{pool: pool}
}

// Create creates a new session
func (r *PostgresSessionRepository) Create(ctx context.Context, session *domain.Session) error {
	query := `
		INSERT INTO sessions (id, user_id, refresh_token, user_agent, ip, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.pool.Exec(ctx, query,
		session.ID,
		session.UserID,
		session.RefreshToken,
		session.UserAgent,
		session.IP,
		session.ExpiresAt,
		session.CreatedAt,
	)
	return err
}

// GetByID retrieves a session by ID
func (r *PostgresSessionRepository) GetByID(ctx context.Context, id string) (*domain.Session, error) {
	query := `
		SELECT id, user_id, refresh_token, user_agent, ip, expires_at, created_at
		FROM sessions
		WHERE id = $1
	`
	session := &domain.Session{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&session.ID,
		&session.UserID,
		&session.RefreshToken,
		&session.UserAgent,
		&session.IP,
		&session.ExpiresAt,
		&session.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return session, nil
}

// GetByRefreshToken retrieves a session by refresh token
func (r *PostgresSessionRepository) GetByRefreshToken(ctx context.Context, token string) (*domain.Session, error) {
	query := `
		SELECT id, user_id, refresh_token, user_agent, ip, expires_at, created_at
		FROM sessions
		WHERE refresh_token = $1 AND expires_at > NOW()
	`
	session := &domain.Session{}
	err := r.pool.QueryRow(ctx, query, token).Scan(
		&session.ID,
		&session.UserID,
		&session.RefreshToken,
		&session.UserAgent,
		&session.IP,
		&session.ExpiresAt,
		&session.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return session, nil
}

// GetByUserID retrieves all sessions for a user
func (r *PostgresSessionRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.Session, error) {
	query := `
		SELECT id, user_id, refresh_token, user_agent, ip, expires_at, created_at
		FROM sessions
		WHERE user_id = $1 AND expires_at > NOW()
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*domain.Session
	for rows.Next() {
		session := &domain.Session{}
		err := rows.Scan(
			&session.ID,
			&session.UserID,
			&session.RefreshToken,
			&session.UserAgent,
			&session.IP,
			&session.ExpiresAt,
			&session.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}
	return sessions, nil
}

// Delete deletes a session
func (r *PostgresSessionRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM sessions WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

// DeleteByUserID deletes all sessions for a user
func (r *PostgresSessionRepository) DeleteByUserID(ctx context.Context, userID string) error {
	query := `DELETE FROM sessions WHERE user_id = $1`
	_, err := r.pool.Exec(ctx, query, userID)
	return err
}

// DeleteExpired deletes all expired sessions
func (r *PostgresSessionRepository) DeleteExpired(ctx context.Context) error {
	query := `DELETE FROM sessions WHERE expires_at < $1`
	_, err := r.pool.Exec(ctx, query, time.Now())
	return err
}
