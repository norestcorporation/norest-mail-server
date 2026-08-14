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
