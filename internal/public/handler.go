package public

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/norest-mail/server/internal/addresses"
	"github.com/norest-mail/server/internal/auth"
	domainspkg "github.com/norest-mail/server/internal/domains"
	"github.com/norest-mail/server/internal/response"
)

type Handler struct {
	domainsService  *domainspkg.Service
	addressesService *addresses.Service
	authService     *auth.Service
}

func NewHandler(domainsService *domainspkg.Service, addressesService *addresses.Service, authService *auth.Service) *Handler {
	return &Handler{
		domainsService:  domainsService,
		addressesService: addressesService,
		authService:     authService,
	}
}

// ListPlatformDomains returns all platform-owned domains available for public registration.
// This endpoint does not require authentication.
func (h *Handler) ListPlatformDomains(w http.ResponseWriter, r *http.Request) {
	domains, err := h.domainsService.ListPlatformDomains(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to list platform domains")
		return
	}

	// Filter to only return active domains with registration enabled
	var availableDomains []domainspkg.Domain
	for _, domain := range domains {
		if domain.Status == "active" && domain.RegistrationEnabled {
			availableDomains = append(availableDomains, domain)
		}
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"domains": availableDomains,
		},
	})
}

// CheckUsernameAvailability checks if a username is available for a specific domain.
// This endpoint does not require authentication.
func (h *Handler) CheckUsernameAvailability(w http.ResponseWriter, r *http.Request) {
	domainName := chi.URLParam(r, "domainName")
	username := chi.URLParam(r, "username")

	if domainName == "" {
		response.Error(w, http.StatusBadRequest, "domain name is required")
		return
	}

	if username == "" {
		response.Error(w, http.StatusBadRequest, "username is required")
		return
	}

	// Get domain by name
	domain, err := h.domainsService.GetDomainByName(r.Context(), domainName)
	if err != nil {
		response.Error(w, http.StatusNotFound, "domain not found")
		return
	}

	// Check if domain is available for registration
	if domain.Status != "active" {
		response.JSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"available": false,
				"reason":    "domain is not active",
			},
		})
		return
	}

	if !domain.RegistrationEnabled {
		response.JSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"available": false,
				"reason":    "registration is not enabled for this domain",
			},
		})
		return
	}

	// Check address availability
	available, err := h.addressesService.CheckAddressAvailability(r.Context(), domain.ID, username)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to check username availability")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"available": available,
			"username":  username,
			"domain":    domainName,
		},
	})
}

// ReserveUsername reserves a username for a specific domain.
// This endpoint does not require authentication but returns a reservation ID.
func (h *Handler) ReserveUsername(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Domain   string `json:"domain"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Username == "" {
		response.Error(w, http.StatusBadRequest, "username is required")
		return
	}

	if req.Domain == "" {
		response.Error(w, http.StatusBadRequest, "domain is required")
		return
	}

	// Get domain by name
	domain, err := h.domainsService.GetDomainByName(r.Context(), req.Domain)
	if err != nil {
		response.Error(w, http.StatusNotFound, "domain not found")
		return
	}

	// Check if domain is available for registration
	if domain.Status != "active" || !domain.RegistrationEnabled {
		response.Error(w, http.StatusBadRequest, "domain is not available for registration")
		return
	}

	// Check username availability
	available, err := h.addressesService.CheckAddressAvailability(r.Context(), domain.ID, req.Username)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to check username availability")
		return
	}

	if !available {
		response.Error(w, http.StatusConflict, "username is not available")
		return
	}

	// Generate a reservation ID (in a real implementation, this would be stored in database)
	reservationID := generateReservationID()

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"reservation_id": reservationID,
			"username":       req.Username,
			"domain":         req.Domain,
			"expires_in":     300, // 5 minutes
		},
	})
}

// RegisterUser registers a new user with the provided credentials and reserved username.
// This endpoint does not require authentication.
func (h *Handler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username     string `json:"username"`
		Domain       string `json:"domain"`
		ReservationID string `json:"reservation_id"`
		DisplayName  string `json:"display_name"`
		Password     string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate required fields
	if req.Username == "" {
		response.Error(w, http.StatusBadRequest, "username is required")
		return
	}
	if req.Domain == "" {
		response.Error(w, http.StatusBadRequest, "domain is required")
		return
	}
	if req.ReservationID == "" {
		response.Error(w, http.StatusBadRequest, "reservation_id is required")
		return
	}
	if req.DisplayName == "" {
		response.Error(w, http.StatusBadRequest, "display_name is required")
		return
	}
	if req.Password == "" {
		response.Error(w, http.StatusBadRequest, "password is required")
		return
	}

	// Validate password strength
	if len(req.Password) < 8 {
		response.Error(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	// Get domain by name
	domain, err := h.domainsService.GetDomainByName(r.Context(), req.Domain)
	if err != nil {
		response.Error(w, http.StatusNotFound, "domain not found")
		return
	}

	// Check if domain is available for registration
	if domain.Status != "active" || !domain.RegistrationEnabled {
		response.Error(w, http.StatusBadRequest, "domain is not available for registration")
		return
	}

	// Construct email address
	email := req.Username + "@" + req.Domain

	// Register the user (this will use the existing auth service)
	authRes, err := h.authService.Register(r.Context(), email, req.Password)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "registration failed")
		return
	}

	// In a real implementation, we would also:
	// 1. Verify the reservation ID is valid and not expired
	// 2. Create the email address for the user
	// 3. Associate the address with the user's account

	response.JSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"user_id":       authRes.ID,
			"email":         authRes.Email,
			"display_name":  req.DisplayName,
			"access_token":  authRes.AccessToken,
			"refresh_token": authRes.RefreshToken,
			"expires_in":    authRes.ExpiresIn,
		},
	})
}

// generateReservationID generates a unique reservation ID
func generateReservationID() string {
	return "res_" + uuid.New().String()
}