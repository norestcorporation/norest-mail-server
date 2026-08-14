package mail

import (
	"context"
	"fmt"

	"github.com/norest-mail/server/internal/stalwart"
)

// Service provides mail authorization and session bridging.
type Service struct {
	db       DB
	stalwart *stalwart.Client
}

// DB defines the minimal interface the mail service needs from the database.
type DB interface {
	GetMailboxByUserID(ctx context.Context, userID string) (Mailbox, error)
	GetAddressByID(ctx context.Context, id string) (Address, error)
}

// Mailbox represents a provisioned mailbox.
type Mailbox struct {
	ID                string
	AddressID         string
	Status            string
	StalwartAccountID *string
}

// Address represents a provisioned address.
type Address struct {
	ID        string
	LocalPart string
	DomainID  string
	Status    string
}

func NewService(db DB, client *stalwart.Client) *Service {
	return &Service{
		db:       db,
		stalwart: client,
	}
}

// MailSessionResponse represents the secure mail session artifact returned to the frontend.
type MailSessionResponse struct {
	Provider       string `json:"provider"`
	JMAPSessionURL string `json:"jmap_session_url"`
	AccessToken    string `json:"access_token"`
	AccountID      string `json:"account_id"`
}

// CreateMailSession verifies the user and mailbox, then provisions a short-lived AppPassword
// via Stalwart JMAP acting as the user's mail access token.
func (s *Service) CreateMailSession(ctx context.Context, userID string, stalwartHost string) (*MailSessionResponse, error) {
	// 1. Verify user's mailbox exists and is active.
	mailbox, err := s.db.GetMailboxByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("getting mailbox: %w", err)
	}

	if mailbox.Status != "active" {
		return nil, fmt.Errorf("mailbox is not active (status: %s)", mailbox.Status)
	}
	
	if mailbox.StalwartAccountID == nil || *mailbox.StalwartAccountID == "" {
		return nil, fmt.Errorf("mailbox not fully provisioned in stalwart")
	}

	// 2. Provision a short-lived user mail token (AppPassword) in Stalwart.
	// We generate an AppPassword programmatically for this session.
	desc := fmt.Sprintf("Norest Session Token - %s", userID)
	secret, err := s.stalwart.CreateAppPassword(ctx, *mailbox.StalwartAccountID, desc)
	if err != nil {
		return nil, fmt.Errorf("provisioning mail token: %w", err)
	}

	jmapURL := fmt.Sprintf("%s/.well-known/jmap", stalwartHost)

	return &MailSessionResponse{
		Provider:       "stalwart",
		JMAPSessionURL: jmapURL,
		AccessToken:    secret,
		AccountID:      *mailbox.StalwartAccountID,
	}, nil
}
