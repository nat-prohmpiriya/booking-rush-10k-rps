package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/domain"
)

// PostgresEventRepository implements EventRepository using PostgreSQL
type PostgresEventRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresEventRepository creates a new PostgresEventRepository
func NewPostgresEventRepository(pool *pgxpool.Pool) *PostgresEventRepository {
	return &PostgresEventRepository{pool: pool}
}

// Create creates a new event
func (r *PostgresEventRepository) Create(ctx context.Context, event *domain.Event) error {
	query := `
		INSERT INTO events (id, name, slug, description, venue_id, start_time, end_time, status, tenant_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.pool.Exec(ctx, query,
		event.ID,
		event.Name,
		event.Slug,
		event.Description,
		event.VenueID,
		event.StartTime,
		event.EndTime,
		event.Status,
		event.TenantID,
		event.CreatedAt,
		event.UpdatedAt,
	)
	return err
}

// GetByID retrieves an event by ID
func (r *PostgresEventRepository) GetByID(ctx context.Context, id string) (*domain.Event, error) {
	query := `
		SELECT id, name, slug, description, venue_id, start_time, end_time, status, tenant_id, created_at, updated_at, deleted_at
		FROM events
		WHERE id = $1 AND deleted_at IS NULL
	`
	event := &domain.Event{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&event.ID,
		&event.Name,
		&event.Slug,
		&event.Description,
		&event.VenueID,
		&event.StartTime,
		&event.EndTime,
		&event.Status,
		&event.TenantID,
		&event.CreatedAt,
		&event.UpdatedAt,
		&event.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return event, nil
}

// GetBySlug retrieves an event by slug
func (r *PostgresEventRepository) GetBySlug(ctx context.Context, slug string) (*domain.Event, error) {
	query := `
		SELECT id, name, slug, description, venue_id, start_time, end_time, status, tenant_id, created_at, updated_at, deleted_at
		FROM events
		WHERE slug = $1 AND deleted_at IS NULL
	`
	event := &domain.Event{}
	err := r.pool.QueryRow(ctx, query, slug).Scan(
		&event.ID,
		&event.Name,
		&event.Slug,
		&event.Description,
		&event.VenueID,
		&event.StartTime,
		&event.EndTime,
		&event.Status,
		&event.TenantID,
		&event.CreatedAt,
		&event.UpdatedAt,
		&event.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return event, nil
}

// GetByTenantID retrieves events by tenant ID
func (r *PostgresEventRepository) GetByTenantID(ctx context.Context, tenantID string, limit, offset int) ([]*domain.Event, error) {
	query := `
		SELECT id, name, slug, description, venue_id, start_time, end_time, status, tenant_id, created_at, updated_at, deleted_at
		FROM events
		WHERE tenant_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.pool.Query(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*domain.Event
	for rows.Next() {
		event := &domain.Event{}
		err := rows.Scan(
			&event.ID,
			&event.Name,
			&event.Slug,
			&event.Description,
			&event.VenueID,
			&event.StartTime,
			&event.EndTime,
			&event.Status,
			&event.TenantID,
			&event.CreatedAt,
			&event.UpdatedAt,
			&event.DeletedAt,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, nil
}

// Update updates an event
func (r *PostgresEventRepository) Update(ctx context.Context, event *domain.Event) error {
	query := `
		UPDATE events
		SET name = $2, slug = $3, description = $4, venue_id = $5, start_time = $6, end_time = $7, status = $8, updated_at = $9
		WHERE id = $1 AND deleted_at IS NULL
	`
	event.UpdatedAt = time.Now()
	result, err := r.pool.Exec(ctx, query,
		event.ID,
		event.Name,
		event.Slug,
		event.Description,
		event.VenueID,
		event.StartTime,
		event.EndTime,
		event.Status,
		event.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return errors.New("event not found")
	}
	return nil
}

// Delete soft deletes an event by ID
func (r *PostgresEventRepository) Delete(ctx context.Context, id string) error {
	query := `
		UPDATE events
		SET deleted_at = $2, updated_at = $2
		WHERE id = $1 AND deleted_at IS NULL
	`
	now := time.Now()
	result, err := r.pool.Exec(ctx, query, id, now)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return errors.New("event not found")
	}
	return nil
}

// ListPublished lists all published events with pagination
func (r *PostgresEventRepository) ListPublished(ctx context.Context, limit, offset int) ([]*domain.Event, int, error) {
	// Count total
	countQuery := `SELECT COUNT(*) FROM events WHERE status = $1 AND deleted_at IS NULL`
	var total int
	err := r.pool.QueryRow(ctx, countQuery, domain.EventStatusPublished).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get events
	query := `
		SELECT id, name, slug, description, venue_id, start_time, end_time, status, tenant_id, created_at, updated_at, deleted_at
		FROM events
		WHERE status = $1 AND deleted_at IS NULL
		ORDER BY start_time ASC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.pool.Query(ctx, query, domain.EventStatusPublished, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []*domain.Event
	for rows.Next() {
		event := &domain.Event{}
		err := rows.Scan(
			&event.ID,
			&event.Name,
			&event.Slug,
			&event.Description,
			&event.VenueID,
			&event.StartTime,
			&event.EndTime,
			&event.Status,
			&event.TenantID,
			&event.CreatedAt,
			&event.UpdatedAt,
			&event.DeletedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		events = append(events, event)
	}
	return events, total, nil
}

// List lists events with filters and pagination
func (r *PostgresEventRepository) List(ctx context.Context, filter *EventFilter, limit, offset int) ([]*domain.Event, int, error) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	conditions = append(conditions, "deleted_at IS NULL")

	if filter != nil {
		if filter.Status != "" {
			conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
			args = append(args, filter.Status)
			argIndex++
		}
		if filter.TenantID != "" {
			conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIndex))
			args = append(args, filter.TenantID)
			argIndex++
		}
		if filter.VenueID != "" {
			conditions = append(conditions, fmt.Sprintf("venue_id = $%d", argIndex))
			args = append(args, filter.VenueID)
			argIndex++
		}
		if filter.Search != "" {
			conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR description ILIKE $%d)", argIndex, argIndex))
			args = append(args, "%"+filter.Search+"%")
			argIndex++
		}
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM events WHERE %s", whereClause)
	var total int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get events
	query := fmt.Sprintf(`
		SELECT id, name, slug, description, venue_id, start_time, end_time, status, tenant_id, created_at, updated_at, deleted_at
		FROM events
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []*domain.Event
	for rows.Next() {
		event := &domain.Event{}
		err := rows.Scan(
			&event.ID,
			&event.Name,
			&event.Slug,
			&event.Description,
			&event.VenueID,
			&event.StartTime,
			&event.EndTime,
			&event.Status,
			&event.TenantID,
			&event.CreatedAt,
			&event.UpdatedAt,
			&event.DeletedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		events = append(events, event)
	}
	return events, total, nil
}

// SlugExists checks if a slug already exists
func (r *PostgresEventRepository) SlugExists(ctx context.Context, slug string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM events WHERE slug = $1 AND deleted_at IS NULL)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, slug).Scan(&exists)
	return exists, err
}
