package users

import (
	"time"

	"github.com/google/uuid"
)

// Status represents the lifecycle state of a Norest user.
type Status string

const (
	StatusPending   Status = "pending"
	StatusActive    Status = "active"
	StatusSuspended Status = "suspended"
	StatusDisabled  Status = "disabled"
)

// User represents a Norest platform user.
// This is a product/control-plane entity — not a Stalwart mail user.
type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // never serialized
	Status       Status    `json:"status"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
