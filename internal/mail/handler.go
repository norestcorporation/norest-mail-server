package mail

import (
	"net/http"

	"github.com/norest-mail/server/internal/auth"
	"github.com/norest-mail/server/internal/response"
)

type Handler struct {
	service      *Service
	stalwartHost string
}

func NewHandler(service *Service, stalwartHost string) *Handler {
	return &Handler{
		service:      service,
		stalwartHost: stalwartHost,
	}
}

func (h *Handler) CreateSession(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	res, err := h.service.CreateMailSession(r.Context(), userID.String(), h.stalwartHost)
	if err != nil {
		// Provide specific error messages for different failure scenarios
		if err.Error() == "mailbox is not active" || err.Error() == "mailbox not fully provisioned in stalwart" {
			response.Error(w, http.StatusServiceUnavailable, "mailbox not ready - please wait for provisioning to complete")
			return
		}
		// Do not expose internal details about stalwart in production, but we return a generic error
		response.Error(w, http.StatusInternalServerError, "failed to create mail session")
		return
	}

	response.JSON(w, http.StatusOK, res)
}

func (h *Handler) GetAccount(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	mailbox, err := h.service.db.GetMailboxByUserID(r.Context(), userID.String())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to get mailbox")
		return
	}

	res := map[string]any{
		"id":                 mailbox.ID,
		"address_id":         mailbox.AddressID,
		"status":             mailbox.Status,
		"stalwart_account_id": mailbox.StalwartAccountID,
	}

	response.JSON(w, http.StatusOK, res)
}

func (h *Handler) GetProvisioningStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	status, err := h.service.GetProvisioningStatus(r.Context(), userID.String())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to get provisioning status")
		return
	}

	response.JSON(w, http.StatusOK, status)
}
