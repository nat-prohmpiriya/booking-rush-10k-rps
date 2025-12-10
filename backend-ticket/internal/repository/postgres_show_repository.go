package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/domain"
)

// PostgresShowRepository implements ShowRepository using PostgreSQL
type PostgresShowRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresShowRepository creates a new PostgresShowRepository
func NewPostgresShowRepository(pool *pgxpool.Pool) *PostgresShowRepository {
	return &PostgresShowRepository{pool: pool}
}

// Create creates a new show
func (r *PostgresShowRepository) Create(ctx context.Context, show *domain.Show) error {
	query := `
		INSERT INTO shows (id, event_id, name, start_time, end_time, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.pool.Exec(ctx, query,
		show.ID,
		show.EventID,
		show.Name,
		show.StartTime,
		show.EndTime,
		show.Status,
		show.CreatedAt,
		show.UpdatedAt,
	)
	return err
}

// GetByID retrieves a show by ID
func (r *PostgresShowRepository) GetByID(ctx context.Context, id string) (*domain.Show, error) {
	query := `
		SELECT id, event_id, name, start_time, end_time, status, created_at, updated_at, deleted_at
		FROM shows
		WHERE id = $1 AND deleted_at IS NULL
	`
	show := &domain.Show{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&show.ID,
		&show.EventID,
		&show.Name,
		&show.StartTime,
		&show.EndTime,
		&show.Status,
		&show.CreatedAt,
		&show.UpdatedAt,
		&show.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return show, nil
}

// GetByEventID retrieves shows by event ID with pagination
func (r *PostgresShowRepository) GetByEventID(ctx context.Context, eventID string, limit, offset int) ([]*domain.Show, int, error) {
	// Count total
	countQuery := `SELECT COUNT(*) FROM shows WHERE event_id = $1 AND deleted_at IS NULL`
	var total int
	err := r.pool.QueryRow(ctx, countQuery, eventID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get shows
	query := `
		SELECT id, event_id, name, start_time, end_time, status, created_at, updated_at, deleted_at
		FROM shows
		WHERE event_id = $1 AND deleted_at IS NULL
		ORDER BY start_time ASC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.pool.Query(ctx, query, eventID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var shows []*domain.Show
	for rows.Next() {
		show := &domain.Show{}
		err := rows.Scan(
			&show.ID,
			&show.EventID,
			&show.Name,
			&show.StartTime,
			&show.EndTime,
			&show.Status,
			&show.CreatedAt,
			&show.UpdatedAt,
			&show.DeletedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		shows = append(shows, show)
	}
	return shows, total, nil
}

// Update updates a show
func (r *PostgresShowRepository) Update(ctx context.Context, show *domain.Show) error {
	query := `
		UPDATE shows
		SET name = $2, start_time = $3, end_time = $4, status = $5, updated_at = $6
		WHERE id = $1 AND deleted_at IS NULL
	`
	show.UpdatedAt = time.Now()
	result, err := r.pool.Exec(ctx, query,
		show.ID,
		show.Name,
		show.StartTime,
		show.EndTime,
		show.Status,
		show.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return errors.New("show not found")
	}
	return nil
}

// Delete soft deletes a show by ID
func (r *PostgresShowRepository) Delete(ctx context.Context, id string) error {
	query := `
		UPDATE shows
		SET deleted_at = $2, updated_at = $2
		WHERE id = $1 AND deleted_at IS NULL
	`
	now := time.Now()
	result, err := r.pool.Exec(ctx, query, id, now)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return errors.New("show not found")
	}
	return nil
}
