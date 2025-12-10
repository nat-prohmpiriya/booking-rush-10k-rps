package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/domain"
)

// PostgresShowZoneRepository implements ShowZoneRepository using PostgreSQL
type PostgresShowZoneRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresShowZoneRepository creates a new PostgresShowZoneRepository
func NewPostgresShowZoneRepository(pool *pgxpool.Pool) *PostgresShowZoneRepository {
	return &PostgresShowZoneRepository{pool: pool}
}

// Create creates a new show zone
func (r *PostgresShowZoneRepository) Create(ctx context.Context, zone *domain.ShowZone) error {
	query := `
		INSERT INTO show_zones (id, show_id, name, price, total_seats, available_seats, description, sort_order, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.pool.Exec(ctx, query,
		zone.ID,
		zone.ShowID,
		zone.Name,
		zone.Price,
		zone.TotalSeats,
		zone.AvailableSeats,
		zone.Description,
		zone.SortOrder,
		zone.CreatedAt,
		zone.UpdatedAt,
	)
	return err
}

// GetByID retrieves a show zone by ID
func (r *PostgresShowZoneRepository) GetByID(ctx context.Context, id string) (*domain.ShowZone, error) {
	query := `
		SELECT id, show_id, name, price, total_seats, available_seats, description, sort_order, created_at, updated_at, deleted_at
		FROM show_zones
		WHERE id = $1 AND deleted_at IS NULL
	`
	zone := &domain.ShowZone{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&zone.ID,
		&zone.ShowID,
		&zone.Name,
		&zone.Price,
		&zone.TotalSeats,
		&zone.AvailableSeats,
		&zone.Description,
		&zone.SortOrder,
		&zone.CreatedAt,
		&zone.UpdatedAt,
		&zone.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return zone, nil
}

// GetByShowID retrieves all zones for a show with pagination
func (r *PostgresShowZoneRepository) GetByShowID(ctx context.Context, showID string, limit, offset int) ([]*domain.ShowZone, int, error) {
	// Count total
	countQuery := `SELECT COUNT(*) FROM show_zones WHERE show_id = $1 AND deleted_at IS NULL`
	var total int
	err := r.pool.QueryRow(ctx, countQuery, showID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get zones
	query := `
		SELECT id, show_id, name, price, total_seats, available_seats, description, sort_order, created_at, updated_at, deleted_at
		FROM show_zones
		WHERE show_id = $1 AND deleted_at IS NULL
		ORDER BY sort_order ASC, name ASC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.pool.Query(ctx, query, showID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var zones []*domain.ShowZone
	for rows.Next() {
		zone := &domain.ShowZone{}
		err := rows.Scan(
			&zone.ID,
			&zone.ShowID,
			&zone.Name,
			&zone.Price,
			&zone.TotalSeats,
			&zone.AvailableSeats,
			&zone.Description,
			&zone.SortOrder,
			&zone.CreatedAt,
			&zone.UpdatedAt,
			&zone.DeletedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		zones = append(zones, zone)
	}
	return zones, total, nil
}

// Update updates a show zone
func (r *PostgresShowZoneRepository) Update(ctx context.Context, zone *domain.ShowZone) error {
	query := `
		UPDATE show_zones
		SET name = $2, price = $3, total_seats = $4, available_seats = $5, description = $6, sort_order = $7, updated_at = $8
		WHERE id = $1 AND deleted_at IS NULL
	`
	zone.UpdatedAt = time.Now()
	result, err := r.pool.Exec(ctx, query,
		zone.ID,
		zone.Name,
		zone.Price,
		zone.TotalSeats,
		zone.AvailableSeats,
		zone.Description,
		zone.SortOrder,
		zone.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return errors.New("show zone not found")
	}
	return nil
}

// Delete soft deletes a show zone by ID
func (r *PostgresShowZoneRepository) Delete(ctx context.Context, id string) error {
	query := `
		UPDATE show_zones
		SET deleted_at = $2, updated_at = $2
		WHERE id = $1 AND deleted_at IS NULL
	`
	now := time.Now()
	result, err := r.pool.Exec(ctx, query, id, now)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return errors.New("show zone not found")
	}
	return nil
}

// UpdateAvailableSeats updates the available seats count
func (r *PostgresShowZoneRepository) UpdateAvailableSeats(ctx context.Context, id string, availableSeats int) error {
	query := `
		UPDATE show_zones
		SET available_seats = $2, updated_at = $3
		WHERE id = $1 AND deleted_at IS NULL
	`
	now := time.Now()
	result, err := r.pool.Exec(ctx, query, id, availableSeats, now)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return errors.New("show zone not found")
	}
	return nil
}
