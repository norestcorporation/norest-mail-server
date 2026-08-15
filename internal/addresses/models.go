package addresses

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidLocalPart = errors.New("invalid local part")
	ErrAddressExists    = errors.New("address already exists")
)

type Status string

const (
	StatusAvailable Status = "AVAILABLE"
	StatusReserved   Status = "RESERVED"
	StatusClaimed    Status = "CLAIMED"
	StatusBlocked    Status = "BLOCKED"
)

type Address struct {
	ID            uuid.UUID  `json:"id"`
	DomainID      uuid.UUID  `json:"domain_id"`
	LocalPart     string     `json:"local_part"`
	Status        Status     `json:"status"`
	ReservedBy    *uuid.UUID `json:"reserved_by,omitempty"`
	ReservedAt    *time.Time `json:"reserved_at,omitempty"`
	ReservedUntil *time.Time `json:"reserved_until,omitempty"`
	ClaimedBy     *uuid.UUID `json:"claimed_by,omitempty"`
	ClaimedAt     *time.Time `json:"claimed_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

var localPartRegex = regexp.MustCompile(`^[a-z0-9\.\-\+]+$`)

func NormalizeLocalPart(localPart string) (string, error) {
	localPart = strings.TrimSpace(localPart)
	localPart = strings.ToLower(localPart)
	
	if localPart == "" {
		return "", ErrInvalidLocalPart
	}

	if strings.HasPrefix(localPart, ".") || strings.HasSuffix(localPart, ".") {
		return "", ErrInvalidLocalPart
	}

	if strings.Contains(localPart, "..") {
		return "", ErrInvalidLocalPart
	}

	if !localPartRegex.MatchString(localPart) {
		return "", ErrInvalidLocalPart
	}

	return localPart, nil
}

type CreateAddressRequest struct {
	LocalPart string `json:"local_part"`
}
