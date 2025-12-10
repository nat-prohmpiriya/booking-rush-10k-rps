package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/domain"
)

// PostgresVenueRepository implements VenueRepository using PostgreSQL
type PostgresVenueRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresVenueRepository creates a new PostgresVenueRepository
func NewPostgresVenueRepository(pool *pgxpool.Pool) *PostgresVenueRepository {
	return &PostgresVenueRepository{pool: pool}
}

// Create creates a new venue
func (r *PostgresVenueRepository) Create(ctx context.Context, venue *domain.Venue) error {
	query := `
		INSERT INTO venues (id, name, address, capacity, tenant_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.pool.Exec(ctx, query,
		venue.ID,
		venue.Name,
		venue.Address,
		venue.Capacity,
		venue.TenantID,
		venue.CreatedAt,
		venue.UpdatedAt,
	)
	return err
}

// GetByID retrieves a venue by ID
func (r *PostgresVenueRepository) GetByID(ctx context.Context, id string) (*domain.Venue, error) {
	query := `
		SELECT id, name, address, capacity, tenant_id, created_at, updated_at
		FROM venues
		WHERE id = $1
	`
	venue := &domain.Venue{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&venue.ID,
		&venue.Name,
		&venue.Address,
		&venue.Capacity,
		&venue.TenantID,
		&venue.CreatedAt,
		&venue.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return venue, nil
}

// GetByTenantID retrieves venues by tenant ID
func (r *PostgresVenueRepository) GetByTenantID(ctx context.Context, tenantID string) ([]*domain.Venue, error) {
	query := `
		SELECT id, name, address, capacity, tenant_id, created_at, updated_at
		FROM venues
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var venues []*domain.Venue
	for rows.Next() {
		venue := &domain.Venue{}
		err := rows.Scan(
			&venue.ID,
			&venue.Name,
			&venue.Address,
			&venue.Capacity,
			&venue.TenantID,
			&venue.CreatedAt,
			&venue.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		venues = append(venues, venue)
	}
	return venues, nil
}

// Update updates a venue
func (r *PostgresVenueRepository) Update(ctx context.Context, venue *domain.Venue) error {
	query := `
		UPDATE venues
		SET name = $2, address = $3, capacity = $4, updated_at = $5
		WHERE id = $1
	`
	venue.UpdatedAt = time.Now()
	result, err := r.pool.Exec(ctx, query,
		venue.ID,
		venue.Name,
		venue.Address,
		venue.Capacity,
		venue.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return errors.New("venue not found")
	}
	return nil
}

// Delete deletes a venue by ID
func (r *PostgresVenueRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM venues WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return errors.New("venue not found")
	}
	return nil
}
