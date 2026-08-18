package mail

import (
	"encoding/json"
	"time"

	"github.com/norest-mail/server/internal/stalwart"
)

// --- Norest REST DTOs ---
// These are the normalized response types returned by the Norest API.
// They do NOT expose raw JMAP structures.

// MailboxResponse is the Norest representation of a mailbox/folder.
type MailboxResponse struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Role          string `json:"role,omitempty"`
	ParentID      string `json:"parent_id,omitempty"`
	TotalEmails   int    `json:"total_emails"`
	UnreadEmails  int    `json:"unread_emails"`
	TotalThreads  int    `json:"total_threads"`
	UnreadThreads int    `json:"unread_threads"`
}

// MailboxListResponse is the paginated list of mailboxes.
type MailboxListResponse struct {
	Mailboxes []MailboxResponse `json:"mailboxes"`
}

// EmailAddressDTO is a normalized email address for Norest responses.
type EmailAddressDTO struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email"`
}

// AttachmentDTO represents a file attachment.
type AttachmentDTO struct {
	BlobID string `json:"blob_id"`
	Type   string `json:"type"`
	Name   string `json:"name,omitempty"`
	Size   int    `json:"size,omitempty"`
}

// MessageResponse is the Norest representation of an email message.
type MessageResponse struct {
	ID            string            `json:"id"`
	ThreadID      string            `json:"thread_id,omitempty"`
	MailboxIDs    []string          `json:"mailbox_ids,omitempty"`
	From          []EmailAddressDTO `json:"from,omitempty"`
	To            []EmailAddressDTO `json:"to,omitempty"`
	CC            []EmailAddressDTO `json:"cc,omitempty"`
	BCC           []EmailAddressDTO `json:"bcc,omitempty"`
	ReplyTo       []EmailAddressDTO `json:"reply_to,omitempty"`
	Subject       string            `json:"subject"`
	Preview       string            `json:"preview,omitempty"`
	ReceivedAt    string            `json:"received_at,omitempty"`
	SentAt        string            `json:"sent_at,omitempty"`
	Size          int               `json:"size,omitempty"`
	HasAttachment bool              `json:"has_attachment"`
	Attachments   []AttachmentDTO   `json:"attachments,omitempty"`
	IsRead        bool              `json:"is_read"`
	IsStarred     bool              `json:"is_starred"`
	IsDraft       bool              `json:"is_draft"`
	// Body content — present only for single-message GET, not list.
	TextBody string `json:"text_body,omitempty"`
	HTMLBody string `json:"html_body,omitempty"`
	// Threading
	MessageID  string   `json:"message_id,omitempty"`
	InReplyTo  []string `json:"in_reply_to,omitempty"`
	References []string `json:"references,omitempty"`
}

// MessageListResponse is the paginated list of messages.
type MessageListResponse struct {
	Messages []MessageResponse `json:"messages"`
	Total    int               `json:"total"`
	Position int               `json:"position"`
}

// ListMessagesOptions controls filtering, sorting, and pagination for message listing.
type ListMessagesOptions struct {
	MailboxID string `json:"mailbox_id,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	Position  int    `json:"position,omitempty"` // For offset pagination (messages)
	Cursor    string `json:"cursor,omitempty"`   // For keyset pagination (threads)
	// Filtering
	From          string `json:"from,omitempty"`
	To            string `json:"to,omitempty"`
	CC            string `json:"cc,omitempty"`
	Subject       string `json:"subject,omitempty"`
	Text          string `json:"text,omitempty"`
	Before        string `json:"before,omitempty"`
	After         string `json:"after,omitempty"`
	HasKeyword    string `json:"has_keyword,omitempty"`
	NotHasKeyword string `json:"not_has_keyword,omitempty"`
	ThreadID      string `json:"thread_id,omitempty"`
	HasAttachment bool   `json:"has_attachment,omitempty"`
}

// ThreadResponse represents a message thread.
type ThreadResponse struct {
	ID            string          `json:"id"`
	Subject       string          `json:"subject"`
	Participants  json.RawMessage `json:"participants"`
	MessageCount  int             `json:"message_count"`
	UnreadCount   int             `json:"unread_count"`
	Snippet       *string         `json:"snippet"`
	LastMessageAt time.Time       `json:"last_message_at"`
}

// ThreadListResponse represents a paginated list of threads.
type ThreadListResponse struct {
	Threads    []ThreadResponse `json:"threads"`
	NextCursor string           `json:"next_cursor,omitempty"`
}

// DraftRequest is the Norest DTO for creating or updating a draft.
type DraftRequest struct {
	To            []EmailAddressDTO `json:"to,omitempty"`
	CC            []EmailAddressDTO `json:"cc,omitempty"`
	BCC           []EmailAddressDTO `json:"bcc,omitempty"`
	Subject       string            `json:"subject,omitempty"`
	TextBody      string            `json:"text_body,omitempty"`
	HTMLBody      string            `json:"html_body,omitempty"`
	AttachmentIDs []string          `json:"attachment_ids,omitempty"`
	// For reply/forward
	InReplyTo   []string        `json:"in_reply_to,omitempty"`
	References  []string        `json:"references,omitempty"`
	Attachments []AttachmentDTO `json:"attachments,omitempty"`
}

// DraftResponse is returned after creating or updating a draft.
type DraftResponse struct {
	ID            string            `json:"id"`
	BlobID        string            `json:"blob_id,omitempty"`
	Message       string            `json:"message,omitempty"`
	To            []EmailAddressDTO `json:"to,omitempty"`
	CC            []EmailAddressDTO `json:"cc,omitempty"`
	BCC           []EmailAddressDTO `json:"bcc,omitempty"`
	Subject       string            `json:"subject,omitempty"`
	TextBody      string            `json:"text_body,omitempty"`
	HTMLBody      string            `json:"html_body,omitempty"`
	AttachmentIDs []string          `json:"attachment_ids,omitempty"`
	CreatedAt     string            `json:"created_at,omitempty"`
	UpdatedAt     string            `json:"updated_at,omitempty"`
}

// SendRequest is the Norest DTO for sending a message.
type SendRequest struct {
	// Either provide draft_id to send an existing draft,
	// or provide the full message content to create-and-send.
	DraftID  string            `json:"draft_id,omitempty"`
	To       []EmailAddressDTO `json:"to,omitempty"`
	CC       []EmailAddressDTO `json:"cc,omitempty"`
	BCC      []EmailAddressDTO `json:"bcc,omitempty"`
	Subject  string            `json:"subject,omitempty"`
	TextBody string            `json:"text_body,omitempty"`
	HTMLBody string            `json:"html_body,omitempty"`
	// For reply/forward
	InReplyTo   []string        `json:"in_reply_to,omitempty"`
	References  []string        `json:"references,omitempty"`
	Attachments []AttachmentDTO `json:"attachments,omitempty"`
	// Idempotency
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

// SendResponse is returned after sending a message.
type SendResponse struct {
	MessageID    string `json:"message_id,omitempty"`
	SubmissionID string `json:"submission_id,omitempty"`
	Status       string `json:"status"` // "sent", "queued", "unknown"
}

// MessageActionResponse is returned after a message action (read, star, etc.)
type MessageActionResponse struct {
	ID     string `json:"id"`
	Action string `json:"action"`
	Status string `json:"status"` // "applied"
}

// MoveRequest is the body for a move operation.
type MoveRequest struct {
	MailboxID string `json:"mailbox_id"`
}

// --- Conversion helpers ---

func toEmailAddressDTOs(addrs []stalwart.EmailAddress) []EmailAddressDTO {
	if len(addrs) == 0 {
		return nil
	}
	result := make([]EmailAddressDTO, len(addrs))
	for i, a := range addrs {
		result[i] = EmailAddressDTO{Name: a.Name, Email: a.Email}
	}
	return result
}

func toStalwartAddresses(addrs []EmailAddressDTO) []stalwart.EmailAddress {
	if len(addrs) == 0 {
		return nil
	}
	result := make([]stalwart.EmailAddress, len(addrs))
	for i, a := range addrs {
		result[i] = stalwart.EmailAddress{Name: a.Name, Email: a.Email}
	}
	return result
}

func mailboxIDsToSlice(ids map[string]bool) []string {
	if len(ids) == 0 {
		return nil
	}
	result := make([]string, 0, len(ids))
	for id, v := range ids {
		if v {
			result = append(result, id)
		}
	}
	return result
}

// emailToMessageResponse converts a JMAP Email to a Norest MessageResponse.
func emailToMessageResponse(e stalwart.Email) MessageResponse {
	msg := MessageResponse{
		ID:            e.ID,
		ThreadID:      e.ThreadID,
		MailboxIDs:    mailboxIDsToSlice(e.MailboxIDs),
		From:          toEmailAddressDTOs(e.From),
		To:            toEmailAddressDTOs(e.To),
		CC:            toEmailAddressDTOs(e.CC),
		BCC:           toEmailAddressDTOs(e.BCC),
		ReplyTo:       toEmailAddressDTOs(e.ReplyTo),
		Subject:       e.Subject,
		Preview:       e.Preview,
		ReceivedAt:    e.ReceivedAt,
		SentAt:        e.SentAt,
		Size:          e.Size,
		HasAttachment: e.HasAttachment,
		IsRead:        e.Keywords["$seen"],
		IsStarred:     e.Keywords["$flagged"],
		IsDraft:       e.Keywords["$draft"],
		InReplyTo:     e.InReplyTo,
		References:    e.References,
	}

	if len(e.MessageId) > 0 {
		msg.MessageID = e.MessageId[0]
	}

	if len(e.Attachments) > 0 {
		msg.Attachments = make([]AttachmentDTO, len(e.Attachments))
		for i, att := range e.Attachments {
			msg.Attachments[i] = AttachmentDTO{
				BlobID: att.BlobID,
				Type:   att.Type,
				Name:   att.Name,
				Size:   att.Size,
			}
		}
	}

	// Extract body text from bodyValues if present
	for _, part := range e.TextBody {
		if bv, ok := e.BodyValues[part.PartID]; ok {
			msg.TextBody = bv.Value
			break
		}
	}
	for _, part := range e.HTMLBody {
		if bv, ok := e.BodyValues[part.PartID]; ok {
			msg.HTMLBody = bv.Value
			break
		}
	}

	return msg
}

// mailboxToResponse converts a JMAP Mailbox to a Norest MailboxResponse.
func mailboxToResponse(m stalwart.Mailbox) MailboxResponse {
	return MailboxResponse{
		ID:            m.ID,
		Name:          m.Name,
		Role:          m.Role,
		ParentID:      m.ParentID,
		TotalEmails:   m.TotalEmails,
		UnreadEmails:  m.UnreadEmails,
		TotalThreads:  m.TotalThreads,
		UnreadThreads: m.UnreadThreads,
	}
}
