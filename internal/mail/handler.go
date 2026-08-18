package mail

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/norest-mail/server/internal/auth"
	"github.com/norest-mail/server/internal/response"
)

type IdempotentState struct {
	Status       string
	ResponseCode int
	ResponseBody []byte
}

type IdempotencyStore interface {
	StartIdempotentRequest(ctx context.Context, userID, idempotencyKey, requestHash string) (*IdempotentState, error)
	CompleteIdempotentRequest(ctx context.Context, userID, idempotencyKey, status string, responseCode int, responseBody []byte) error
	ClearIdempotentRequest(ctx context.Context, userID, idempotencyKey string) error
}

type Handler struct {
	service      *Service
	idempotency  IdempotencyStore
	stalwartHost string
}

func NewHandler(service *Service, idempotency IdempotencyStore, stalwartHost string) *Handler {
	return &Handler{
		service:      service,
		idempotency:  idempotency,
		stalwartHost: stalwartHost,
	}
}

// --- Error Translation ---

// handleMailError maps mail service errors to appropriate HTTP responses.
// Uses the existing project error response structure.
func handleMailError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	slog.Debug("mail operation error", "error", err)

	switch {
	// Not found
	case errors.Is(err, ErrMessageNotFound):
		response.Error(w, http.StatusNotFound, "message_not_found")
	case errors.Is(err, ErrMailboxNotFound):
		response.Error(w, http.StatusNotFound, "mailbox_not_found")
	case errors.Is(err, ErrDraftNotFound):
		response.Error(w, http.StatusNotFound, "draft_not_found")
	case errors.Is(err, ErrIdentityNotFound):
		response.Error(w, http.StatusNotFound, "identity_not_found")

	// Authorization
	case errors.Is(err, ErrUnauthorizedResource):
		response.Error(w, http.StatusForbidden, "unauthorized_resource")
	case errors.Is(err, ErrAccountSuspended):
		response.Error(w, http.StatusForbidden, "account_suspended")
	case errors.Is(err, ErrAccountNotActive):
		response.Error(w, http.StatusForbidden, "account_not_active")
	case errors.Is(err, ErrMailboxNotReady):
		response.Error(w, http.StatusServiceUnavailable, "mailbox_not_ready")

	// Validation
	case errors.Is(err, ErrInvalidRecipient):
		response.Error(w, http.StatusUnprocessableEntity, "invalid_recipient")
	case errors.Is(err, ErrMissingRecipient):
		response.Error(w, http.StatusUnprocessableEntity, "missing_recipient")
	case errors.Is(err, ErrMissingSubject):
		response.Error(w, http.StatusUnprocessableEntity, "missing_subject")
	case errors.Is(err, ErrMessageRejected):
		response.Error(w, http.StatusUnprocessableEntity, "message_rejected")

	// Quota/policy
	case errors.Is(err, ErrQuotaExceeded):
		response.Error(w, http.StatusTooManyRequests, "quota_exceeded")
	case errors.Is(err, ErrRateLimitExceeded):
		response.Error(w, http.StatusTooManyRequests, "rate_limit_exceeded")
	case errors.Is(err, ErrAttachmentTooLarge):
		response.Error(w, http.StatusRequestEntityTooLarge, "attachment_too_large")

	// Infrastructure
	case errors.Is(err, ErrStalwartUnavailable):
		response.Error(w, http.StatusBadGateway, "mail_service_unavailable")
	case errors.Is(err, ErrJMAPTimeout):
		response.Error(w, http.StatusGatewayTimeout, "mail_service_timeout")

	// Delivery ambiguity — critical: this is NOT a failure
	case errors.Is(err, ErrDeliveryAmbiguous):
		// Return 202 Accepted — the message may have been sent.
		// The client should NOT retry blindly.
		response.JSON(w, http.StatusAccepted, map[string]string{
			"status": "delivery_status_unknown",
			"error":  "the send operation timed out but the message may have been delivered",
		})

	// Idempotency
	case errors.Is(err, ErrDuplicateSend):
		response.Error(w, http.StatusConflict, "duplicate_send_request")

	// Default
	default:
		slog.Error("unhandled mail error", "error", err)
		response.Error(w, http.StatusInternalServerError, "internal_error")
	}
}

// --- Existing Handlers (preserved) ---

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
		"id":                  mailbox.ID,
		"address_id":          mailbox.AddressID,
		"status":              mailbox.Status,
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

// --- Mailbox Handlers ---

// ListMailboxes handles GET /v1/mail/mailboxes
func (h *Handler) ListMailboxes(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	result, err := h.service.ListMailboxes(r.Context(), userID.String())
	if err != nil {
		handleMailError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// GetMailbox handles GET /v1/mail/mailboxes/{id}
func (h *Handler) GetMailbox(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	mailboxID := chi.URLParam(r, "id")
	if mailboxID == "" {
		response.Error(w, http.StatusBadRequest, "mailbox_id_required")
		return
	}

	result, err := h.service.GetMailbox(r.Context(), userID.String(), mailboxID)
	if err != nil {
		handleMailError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// --- Message Handlers ---

// ListMessages handles GET /v1/mail/messages
func (h *Handler) ListMessages(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	q := r.URL.Query()
	opts := ListMessagesOptions{
		MailboxID:     q.Get("mailbox_id"),
		From:          q.Get("from"),
		To:            q.Get("to"),
		Subject:       q.Get("subject"),
		Text:          q.Get("text"),
		Before:        q.Get("before"),
		After:         q.Get("after"),
		HasKeyword:    q.Get("has_keyword"),
		NotHasKeyword: q.Get("not_has_keyword"),
		CC:            q.Get("cc"),
		ThreadID:      q.Get("thread_id"),
		HasAttachment: q.Get("has_attachment") == "true",
	}

	if limitStr := q.Get("limit"); limitStr != "" {
		var limit int
		if _, err := fmt.Sscanf(limitStr, "%d", &limit); err == nil {
			opts.Limit = limit
		}
	}
	if posStr := q.Get("position"); posStr != "" {
		var pos int
		if _, err := fmt.Sscanf(posStr, "%d", &pos); err == nil {
			opts.Position = pos
		}
	}

	result, err := h.service.ListMessages(r.Context(), userID.String(), opts)
	if err != nil {
		handleMailError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// GetMessage handles GET /v1/mail/messages/{id}
func (h *Handler) GetMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	messageID := chi.URLParam(r, "id")
	if messageID == "" {
		response.Error(w, http.StatusBadRequest, "message_id_required")
		return
	}

	result, err := h.service.GetMessage(r.Context(), userID.String(), messageID)
	if err != nil {
		handleMailError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// --- Draft Handlers ---

// CreateDraft handles POST /v1/mail/drafts
func (h *Handler) CreateDraft(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req DraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid_request_body")
		return
	}

	result, err := h.service.CreateDraft(r.Context(), userID.String(), req)
	if err != nil {
		handleMailError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, result)
}

// GetDraft handles GET /v1/mail/drafts/{id}
func (h *Handler) GetDraft(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	draftID := chi.URLParam(r, "id")
	if draftID == "" {
		response.Error(w, http.StatusBadRequest, "draft_id_required")
		return
	}

	result, err := h.service.GetDraft(r.Context(), userID.String(), draftID)
	if err != nil {
		handleMailError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// UpdateDraft handles PUT /v1/mail/drafts/{id}
func (h *Handler) UpdateDraft(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	draftID := chi.URLParam(r, "id")
	if draftID == "" {
		response.Error(w, http.StatusBadRequest, "draft_id_required")
		return
	}

	var req DraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid_request_body")
		return
	}

	result, err := h.service.UpdateDraft(r.Context(), userID.String(), draftID, req)
	if err != nil {
		handleMailError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// DeleteDraft handles DELETE /v1/mail/drafts/{id}
func (h *Handler) DeleteDraft(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	draftID := chi.URLParam(r, "id")
	if draftID == "" {
		response.Error(w, http.StatusBadRequest, "draft_id_required")
		return
	}

	if err := h.service.DeleteDraft(r.Context(), userID.String(), draftID); err != nil {
		handleMailError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"id":     draftID,
		"status": "deleted",
	})
}

// --- Send Handler ---

// SendMessage handles POST /v1/mail/send
func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{
				"code":    "idempotency_key_required",
				"message": "Idempotency-Key header is required for send requests",
			},
		})
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid_request_body")
		return
	}
	r.Body.Close()

	var req SendRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid_request_body")
		return
	}

	// req.IdempotencyKey in SendRequest is no longer used, but we'll populate it just in case
	req.IdempotencyKey = idempotencyKey

	// Compute payload hash
	hashBytes := sha256.Sum256(bodyBytes)
	hash := hex.EncodeToString(hashBytes[:])

	// 1. Acquire durable idempotency record
	state, err := h.idempotency.StartIdempotentRequest(r.Context(), userID.String(), idempotencyKey, hash)
	if err != nil {
		if errors.Is(err, ErrIdempotencyMismatch) {
			response.Error(w, http.StatusBadRequest, "idempotency_mismatch")
			return
		}
		if errors.Is(err, ErrIdempotencyInProgress) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{
					"code": "idempotency_in_progress",
				},
			})
			return
		}
		handleMailError(w, err)
		return
	}

	// 2. If already COMPLETED or AMBIGUOUS, return cached result
	if state != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(state.ResponseCode)
		w.Write(state.ResponseBody)
		return
	}

	// 3. Call service
	result, err := h.service.SendMessage(r.Context(), userID.String(), req)
	if err != nil {
		// Distinguish between AMBIGUOUS and FAILED_RETRYABLE
		if errors.Is(err, ErrDeliveryAmbiguous) && result != nil {
			respBytes, _ := json.Marshal(result)
			_ = h.idempotency.CompleteIdempotentRequest(r.Context(), userID.String(), idempotencyKey, "AMBIGUOUS", http.StatusAccepted, respBytes)
			response.JSON(w, http.StatusAccepted, result)
			return
		}

		// Any other error means it failed before submission or was safely aborted
		_ = h.idempotency.ClearIdempotentRequest(r.Context(), userID.String(), idempotencyKey)
		handleMailError(w, err)
		return
	}

	// 4. On success
	respBytes, _ := json.Marshal(result)
	_ = h.idempotency.CompleteIdempotentRequest(r.Context(), userID.String(), idempotencyKey, "COMPLETED", http.StatusOK, respBytes)
	response.JSON(w, http.StatusOK, result)
}

// --- Message Action Handlers ---

// MarkRead handles POST /v1/mail/messages/{id}/read
func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	h.handleAction(w, r, func(ctx context.Context, userID, messageID string) (*MessageActionResponse, error) {
		return h.service.MarkRead(ctx, userID, messageID)
	})
}

// MarkUnread handles POST /v1/mail/messages/{id}/unread
func (h *Handler) MarkUnread(w http.ResponseWriter, r *http.Request) {
	h.handleAction(w, r, func(ctx context.Context, userID, messageID string) (*MessageActionResponse, error) {
		return h.service.MarkUnread(ctx, userID, messageID)
	})
}

// StarMessage handles POST /v1/mail/messages/{id}/star
func (h *Handler) StarMessage(w http.ResponseWriter, r *http.Request) {
	h.handleAction(w, r, func(ctx context.Context, userID, messageID string) (*MessageActionResponse, error) {
		return h.service.StarMessage(ctx, userID, messageID)
	})
}

// UnstarMessage handles POST /v1/mail/messages/{id}/unstar
func (h *Handler) UnstarMessage(w http.ResponseWriter, r *http.Request) {
	h.handleAction(w, r, func(ctx context.Context, userID, messageID string) (*MessageActionResponse, error) {
		return h.service.UnstarMessage(ctx, userID, messageID)
	})
}

// ArchiveMessage handles POST /v1/mail/messages/{id}/archive
func (h *Handler) ArchiveMessage(w http.ResponseWriter, r *http.Request) {
	h.handleAction(w, r, func(ctx context.Context, userID, messageID string) (*MessageActionResponse, error) {
		return h.service.ArchiveMessage(ctx, userID, messageID)
	})
}

// TrashMessage handles POST /v1/mail/messages/{id}/trash
func (h *Handler) TrashMessage(w http.ResponseWriter, r *http.Request) {
	h.handleAction(w, r, func(ctx context.Context, userID, messageID string) (*MessageActionResponse, error) {
		return h.service.TrashMessage(ctx, userID, messageID)
	})
}

// RestoreMessage handles POST /v1/mail/messages/{id}/restore
func (h *Handler) RestoreMessage(w http.ResponseWriter, r *http.Request) {
	h.handleAction(w, r, func(ctx context.Context, userID, messageID string) (*MessageActionResponse, error) {
		return h.service.RestoreMessage(ctx, userID, messageID)
	})
}

// SpamMessage handles POST /v1/mail/messages/{id}/spam
func (h *Handler) SpamMessage(w http.ResponseWriter, r *http.Request) {
	h.handleAction(w, r, func(ctx context.Context, userID, messageID string) (*MessageActionResponse, error) {
		return h.service.SpamMessage(ctx, userID, messageID)
	})
}

// MoveMessage handles POST /v1/mail/messages/{id}/move
func (h *Handler) MoveMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	messageID := chi.URLParam(r, "id")
	if messageID == "" {
		response.Error(w, http.StatusBadRequest, "message_id_required")
		return
	}

	var req MoveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid_request_body")
		return
	}
	if req.MailboxID == "" {
		response.Error(w, http.StatusBadRequest, "mailbox_id_required")
		return
	}

	result, err := h.service.MoveMessage(r.Context(), userID.String(), messageID, req.MailboxID)
	if err != nil {
		handleMailError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// handleAction is a shared helper for simple message action handlers.
func (h *Handler) handleAction(w http.ResponseWriter, r *http.Request,
	action func(ctx context.Context, userID, messageID string) (*MessageActionResponse, error)) {

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	messageID := chi.URLParam(r, "id")
	if messageID == "" {
		response.Error(w, http.StatusBadRequest, "message_id_required")
		return
	}

	result, err := action(r.Context(), userID.String(), messageID)
	if err != nil {
		handleMailError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// --- Replies and Forwarding ---

func (h *Handler) ReplyMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	messageID := chi.URLParam(r, "id")
	if messageID == "" {
		response.Error(w, http.StatusBadRequest, "message_id_required")
		return
	}

	var req SendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid_request")
		return
	}

	// Idempotency check logic is inside SendMessage which is called by ReplyMessage,
	// but the client-supplied idempotency key is in req.IdempotencyKey.

	result, err := h.service.ReplyMessage(r.Context(), userID.String(), messageID, req)
	if err != nil {
		handleMailError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

func (h *Handler) ReplyAllMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	messageID := chi.URLParam(r, "id")
	if messageID == "" {
		response.Error(w, http.StatusBadRequest, "message_id_required")
		return
	}

	var req SendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid_request")
		return
	}

	result, err := h.service.ReplyAllMessage(r.Context(), userID.String(), messageID, req)
	if err != nil {
		handleMailError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

func (h *Handler) ForwardMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	messageID := chi.URLParam(r, "id")
	if messageID == "" {
		response.Error(w, http.StatusBadRequest, "message_id_required")
		return
	}

	var req SendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid_request")
		return
	}

	result, err := h.service.ForwardMessage(r.Context(), userID.String(), messageID, req)
	if err != nil {
		handleMailError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// --- Sync ---

func (h *Handler) SyncMail(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	result, err := h.service.SyncMail(r.Context(), userID.String())
	if err != nil {
		if err.Error() == "resync_required" {
			response.Error(w, http.StatusConflict, "resync_required")
			return
		}
		handleMailError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// --- Threads ---

func (h *Handler) ListThreads(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	opts := ListMessagesOptions{} // same filter parsing as ListMessages can be done
	opts.MailboxID = r.URL.Query().Get("mailbox_id")
	opts.Cursor = r.URL.Query().Get("cursor")
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &opts.Limit)
	}

	result, err := h.service.ListThreads(r.Context(), userID.String(), opts)
	if err != nil {
		handleMailError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

func (h *Handler) GetThread(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	threadID := chi.URLParam(r, "id")
	if threadID == "" {
		response.Error(w, http.StatusBadRequest, "thread_id_required")
		return
	}

	result, err := h.service.GetThread(r.Context(), userID.String(), threadID)
	if err != nil {
		handleMailError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

func (h *Handler) GetThreadMessages(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	threadID := chi.URLParam(r, "id")
	if threadID == "" {
		response.Error(w, http.StatusBadRequest, "thread_id_required")
		return
	}

	result, err := h.service.GetThreadMessages(r.Context(), userID.String(), threadID)
	if err != nil {
		handleMailError(w, err)
		return
	}

	// For simplicity, return a standard array response
	response.JSON(w, http.StatusOK, map[string]any{"messages": result})
}

// --- Attachments ---

func (h *Handler) UploadAttachment(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// The router limits the request body to 25MB using RequestSizeLimit(25<<20).
	// We can stream directly to Stalwart.
	attachment, err := h.service.UploadAttachment(r.Context(), userID.String(), contentType, r.Body)
	if err != nil {
		handleMailError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, attachment)
}

func (h *Handler) DownloadAttachment(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	blobID := chi.URLParam(r, "blob_id")
	if blobID == "" {
		response.Error(w, http.StatusBadRequest, "blob_id_required")
		return
	}

	// We default expected type to application/octet-stream as we don't store it locally,
	// but stalwart will respond with the correct type.
	// Norest doesn't store the exact name either, so we just use a generic name or "attachment".
	body, contentType, err := h.service.DownloadAttachment(r.Context(), userID.String(), blobID, "application/octet-stream", "attachment")
	if err != nil {
		handleMailError(w, err)
		return
	}
	defer body.Close()

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "attachment")
	w.WriteHeader(http.StatusOK)

	// Stream to client
	_, _ = io.Copy(w, body)
}
