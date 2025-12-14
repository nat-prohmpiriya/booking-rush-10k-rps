package dto

// CreateShowZoneRequest represents the request to create a new show zone
type CreateShowZoneRequest struct {
	ShowID      string  `json:"-"` // Set from URL param
	Name        string  `json:"name" binding:"required,min=1,max=200"`
	Price       float64 `json:"price" binding:"required,gte=0"`
	TotalSeats  int     `json:"total_seats" binding:"required,gt=0"`
	Description string  `json:"description" binding:"omitempty,max=1000"`
	SortOrder   int     `json:"sort_order" binding:"omitempty,gte=0"`
}

// Validate validates the CreateShowZoneRequest
func (r *CreateShowZoneRequest) Validate() (bool, string) {
	if r.Name == "" {
		return false, "Zone name is required"
	}
	if r.Price < 0 {
		return false, "Price must be greater than or equal to 0"
	}
	if r.TotalSeats <= 0 {
		return false, "Total seats must be greater than 0"
	}
	return true, ""
}

// UpdateShowZoneRequest represents the request to update a show zone
type UpdateShowZoneRequest struct {
	Name        string   `json:"name" binding:"omitempty,min=1,max=200"`
	Price       *float64 `json:"price" binding:"omitempty,gte=0"`
	TotalSeats  *int     `json:"total_seats" binding:"omitempty,gt=0"`
	Description string   `json:"description" binding:"omitempty,max=1000"`
	SortOrder   *int     `json:"sort_order" binding:"omitempty,gte=0"`
	IsActive    *bool    `json:"is_active" binding:"omitempty"`
}

// Validate validates the UpdateShowZoneRequest
func (r *UpdateShowZoneRequest) Validate() (bool, string) {
	if r.Name == "" && r.Price == nil && r.TotalSeats == nil && r.Description == "" && r.SortOrder == nil && r.IsActive == nil {
		return false, "At least one field must be provided for update"
	}
	if r.Price != nil && *r.Price < 0 {
		return false, "Price must be greater than or equal to 0"
	}
	if r.TotalSeats != nil && *r.TotalSeats <= 0 {
		return false, "Total seats must be greater than 0"
	}
	return true, ""
}

// ShowZoneResponse represents the response for a show zone
type ShowZoneResponse struct {
	ID             string  `json:"id"`
	ShowID         string  `json:"show_id"`
	Name           string  `json:"name"`
	Description    string  `json:"description"`
	Color          string  `json:"color"`
	Price          float64 `json:"price"`
	Currency       string  `json:"currency"`
	TotalSeats     int     `json:"total_seats"`
	AvailableSeats int     `json:"available_seats"`
	ReservedSeats  int     `json:"reserved_seats"`
	SoldSeats      int     `json:"sold_seats"`
	MinPerOrder    int     `json:"min_per_order"`
	MaxPerOrder    int     `json:"max_per_order"`
	IsActive       bool    `json:"is_active"`
	SortOrder      int     `json:"sort_order"`
	SaleStartAt    *string `json:"sale_start_at,omitempty"`
	SaleEndAt      *string `json:"sale_end_at,omitempty"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

// ShowZoneListResponse represents a list of show zones
type ShowZoneListResponse struct {
	Zones  []*ShowZoneResponse `json:"zones"`
	Total  int                 `json:"total"`
	Limit  int                 `json:"limit"`
	Offset int                 `json:"offset"`
}

// ShowZoneListFilter represents filters for listing show zones
type ShowZoneListFilter struct {
	ShowID   string `form:"-"`
	IsActive *bool  `form:"is_active"`
	Limit    int    `form:"limit"`
	Offset   int    `form:"offset"`
}

// SetDefaults sets default values for pagination
func (f *ShowZoneListFilter) SetDefaults() {
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 20
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
}
