package registration

import (
	"context"

	"github.com/google/uuid"
	"github.com/norest-mail/server/internal/auth"
	"github.com/norest-mail/server/internal/domains"
)

type Service struct {
	authService    *auth.Service
	domainsService *domains.Service
}

func NewService(authService *auth.Service, domainsService *domains.Service) *Service {
	return &Service{
		authService:    authService,
		domainsService: domainsService,
	}
}

// GetRegistrationStatus retrieves the complete registration status for a user.
func (s *Service) GetRegistrationStatus(ctx context.Context, userID uuid.UUID) (*auth.RegistrationFlowResponse, error) {
	// Get user details
	user, err := s.authService.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Get user's domains
	userDomains, err := s.domainsService.ListDomains(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Determine registration status
	status := auth.RegistrationStatusPending
	var requiresAction *string
	var domainID *uuid.UUID
	var domainName *string
	domainVerified := false

	if len(userDomains) > 0 {
		domain := &userDomains[0]
		domainID = &domain.ID
		domainName = &domain.Name
		status = auth.RegistrationStatusDomainAdded
		action := "verify_domain"
		requiresAction = &action

		if domain.VerificationStatus == string(domains.VerificationVerified) {
			domainVerified = true
			status = auth.RegistrationStatusVerified
			action = "register_address"
			requiresAction = &action
		} else if domain.VerificationStatus == string(domains.VerificationVerifying) {
			status = auth.RegistrationStatusVerifying
			action = "wait_for_verification"
			requiresAction = &action
		}
	}

	return &auth.RegistrationFlowResponse{
		ID:             user.ID,
		Email:          user.Email,
		Status:         status,
		RequiresAction: requiresAction,
		DomainID:       domainID,
		DomainName:     domainName,
		DomainVerified: domainVerified,
		// AddressID and mailbox status would be populated by checking addresses/mailboxes tables
		AddressID:          nil,
		MailboxProvisioned: false,
		ReadyForMail:       false,
	}, nil
}