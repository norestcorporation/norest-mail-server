package mail

import (
	"context"
	"net/http"

	"github.com/norest-mail/server/internal/auth"
	"github.com/norest-mail/server/internal/response"
)

func (h *Handler) SearchMessages(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	query := r.URL.Query().Get("query")
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	subject := r.URL.Query().Get("subject")
	mailbox := r.URL.Query().Get("mailbox")
	hasAttachment := r.URL.Query().Get("has_attachment") == "true"

	filter := map[string]any{}
	
	if query != "" {
		filter["text"] = query
	}
	if from != "" {
		filter["from"] = from
	}
	if to != "" {
		filter["to"] = to
	}
	if subject != "" {
		filter["subject"] = subject
	}
	if mailbox != "" {
		filter["inMailbox"] = mailbox
	}
	if hasAttachment {
		filter["hasAttachment"] = true
	}
	
	if len(filter) == 0 {
		response.OK(w, map[string]any{"ids": []string{}})
		return
	}

	ids, err := h.service.SearchMessages(r.Context(), userID.String(), filter)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to search messages")
		return
	}

	response.OK(w, map[string]any{"ids": ids})
}

func (s *Service) SearchMessages(ctx context.Context, userID string, filter map[string]any) ([]string, error) {
	acc, err := s.resolveUserAccount(ctx, userID)
	if err != nil {
		return nil, err
	}

	res, err := s.stalwart.EmailQuery(ctx, acc.StalwartAccountID, filter, nil, 100)
	if err != nil {
		return nil, err
	}

	return res.IDs, nil
}
