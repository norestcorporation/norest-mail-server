package addresses

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/norest-mail/server/internal/auth"
	"github.com/norest-mail/server/internal/domains"
	"github.com/norest-mail/server/internal/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// CheckAvailability checks if an address is available for reservation.
func (h *Handler) CheckAvailability(w http.ResponseWriter, r *http.Request) {
	domainID, err := uuid.Parse(chi.URLParam(r, "domainID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid domain id")
		return
	}

	localPart := chi.URLParam(r, "localPart")
	if localPart == "" {
		response.Error(w, http.StatusBadRequest, "local part is required")
		return
	}

	available, err := h.service.CheckAddressAvailability(r.Context(), domainID, localPart)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to check availability")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"available": available,
		"local_part": localPart,
		"domain_id": domainID,
	})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
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

	var req CreateAddressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	address, err := h.service.CreateAddress(r.Context(), userID, domainID, req.LocalPart)
	if err != nil {
		if err == ErrInvalidLocalPart {
			response.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		if err == ErrAddressExists {
			response.Error(w, http.StatusConflict, err.Error())
			return
		}
		if err == domains.ErrDomainNotFound {
			response.Error(w, http.StatusNotFound, "domain not found")
			return
		}
		if err.Error() == "quota exceeded" {
			response.Error(w, http.StatusForbidden, err.Error())
			return
		}
		if err.Error() == "account suspended" {
			response.Error(w, http.StatusForbidden, err.Error())
			return
		}
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response.JSON(w, http.StatusCreated, address)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
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

	addressesList, err := h.service.ListAddresses(r.Context(), userID, domainID)
	if err != nil {
		if err == domains.ErrDomainNotFound {
			response.Error(w, http.StatusNotFound, "domain not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response.JSON(w, http.StatusOK, addressesList)
}
