package models

import "time"

type UserStatus string

const (
	UserStatusActive  UserStatus = "active"
	UserStatusBlocked UserStatus = "blocked"
)

type UserRole string

const (
	UserRoleUser  UserRole = "user"
	UserRoleAdmin UserRole = "admin"
)

type User struct {
	ID          string     `json:"id"`
	Email       string     `json:"email"`
	DisplayName string     `json:"display_name"`
	Status      UserStatus `json:"status"`
	Role        UserRole   `json:"role"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// UserResponse is the public representation of a user
type UserResponse struct {
	ID          string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Email       string `json:"email" example:"user@example.com"`
	DisplayName string `json:"display_name" example:"John Doe"`
	Role        string `json:"role" example:"user"`
	Status      string `json:"status" example:"active"`
}

func (u *User) ToResponse() *UserResponse {
	return &UserResponse{
		ID:          u.ID,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		Role:        string(u.Role),
		Status:      string(u.Status),
	}
}
