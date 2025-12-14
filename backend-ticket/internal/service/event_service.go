package service

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/dto"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/repository"
)

// Common errors
var (
	ErrEventNotFound      = errors.New("event not found")
	ErrEventAlreadyExists = errors.New("event with this slug already exists")
	ErrInvalidEventStatus = errors.New("invalid event status transition")
	ErrUnauthorized       = errors.New("unauthorized to perform this action")
)

// eventService implements EventService
type eventService struct {
	eventRepo repository.EventRepository
}

// NewEventService creates a new EventService
func NewEventService(eventRepo repository.EventRepository) EventService {
	return &eventService{
		eventRepo: eventRepo,
	}
}

// CreateEvent creates a new event
func (s *eventService) CreateEvent(ctx context.Context, req *dto.CreateEventRequest) (*domain.Event, error) {
	// Validate request
	if valid, msg := req.Validate(); !valid {
		return nil, errors.New(msg)
	}

	// Generate slug from name
	slug := generateSlug(req.Name)

	// Ensure slug is unique
	slug, err := s.ensureUniqueSlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	maxTickets := 10
	if req.MaxTicketsPerUser > 0 {
		maxTickets = req.MaxTicketsPerUser
	}

	event := &domain.Event{
		ID:                uuid.New().String(),
		TenantID:          req.TenantID,
		OrganizerID:       req.OrganizerID,
		CategoryID:        req.CategoryID,
		Name:              req.Name,
		Slug:              slug,
		Description:       req.Description,
		ShortDescription:  req.ShortDescription,
		PosterURL:         req.PosterURL,
		BannerURL:         req.BannerURL,
		Gallery:           req.Gallery,
		VenueName:         req.VenueName,
		VenueAddress:      req.VenueAddress,
		City:              req.City,
		Country:           req.Country,
		Latitude:          req.Latitude,
		Longitude:         req.Longitude,
		MaxTicketsPerUser: maxTickets,
		BookingStartAt:    req.BookingStartAt,
		BookingEndAt:      req.BookingEndAt,
		Status:            domain.EventStatusDraft,
		IsFeatured:        false,
		IsPublic:          true,
		MetaTitle:         req.MetaTitle,
		MetaDescription:   req.MetaDescription,
		Settings:          "{}",
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if event.Gallery == nil {
		event.Gallery = []string{}
	}

	if err := s.eventRepo.Create(ctx, event); err != nil {
		return nil, err
	}

	return event, nil
}

// GetEventByID retrieves an event by ID
func (s *eventService) GetEventByID(ctx context.Context, id string) (*domain.Event, error) {
	event, err := s.eventRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if event == nil {
		return nil, ErrEventNotFound
	}
	return event, nil
}

// GetEventBySlug retrieves an event by slug
func (s *eventService) GetEventBySlug(ctx context.Context, slug string) (*domain.Event, error) {
	event, err := s.eventRepo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if event == nil {
		return nil, ErrEventNotFound
	}
	return event, nil
}

// ListEvents lists events with filters and pagination
func (s *eventService) ListEvents(ctx context.Context, filter *dto.EventListFilter) ([]*domain.Event, int, error) {
	filter.SetDefaults()

	repoFilter := &repository.EventFilter{
		Status:      filter.Status,
		TenantID:    filter.TenantID,
		OrganizerID: filter.OrganizerID,
		CategoryID:  filter.CategoryID,
		City:        filter.City,
		Search:      filter.Search,
	}

	return s.eventRepo.List(ctx, repoFilter, filter.Limit, filter.Offset)
}

// ListPublishedEvents lists all published public events
func (s *eventService) ListPublishedEvents(ctx context.Context, limit, offset int) ([]*domain.Event, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.eventRepo.ListPublished(ctx, limit, offset)
}

// UpdateEvent updates an event
func (s *eventService) UpdateEvent(ctx context.Context, id string, req *dto.UpdateEventRequest) (*domain.Event, error) {
	// Validate request
	if valid, msg := req.Validate(); !valid {
		return nil, errors.New(msg)
	}

	// Get existing event
	event, err := s.eventRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if event == nil {
		return nil, ErrEventNotFound
	}

	// Update fields
	if req.Name != "" {
		event.Name = req.Name
		// Regenerate slug if name changed
		slug := generateSlug(req.Name)
		slug, err = s.ensureUniqueSlugExcluding(ctx, slug, event.ID)
		if err != nil {
			return nil, err
		}
		event.Slug = slug
	}
	if req.Description != "" {
		event.Description = req.Description
	}
	if req.ShortDescription != "" {
		event.ShortDescription = req.ShortDescription
	}
	if req.CategoryID != nil {
		event.CategoryID = req.CategoryID
	}
	if req.PosterURL != "" {
		event.PosterURL = req.PosterURL
	}
	if req.BannerURL != "" {
		event.BannerURL = req.BannerURL
	}
	if req.Gallery != nil {
		event.Gallery = req.Gallery
	}
	if req.VenueName != "" {
		event.VenueName = req.VenueName
	}
	if req.VenueAddress != "" {
		event.VenueAddress = req.VenueAddress
	}
	if req.City != "" {
		event.City = req.City
	}
	if req.Country != "" {
		event.Country = req.Country
	}
	if req.Latitude != nil {
		event.Latitude = req.Latitude
	}
	if req.Longitude != nil {
		event.Longitude = req.Longitude
	}
	if req.MaxTicketsPerUser != nil {
		event.MaxTicketsPerUser = *req.MaxTicketsPerUser
	}
	if req.BookingStartAt != nil {
		event.BookingStartAt = req.BookingStartAt
	}
	if req.BookingEndAt != nil {
		event.BookingEndAt = req.BookingEndAt
	}
	if req.IsFeatured != nil {
		event.IsFeatured = *req.IsFeatured
	}
	if req.IsPublic != nil {
		event.IsPublic = *req.IsPublic
	}
	if req.Status != nil {
		// Validate status
		validStatuses := map[string]bool{
			domain.EventStatusDraft:     true,
			domain.EventStatusPublished: true,
			domain.EventStatusCancelled: true,
			domain.EventStatusCompleted: true,
		}
		if !validStatuses[*req.Status] {
			return nil, ErrInvalidEventStatus
		}
		event.Status = *req.Status
		// Set published_at timestamp when status changes to published
		if *req.Status == domain.EventStatusPublished && event.PublishedAt == nil {
			now := time.Now()
			event.PublishedAt = &now
		}
	}
	if req.MetaTitle != "" {
		event.MetaTitle = req.MetaTitle
	}
	if req.MetaDescription != "" {
		event.MetaDescription = req.MetaDescription
	}

	if err := s.eventRepo.Update(ctx, event); err != nil {
		return nil, err
	}

	return event, nil
}

// DeleteEvent soft deletes an event
func (s *eventService) DeleteEvent(ctx context.Context, id string) error {
	// Check event exists
	event, err := s.eventRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if event == nil {
		return ErrEventNotFound
	}

	return s.eventRepo.Delete(ctx, id)
}

// PublishEvent publishes an event
func (s *eventService) PublishEvent(ctx context.Context, id string) (*domain.Event, error) {
	event, err := s.eventRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if event == nil {
		return nil, ErrEventNotFound
	}

	// Only draft events can be published
	if event.Status != domain.EventStatusDraft {
		return nil, ErrInvalidEventStatus
	}

	event.Status = domain.EventStatusPublished
	now := time.Now()
	event.PublishedAt = &now

	if err := s.eventRepo.Update(ctx, event); err != nil {
		return nil, err
	}

	return event, nil
}

// generateSlug generates a URL-friendly slug from a string
func generateSlug(s string) string {
	// Convert to lowercase
	s = strings.ToLower(s)

	// Replace spaces and special characters with hyphens
	var builder strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
		} else if r == ' ' || r == '-' || r == '_' {
			builder.WriteRune('-')
		}
	}
	slug := builder.String()

	// Remove consecutive hyphens
	re := regexp.MustCompile(`-+`)
	slug = re.ReplaceAllString(slug, "-")

	// Trim hyphens from start and end
	slug = strings.Trim(slug, "-")

	return slug
}

// ensureUniqueSlug ensures the slug is unique by appending a number if needed
func (s *eventService) ensureUniqueSlug(ctx context.Context, slug string) (string, error) {
	baseSlug := slug
	counter := 1

	for {
		exists, err := s.eventRepo.SlugExists(ctx, slug)
		if err != nil {
			return "", err
		}
		if !exists {
			return slug, nil
		}
		counter++
		slug = baseSlug + "-" + string(rune('0'+counter%10))
		if counter > 10 {
			// Use UUID suffix for high collision scenarios
			slug = baseSlug + "-" + uuid.New().String()[:8]
			return slug, nil
		}
	}
}

// ensureUniqueSlugExcluding ensures the slug is unique excluding the current event
func (s *eventService) ensureUniqueSlugExcluding(ctx context.Context, slug string, excludeID string) (string, error) {
	baseSlug := slug
	counter := 1

	for {
		event, err := s.eventRepo.GetBySlug(ctx, slug)
		if err != nil {
			return "", err
		}
		if event == nil || event.ID == excludeID {
			return slug, nil
		}
		counter++
		slug = baseSlug + "-" + string(rune('0'+counter%10))
		if counter > 10 {
			slug = baseSlug + "-" + uuid.New().String()[:8]
			return slug, nil
		}
	}
}
