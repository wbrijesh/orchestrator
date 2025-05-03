package database

import (
	"time"

	"github.com/google/uuid"
)

// User holds a row from users.
type User struct {
	ID           uuid.UUID
	Email        string
	FirstName    string
	LastName     string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type AuthRequest struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Password  string `json:"password"`
}

// APIResponse provides a consistent structure for all API responses
type APIResponse struct {
	Error string      `json:"error"`
	Data  interface{} `json:"data"`
}

type AuthResponse struct {
	Token string    `json:"token"`
	User  *UserView `json:"user"`
}

// UserView hides the password hash.
type UserView struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}
