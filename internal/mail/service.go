package mail

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/norest-mail/server/internal/stalwart"
)

// Service provides mail operations through the Norest REST API.
// All operations resolve ownership from the authenticated user's JWT.
// The admin JMAP client is used with the user's Stalwart account ID.
type Service struct {
	db          DB
	stalwart    *stalwart.Client
	idempotency *IdempotencyStore
	reconciler  *ReconciliationStore
}

// DB defines the database interface the mail service requires.
// The implementations of these methods live in the db package;
// the router bridges db types to mail types.
type DB interface {
	GetMailboxByUserID(ctx context.Context, userID string) (Mailbox, error)
	GetAddressByID(ctx context.Context, id string) (Address, error)
	GetMailboxMappingByRole(ctx context.Context, mailboxID, role string) (string, error)
	GetSyncState(ctx context.Context, mailboxID string) (*SyncState, error)
	UpdateSyncState(ctx context.Context, mailboxID, state, status, errMsg string) error

	// Threading
	ListThreads(ctx context.Context, accountID, mailboxID string, limit int, cursorTime *time.Time, cursorID string) ([]ThreadData, error)
	GetThreadAccountScoped(ctx context.Context, accountID, threadID string) (*ThreadData, error)
	GetMessagesByThread(ctx context.Context, accountID, threadID string) ([]MessageData, error)
	GetThreadIDByStalwartID(ctx context.Context, accountID, stalwartEmailID string) (string, error)

	// Delivery Status
	CreateDeliveryStatus(ctx context.Context, messageID, submissionID, userID, mailboxID, recipientEmail, subject string) error
	GetPendingDeliveryStatus(ctx context.Context, batchSize int) ([]map[string]any, error)
	UpdateDeliveryStatus(ctx context.Context, submissionID, recipientEmail, status, errorMessage, errorType, smtpReply string, isPermanent bool) error
	UpdateDeliveryStatusWithOutbox(ctx context.Context, submissionID, recipientEmail, userID, messageID, status, errorMessage, errorType, smtpReply string, isPermanent bool) error
	GetDeliveryStatusesByMessageID(ctx context.Context, messageID string) ([]DeliveryStatusRecord, error)
	TryClaimBounceGeneration(ctx context.Context, submissionID, recipientEmail string) (bool, error)
	SaveBounceEmailID(ctx context.Context, submissionID, recipientEmail, bounceEmailID string) error
	GetRFCMessageID(ctx context.Context, stalwartEmailID string) (string, error)

	// Reactions
	ToggleReaction(ctx context.Context, messageID, userID, emoji string) (bool, error)
	GetReactionsForMessage(ctx context.Context, messageID string) ([]EmailReaction, error)
}

// EmailReaction represents a reaction on an email.
type EmailReaction struct {
	ID        string    `json:"id"`
	MessageID string    `json:"message_id"`
	UserEmail string    `json:"user_email"`
	Emoji     string    `json:"emoji"`
	CreatedAt time.Time `json:"created_at"`
}

// ThreadData represents the Norest-owned thread projection returned by the DB.
type ThreadData struct {
	ID            string
	AccountID     string
	Subject       string
	Participants  []byte
	MessageCount  int
	UnreadCount   int
	Snippet       *string
	LastMessageAt time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// MessageData represents the Norest-owned message projection returned by the DB.
type MessageData struct {
	ID              string
	AccountID       string
	ThreadID        string
	StalwartEmailID string
	MessageID       *string
	InReplyTo       *string
	ReferencesChain []string
	Subject         *string
	Sender          []byte
	Recipients      []byte
	ReceivedAt      *time.Time
	SentAt          *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// SyncState represents the incremental sync state of a mailbox.
type SyncState struct {
	State        string
	LastSyncedAt *string // timestamp as string or time.Time, let's use a simple string for simplicity or just keep it simple
	Status       string
	ErrorMessage *string
}

// Mailbox represents a provisioned mailbox.
type Mailbox struct {
	ID                string
	AddressID         string
	Status            string
	StalwartAccountID *string
}

// Address represents a provisioned address.
type Address struct {
	ID        string
	LocalPart string
	DomainID  string
	Status    string
}

func NewService(db DB, client *stalwart.Client, pool *pgxpool.Pool) *Service {
	return &Service{
		db:         db,
		stalwart:   client,
		reconciler: NewReconciliationStore(pool),
	}
}

// resolvedAccount holds the validated user context for a mail operation.
type resolvedAccount struct {
	UserID            string
	MailboxID         string
	StalwartAccountID string
}

// resolveUserAccount derives the Stalwart account ID from an authenticated user.
// This is the security boundary: every mail operation must call this first.
//
// Flow:
//
//	JWT user_id → Norest mailbox → Stalwart account_id
//
// Returns an error if the user has no mailbox, the mailbox is not active,
// or the Stalwart account is not provisioned.
func (s *Service) resolveUserAccount(ctx context.Context, userID string) (*resolvedAccount, error) {
	mailbox, err := s.db.GetMailboxByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMailboxNotFound, err)
	}

	if mailbox.Status != "active" {
		return nil, fmt.Errorf("%w: status is %s", ErrAccountNotActive, mailbox.Status)
	}

	if mailbox.StalwartAccountID == nil || *mailbox.StalwartAccountID == "" {
		return nil, ErrMailboxNotReady
	}

	return &resolvedAccount{
		UserID:            userID,
		MailboxID:         mailbox.ID,
		StalwartAccountID: *mailbox.StalwartAccountID,
	}, nil
}

// resolveStalwartMailboxID looks up the Stalwart mailbox ID for a given role
// using the persisted Norest mailbox mappings (not hardcoded).
func (s *Service) resolveStalwartMailboxID(ctx context.Context, norestMailboxID, role string) (string, error) {
	stalwartMailboxID, err := s.db.GetMailboxMappingByRole(ctx, norestMailboxID, role)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrMailboxNotFound, err)
	}
	return stalwartMailboxID, nil
}

// --- Mail Session (existing) ---

// MailSessionResponse represents the secure mail session artifact returned to the frontend.
type MailSessionResponse struct {
	Provider       string `json:"provider"`
	JMAPSessionURL string `json:"jmap_session_url"`
	AccessToken    string `json:"access_token"`
	AccountID      string `json:"account_id"`
}

// ProvisioningStatus represents the current provisioning state of a user's mailbox.
type ProvisioningStatus struct {
	MailboxID         string  `json:"mailbox_id"`
	AddressID         string  `json:"address_id"`
	Status            string  `json:"status"`
	StalwartAccountID *string `json:"stalwart_account_id"`
	ReadyForSession   bool    `json:"ready_for_session"`
}

// CreateMailSession verifies the user and mailbox, then provisions a short-lived AppPassword
// via Stalwart JMAP acting as the user's mail access token.
func (s *Service) CreateMailSession(ctx context.Context, userID string, stalwartHost string) (*MailSessionResponse, error) {
	// 1. Verify user's mailbox exists and is active.
	mailbox, err := s.db.GetMailboxByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("getting mailbox: %w", err)
	}

	if mailbox.Status != "active" {
		return nil, fmt.Errorf("mailbox is not active")
	}

	if mailbox.StalwartAccountID == nil || *mailbox.StalwartAccountID == "" {
		return nil, fmt.Errorf("mailbox not fully provisioned in stalwart")
	}

	// 2. Provision a short-lived user mail token (AppPassword) in Stalwart.
	// We generate an AppPassword programmatically for this session.
	desc := fmt.Sprintf("Norest Session Token - %s", userID)
	secret, err := s.stalwart.CreateAppPassword(ctx, *mailbox.StalwartAccountID, desc)
	if err != nil {
		return nil, fmt.Errorf("provisioning mail token: %w", err)
	}

	jmapURL := fmt.Sprintf("%s/.well-known/jmap", stalwartHost)

	return &MailSessionResponse{
		Provider:       "stalwart",
		JMAPSessionURL: jmapURL,
		AccessToken:    secret,
		AccountID:      *mailbox.StalwartAccountID,
	}, nil
}

// GetProvisioningStatus returns the current provisioning state of the user's mailbox.
func (s *Service) GetProvisioningStatus(ctx context.Context, userID string) (*ProvisioningStatus, error) {
	mailbox, err := s.db.GetMailboxByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("getting mailbox: %w", err)
	}

	ready := mailbox.Status == "active" && mailbox.StalwartAccountID != nil && *mailbox.StalwartAccountID != ""

	return &ProvisioningStatus{
		MailboxID:         mailbox.ID,
		AddressID:         mailbox.AddressID,
		Status:            mailbox.Status,
		StalwartAccountID: mailbox.StalwartAccountID,
		ReadyForSession:   ready,
	}, nil
}

// --- Mailboxes ---

// ListMailboxes returns all mailboxes (folders) for the authenticated user.
func (s *Service) ListMailboxes(ctx context.Context, userID string) (*MailboxListResponse, error) {
	acct, err := s.resolveUserAccount(ctx, userID)
	if err != nil {
		return nil, err
	}

	resp, err := s.stalwart.MailboxGet(ctx, acct.StalwartAccountID, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStalwartUnavailable, err)
	}

	mailboxes := make([]MailboxResponse, len(resp.List))
	for i, m := range resp.List {
		mailboxes[i] = mailboxToResponse(m)
	}

	return &MailboxListResponse{Mailboxes: mailboxes}, nil
}

// GetMailbox returns a specific mailbox by its JMAP mailbox ID.
// The mailbox must belong to the authenticated user's Stalwart account.
func (s *Service) GetMailbox(ctx context.Context, userID, mailboxID string) (*MailboxResponse, error) {
	acct, err := s.resolveUserAccount(ctx, userID)
	if err != nil {
		return nil, err
	}

	resp, err := s.stalwart.MailboxGet(ctx, acct.StalwartAccountID, []string{mailboxID})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStalwartUnavailable, err)
	}

	if len(resp.List) == 0 {
		return nil, ErrMailboxNotFound
	}

	result := mailboxToResponse(resp.List[0])
	return &result, nil
}

// --- Messages ---

// ListMessages returns messages matching the given options.
// The mailbox_id filter is resolved through the user's account.
func (s *Service) ListMessages(ctx context.Context, userID string, opts ListMessagesOptions) (*MessageListResponse, error) {
	acct, err := s.resolveUserAccount(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Build JMAP filter from Norest options
	filter := make(map[string]any)
	if opts.MailboxID != "" {
		filter["inMailbox"] = opts.MailboxID
	}
	if opts.From != "" {
		filter["from"] = opts.From
	}
	if opts.To != "" {
		filter["to"] = opts.To
	}
	if opts.Subject != "" {
		filter["subject"] = opts.Subject
	}
	if opts.Text != "" {
		filter["text"] = opts.Text
	}
	if opts.Before != "" {
		filter["before"] = opts.Before
	}
	if opts.After != "" {
		filter["after"] = opts.After
	}
	if opts.HasKeyword != "" {
		filter["hasKeyword"] = opts.HasKeyword
	}
	if opts.NotHasKeyword != "" {
		filter["notKeyword"] = opts.NotHasKeyword
	}
	if opts.CC != "" {
		filter["cc"] = opts.CC
	}
	if opts.ThreadID != "" {
		// some JMAP servers do not support threadId as a standard Email/query filter, but we'll try
		filter["threadId"] = opts.ThreadID
	}
	if opts.HasAttachment {
		filter["hasAttachment"] = true
	}

	// Default sort: newest first
	sort := []map[string]any{
		{"property": "receivedAt", "isAscending": false},
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	// Query for IDs
	queryResp, err := s.stalwart.EmailQueryWithPosition(ctx, acct.StalwartAccountID, filter, sort, opts.Position, limit)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStalwartUnavailable, err)
	}

	if len(queryResp.IDs) == 0 {
		return &MessageListResponse{
			Messages: []MessageResponse{},
			Total:    queryResp.Total,
			Position: queryResp.Position,
		}, nil
	}

	// Fetch the actual email objects (summary properties only for list)
	getResp, err := s.stalwart.EmailGet(ctx, acct.StalwartAccountID, queryResp.IDs, []string{
		"id", "threadId", "mailboxIds", "keywords", "size", "receivedAt",
		"from", "to", "cc", "subject", "preview", "hasAttachment", "sentAt",
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStalwartUnavailable, err)
	}

	messages := make([]MessageResponse, len(getResp.List))
	for i, e := range getResp.List {
		messages[i] = emailToMessageResponse(e)
	}

	return &MessageListResponse{
		Messages: messages,
		Total:    queryResp.Total,
		Position: queryResp.Position,
	}, nil
}

// GetMessage returns a single message by ID with full body content.
// Ownership is verified: JWT → mailbox → Stalwart account → Email/get scoped.
func (s *Service) GetMessage(ctx context.Context, userID, messageID string) (*MessageResponse, error) {
	acct, err := s.resolveUserAccount(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Email/get scoped to this user's Stalwart account.
	// If the messageID belongs to a different account, Stalwart will return it in notFound.
	resp, err := s.stalwart.EmailGetWithBody(ctx, acct.StalwartAccountID, []string{messageID})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStalwartUnavailable, err)
	}

	// Check if the message was found
	if len(resp.List) == 0 {
		// If the ID is in NotFound, the message doesn't exist in this account.
		// This also protects against cross-user access — a forged ID for another
		// user's account will simply not be found in this user's account.
		return nil, ErrMessageNotFound
	}

	msg := emailToMessageResponse(resp.List[0])

	// Override ThreadID with Norest ThreadID
	if threadID, err := s.db.GetThreadIDByStalwartID(ctx, acct.MailboxID, msg.ID); err == nil {
		msg.ThreadID = threadID
	}

	// Fetch delivery statuses
	if deliveryStatuses, err := s.db.GetDeliveryStatusesByMessageID(ctx, msg.ID); err == nil && len(deliveryStatuses) > 0 {
		msg.DeliveryStatuses = deliveryStatuses
	} else if err != nil {
		slog.Error("failed to get delivery statuses", "error", err, "message_id", msg.ID)
	}

	// Fetch reactions
	if reactions, err := s.db.GetReactionsForMessage(ctx, msg.ID); err == nil && len(reactions) > 0 {
		msg.Reactions = reactions
	} else if err != nil {
		slog.Error("failed to get reactions", "error", err, "message_id", msg.ID)
	}

	return &msg, nil
}

// --- Drafts ---

// CreateDraft creates a new draft email in the user's Drafts mailbox.
func (s *Service) CreateDraft(ctx context.Context, userID string, req DraftRequest) (*DraftResponse, error) {
	acct, err := s.resolveUserAccount(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Resolve the Drafts mailbox ID from persisted mappings (not hardcoded)
	draftsMailboxID, err := s.resolveStalwartMailboxID(ctx, acct.MailboxID, "drafts")
	if err != nil {
		return nil, fmt.Errorf("cannot resolve drafts mailbox: %w", err)
	}

	// Resolve the user's sending address for the From field
	address, err := s.resolveUserAddress(ctx, acct)
	if err != nil {
		return nil, err
	}

	// Build the JMAP Email object for creation
	createObj := map[string]any{
		"mailboxIds": map[string]bool{draftsMailboxID: true},
		"keywords":   map[string]bool{"$draft": true, "$seen": true},
		"from":       []stalwart.EmailAddress{{Email: address}},
	}

	if len(req.To) > 0 {
		createObj["to"] = toStalwartAddresses(req.To)
	}
	if len(req.CC) > 0 {
		createObj["cc"] = toStalwartAddresses(req.CC)
	}
	if len(req.BCC) > 0 {
		createObj["bcc"] = toStalwartAddresses(req.BCC)
	}
	if req.Subject != "" {
		createObj["subject"] = req.Subject
	}
	if len(req.InReplyTo) > 0 {
		createObj["inReplyTo"] = req.InReplyTo
	}
	if len(req.References) > 0 {
		createObj["references"] = req.References
	}
	if len(req.Attachments) > 0 {
		createObj["attachments"] = buildAttachments(req.Attachments)
	}
	if len(req.AttachmentIDs) > 0 {
		// Convert attachment IDs to attachment format
		attachments := make([]map[string]any, len(req.AttachmentIDs))
		for i, id := range req.AttachmentIDs {
			attachments[i] = map[string]any{
				"blobId": id,
			}
		}
		createObj["attachments"] = attachments
	}

	// Build body
	bodyValue := map[string]any{}
	if req.HTMLBody != "" {
		bodyValue["value"] = req.HTMLBody
		createObj["htmlBody"] = []map[string]any{
			{"partId": "body", "type": "text/html"},
		}
	} else if req.TextBody != "" {
		bodyValue["value"] = req.TextBody
		createObj["textBody"] = []map[string]any{
			{"partId": "body", "type": "text/plain"},
		}
	} else {
		bodyValue["value"] = ""
		createObj["textBody"] = []map[string]any{
			{"partId": "body", "type": "text/plain"},
		}
	}
	createObj["bodyValues"] = map[string]any{
		"body": bodyValue,
	}

	createKey := "draft0"

	intentPayload := map[string]any{"action": "create_draft"}
	reconID, err := s.reconciler.LogIntent(ctx, userID, "", "draft.create", intentPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to log intent: %w", err)
	}

	resp, err := s.stalwart.EmailSet(ctx, acct.StalwartAccountID,
		map[string]any{createKey: createObj}, nil, nil)
	if err != nil {
		s.reconciler.MarkUnknown(ctx, reconID, err)
		return nil, fmt.Errorf("creating draft: %w", err)
	}

	created, ok := resp.Created[createKey]
	if !ok {
		s.reconciler.MarkFailed(ctx, reconID, fmt.Errorf("no created object"))
		return nil, fmt.Errorf("draft creation did not return created object")
	}

	eventPayload := map[string]any{"message_id": created.ID}
	if markErr := s.reconciler.MarkSuccess(ctx, reconID, userID, "message.created", eventPayload, resp); markErr != nil {
		slog.Error("failed to mark success in reconciliation", "error", markErr, "recon_id", reconID)
	}

	// Get the created draft with full details to return complete response
	createdDraft, err := s.stalwart.EmailGetWithBody(ctx, acct.StalwartAccountID, []string{created.ID})
	if err != nil {
		slog.Error("failed to fetch created draft details", "error", err)
		// Still return success even if we can't fetch details
		return &DraftResponse{
			ID:      created.ID,
			BlobID:  created.BlobID,
			Message: "draft created",
		}, nil
	}

	draftMsg := emailToMessageResponse(createdDraft.List[0])

	return &DraftResponse{
		ID:       created.ID,
		BlobID:   created.BlobID,
		Message:  "draft created",
		To:       draftMsg.To,
		CC:       draftMsg.CC,
		BCC:      draftMsg.BCC,
		Subject:  draftMsg.Subject,
		TextBody: draftMsg.TextBody,
		HTMLBody: draftMsg.HTMLBody,
		AttachmentIDs: func() []string {
			ids := make([]string, len(draftMsg.Attachments))
			for i, att := range draftMsg.Attachments {
				ids[i] = att.BlobID
			}
			return ids
		}(),
		CreatedAt: draftMsg.SentAt,
		UpdatedAt: draftMsg.SentAt,
	}, nil
}

// GetDraft returns a draft by ID. Validates that it's in the Drafts mailbox.
func (s *Service) GetDraft(ctx context.Context, userID, draftID string) (*DraftResponse, error) {
	msg, err := s.GetMessage(ctx, userID, draftID)
	if err != nil {
		return nil, ErrDraftNotFound
	}

	if !msg.IsDraft {
		return nil, ErrDraftNotFound
	}

	// Convert MessageResponse to DraftResponse
	attachmentIDs := make([]string, len(msg.Attachments))
	for i, att := range msg.Attachments {
		attachmentIDs[i] = att.BlobID
	}

	return &DraftResponse{
		ID:            msg.ID,
		BlobID:        "", // Not available in MessageResponse
		Message:       "draft retrieved",
		To:            msg.To,
		CC:            msg.CC,
		BCC:           msg.BCC,
		Subject:       msg.Subject,
		TextBody:      msg.TextBody,
		HTMLBody:      msg.HTMLBody,
		AttachmentIDs: attachmentIDs,
		CreatedAt:     msg.SentAt,
		UpdatedAt:     msg.ReceivedAt,
	}, nil
}

// UpdateDraft updates an existing draft. This creates a new JMAP Email and destroys the old one.
// JMAP emails are immutable, so "updating" a draft means creating a replacement.
func (s *Service) UpdateDraft(ctx context.Context, userID, draftID string, req DraftRequest) (*DraftResponse, error) {
	acct, err := s.resolveUserAccount(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Verify the draft exists and belongs to this user
	existing, err := s.stalwart.EmailGetWithBody(ctx, acct.StalwartAccountID, []string{draftID})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStalwartUnavailable, err)
	}
	if len(existing.List) == 0 {
		return nil, ErrDraftNotFound
	}

	oldEmail := existing.List[0]
	if !oldEmail.Keywords["$draft"] {
		return nil, ErrDraftNotFound
	}

	// Resolve the Drafts mailbox ID
	draftsMailboxID, err := s.resolveStalwartMailboxID(ctx, acct.MailboxID, "drafts")
	if err != nil {
		return nil, fmt.Errorf("cannot resolve drafts mailbox: %w", err)
	}

	// Build the updated email object — merge existing with new values
	createObj := map[string]any{
		"mailboxIds": map[string]bool{draftsMailboxID: true},
		"keywords":   map[string]bool{"$draft": true, "$seen": true},
		"from":       oldEmail.From,
	}

	// Apply updates — use new values if provided, else keep existing
	if len(req.To) > 0 {
		createObj["to"] = toStalwartAddresses(req.To)
	} else if len(oldEmail.To) > 0 {
		createObj["to"] = oldEmail.To
	}
	if len(req.CC) > 0 {
		createObj["cc"] = toStalwartAddresses(req.CC)
	} else if len(oldEmail.CC) > 0 {
		createObj["cc"] = oldEmail.CC
	}
	if len(req.BCC) > 0 {
		createObj["bcc"] = toStalwartAddresses(req.BCC)
	} else if len(oldEmail.BCC) > 0 {
		createObj["bcc"] = oldEmail.BCC
	}
	if req.Subject != "" {
		createObj["subject"] = req.Subject
	} else {
		createObj["subject"] = oldEmail.Subject
	}
	if len(req.InReplyTo) > 0 {
		createObj["inReplyTo"] = req.InReplyTo
	} else if len(oldEmail.InReplyTo) > 0 {
		createObj["inReplyTo"] = oldEmail.InReplyTo
	}
	if len(req.References) > 0 {
		createObj["references"] = req.References
	} else if len(oldEmail.References) > 0 {
		createObj["references"] = oldEmail.References
	}
	if len(req.Attachments) > 0 {
		createObj["attachments"] = buildAttachments(req.Attachments)
	} else if len(req.AttachmentIDs) > 0 {
		// Convert attachment IDs to attachment format
		attachments := make([]map[string]any, len(req.AttachmentIDs))
		for i, id := range req.AttachmentIDs {
			attachments[i] = map[string]any{
				"blobId": id,
			}
		}
		createObj["attachments"] = attachments
	} else if len(oldEmail.Attachments) > 0 {
		createObj["attachments"] = oldEmail.Attachments
	}

	// Build body
	bodyValue := map[string]any{}
	if req.HTMLBody != "" {
		bodyValue["value"] = req.HTMLBody
		createObj["htmlBody"] = []map[string]any{
			{"partId": "body", "type": "text/html"},
		}
	} else if req.TextBody != "" {
		bodyValue["value"] = req.TextBody
		createObj["textBody"] = []map[string]any{
			{"partId": "body", "type": "text/plain"},
		}
	} else {
		// Keep existing body if no update provided
		existingBody := ""
		for _, part := range oldEmail.TextBody {
			if bv, ok := oldEmail.BodyValues[part.PartID]; ok {
				existingBody = bv.Value
				break
			}
		}
		bodyValue["value"] = existingBody
		createObj["textBody"] = []map[string]any{
			{"partId": "body", "type": "text/plain"},
		}
	}
	createObj["bodyValues"] = map[string]any{
		"body": bodyValue,
	}

	// Create new draft and destroy old one in a single Email/set call
	createKey := "draft_update0"

	intentPayload := map[string]any{"action": "update_draft", "old_draft_id": draftID}
	reconID, err := s.reconciler.LogIntent(ctx, userID, "", "draft.update", intentPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to log intent: %w", err)
	}

	resp, err := s.stalwart.EmailSet(ctx, acct.StalwartAccountID,
		map[string]any{createKey: createObj},
		nil,
		[]string{draftID},
	)
	if err != nil {
		s.reconciler.MarkUnknown(ctx, reconID, err)
		return nil, fmt.Errorf("updating draft: %w", err)
	}

	created, ok := resp.Created[createKey]
	if !ok {
		s.reconciler.MarkFailed(ctx, reconID, fmt.Errorf("no created object"))
		return nil, fmt.Errorf("draft update did not return created object")
	}

	eventPayload := map[string]any{"message_id": created.ID, "old_message_id": draftID}
	if markErr := s.reconciler.MarkSuccess(ctx, reconID, userID, "message.updated", eventPayload, resp); markErr != nil {
		slog.Error("failed to mark success in reconciliation", "error", markErr, "recon_id", reconID)
	}

	// Get the updated draft with full details to return complete response
	updatedDraft, err := s.stalwart.EmailGetWithBody(ctx, acct.StalwartAccountID, []string{created.ID})
	if err != nil {
		slog.Error("failed to fetch updated draft details", "error", err)
		// Still return success even if we can't fetch details
		return &DraftResponse{
			ID:      created.ID,
			BlobID:  created.BlobID,
			Message: "draft updated",
		}, nil
	}

	draftMsg := emailToMessageResponse(updatedDraft.List[0])

	return &DraftResponse{
		ID:       created.ID,
		BlobID:   created.BlobID,
		Message:  "draft updated",
		To:       draftMsg.To,
		CC:       draftMsg.CC,
		BCC:      draftMsg.BCC,
		Subject:  draftMsg.Subject,
		TextBody: draftMsg.TextBody,
		HTMLBody: draftMsg.HTMLBody,
		AttachmentIDs: func() []string {
			ids := make([]string, len(draftMsg.Attachments))
			for i, att := range draftMsg.Attachments {
				ids[i] = att.BlobID
			}
			return ids
		}(),
		CreatedAt: draftMsg.SentAt,
		UpdatedAt: draftMsg.SentAt,
	}, nil
}

// DeleteDraft deletes a draft by destroying it via Email/set.
func (s *Service) DeleteDraft(ctx context.Context, userID, draftID string) error {
	acct, err := s.resolveUserAccount(ctx, userID)
	if err != nil {
		return err
	}

	// Verify the draft exists and has the $draft keyword
	existing, err := s.stalwart.EmailGet(ctx, acct.StalwartAccountID, []string{draftID}, []string{"id", "keywords"})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrStalwartUnavailable, err)
	}
	if len(existing.List) == 0 {
		return ErrDraftNotFound
	}
	if !existing.List[0].Keywords["$draft"] {
		return ErrDraftNotFound
	}

	intentPayload := map[string]any{"action": "delete_draft", "draft_id": draftID}
	reconID, err := s.reconciler.LogIntent(ctx, userID, "", "draft.delete", intentPayload)
	if err != nil {
		return fmt.Errorf("failed to log intent: %w", err)
	}

	_, err = s.stalwart.EmailSet(ctx, acct.StalwartAccountID, nil, nil, []string{draftID})
	if err != nil {
		s.reconciler.MarkUnknown(ctx, reconID, err)
		return fmt.Errorf("deleting draft: %w", err)
	}

	eventPayload := map[string]any{"message_id": draftID}
	if markErr := s.reconciler.MarkSuccess(ctx, reconID, userID, "message.deleted", eventPayload, nil); markErr != nil {
		slog.Error("failed to mark success in reconciliation", "error", markErr, "recon_id", reconID)
	}

	return nil
}

// --- Send ---

// SendMessage sends an email. It supports two modes:
// 1. Send an existing draft (draft_id is provided)
// 2. Create and send in one step (full message content provided)
//
// Idempotency: If an idempotency_key is provided, the service should check
// whether a submission with this key has already been processed. This prevents
// duplicate sends when the client retries after a timeout.
func (s *Service) SendMessage(ctx context.Context, userID string, req SendRequest) (*SendResponse, error) {
	acct, err := s.resolveUserAccount(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Validate the request has recipients
	if req.DraftID == "" && len(req.To) == 0 && len(req.CC) == 0 && len(req.BCC) == 0 {
		return nil, ErrMissingRecipient
	}

	// Discover the user's sending identity
	identity, err := s.stalwart.FindPrimaryIdentity(ctx, acct.StalwartAccountID)
	if err != nil {
		slog.Warn("failed to find sending identity, will try without", "error", err, "account", acct.StalwartAccountID)
		// Stalwart may auto-assign identity — proceed with empty identity ID
	}

	var identityID string
	if identity != nil {
		identityID = identity.ID
	}

	// Resolve Sent mailbox for post-send move
	sentMailboxID, err := s.resolveStalwartMailboxID(ctx, acct.MailboxID, "sent")
	if err != nil {
		slog.Warn("could not resolve sent mailbox", "error", err)
	}

	var emailID string
	var subject string
	recipients := make(map[string]struct{})
	addRecips := func(addrs []EmailAddressDTO) {
		for _, a := range addrs {
			if a.Email != "" {
				recipients[a.Email] = struct{}{}
			}
		}
	}
	addStalwartRecips := func(addrs []stalwart.EmailAddress) {
		for _, a := range addrs {
			if a.Email != "" {
				recipients[a.Email] = struct{}{}
			}
		}
	}

	if req.DraftID != "" {
		// Mode 1: Send existing draft
		// Verify draft exists and belongs to this user
		existing, draftErr := s.stalwart.EmailGet(ctx, acct.StalwartAccountID, []string{req.DraftID}, []string{"id", "keywords", "to", "cc", "bcc", "subject"})
		if draftErr != nil {
			return nil, fmt.Errorf("%w: %v", ErrStalwartUnavailable, draftErr)
		}
		if len(existing.List) == 0 {
			return nil, ErrDraftNotFound
		}

		// Validate the draft has recipients
		draft := existing.List[0]
		if len(draft.To) == 0 && len(draft.CC) == 0 && len(draft.BCC) == 0 {
			return nil, ErrMissingRecipient
		}

		subject = draft.Subject
		addStalwartRecips(draft.To)
		addStalwartRecips(draft.CC)
		addStalwartRecips(draft.BCC)

		emailID = req.DraftID
	} else {
		// Mode 2: Create email and send
		subject = req.Subject
		addRecips(req.To)
		addRecips(req.CC)
		addRecips(req.BCC)
		draftsMailboxID, draftErr := s.resolveStalwartMailboxID(ctx, acct.MailboxID, "drafts")
		if draftErr != nil {
			return nil, fmt.Errorf("cannot resolve drafts mailbox: %w", draftErr)
		}

		address, addrErr := s.resolveUserAddress(ctx, acct)
		if addrErr != nil {
			return nil, addrErr
		}

		createObj := map[string]any{
			"mailboxIds": map[string]bool{draftsMailboxID: true},
			"keywords":   map[string]bool{"$draft": true, "$seen": true},
			"from":       []stalwart.EmailAddress{{Email: address}},
			"to":         toStalwartAddresses(req.To),
		}
		if len(req.CC) > 0 {
			createObj["cc"] = toStalwartAddresses(req.CC)
		}
		if len(req.BCC) > 0 {
			createObj["bcc"] = toStalwartAddresses(req.BCC)
		}
		if req.Subject != "" {
			createObj["subject"] = req.Subject
		}
		if len(req.InReplyTo) > 0 {
			createObj["inReplyTo"] = req.InReplyTo
		}
		if len(req.References) > 0 {
			createObj["references"] = req.References
		}
		if len(req.Attachments) > 0 {
			createObj["attachments"] = buildAttachments(req.Attachments)
		}

		bodyValue := map[string]any{}
		if req.HTMLBody != "" {
			bodyValue["value"] = req.HTMLBody
			createObj["htmlBody"] = []map[string]any{
				{"partId": "body", "type": "text/html"},
			}
		} else {
			bodyValue["value"] = req.TextBody
			createObj["textBody"] = []map[string]any{
				{"partId": "body", "type": "text/plain"},
			}
		}
		createObj["bodyValues"] = map[string]any{
			"body": bodyValue,
		}

		createKey := "send_email0"
		emailResp, createErr := s.stalwart.EmailSet(ctx, acct.StalwartAccountID,
			map[string]any{createKey: createObj}, nil, nil)
		if createErr != nil {
			return nil, fmt.Errorf("creating email for send: %w", createErr)
		}

		created, ok := emailResp.Created[createKey]
		if !ok {
			return nil, fmt.Errorf("email creation for send did not return created object")
		}
		emailID = created.ID
	}

	// Build post-send update: move from Drafts to Sent, remove $draft keyword
	var onSuccessUpdate map[string]any
	if sentMailboxID != "" {
		draftsMailboxID, _ := s.resolveStalwartMailboxID(ctx, acct.MailboxID, "drafts")
		updatePatch := map[string]any{
			"keywords/$draft": nil, // Remove $draft keyword
		}
		if sentMailboxID != "" {
			updatePatch["mailboxIds/"+sentMailboxID] = true
		}
		if draftsMailboxID != "" {
			updatePatch["mailboxIds/"+draftsMailboxID] = nil // Remove from Drafts
		}
		onSuccessUpdate = updatePatch
	}

	// Submit the email
	slog.Info("submitting email",
		"user_id", userID,
		"email_id", emailID,
		"identity_id", identityID,
		"has_on_success_update", onSuccessUpdate != nil,
	)

	intentPayload := map[string]any{"email_id": emailID, "identity_id": identityID}
	reconID, err := s.reconciler.LogIntent(ctx, userID, req.IdempotencyKey, "message.send", intentPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to log intent: %w", err)
	}

	subResp, err := s.stalwart.EmailSubmissionSet(ctx, acct.StalwartAccountID, emailID, identityID, onSuccessUpdate)
	if err != nil {
		if isTimeoutOrNetworkError(err) {
			s.reconciler.MarkUnknown(ctx, reconID, err)
			slog.Error("send submission timeout — delivery status ambiguous",
				"user_id", userID,
				"email_id", emailID,
				"error", err,
			)
			return &SendResponse{
				MessageID: emailID,
				Status:    "unknown",
			}, ErrDeliveryAmbiguous
		}
		s.reconciler.MarkFailed(ctx, reconID, err)
		return nil, fmt.Errorf("submitting email: %w", err)
	}

	eventPayload := map[string]any{"message_id": emailID}
	if markErr := s.reconciler.MarkSuccess(ctx, reconID, userID, "message.sent", eventPayload, subResp); markErr != nil {
		slog.Error("failed to mark success in reconciliation", "error", markErr, "recon_id", reconID)
	}

	// Extract the submission result
	var submissionID string
	for _, sub := range subResp.Created {
		submissionID = sub.ID
		break
	}

	// Insert delivery status records for each recipient
	for recEmail := range recipients {
		err := s.db.CreateDeliveryStatus(ctx, emailID, submissionID, userID, acct.MailboxID, recEmail, subject)
		if err != nil {
			slog.Error("failed to create delivery status", "error", err, "recipient", recEmail)
		}
	}

	return &SendResponse{
		MessageID:    emailID,
		SubmissionID: submissionID,
		Status:       "submitted",
	}, nil
}

// --- Message Actions ---

// MarkRead marks a message as read (adds $seen keyword).
func (s *Service) MarkRead(ctx context.Context, userID, messageID string) (*MessageActionResponse, error) {
	return s.updateKeyword(ctx, userID, messageID, "$seen", true, "mark_read")
}

// MarkUnread marks a message as unread (removes $seen keyword).
func (s *Service) MarkUnread(ctx context.Context, userID, messageID string) (*MessageActionResponse, error) {
	return s.updateKeyword(ctx, userID, messageID, "$seen", false, "mark_unread")
}

// StarMessage stars a message (adds $flagged keyword).
func (s *Service) StarMessage(ctx context.Context, userID, messageID string) (*MessageActionResponse, error) {
	return s.updateKeyword(ctx, userID, messageID, "$flagged", true, "star")
}

// UnstarMessage unstars a message (removes $flagged keyword).
func (s *Service) UnstarMessage(ctx context.Context, userID, messageID string) (*MessageActionResponse, error) {
	return s.updateKeyword(ctx, userID, messageID, "$flagged", false, "unstar")
}

// ArchiveMessage moves a message to the Archive mailbox.
func (s *Service) ArchiveMessage(ctx context.Context, userID, messageID string) (*MessageActionResponse, error) {
	return s.moveToRole(ctx, userID, messageID, "archive", "archive")
}

// TrashMessage moves a message to the Trash mailbox.
func (s *Service) TrashMessage(ctx context.Context, userID, messageID string) (*MessageActionResponse, error) {
	return s.moveToRole(ctx, userID, messageID, "trash", "trash")
}

// RestoreMessage moves a message from Trash back to the Inbox.
func (s *Service) RestoreMessage(ctx context.Context, userID, messageID string) (*MessageActionResponse, error) {
	return s.moveToRole(ctx, userID, messageID, "inbox", "restore")
}

// SpamMessage moves a message to the Junk/Spam mailbox.
func (s *Service) SpamMessage(ctx context.Context, userID, messageID string) (*MessageActionResponse, error) {
	return s.moveToRole(ctx, userID, messageID, "junk", "spam")
}

// MoveMessage moves a message to a specified mailbox.
func (s *Service) MoveMessage(ctx context.Context, userID, messageID, targetMailboxID string) (*MessageActionResponse, error) {
	acct, err := s.resolveUserAccount(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Verify the message belongs to this user
	existing, err := s.stalwart.EmailGet(ctx, acct.StalwartAccountID, []string{messageID}, []string{"id", "mailboxIds"})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStalwartUnavailable, err)
	}
	if len(existing.List) == 0 {
		return nil, ErrMessageNotFound
	}

	// Build update: set ONLY the target mailbox (removes all others)
	update := map[string]any{
		"mailboxIds": map[string]bool{targetMailboxID: true},
	}

	_, err = s.stalwart.EmailSet(ctx, acct.StalwartAccountID, nil,
		map[string]any{messageID: update}, nil)
	if err != nil {
		return nil, fmt.Errorf("moving message: %w", err)
	}

	return &MessageActionResponse{
		ID:     messageID,
		Action: "move",
		Status: "applied",
	}, nil
}

// --- Internal helpers ---

// updateKeyword adds or removes a keyword on a message.
// This is idempotent: setting a keyword that already exists is a no-op.
func (s *Service) updateKeyword(ctx context.Context, userID, messageID, keyword string, add bool, action string) (*MessageActionResponse, error) {
	acct, err := s.resolveUserAccount(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Verify the message belongs to this user
	existing, err := s.stalwart.EmailGet(ctx, acct.StalwartAccountID, []string{messageID}, []string{"id", "keywords"})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStalwartUnavailable, err)
	}
	if len(existing.List) == 0 {
		return nil, ErrMessageNotFound
	}

	// Build the patch: use JMAP patch notation for keywords
	var value any
	if add {
		value = true
	} else {
		value = nil // null removes the key in JMAP patch
	}

	update := map[string]any{
		"keywords/" + keyword: value,
	}

	intentPayload := map[string]any{"message_id": messageID, "keyword": keyword, "add": add}
	reconID, err := s.reconciler.LogIntent(ctx, userID, "", "message.update", intentPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to log intent: %w", err)
	}

	_, err = s.stalwart.EmailSet(ctx, acct.StalwartAccountID, nil,
		map[string]any{messageID: update}, nil)
	if err != nil {
		s.reconciler.MarkUnknown(ctx, reconID, err)
		return nil, fmt.Errorf("updating keyword: %w", err)
	}

	eventPayload := map[string]any{"message_id": messageID, "keyword": keyword, "add": add}
	if markErr := s.reconciler.MarkSuccess(ctx, reconID, userID, "message.updated", eventPayload, nil); markErr != nil {
		slog.Error("failed to mark success in reconciliation", "error", markErr, "recon_id", reconID)
	}

	return &MessageActionResponse{
		ID:     messageID,
		Action: action,
		Status: "applied",
	}, nil
}

// moveToRole moves a message to a mailbox identified by its JMAP role.
func (s *Service) moveToRole(ctx context.Context, userID, messageID, role, action string) (*MessageActionResponse, error) {
	acct, err := s.resolveUserAccount(ctx, userID)
	if err != nil {
		return nil, err
	}

	targetMailboxID, err := s.resolveStalwartMailboxID(ctx, acct.MailboxID, role)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve %s mailbox: %w", role, err)
	}

	// Verify the message belongs to this user
	existing, err := s.stalwart.EmailGet(ctx, acct.StalwartAccountID, []string{messageID}, []string{"id", "mailboxIds"})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStalwartUnavailable, err)
	}
	if len(existing.List) == 0 {
		return nil, ErrMessageNotFound
	}

	update := map[string]any{
		"mailboxIds": map[string]bool{targetMailboxID: true},
	}

	intentPayload := map[string]any{"message_id": messageID, "role": role, "action": action}
	reconID, err := s.reconciler.LogIntent(ctx, userID, "", "message.move", intentPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to log intent: %w", err)
	}

	_, err = s.stalwart.EmailSet(ctx, acct.StalwartAccountID, nil,
		map[string]any{messageID: update}, nil)
	if err != nil {
		s.reconciler.MarkUnknown(ctx, reconID, err)
		return nil, fmt.Errorf("moving message to %s: %w", role, err)
	}

	eventPayload := map[string]any{"message_id": messageID, "role": role}
	if markErr := s.reconciler.MarkSuccess(ctx, reconID, userID, "message.moved", eventPayload, nil); markErr != nil {
		slog.Error("failed to mark success in reconciliation", "error", markErr, "recon_id", reconID)
	}

	return &MessageActionResponse{
		ID:     messageID,
		Action: action,
		Status: "applied",
	}, nil
}

// resolveUserAddress gets the email address for the authenticated user.
func (s *Service) resolveUserAddress(ctx context.Context, acct *resolvedAccount) (string, error) {
	// Try to get the full email address from the Stalwart identity first
	identity, err := s.stalwart.FindPrimaryIdentity(ctx, acct.StalwartAccountID)
	if err == nil && identity != nil && identity.Email != "" {
		return identity.Email, nil
	}

	// Fallback: resolve from the Norest DB
	mailbox, err := s.db.GetMailboxByUserID(ctx, acct.UserID)
	if err != nil {
		return "", fmt.Errorf("resolving user address: %w", err)
	}
	addr, err := s.db.GetAddressByID(ctx, mailbox.AddressID)
	if err != nil {
		return "", fmt.Errorf("resolving address: %w", err)
	}

	// The local_part alone won't work as a From address.
	// Log a warning — the identity discovery should be the primary path.
	slog.Warn("could not discover identity from Stalwart, using local part only",
		"local_part", addr.LocalPart,
		"account_id", acct.StalwartAccountID,
	)
	return addr.LocalPart, nil
}

// --- Replies & Forwarding ---

// ReplyMessage constructs a reply to a specific message and sends it.
func (s *Service) ReplyMessage(ctx context.Context, userID, messageID string, req SendRequest) (*SendResponse, error) {
	_, err := s.resolveUserAccount(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Resolve the source message
	orig, err := s.GetMessage(ctx, userID, messageID)
	if err != nil {
		return nil, err
	}

	// If To is not provided, fallback to the sender of the original message
	if len(req.To) == 0 {
		if len(orig.ReplyTo) > 0 {
			req.To = orig.ReplyTo
		} else if len(orig.From) > 0 {
			req.To = orig.From
		}
	}

	// Prepend "Re: " to subject if not present, but respect provided subject if given
	if req.Subject == "" {
		subject := orig.Subject
		if !strings.HasPrefix(strings.ToLower(subject), "re: ") {
			subject = "Re: " + subject
		}
		req.Subject = subject
	}

	// Set threading headers
	if orig.MessageID != "" {
		req.InReplyTo = []string{orig.MessageID}

		refs := make([]string, 0, len(orig.References)+1)
		refs = append(refs, orig.References...)
		refs = append(refs, orig.MessageID)
		req.References = refs
	} else {
		// Fallback to JMAP ID if Message-ID is somehow missing (though it shouldn't be)
		req.InReplyTo = []string{orig.ID}

		refs := make([]string, 0, len(orig.References)+1)
		refs = append(refs, orig.References...)
		refs = append(refs, orig.ID)
		req.References = refs
	}

	// Send message using existing send logic
	return s.SendMessage(ctx, userID, req)
}

// ReplyAllMessage constructs a reply-all to a specific message and sends it.
func (s *Service) ReplyAllMessage(ctx context.Context, userID, messageID string, req SendRequest) (*SendResponse, error) {
	acct, err := s.resolveUserAccount(ctx, userID)
	if err != nil {
		return nil, err
	}

	orig, err := s.GetMessage(ctx, userID, messageID)
	if err != nil {
		return nil, err
	}

	userAddress, err := s.resolveUserAddress(ctx, acct)
	if err != nil {
		return nil, err
	}

	// Only fallback if To is not provided by the client
	if len(req.To) == 0 {
		var to []EmailAddressDTO
		if len(orig.ReplyTo) > 0 {
			to = orig.ReplyTo
		} else if len(orig.From) > 0 {
			to = orig.From
		}
		req.To = to
	}

	// Only fallback if CC is not provided by the client
	if len(req.CC) == 0 {
		ccMap := make(map[string]EmailAddressDTO)

		addRecipients := func(addrs []EmailAddressDTO) {
			for _, a := range addrs {
				// Don't CC ourselves or the primary To
				if strings.EqualFold(a.Email, userAddress) {
					continue
				}
				isTo := false
				for _, t := range req.To {
					if strings.EqualFold(t.Email, a.Email) {
						isTo = true
						break
					}
				}
				if !isTo {
					ccMap[strings.ToLower(a.Email)] = a
				}
			}
		}

		addRecipients(orig.To)
		addRecipients(orig.CC)

		var cc []EmailAddressDTO
		for _, a := range ccMap {
			cc = append(cc, a)
		}
		req.CC = cc
	}

	if req.Subject == "" {
		subject := orig.Subject
		if !strings.HasPrefix(strings.ToLower(subject), "re: ") {
			subject = "Re: " + subject
		}
		req.Subject = subject
	}

	if orig.MessageID != "" {
		req.InReplyTo = []string{orig.MessageID}

		refs := make([]string, 0, len(orig.References)+1)
		refs = append(refs, orig.References...)
		refs = append(refs, orig.MessageID)
		req.References = refs
	} else {
		req.InReplyTo = []string{orig.ID}

		refs := make([]string, 0, len(orig.References)+1)
		refs = append(refs, orig.References...)
		refs = append(refs, orig.ID)
		req.References = refs
	}

	return s.SendMessage(ctx, userID, req)
}

// ForwardMessage forwards a specific message to new recipients.
func (s *Service) ForwardMessage(ctx context.Context, userID, messageID string, req SendRequest) (*SendResponse, error) {
	_, err := s.resolveUserAccount(ctx, userID)
	if err != nil {
		return nil, err
	}

	orig, err := s.GetMessage(ctx, userID, messageID)
	if err != nil {
		return nil, err
	}

	if req.Subject == "" {
		subject := orig.Subject
		if !strings.HasPrefix(strings.ToLower(subject), "fwd: ") {
			subject = "Fwd: " + subject
		}
		req.Subject = subject
	}

	// Combine provided attachments with original attachments
	if len(orig.Attachments) > 0 {
		req.Attachments = append(req.Attachments, orig.Attachments...)
	}

	// Forward doesn't set In-Reply-To or References per standard conventions,
	// but some clients do. We will follow standard where Fwd is a new thread.

	return s.SendMessage(ctx, userID, req)
}

// --- Attachments ---

// UploadAttachment streams a file to the user's Stalwart account and returns the Attachment metadata.
func (s *Service) UploadAttachment(ctx context.Context, userID, contentType string, body io.Reader) (*AttachmentDTO, error) {
	acct, err := s.resolveUserAccount(ctx, userID)
	if err != nil {
		return nil, err
	}

	blobResp, err := s.stalwart.UploadBlob(ctx, acct.StalwartAccountID, contentType, body)
	if err != nil {
		return nil, fmt.Errorf("uploading blob: %w", err)
	}

	return &AttachmentDTO{
		BlobID: blobResp.BlobID,
		Type:   blobResp.Type,
		Size:   blobResp.Size,
	}, nil
}

// DownloadAttachment streams a file from the user's Stalwart account.
// The caller is responsible for closing the returned io.ReadCloser.
func (s *Service) DownloadAttachment(ctx context.Context, userID, blobID, expectedType, name string) (io.ReadCloser, string, error) {
	acct, err := s.resolveUserAccount(ctx, userID)
	if err != nil {
		return nil, "", err
	}

	return s.stalwart.DownloadBlob(ctx, acct.StalwartAccountID, blobID, expectedType, name)
}

// isTimeoutOrNetworkError determines if an error is a timeout or network error
// that means the operation result is ambiguous (the operation may have succeeded).
// --- Threads ---

// ListThreads lists threads for a mailbox with pagination and filtering.
func (s *Service) ListThreads(ctx context.Context, userID string, opts ListMessagesOptions) (*ThreadListResponse, error) {
	acct, err := s.resolveUserAccount(ctx, userID)
	if err != nil {
		return nil, err
	}

	stMailboxID := ""
	if opts.MailboxID != "" {
		// Check if the provided ID is already a Stalwart mailbox ID (single character like "a", "b", "c")
		// or a Norest mailbox UUID. If it's short, assume it's already a Stalwart ID.
		if len(opts.MailboxID) <= 2 {
			stMailboxID = opts.MailboxID
		} else {
			// Otherwise, try to resolve it as a Norest mailbox UUID
			stMailboxID, err = s.resolveStalwartMailboxID(ctx, acct.MailboxID, opts.MailboxID)
			if err != nil {
				return nil, err
			}
		}
	}

	limit := opts.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var cursorTime *time.Time
	var cursorID string
	if opts.Cursor != "" {
		decoded, err := base64.StdEncoding.DecodeString(opts.Cursor)
		if err == nil {
			parts := strings.Split(string(decoded), "|")
			if len(parts) == 2 {
				t, err := time.Parse(time.RFC3339Nano, parts[0])
				if err == nil {
					cursorTime = &t
					cursorID = parts[1]
				}
			}
		}
	}

	dbThreads, err := s.db.ListThreads(ctx, acct.MailboxID, stMailboxID, limit, cursorTime, cursorID)
	if err != nil {
		return nil, fmt.Errorf("listing threads from db: %w", err)
	}

	threads := make([]ThreadResponse, 0, len(dbThreads))
	var nextCursor string

	for i, t := range dbThreads {
		threads = append(threads, ThreadResponse{
			ID:            t.ID,
			Subject:       t.Subject,
			Participants:  t.Participants,
			MessageCount:  t.MessageCount,
			UnreadCount:   t.UnreadCount,
			Snippet:       t.Snippet,
			LastMessageAt: t.LastMessageAt,
		})

		if i == len(dbThreads)-1 {
			nextCursorStr := fmt.Sprintf("%s|%s", t.LastMessageAt.Format(time.RFC3339Nano), t.ID)
			nextCursor = base64.StdEncoding.EncodeToString([]byte(nextCursorStr))
		}
	}

	return &ThreadListResponse{
		Threads:    threads,
		NextCursor: nextCursor,
	}, nil
}

// GetThread gets a single thread.
func (s *Service) GetThread(ctx context.Context, userID, threadID string) (*ThreadResponse, error) {
	acct, err := s.resolveUserAccount(ctx, userID)
	if err != nil {
		return nil, err
	}

	t, err := s.db.GetThreadAccountScoped(ctx, acct.MailboxID, threadID)
	if err != nil {
		return nil, ErrMessageNotFound
	}

	return &ThreadResponse{
		ID:            t.ID,
		Subject:       t.Subject,
		Participants:  t.Participants,
		MessageCount:  t.MessageCount,
		UnreadCount:   t.UnreadCount,
		Snippet:       t.Snippet,
		LastMessageAt: t.LastMessageAt,
	}, nil
}

// GetThreadMessages fetches all messages within a thread.
func (s *Service) GetThreadMessages(ctx context.Context, userID, threadID string) ([]MessageResponse, error) {
	acct, err := s.resolveUserAccount(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 1. Validate thread ownership
	_, err = s.db.GetThreadAccountScoped(ctx, acct.MailboxID, threadID)
	if err != nil {
		return nil, ErrMessageNotFound
	}

	// 2. Fetch all messages in the thread from PostgreSQL (chronologically ordered)
	dbMessages, err := s.db.GetMessagesByThread(ctx, acct.MailboxID, threadID)
	if err != nil {
		return nil, fmt.Errorf("fetching thread messages from db: %w", err)
	}

	if len(dbMessages) == 0 {
		return []MessageResponse{}, nil
	}

	// 3. Fetch full bodies from Stalwart
	emailIDs := make([]string, 0, len(dbMessages))
	for _, m := range dbMessages {
		emailIDs = append(emailIDs, m.StalwartEmailID)
	}

	// Bulk fetch from Stalwart to get bodies and attachments
	emailResp, err := s.stalwart.EmailGetWithBody(ctx, acct.StalwartAccountID, emailIDs)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStalwartUnavailable, err)
	}

	stalwartMap := make(map[string]*stalwart.Email)
	for _, e := range emailResp.List {
		stalwartMap[e.ID] = &e
	}

	messages := make([]MessageResponse, 0, len(dbMessages))
	for _, m := range dbMessages {
		stalwartEmail, ok := stalwartMap[m.StalwartEmailID]
		if !ok {
			// fallback/skip if stalwart is out of sync somehow
			continue
		}

		// Map from Stalwart Email to MessageResponse using existing logic
		resp := emailToMessageResponse(*stalwartEmail)

		// Override Norest fields
		resp.ThreadID = m.ThreadID

		// Fetch reactions
		if reactions, err := s.db.GetReactionsForMessage(ctx, resp.ID); err == nil && len(reactions) > 0 {
			resp.Reactions = reactions
		} else if err != nil {
			slog.Error("failed to get reactions for thread message", "error", err, "message_id", resp.ID)
		}

		messages = append(messages, resp)
	}

	return messages, nil
}

func isTimeoutOrNetworkError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "timeout") ||
		contains(errStr, "deadline exceeded") ||
		contains(errStr, "connection refused") ||
		contains(errStr, "connection reset") ||
		contains(errStr, "EOF")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// --- Sync ---

// SyncResponse contains the changes fetched from the server.
type SyncResponse struct {
	MailboxChanges *stalwart.ChangesResponse `json:"mailbox_changes,omitempty"`
	EmailChanges   *stalwart.ChangesResponse `json:"email_changes,omitempty"`
	ThreadChanges  *stalwart.ChangesResponse `json:"thread_changes,omitempty"`
	NewState       string                    `json:"new_state"`
}

// SyncMail performs an incremental sync for the user's mailbox.
func (s *Service) SyncMail(ctx context.Context, userID string) (*SyncResponse, error) {
	acct, err := s.resolveUserAccount(ctx, userID)
	if err != nil {
		return nil, err
	}

	state, err := s.db.GetSyncState(ctx, acct.MailboxID)
	if err != nil {
		return nil, err
	}

	sinceState := ""
	if state != nil {
		sinceState = state.State
	}

	// Update status to syncing
	if err := s.db.UpdateSyncState(ctx, acct.MailboxID, sinceState, "syncing", ""); err != nil {
		slog.Error("Failed to update sync state to syncing", "error", err)
	}

	// If sinceState is empty, we need to get the current state.
	// For simplicity, if sinceState is empty, we just query Email/changes with an empty string,
	// which might result in cannotCalculateChanges, in which case we'll fetch Email/get to get a valid state.
	// We'll rely on the client to do a full sync.

	var emailChanges *stalwart.ChangesResponse
	var mailboxChanges *stalwart.ChangesResponse
	var threadChanges *stalwart.ChangesResponse
	newState := sinceState

	if sinceState != "" {
		// Fetch Mailbox/changes
		mc, err := s.stalwart.GetChanges(ctx, acct.StalwartAccountID, "Mailbox", sinceState)
		if err == nil {
			mailboxChanges = mc
			newState = mc.NewState
		} else if err.Error() == "cannotCalculateChanges" {
			// Handle resync
			if err := s.db.UpdateSyncState(ctx, acct.MailboxID, "", "error", "cannotCalculateChanges"); err != nil {
				slog.Error("Failed to update sync state", "error", err)
			}
			return nil, fmt.Errorf("resync_required")
		} else {
			_ = s.db.UpdateSyncState(ctx, acct.MailboxID, sinceState, "error", err.Error())
			return nil, err
		}

		// Fetch Thread/changes
		tc, err := s.stalwart.GetChanges(ctx, acct.StalwartAccountID, "Thread", sinceState)
		if err == nil {
			threadChanges = tc
			newState = tc.NewState
		} else if err.Error() == "cannotCalculateChanges" {
			_ = s.db.UpdateSyncState(ctx, acct.MailboxID, "", "error", "cannotCalculateChanges")
			return nil, fmt.Errorf("resync_required")
		} else {
			_ = s.db.UpdateSyncState(ctx, acct.MailboxID, sinceState, "error", err.Error())
			return nil, err
		}

		// Fetch Email/changes
		ec, err := s.stalwart.GetChanges(ctx, acct.StalwartAccountID, "Email", sinceState)
		if err == nil {
			emailChanges = ec
			newState = ec.NewState
		} else if err.Error() == "cannotCalculateChanges" {
			_ = s.db.UpdateSyncState(ctx, acct.MailboxID, "", "error", "cannotCalculateChanges")
			return nil, fmt.Errorf("resync_required")
		} else {
			_ = s.db.UpdateSyncState(ctx, acct.MailboxID, sinceState, "error", err.Error())
			return nil, err
		}
	} else {
		// If sinceState is empty, we just fetch a known state and return it.
		// Email/get with no ids returns the current state.
		res, err := s.stalwart.EmailGet(ctx, acct.StalwartAccountID, []string{}, []string{"id"})
		if err != nil {
			_ = s.db.UpdateSyncState(ctx, acct.MailboxID, "", "error", err.Error())
			return nil, err
		}
		newState = res.State
	}

	// Update sync state to idle with new state
	if err := s.db.UpdateSyncState(ctx, acct.MailboxID, newState, "idle", ""); err != nil {
		slog.Error("Failed to update sync state to idle", "error", err)
	}

	return &SyncResponse{
		MailboxChanges: mailboxChanges,
		EmailChanges:   emailChanges,
		ThreadChanges:  threadChanges,
		NewState:       newState,
	}, nil
}

func buildAttachments(atts []AttachmentDTO) []map[string]any {
	if len(atts) == 0 {
		return nil
	}
	res := make([]map[string]any, len(atts))
	for i, a := range atts {
		res[i] = map[string]any{
			"blobId":      a.BlobID,
			"type":        a.Type,
			"name":        a.Name,
			"disposition": "attachment",
		}
	}
	return res
}

// ToggleReaction adds or removes an emoji reaction for a user on a message.
func (s *Service) ToggleReaction(ctx context.Context, messageID, userID, emoji string) (bool, error) {
	acct, err := s.resolveUserAccount(ctx, userID)
	if err != nil {
		return false, err
	}

	userEmail, err := s.resolveUserAddress(ctx, acct)
	if err != nil {
		return false, err
	}

	// Make sure message belongs to user
	existing, err := s.stalwart.EmailGet(ctx, acct.StalwartAccountID, []string{messageID}, []string{"id"})
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrStalwartUnavailable, err)
	}
	if len(existing.List) == 0 {
		return false, ErrMessageNotFound
	}

	return s.db.ToggleReaction(ctx, messageID, userEmail, emoji)
}
