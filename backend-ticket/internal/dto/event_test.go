package dto

import (
	"testing"
	"time"
)

func TestCreateEventRequest_Validate(t *testing.T) {
	futureTime := time.Now().Add(24 * time.Hour)
	futureEndTime := time.Now().Add(48 * time.Hour)

	tests := []struct {
		name    string
		req     CreateEventRequest
		want    bool
		wantMsg string
	}{
		{
			name: "valid request",
			req: CreateEventRequest{
				Name: "Concert",
			},
			want:    true,
			wantMsg: "",
		},
		{
			name: "valid request with booking times",
			req: CreateEventRequest{
				Name:           "Concert",
				BookingStartAt: &futureTime,
				BookingEndAt:   &futureEndTime,
			},
			want:    true,
			wantMsg: "",
		},
		{
			name:    "missing name",
			req:     CreateEventRequest{},
			want:    false,
			wantMsg: "Event name is required",
		},
		{
			name: "negative max_tickets_per_user",
			req: CreateEventRequest{
				Name:              "Concert",
				MaxTicketsPerUser: -1,
			},
			want:    false,
			wantMsg: "Max tickets per user cannot be negative",
		},
		{
			name: "booking_end_at before booking_start_at",
			req: CreateEventRequest{
				Name:           "Concert",
				BookingStartAt: &futureEndTime,
				BookingEndAt:   &futureTime,
			},
			want:    false,
			wantMsg: "Booking end time must be after booking start time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, msg := tt.req.Validate()
			if got != tt.want {
				t.Errorf("Validate() got = %v, want %v", got, tt.want)
			}
			if msg != tt.wantMsg {
				t.Errorf("Validate() msg = %v, want %v", msg, tt.wantMsg)
			}
		})
	}
}

func TestUpdateEventRequest_Validate(t *testing.T) {
	futureTime := time.Now().Add(24 * time.Hour)
	futureEndTime := time.Now().Add(48 * time.Hour)
	negativeValue := -1

	tests := []struct {
		name    string
		req     UpdateEventRequest
		want    bool
		wantMsg string
	}{
		{
			name: "valid name update",
			req: UpdateEventRequest{
				Name: "Updated Concert",
			},
			want:    true,
			wantMsg: "",
		},
		{
			name: "valid description update",
			req: UpdateEventRequest{
				Description: "New description",
			},
			want:    true,
			wantMsg: "",
		},
		{
			name: "valid time update",
			req: UpdateEventRequest{
				BookingStartAt: &futureTime,
				BookingEndAt:   &futureEndTime,
			},
			want:    true,
			wantMsg: "",
		},
		{
			name:    "empty request is valid",
			req:     UpdateEventRequest{},
			want:    true,
			wantMsg: "",
		},
		{
			name: "booking_end_at before booking_start_at",
			req: UpdateEventRequest{
				BookingStartAt: &futureEndTime,
				BookingEndAt:   &futureTime,
			},
			want:    false,
			wantMsg: "Booking end time must be after booking start time",
		},
		{
			name: "negative max_tickets_per_user",
			req: UpdateEventRequest{
				MaxTicketsPerUser: &negativeValue,
			},
			want:    false,
			wantMsg: "Max tickets per user cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, msg := tt.req.Validate()
			if got != tt.want {
				t.Errorf("Validate() got = %v, want %v", got, tt.want)
			}
			if msg != tt.wantMsg {
				t.Errorf("Validate() msg = %v, want %v", msg, tt.wantMsg)
			}
		})
	}
}
