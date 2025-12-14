package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/domain"
)

// seatZoneColumns defines columns for seat_zones table
const seatZoneColumns = `id, show_id, name, COALESCE(description, '') as description,
	COALESCE(color, '') as color, price, COALESCE(currency, 'THB') as currency,
	total_seats, available_seats, COALESCE(reserved_seats, 0) as reserved_seats,
	COALESCE(sold_seats, 0) as sold_seats, COALESCE(min_per_order, 1) as min_per_order,
	COALESCE(max_per_order, 10) as max_per_order, COALESCE(is_active, true) as is_active,
	COALESCE(sort_order, 0) as sort_order, sale_start_at, sale_end_at,
	created_at, updated_at, deleted_at`

// PostgresShowZoneRepository implements ShowZoneRepository using PostgreSQL
type PostgresShowZoneRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresShowZoneRepository creates a new PostgresShowZoneRepository
func NewPostgresShowZoneRepository(pool *pgxpool.Pool) *PostgresShowZoneRepository {
	return &PostgresShowZoneRepository{pool: pool}
}

// scanZone scans a row into a ShowZone struct
func (r *PostgresShowZoneRepository) scanZone(row pgx.Row) (*domain.ShowZone, error) {
	zone := &domain.ShowZone{}
	err := row.Scan(
		&zone.ID,
		&zone.ShowID,
		&zone.Name,
		&zone.Description,
		&zone.Color,
		&zone.Price,
		&zone.Currency,
		&zone.TotalSeats,
		&zone.AvailableSeats,
		&zone.ReservedSeats,
		&zone.SoldSeats,
		&zone.MinPerOrder,
		&zone.MaxPerOrder,
		&zone.IsActive,
		&zone.SortOrder,
		&zone.SaleStartAt,
		&zone.SaleEndAt,
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

// Create creates a new show zone
func (r *PostgresShowZoneRepository) Create(ctx context.Context, zone *domain.ShowZone) error {
	query := `
		INSERT INTO seat_zones (id, show_id, name, description, color, price, currency,
			total_seats, available_seats, min_per_order, max_per_order, is_active,
			sort_order, sale_start_at, sale_end_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`
	_, err := r.pool.Exec(ctx, query,
		zone.ID,
		zone.ShowID,
		zone.Name,
		zone.Description,
		zone.Color,
		zone.Price,
		zone.Currency,
		zone.TotalSeats,
		zone.AvailableSeats,
		zone.MinPerOrder,
		zone.MaxPerOrder,
		zone.IsActive,
		zone.SortOrder,
		zone.SaleStartAt,
		zone.SaleEndAt,
		zone.CreatedAt,
		zone.UpdatedAt,
	)
	return err
}

// GetByID retrieves a show zone by ID
func (r *PostgresShowZoneRepository) GetByID(ctx context.Context, id string) (*domain.ShowZone, error) {
	query := `SELECT ` + seatZoneColumns + ` FROM seat_zones WHERE id = $1 AND deleted_at IS NULL`
	return r.scanZone(r.pool.QueryRow(ctx, query, id))
}

// GetByShowID retrieves all zones for a show with pagination and optional is_active filter
func (r *PostgresShowZoneRepository) GetByShowID(ctx context.Context, showID string, isActive *bool, limit, offset int) ([]*domain.ShowZone, int, error) {
	// Build WHERE clause
	whereClause := "show_id = $1 AND deleted_at IS NULL"
	args := []interface{}{showID}
	argIndex := 2

	if isActive != nil {
		whereClause += fmt.Sprintf(" AND is_active = $%d", argIndex)
		args = append(args, *isActive)
		argIndex++
	}

	// Count total
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM seat_zones WHERE %s`, whereClause)
	var total int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get zones
	query := fmt.Sprintf(`SELECT %s FROM seat_zones
		WHERE %s
		ORDER BY sort_order ASC, name ASC
		LIMIT $%d OFFSET $%d`, seatZoneColumns, whereClause, argIndex, argIndex+1)
	args = append(args, limit, offset)
	rows, err := r.pool.Query(ctx, query, args...)
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
			&zone.Description,
			&zone.Color,
			&zone.Price,
			&zone.Currency,
			&zone.TotalSeats,
			&zone.AvailableSeats,
			&zone.ReservedSeats,
			&zone.SoldSeats,
			&zone.MinPerOrder,
			&zone.MaxPerOrder,
			&zone.IsActive,
			&zone.SortOrder,
			&zone.SaleStartAt,
			&zone.SaleEndAt,
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
		UPDATE seat_zones
		SET name = $2, description = $3, color = $4, price = $5, total_seats = $6,
			available_seats = $7, min_per_order = $8, max_per_order = $9, is_active = $10,
			sort_order = $11, updated_at = $12
		WHERE id = $1 AND deleted_at IS NULL
	`
	zone.UpdatedAt = time.Now()
	result, err := r.pool.Exec(ctx, query,
		zone.ID,
		zone.Name,
		zone.Description,
		zone.Color,
		zone.Price,
		zone.TotalSeats,
		zone.AvailableSeats,
		zone.MinPerOrder,
		zone.MaxPerOrder,
		zone.IsActive,
		zone.SortOrder,
		zone.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return errors.New("seat zone not found")
	}
	return nil
}

// Delete soft deletes a show zone by ID
func (r *PostgresShowZoneRepository) Delete(ctx context.Context, id string) error {
	query := `
		UPDATE seat_zones
		SET deleted_at = $2, updated_at = $2
		WHERE id = $1 AND deleted_at IS NULL
	`
	now := time.Now()
	result, err := r.pool.Exec(ctx, query, id, now)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return errors.New("seat zone not found")
	}
	return nil
}

// UpdateAvailableSeats updates the available seats count
func (r *PostgresShowZoneRepository) UpdateAvailableSeats(ctx context.Context, id string, availableSeats int) error {
	query := `
		UPDATE seat_zones
		SET available_seats = $2, updated_at = $3
		WHERE id = $1 AND deleted_at IS NULL
	`
	now := time.Now()
	result, err := r.pool.Exec(ctx, query, id, availableSeats, now)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return errors.New("seat zone not found")
	}
	return nil
}

// ListActive retrieves all active zones for inventory sync
func (r *PostgresShowZoneRepository) ListActive(ctx context.Context) ([]*domain.ShowZone, error) {
	query := `SELECT ` + seatZoneColumns + ` FROM seat_zones
		WHERE is_active = true AND deleted_at IS NULL
		ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var zones []*domain.ShowZone
	for rows.Next() {
		zone := &domain.ShowZone{}
		err := rows.Scan(
			&zone.ID,
			&zone.ShowID,
			&zone.Name,
			&zone.Description,
			&zone.Color,
			&zone.Price,
			&zone.Currency,
			&zone.TotalSeats,
			&zone.AvailableSeats,
			&zone.ReservedSeats,
			&zone.SoldSeats,
			&zone.MinPerOrder,
			&zone.MaxPerOrder,
			&zone.IsActive,
			&zone.SortOrder,
			&zone.SaleStartAt,
			&zone.SaleEndAt,
			&zone.CreatedAt,
			&zone.UpdatedAt,
			&zone.DeletedAt,
		)
		if err != nil {
			return nil, err
		}
		zones = append(zones, zone)
	}
	return zones, nil
}
