package dto

import "testing"

func TestRegisterRequest_ValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		want     bool
		wantMsg  string
	}{
		{
			name:     "valid password",
			password: "Password1!",
			want:     true,
			wantMsg:  "",
		},
		{
			name:     "valid complex password",
			password: "MyP@ssw0rd123!",
			want:     true,
			wantMsg:  "",
		},
		{
			name:     "too short",
			password: "Pass1!",
			want:     false,
			wantMsg:  "Password must be at least 8 characters",
		},
		{
			name:     "no uppercase",
			password: "password1!",
			want:     false,
			wantMsg:  "Password must contain at least one uppercase letter",
		},
		{
			name:     "no lowercase",
			password: "PASSWORD1!",
			want:     false,
			wantMsg:  "Password must contain at least one lowercase letter",
		},
		{
			name:     "no digit",
			password: "Password!",
			want:     false,
			wantMsg:  "Password must contain at least one digit",
		},
		{
			name:     "no special character",
			password: "Password1",
			want:     false,
			wantMsg:  "Password must contain at least one special character",
		},
		{
			name:     "only lowercase",
			password: "password",
			want:     false,
			wantMsg:  "Password must contain at least one uppercase letter",
		},
		{
			name:     "only numbers",
			password: "12345678",
			want:     false,
			wantMsg:  "Password must contain at least one uppercase letter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &RegisterRequest{Password: tt.password}
			got, msg := req.ValidatePassword()
			if got != tt.want {
				t.Errorf("ValidatePassword() got = %v, want %v", got, tt.want)
			}
			if msg != tt.wantMsg {
				t.Errorf("ValidatePassword() msg = %v, want %v", msg, tt.wantMsg)
			}
		})
	}
}

func TestRegisterRequest_ValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		want    bool
		wantMsg string
	}{
		{
			name:    "valid email",
			email:   "test@example.com",
			want:    true,
			wantMsg: "",
		},
		{
			name:    "valid email with subdomain",
			email:   "test@mail.example.com",
			want:    true,
			wantMsg: "",
		},
		{
			name:    "valid email with plus",
			email:   "test+tag@example.com",
			want:    true,
			wantMsg: "",
		},
		{
			name:    "valid email with dots",
			email:   "test.user@example.com",
			want:    true,
			wantMsg: "",
		},
		{
			name:    "invalid - no @",
			email:   "testexample.com",
			want:    false,
			wantMsg: "Invalid email format",
		},
		{
			name:    "invalid - no domain",
			email:   "test@",
			want:    false,
			wantMsg: "Invalid email format",
		},
		{
			name:    "invalid - no TLD",
			email:   "test@example",
			want:    false,
			wantMsg: "Invalid email format",
		},
		{
			name:    "invalid - double @",
			email:   "test@@example.com",
			want:    false,
			wantMsg: "Invalid email format",
		},
		{
			name:    "invalid - spaces",
			email:   "test @example.com",
			want:    false,
			wantMsg: "Invalid email format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &RegisterRequest{Email: tt.email}
			got, msg := req.ValidateEmail()
			if got != tt.want {
				t.Errorf("ValidateEmail() got = %v, want %v", got, tt.want)
			}
			if msg != tt.wantMsg {
				t.Errorf("ValidateEmail() msg = %v, want %v", msg, tt.wantMsg)
			}
		})
	}
}
