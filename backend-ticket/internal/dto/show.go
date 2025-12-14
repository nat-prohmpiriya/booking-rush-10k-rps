package dto

import "time"

// CreateShowRequest represents the request to create a new show
type CreateShowRequest struct {
	EventID     string     `json:"-"` // Set from URL param
	Name        string     `json:"name" binding:"max=255"`
	ShowDate    string     `json:"show_date" binding:"required"` // Format: YYYY-MM-DD (e.g., 2006-01-02)
	StartTime   string     `json:"start_time" binding:"required"` // Format: ISO 8601 (e.g., 2006-01-02T15:04:05+07:00) or time-only (e.g., 15:04:05)
	EndTime     string     `json:"end_time"`                      // Format: ISO 8601 or time-only
	DoorsOpenAt string     `json:"doors_open_at"`                 // Format: ISO 8601 or time-only
	SaleStartAt *time.Time `json:"sale_start_at"`
	SaleEndAt   *time.Time `json:"sale_end_at"`
}

// Validate validates the CreateShowRequest
func (r *CreateShowRequest) Validate() (bool, string) {
	if r.ShowDate == "" {
		return false, "Show date is required"
	}
	if r.StartTime == "" {
		return false, "Start time is required"
	}
	return true, ""
}

// UpdateShowRequest represents the request to update a show
type UpdateShowRequest struct {
	Name        string     `json:"name" binding:"omitempty,max=255"`
	ShowDate    string     `json:"show_date"`
	StartTime   string     `json:"start_time"`
	EndTime     string     `json:"end_time"`
	DoorsOpenAt string     `json:"doors_open_at"`
	Status      string     `json:"status"`
	SaleStartAt *time.Time `json:"sale_start_at"`
	SaleEndAt   *time.Time `json:"sale_end_at"`
}

// Validate validates the UpdateShowRequest
func (r *UpdateShowRequest) Validate() (bool, string) {
	return true, ""
}

// ShowResponse represents the response for a show
type ShowResponse struct {
	ID            string  `json:"id"`
	EventID       string  `json:"event_id"`
	Name          string  `json:"name"`
	ShowDate      string  `json:"show_date"`
	StartTime     string  `json:"start_time"`
	EndTime       string  `json:"end_time"`
	DoorsOpenAt   *string `json:"doors_open_at,omitempty"`
	Status        string  `json:"status"`
	SaleStartAt   *string `json:"sale_start_at,omitempty"`
	SaleEndAt     *string `json:"sale_end_at,omitempty"`
	TotalCapacity int     `json:"total_capacity"`
	ReservedCount int     `json:"reserved_count"`
	SoldCount     int     `json:"sold_count"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

// ShowListResponse represents a list of shows
type ShowListResponse struct {
	Shows  []*ShowResponse `json:"shows"`
	Total  int             `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

// ShowListFilter represents filters for listing shows
type ShowListFilter struct {
	EventID string `form:"-"`
	Limit   int    `form:"limit"`
	Offset  int    `form:"offset"`
}

// SetDefaults sets default values for pagination
func (f *ShowListFilter) SetDefaults() {
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 20
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
}
