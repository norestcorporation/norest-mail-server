package domains

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/norest-mail/server/internal/auth"
	"github.com/norest-mail/server/internal/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ListPlatformDomains returns all platform domains available for registration.
func (h *Handler) ListPlatformDomains(w http.ResponseWriter, r *http.Request) {
	domains, err := h.service.ListPlatformDomains(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to list platform domains")
		return
	}

	response.JSON(w, http.StatusOK, domains)
}

// CheckDomainByName checks if a domain exists and is available for registration.
func (h *Handler) CheckDomainByName(w http.ResponseWriter, r *http.Request) {
	domainName := chi.URLParam(r, "name")
	if domainName == "" {
		response.Error(w, http.StatusBadRequest, "domain name is required")
		return
	}

	domain, err := h.service.GetDomainByName(r.Context(), domainName)
	if err != nil {
		response.Error(w, http.StatusNotFound, "domain not found")
		return
	}

	// Check if domain is available for registration
	if domain.Status != string(StatusActive) {
		response.JSON(w, http.StatusOK, map[string]interface{}{
			"exists": true,
			"available": false,
			"reason": "domain is not active",
		})
		return
	}

	if !domain.RegistrationEnabled {
		response.JSON(w, http.StatusOK, map[string]interface{}{
			"exists": true,
			"available": false,
			"reason": "registration is not enabled for this domain",
		})
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"exists": true,
		"available": true,
		"domain": domain,
	})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateDomainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Check if this is a platform domain creation (admin only)
	if req.OwnershipType == string(OwnershipTypePlatform) {
		// Verify admin access
		isAdmin, ok := auth.IsAdminFromContext(r.Context())
		if !ok || !isAdmin {
			response.Error(w, http.StatusForbidden, "admin access required for platform domains")
			return
		}

		domain, err := h.service.CreatePlatformDomain(r.Context(), req.Name, req.OwnershipType, req.RegistrationEnabled)
		if err != nil {
			if err == ErrInvalidDomain {
				response.Error(w, http.StatusBadRequest, err.Error())
				return
			}
			if err == ErrDomainExists {
				response.Error(w, http.StatusConflict, err.Error())
				return
			}
			log.Printf("CreatePlatformDomain error: %v", err)
			response.Error(w, http.StatusInternalServerError, err.Error())
			return
		}

		response.JSON(w, http.StatusCreated, domain)
		return
	}

	// Regular user domain creation
	domain, err := h.service.CreateDomain(r.Context(), userID, req.Name)
	if err != nil {
		if err == ErrInvalidDomain {
			response.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		if err == ErrDomainExists {
			response.Error(w, http.StatusConflict, err.Error())
			return
		}
		// log error to console for debugging
		log.Printf("CreateDomain error: %v", err)
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, http.StatusCreated, domain)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	domains, err := h.service.ListDomains(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response.JSON(w, http.StatusOK, domains)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	domainID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid domain id")
		return
	}

	domain, err := h.service.GetDomain(r.Context(), domainID, userID)
	if err != nil {
		if err == ErrDomainNotFound {
			response.Error(w, http.StatusNotFound, "domain not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response.JSON(w, http.StatusOK, domain)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	domainID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid domain id")
		return
	}

	err = h.service.DeleteDomain(r.Context(), domainID, userID)
	if err != nil {
		if err == ErrDomainNotFound {
			response.Error(w, http.StatusNotFound, "domain not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) StartVerification(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	domainID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid domain id")
		return
	}

	domain, err := h.service.StartVerification(r.Context(), domainID, userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, domain)
}

func (h *Handler) GetVerification(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	domainID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid domain id")
		return
	}

	domain, err := h.service.GetDomain(r.Context(), domainID, userID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "domain not found")
		return
	}

	// If the domain has a temporary plaintext token (from verification start), use it
	// Otherwise, indicate the user needs to restart verification
	dnsValue := "norest-verification=TOKEN_FROM_START_VERIFICATION"
	message := "If you lost your verification token, restart verification to get a new one"
	
	if domain.VerificationToken != nil && *domain.VerificationToken != "" {
		dnsValue = "norest-verification=" + *domain.VerificationToken
		message = "Configure this TXT record in your DNS"
	}
	
	res := map[string]interface{}{
		"type":  "TXT",
		"name":  "_norest-verification." + domain.Name,
		"value": dnsValue,
		"status": domain.VerificationStatus,
		"message": message,
	}

	response.JSON(w, http.StatusOK, res)
}
