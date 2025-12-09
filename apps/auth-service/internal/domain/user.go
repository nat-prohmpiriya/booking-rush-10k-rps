package domain

import (
	"time"
)

// Role represents user role
type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

// User represents a user entity
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // Never serialize password
	Name         string    `json:"name"`
	Role         Role      `json:"role"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Session represents a user session
type Session struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	RefreshToken string    `json:"refresh_token"`
	UserAgent    string    `json:"user_agent"`
	IP           string    `json:"ip"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
}

// TokenPair represents access and refresh tokens
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // seconds until access token expires
}

// Claims represents JWT claims
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   Role   `json:"role"`
}
