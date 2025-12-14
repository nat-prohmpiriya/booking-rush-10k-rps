package dto

import "time"

// CreateEventRequest represents the request to create a new event
type CreateEventRequest struct {
	Name              string     `json:"name" binding:"required,min=1,max=255"`
	Description       string     `json:"description"`
	ShortDescription  string     `json:"short_description" binding:"max=500"`
	CategoryID        *string    `json:"category_id"`
	PosterURL         string     `json:"poster_url"`
	BannerURL         string     `json:"banner_url"`
	Gallery           []string   `json:"gallery"`
	VenueName         string     `json:"venue_name" binding:"max=255"`
	VenueAddress      string     `json:"venue_address"`
	City              string     `json:"city" binding:"max=100"`
	Country           string     `json:"country" binding:"max=100"`
	Latitude          *float64   `json:"latitude"`
	Longitude         *float64   `json:"longitude"`
	MaxTicketsPerUser int        `json:"max_tickets_per_user"`
	BookingStartAt    *time.Time `json:"booking_start_at"`
	BookingEndAt      *time.Time `json:"booking_end_at"`
	MetaTitle         string     `json:"meta_title" binding:"max=255"`
	MetaDescription   string     `json:"meta_description" binding:"max=500"`
	TenantID          string     `json:"-"` // Set from context
	OrganizerID       string     `json:"-"` // Set from context
}

// Validate validates the CreateEventRequest
func (r *CreateEventRequest) Validate() (bool, string) {
	if r.Name == "" {
		return false, "Event name is required"
	}
	if r.MaxTicketsPerUser < 0 {
		return false, "Max tickets per user cannot be negative"
	}
	if r.BookingStartAt != nil && r.BookingEndAt != nil && r.BookingEndAt.Before(*r.BookingStartAt) {
		return false, "Booking end time must be after booking start time"
	}
	return true, ""
}

// UpdateEventRequest represents the request to update an event
type UpdateEventRequest struct {
	Name              string     `json:"name" binding:"omitempty,min=1,max=255"`
	Description       string     `json:"description"`
	ShortDescription  string     `json:"short_description" binding:"max=500"`
	CategoryID        *string    `json:"category_id"`
	PosterURL         string     `json:"poster_url"`
	BannerURL         string     `json:"banner_url"`
	Gallery           []string   `json:"gallery"`
	VenueName         string     `json:"venue_name" binding:"max=255"`
	VenueAddress      string     `json:"venue_address"`
	City              string     `json:"city" binding:"max=100"`
	Country           string     `json:"country" binding:"max=100"`
	Latitude          *float64   `json:"latitude"`
	Longitude         *float64   `json:"longitude"`
	MaxTicketsPerUser *int       `json:"max_tickets_per_user"`
	BookingStartAt    *time.Time `json:"booking_start_at"`
	BookingEndAt      *time.Time `json:"booking_end_at"`
	Status            *string    `json:"status"` // draft, published, cancelled, completed
	IsFeatured        *bool      `json:"is_featured"`
	IsPublic          *bool      `json:"is_public"`
	MetaTitle         string     `json:"meta_title" binding:"max=255"`
	MetaDescription   string     `json:"meta_description" binding:"max=500"`
}

// Validate validates the UpdateEventRequest
func (r *UpdateEventRequest) Validate() (bool, string) {
	if r.BookingStartAt != nil && r.BookingEndAt != nil && r.BookingEndAt.Before(*r.BookingStartAt) {
		return false, "Booking end time must be after booking start time"
	}
	if r.MaxTicketsPerUser != nil && *r.MaxTicketsPerUser < 0 {
		return false, "Max tickets per user cannot be negative"
	}
	return true, ""
}

// EventResponse represents the response for an event
type EventResponse struct {
	ID                string   `json:"id"`
	TenantID          string   `json:"tenant_id"`
	OrganizerID       string   `json:"organizer_id"`
	CategoryID        *string  `json:"category_id,omitempty"`
	Name              string   `json:"name"`
	Slug              string   `json:"slug"`
	Description       string   `json:"description"`
	ShortDescription  string   `json:"short_description"`
	PosterURL         string   `json:"poster_url"`
	BannerURL         string   `json:"banner_url"`
	Gallery           []string `json:"gallery"`
	VenueName         string   `json:"venue_name"`
	VenueAddress      string   `json:"venue_address"`
	City              string   `json:"city"`
	Country           string   `json:"country"`
	Latitude          *float64 `json:"latitude,omitempty"`
	Longitude         *float64 `json:"longitude,omitempty"`
	MaxTicketsPerUser int      `json:"max_tickets_per_user"`
	BookingStartAt    *string  `json:"booking_start_at,omitempty"`
	BookingEndAt      *string  `json:"booking_end_at,omitempty"`
	Status            string   `json:"status"`
	SaleStatus        string   `json:"sale_status"` // Aggregated from shows: scheduled, on_sale, sold_out, cancelled, completed
	IsFeatured        bool     `json:"is_featured"`
	IsPublic          bool     `json:"is_public"`
	MetaTitle         string   `json:"meta_title"`
	MetaDescription   string   `json:"meta_description"`
	MinPrice          float64  `json:"min_price"`
	PublishedAt       *string  `json:"published_at,omitempty"`
	CreatedAt         string   `json:"created_at"`
	UpdatedAt         string   `json:"updated_at"`
}

// EventListResponse represents a list of events
type EventListResponse struct {
	Events []*EventResponse `json:"events"`
	Total  int              `json:"total"`
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
}

// EventListFilter represents filters for listing events
type EventListFilter struct {
	Status      string `form:"status"`
	TenantID    string `form:"-"`
	OrganizerID string `form:"-"`
	CategoryID  string `form:"category_id"`
	City        string `form:"city"`
	Search      string `form:"search"`
	Limit       int    `form:"limit"`
	Offset      int    `form:"offset"`
}

// SetDefaults sets default values for pagination
func (f *EventListFilter) SetDefaults() {
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 20
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
}
