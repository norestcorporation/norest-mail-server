package registration

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/norest-mail/server/internal/addresses"
	"github.com/norest-mail/server/internal/auth"
	"github.com/norest-mail/server/internal/domains"
	"github.com/norest-mail/server/internal/response"
)

type Handler struct {
	authService    *auth.Service
	domainsService *domains.Service
	addressesService *addresses.Service
	pool           *pgxpool.Pool
}

func NewHandler(authService *auth.Service, domainsService *domains.Service, addressesService *addresses.Service, pool *pgxpool.Pool) *Handler {
	return &Handler{
		authService:    authService,
		domainsService: domainsService,
		addressesService: addressesService,
		pool:           pool,
	}
}



// GetRegistrationStatus returns the current registration status for the user.
func (h *Handler) GetRegistrationStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get user details
	user, err := h.authService.GetUserByID(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to get user")
		return
	}

	// Extract domain from email
	domainName := auth.ExtractDomainFromEmail(user.Email)
	if domainName == "" {
		response.Error(w, http.StatusBadRequest, "invalid email format")
		return
	}

	// Determine domain type and get domain record
	domain, err := h.domainsService.GetDomainByName(r.Context(), domainName)
	domainType := auth.DomainType("")
	var domainID *uuid.UUID
	var domainVerified bool
	var status auth.RegistrationStatus
	var requiresAction *string
	var addressID *uuid.UUID
	var mailboxProvisioned bool
	var readyForMail bool

	if err != nil {
		// Domain doesn't exist in system
		domainType = auth.DomainTypeCustom
		status = auth.RegistrationStatusPending
		action := "add_domain"
		requiresAction = &action
	} else {
		// Domain exists in system
		if domain.OwnershipType == string(domains.OwnershipTypePlatform) {
			domainType = auth.DomainTypePlatform
			domainID = &domain.ID
			domainVerified = true // Platform domains are pre-verified
		} else {
			domainType = auth.DomainTypeCustom
			domainID = &domain.ID
			domainVerified = domain.VerificationStatus == string(domains.VerificationVerified)
		}

		// Determine status based on domain state
		if domainType == auth.DomainTypePlatform {
			// Platform domain flow
			status = auth.RegistrationStatusProvisioning
			requiresAction = nil // No user action needed
		} else {
			// Custom domain flow
			switch domain.VerificationStatus {
			case string(domains.VerificationPending):
				status = auth.RegistrationStatusDomainAdded
				action := "start_verification"
				requiresAction = &action
			case string(domains.VerificationVerifying):
				status = auth.RegistrationStatusVerifying
				action := "wait_for_verification"
				requiresAction = &action
			case string(domains.VerificationVerified):
				status = auth.RegistrationStatusVerified
				action := "register_address"
				requiresAction = &action
			case string(domains.VerificationFailed):
				status = auth.RegistrationStatusDomainAdded
				action := "retry_verification"
				requiresAction = &action
			default:
				status = auth.RegistrationStatusPending
				action := "add_domain"
				requiresAction = &action
			}
		}
	}

	// Check for address and mailbox if domain is verified
	if domainVerified && domainID != nil {
		// Extract local part from email
		atIndex := strings.LastIndex(user.Email, "@")
		if atIndex != -1 {
			localPart := user.Email[:atIndex]
			
			// Check if address exists
			address, err := h.addressesService.GetAddressByDomainAndLocalPart(r.Context(), *domainID, localPart)
			if err == nil {
				addressID = &address.ID
				
				// Check if address is claimed
				if address.Status == addresses.StatusClaimed {
					mailboxProvisioned = true
					
					// Check mailbox status and initial sync completion
					// Also check user status to ensure lifecycle consistency
					var mailboxStatus string
					var initialSyncCompleted bool
					var userStatus string
					err := h.pool.QueryRow(r.Context(),
						`SELECT m.status, m.initial_sync_checkpoint IS NOT NULL as initial_sync_completed, u.status
						 FROM mailboxes m
						 JOIN addresses a ON m.address_id = a.id
						 JOIN users u ON a.claimed_by = u.id
						 WHERE m.address_id = $1`,
						address.ID,
					).Scan(&mailboxStatus, &initialSyncCompleted, &userStatus)
					
					if err == nil {
						// Registration is ACTIVE only when all conditions are met:
						// 1. User status = active (worker transitioned it)
						// 2. Mailbox status = active (worker activated it)
						// 3. Initial sync checkpoint persisted (worker completed sync)
						if userStatus == "active" && mailboxStatus == "active" && initialSyncCompleted {
							status = auth.RegistrationStatusActive
							readyForMail = true
						} else {
							status = auth.RegistrationStatusProvisioning
							readyForMail = false
						}
					} else {
						// If mailbox not found, still in provisioning
						status = auth.RegistrationStatusProvisioning
						readyForMail = false
					}
				}
			}
		}
	}

	flowRes := &auth.RegistrationFlowResponse{
		ID:                 user.ID,
		Email:              user.Email,
		DomainType:         domainType,
		Status:             status,
		RequiresAction:     requiresAction,
		DomainID:           domainID,
		DomainName:         &domainName,
		DomainVerified:     domainVerified,
		AddressID:          addressID,
		MailboxProvisioned: mailboxProvisioned,
		ReadyForMail:       readyForMail,
	}

	response.JSON(w, http.StatusOK, flowRes)
}

// InitiateDomainVerification starts the domain verification process.
func (h *Handler) InitiateDomainVerification(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	domainID, err := uuid.Parse(chi.URLParam(r, "domainID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid domain id")
		return
	}

	// Start verification
	domain, err := h.domainsService.StartVerification(r.Context(), domainID, userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Return verification instructions
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"domain_id":   domain.ID,
		"domain_name": domain.Name,
		"status":      domain.VerificationStatus,
		"verification_token": domain.VerificationToken,
		"dns_record": map[string]interface{}{
			"type":  "TXT",
			"name":  "_norest-verification." + domain.Name,
			"value": "norest-verification=" + *domain.VerificationToken,
		},
		"message": "Configure this TXT record in your DNS provider. The background worker will verify automatically.",
	})
}

// CheckDomainVerificationStatus checks the current domain verification status.
func (h *Handler) CheckDomainVerificationStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	domainID, err := uuid.Parse(chi.URLParam(r, "domainID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid domain id")
		return
	}

	// Get domain
	domain, err := h.domainsService.GetDomain(r.Context(), domainID, userID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "domain not found")
		return
	}

	// Perform DNS check to give real-time status
	dnsChecker := domains.NewDNSChecker()
	var dnsResult *domains.DNSVerificationResult
	if domain.VerificationTokenHash != nil {
		dnsResult, _ = dnsChecker.PerformFullVerification(r.Context(), domain.Name, *domain.VerificationTokenHash)
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"domain_id":            domain.ID,
		"domain_name":          domain.Name,
		"verification_status":  domain.VerificationStatus,
		"registration_enabled": domain.RegistrationEnabled,
		"dns_check":            dnsResult,
		"next_action": func() string {
			switch domain.VerificationStatus {
			case string(domains.VerificationPending):
				return "start_verification"
			case string(domains.VerificationVerifying):
				return "wait_for_dns_propagation"
			case string(domains.VerificationVerified):
				return "register_address"
			case string(domains.VerificationFailed):
				return "retry_verification"
			default:
				return "unknown"
			}
		}(),
	})
}