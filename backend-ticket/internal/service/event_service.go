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
	ErrVenueNotFound      = errors.New("venue not found")
)

// eventService implements EventService
type eventService struct {
	eventRepo repository.EventRepository
	venueRepo repository.VenueRepository
}

// NewEventService creates a new EventService
func NewEventService(eventRepo repository.EventRepository, venueRepo repository.VenueRepository) EventService {
	return &eventService{
		eventRepo: eventRepo,
		venueRepo: venueRepo,
	}
}

// CreateEvent creates a new event
func (s *eventService) CreateEvent(ctx context.Context, req *dto.CreateEventRequest) (*domain.Event, error) {
	// Validate request
	if valid, msg := req.Validate(); !valid {
		return nil, errors.New(msg)
	}

	// Verify venue exists
	venue, err := s.venueRepo.GetByID(ctx, req.VenueID)
	if err != nil {
		return nil, err
	}
	if venue == nil {
		return nil, ErrVenueNotFound
	}

	// Generate slug from name
	slug := generateSlug(req.Name)

	// Ensure slug is unique
	slug, err = s.ensureUniqueSlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	event := &domain.Event{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Slug:        slug,
		Description: req.Description,
		VenueID:     req.VenueID,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
		Status:      domain.EventStatusDraft,
		TenantID:    req.TenantID,
		CreatedAt:   now,
		UpdatedAt:   now,
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
		Status:   filter.Status,
		TenantID: filter.TenantID,
		VenueID:  filter.VenueID,
		Search:   filter.Search,
	}

	return s.eventRepo.List(ctx, repoFilter, filter.Limit, filter.Offset)
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
	if !req.StartTime.IsZero() {
		event.StartTime = req.StartTime
	}
	if !req.EndTime.IsZero() {
		event.EndTime = req.EndTime
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
