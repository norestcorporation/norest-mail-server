package billing

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/norest-mail/server/internal/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type WebhookRequest struct {
	Provider    string    `json:"provider"`
	EventID     string    `json:"event_id"`
	Type        string    `json:"type"`
	AccountID   uuid.UUID `json:"account_id"`
	PlanCode    string    `json:"plan_code"`
	PayloadHash string    `json:"payload_hash"`
}

func (h *Handler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	var req WebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	processed, err := h.service.HandleWebhook(r.Context(), req.Provider, req.EventID, req.Type, req.PayloadHash, req.AccountID, req.PlanCode)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, map[string]bool{"processed": processed})
}
