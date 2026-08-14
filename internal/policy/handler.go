package policy

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/norest-mail/server/internal/auth"
)

type Handler struct {
	policySvc *Service
}

func NewHandler(policySvc *Service) *Handler {
	return &Handler{
		policySvc: policySvc,
	}
}

type UsageResponse struct {
	Domains   Usage `json:"domains"`
	Mailboxes Usage `json:"mailboxes"`
	Addresses Usage `json:"addresses"`
}

type Usage struct {
	Used  int `json:"used"`
	Limit int `json:"limit"`
}

func (h *Handler) GetUsage(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	ent, err := h.policySvc.GetEntitlement(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	res := UsageResponse{
		Domains:   Usage{Used: ent.CurrentDomains, Limit: ent.Plan.MaxDomains},
		Mailboxes: Usage{Used: ent.CurrentMailboxes, Limit: ent.Plan.MaxMailboxes},
		Addresses: Usage{Used: ent.CurrentAddresses, Limit: ent.Plan.MaxAddresses},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (h *Handler) SuspendAccount(w http.ResponseWriter, r *http.Request) {
	// RequireAdmin middleware already checked permissions
	// Get account ID from URL
	accountIDStr := chi.URLParam(r, "id")
	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		http.Error(w, "invalid account id", http.StatusBadRequest)
		return
	}

	err = h.policySvc.SuspendAccount(r.Context(), accountID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ReactivateAccount(w http.ResponseWriter, r *http.Request) {
	// RequireAdmin middleware already checked permissions
	accountIDStr := chi.URLParam(r, "id")
	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		http.Error(w, "invalid account id", http.StatusBadRequest)
		return
	}

	err = h.policySvc.ReactivateAccount(r.Context(), accountID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
