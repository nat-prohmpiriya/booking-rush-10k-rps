package repository

import (
	"context"
	"encoding/json"
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

// eventColumns defines the columns to select for events
// Using COALESCE for nullable string columns to avoid scan errors
const eventColumns = `id, tenant_id, organizer_id, category_id, name, slug,
	COALESCE(description, '') as description,
	COALESCE(short_description, '') as short_description,
	COALESCE(poster_url, '') as poster_url,
	COALESCE(banner_url, '') as banner_url,
	COALESCE(gallery, '[]'::jsonb) as gallery,
	COALESCE(venue_name, '') as venue_name,
	COALESCE(venue_address, '') as venue_address,
	COALESCE(city, '') as city,
	COALESCE(country, '') as country,
	latitude, longitude, max_tickets_per_user, booking_start_at,
	booking_end_at, status, is_featured, is_public,
	COALESCE(meta_title, '') as meta_title,
	COALESCE(meta_description, '') as meta_description,
	COALESCE(settings, '{}'::jsonb) as settings,
	0 as min_price,
	published_at, created_at, updated_at, deleted_at`

// eventColumnsWithPrice includes min_price for queries with price aggregation
const eventColumnsWithPrice = `e.id, e.tenant_id, e.organizer_id, e.category_id, e.name, e.slug,
	COALESCE(e.description, '') as description,
	COALESCE(e.short_description, '') as short_description,
	COALESCE(e.poster_url, '') as poster_url,
	COALESCE(e.banner_url, '') as banner_url,
	COALESCE(e.gallery, '[]'::jsonb) as gallery,
	COALESCE(e.venue_name, '') as venue_name,
	COALESCE(e.venue_address, '') as venue_address,
	COALESCE(e.city, '') as city,
	COALESCE(e.country, '') as country,
	e.latitude, e.longitude, e.max_tickets_per_user, e.booking_start_at,
	e.booking_end_at, e.status, e.is_featured, e.is_public,
	COALESCE(e.meta_title, '') as meta_title,
	COALESCE(e.meta_description, '') as meta_description,
	COALESCE(e.settings, '{}'::jsonb) as settings,
	COALESCE(MIN(sz.price), 0) as min_price,
	e.published_at, e.created_at, e.updated_at, e.deleted_at`

// scanEvent scans a row into an Event struct
func (r *PostgresEventRepository) scanEvent(row pgx.Row) (*domain.Event, error) {
	event := &domain.Event{}
	var galleryJSON []byte
	var settingsJSON []byte

	err := row.Scan(
		&event.ID,
		&event.TenantID,
		&event.OrganizerID,
		&event.CategoryID,
		&event.Name,
		&event.Slug,
		&event.Description,
		&event.ShortDescription,
		&event.PosterURL,
		&event.BannerURL,
		&galleryJSON,
		&event.VenueName,
		&event.VenueAddress,
		&event.City,
		&event.Country,
		&event.Latitude,
		&event.Longitude,
		&event.MaxTicketsPerUser,
		&event.BookingStartAt,
		&event.BookingEndAt,
		&event.Status,
		&event.IsFeatured,
		&event.IsPublic,
		&event.MetaTitle,
		&event.MetaDescription,
		&settingsJSON,
		&event.MinPrice,
		&event.PublishedAt,
		&event.CreatedAt,
		&event.UpdatedAt,
		&event.DeletedAt,
	)
	if err != nil {
		return nil, err
	}

	// Parse gallery JSON
	if galleryJSON != nil {
		json.Unmarshal(galleryJSON, &event.Gallery)
	}
	if event.Gallery == nil {
		event.Gallery = []string{}
	}

	// Parse settings JSON
	if settingsJSON != nil {
		event.Settings = string(settingsJSON)
	}

	return event, nil
}

// scanEvents scans multiple rows into Event structs
func (r *PostgresEventRepository) scanEvents(rows pgx.Rows) ([]*domain.Event, error) {
	var events []*domain.Event
	for rows.Next() {
		event := &domain.Event{}
		var galleryJSON []byte
		var settingsJSON []byte

		err := rows.Scan(
			&event.ID,
			&event.TenantID,
			&event.OrganizerID,
			&event.CategoryID,
			&event.Name,
			&event.Slug,
			&event.Description,
			&event.ShortDescription,
			&event.PosterURL,
			&event.BannerURL,
			&galleryJSON,
			&event.VenueName,
			&event.VenueAddress,
			&event.City,
			&event.Country,
			&event.Latitude,
			&event.Longitude,
			&event.MaxTicketsPerUser,
			&event.BookingStartAt,
			&event.BookingEndAt,
			&event.Status,
			&event.IsFeatured,
			&event.IsPublic,
			&event.MetaTitle,
			&event.MetaDescription,
			&settingsJSON,
			&event.MinPrice,
			&event.PublishedAt,
			&event.CreatedAt,
			&event.UpdatedAt,
			&event.DeletedAt,
		)
		if err != nil {
			return nil, err
		}

		// Parse gallery JSON
		if galleryJSON != nil {
			json.Unmarshal(galleryJSON, &event.Gallery)
		}
		if event.Gallery == nil {
			event.Gallery = []string{}
		}

		// Parse settings JSON
		if settingsJSON != nil {
			event.Settings = string(settingsJSON)
		}

		events = append(events, event)
	}
	return events, nil
}

// Create creates a new event
func (r *PostgresEventRepository) Create(ctx context.Context, event *domain.Event) error {
	query := `
		INSERT INTO events (
			id, tenant_id, organizer_id, category_id, name, slug, description,
			short_description, poster_url, banner_url, gallery, venue_name, venue_address,
			city, country, latitude, longitude, max_tickets_per_user, booking_start_at,
			booking_end_at, status, is_featured, is_public, meta_title, meta_description,
			settings, published_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29
		)
	`

	galleryJSON, _ := json.Marshal(event.Gallery)
	settingsJSON := event.Settings
	if settingsJSON == "" {
		settingsJSON = "{}"
	}

	_, err := r.pool.Exec(ctx, query,
		event.ID,
		event.TenantID,
		event.OrganizerID,
		event.CategoryID,
		event.Name,
		event.Slug,
		event.Description,
		event.ShortDescription,
		event.PosterURL,
		event.BannerURL,
		galleryJSON,
		event.VenueName,
		event.VenueAddress,
		event.City,
		event.Country,
		event.Latitude,
		event.Longitude,
		event.MaxTicketsPerUser,
		event.BookingStartAt,
		event.BookingEndAt,
		event.Status,
		event.IsFeatured,
		event.IsPublic,
		event.MetaTitle,
		event.MetaDescription,
		settingsJSON,
		event.PublishedAt,
		event.CreatedAt,
		event.UpdatedAt,
	)
	return err
}

// GetByID retrieves an event by ID
func (r *PostgresEventRepository) GetByID(ctx context.Context, id string) (*domain.Event, error) {
	query := fmt.Sprintf(`SELECT %s FROM events WHERE id = $1 AND deleted_at IS NULL`, eventColumns)
	event, err := r.scanEvent(r.pool.QueryRow(ctx, query, id))
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
	query := fmt.Sprintf(`SELECT %s FROM events WHERE slug = $1 AND deleted_at IS NULL`, eventColumns)
	event, err := r.scanEvent(r.pool.QueryRow(ctx, query, slug))
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
	query := fmt.Sprintf(`
		SELECT %s FROM events
		WHERE tenant_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, eventColumns)

	rows, err := r.pool.Query(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanEvents(rows)
}

// Update updates an event
func (r *PostgresEventRepository) Update(ctx context.Context, event *domain.Event) error {
	query := `
		UPDATE events SET
			name = $2, slug = $3, description = $4, short_description = $5,
			poster_url = $6, banner_url = $7, gallery = $8, venue_name = $9,
			venue_address = $10, city = $11, country = $12, latitude = $13,
			longitude = $14, max_tickets_per_user = $15, booking_start_at = $16,
			booking_end_at = $17, status = $18, is_featured = $19, is_public = $20,
			meta_title = $21, meta_description = $22, settings = $23, updated_at = $24
		WHERE id = $1 AND deleted_at IS NULL
	`

	galleryJSON, _ := json.Marshal(event.Gallery)
	settingsJSON := event.Settings
	if settingsJSON == "" {
		settingsJSON = "{}"
	}

	event.UpdatedAt = time.Now()
	result, err := r.pool.Exec(ctx, query,
		event.ID,
		event.Name,
		event.Slug,
		event.Description,
		event.ShortDescription,
		event.PosterURL,
		event.BannerURL,
		galleryJSON,
		event.VenueName,
		event.VenueAddress,
		event.City,
		event.Country,
		event.Latitude,
		event.Longitude,
		event.MaxTicketsPerUser,
		event.BookingStartAt,
		event.BookingEndAt,
		event.Status,
		event.IsFeatured,
		event.IsPublic,
		event.MetaTitle,
		event.MetaDescription,
		settingsJSON,
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

// ListPublished lists all published events with pagination and min price
func (r *PostgresEventRepository) ListPublished(ctx context.Context, limit, offset int) ([]*domain.Event, int, error) {
	// Count total
	countQuery := `SELECT COUNT(*) FROM events WHERE status = $1 AND deleted_at IS NULL AND is_public = true`
	var total int
	err := r.pool.QueryRow(ctx, countQuery, domain.EventStatusPublished).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get events with min_price from seat_zones
	query := fmt.Sprintf(`
		SELECT %s
		FROM events e
		LEFT JOIN shows s ON s.event_id = e.id AND s.deleted_at IS NULL
		LEFT JOIN seat_zones sz ON sz.show_id = s.id AND sz.deleted_at IS NULL
		WHERE e.status = $1 AND e.deleted_at IS NULL AND e.is_public = true
		GROUP BY e.id, e.tenant_id, e.organizer_id, e.category_id, e.name, e.slug,
			e.description, e.short_description, e.poster_url, e.banner_url, e.gallery,
			e.venue_name, e.venue_address, e.city, e.country, e.latitude, e.longitude,
			e.max_tickets_per_user, e.booking_start_at, e.booking_end_at, e.status,
			e.is_featured, e.is_public, e.meta_title, e.meta_description, e.settings,
			e.published_at, e.created_at, e.updated_at, e.deleted_at
		ORDER BY e.created_at DESC
		LIMIT $2 OFFSET $3
	`, eventColumnsWithPrice)

	rows, err := r.pool.Query(ctx, query, domain.EventStatusPublished, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	events, err := r.scanEvents(rows)
	if err != nil {
		return nil, 0, err
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
		if filter.OrganizerID != "" {
			conditions = append(conditions, fmt.Sprintf("organizer_id = $%d", argIndex))
			args = append(args, filter.OrganizerID)
			argIndex++
		}
		if filter.Search != "" {
			conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR description ILIKE $%d)", argIndex, argIndex))
			args = append(args, "%"+filter.Search+"%")
			argIndex++
		}
		if filter.City != "" {
			conditions = append(conditions, fmt.Sprintf("city = $%d", argIndex))
			args = append(args, filter.City)
			argIndex++
		}
		if filter.CategoryID != "" {
			conditions = append(conditions, fmt.Sprintf("category_id = $%d", argIndex))
			args = append(args, filter.CategoryID)
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
		SELECT %s FROM events
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, eventColumns, whereClause, argIndex, argIndex+1)

	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	events, err := r.scanEvents(rows)
	if err != nil {
		return nil, 0, err
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
