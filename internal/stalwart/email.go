package stalwart

import (
	"context"
	"fmt"
)

// Email represents a JMAP Email object (RFC 8621).
type Email struct {
	ID            string          `json:"id,omitempty"`
	BlobID        string          `json:"blobId,omitempty"`
	ThreadID      string          `json:"threadId,omitempty"`
	MailboxIDs    map[string]bool `json:"mailboxIds,omitempty"`
	Keywords      map[string]bool `json:"keywords,omitempty"`
	Size          int             `json:"size,omitempty"`
	ReceivedAt    string          `json:"receivedAt,omitempty"`
	From          []EmailAddress  `json:"from,omitempty"`
	To            []EmailAddress  `json:"to,omitempty"`
	CC            []EmailAddress  `json:"cc,omitempty"`
	BCC           []EmailAddress  `json:"bcc,omitempty"`
	ReplyTo       []EmailAddress  `json:"replyTo,omitempty"`
	Subject       string          `json:"subject,omitempty"`
	Preview       string          `json:"preview,omitempty"`
	HasAttachment bool            `json:"hasAttachment,omitempty"`
	MessageId     []string        `json:"messageId,omitempty"`
	InReplyTo     []string        `json:"inReplyTo,omitempty"`
	References    []string        `json:"references,omitempty"`
	SentAt        string          `json:"sentAt,omitempty"`
	// Body content — only populated when explicitly requested via properties.
	TextBody []EmailBodyPart `json:"textBody,omitempty"`
	HTMLBody []EmailBodyPart `json:"htmlBody,omitempty"`
	Attachments []EmailBodyPart `json:"attachments,omitempty"`
	BodyValues map[string]EmailBodyValue `json:"bodyValues,omitempty"`
}

// EmailBodyPart references a body part in JMAP Email.
type EmailBodyPart struct {
	PartID      string `json:"partId,omitempty"`
	BlobID      string `json:"blobId,omitempty"`
	Type        string `json:"type,omitempty"`
	Name        string `json:"name,omitempty"`
	Disposition string `json:"disposition,omitempty"`
	Charset     string `json:"charset,omitempty"`
	Size        int    `json:"size,omitempty"`
}

// EmailBodyValue holds the actual text content for a body part.
type EmailBodyValue struct {
	Value              string `json:"value"`
	IsEncodingProblem  bool   `json:"isEncodingProblem,omitempty"`
	IsTruncated        bool   `json:"isTruncated,omitempty"`
}

// EmailAddress represents a JMAP email address.
type EmailAddress struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email"`
}

// EmailQueryResponse represents the response to an Email/query JMAP call.
type EmailQueryResponse struct {
	AccountID           string   `json:"accountId"`
	QueryState          string   `json:"queryState"`
	CanCalculateChanges bool     `json:"canCalculateChanges"`
	Position            int      `json:"position"`
	Total               int      `json:"total"`
	IDs                 []string `json:"ids"`
}

// EmailGetResponse represents the response to an Email/get JMAP call.
type EmailGetResponse struct {
	AccountID string   `json:"accountId"`
	State     string   `json:"state"`
	List      []Email  `json:"list"`
	NotFound  []string `json:"notFound"`
}

// EmailSetResponse represents the response to an Email/set JMAP call.
type EmailSetResponse struct {
	AccountID  string                       `json:"accountId"`
	OldState   string                       `json:"oldState"`
	NewState   string                       `json:"newState"`
	Created    map[string]Email             `json:"created"`
	Updated    map[string]any               `json:"updated"`
	Destroyed  []string                     `json:"destroyed"`
	NotCreated map[string]EmailSetError     `json:"notCreated"`
	NotUpdated map[string]EmailSetError     `json:"notUpdated"`
	NotDestroyed map[string]EmailSetError   `json:"notDestroyed"`
}

// EmailSetError represents an error for a specific object in Email/set.
type EmailSetError struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

// EmailQuery queries for email IDs matching the given filter.
func (c *Client) EmailQuery(ctx context.Context, accountID string, filter map[string]any, sort []map[string]any, limit int) (*EmailQueryResponse, error) {
	args := map[string]any{
		"accountId": accountID,
	}
	if filter != nil {
		args["filter"] = filter
	}
	if sort != nil {
		args["sort"] = sort
	}
	if limit > 0 {
		args["limit"] = limit
	}
	args["calculateTotal"] = true

	methodCalls := []any{
		[]any{"Email/query", args, "eq0"},
	}

	var result EmailQueryResponse
	if err := c.callJMAPFirst(ctx, jmapMailUsing, methodCalls, &result); err != nil {
		return nil, fmt.Errorf("Email/query: %w", err)
	}

	return &result, nil
}

// EmailQueryWithPosition queries for email IDs with pagination position support.
func (c *Client) EmailQueryWithPosition(ctx context.Context, accountID string, filter map[string]any, sort []map[string]any, position, limit int) (*EmailQueryResponse, error) {
	args := map[string]any{
		"accountId": accountID,
	}
	if filter != nil {
		args["filter"] = filter
	}
	if sort != nil {
		args["sort"] = sort
	}
	if position > 0 {
		args["position"] = position
	}
	if limit > 0 {
		args["limit"] = limit
	}
	args["calculateTotal"] = true

	methodCalls := []any{
		[]any{"Email/query", args, "eq0"},
	}

	var result EmailQueryResponse
	if err := c.callJMAPFirst(ctx, jmapMailUsing, methodCalls, &result); err != nil {
		return nil, fmt.Errorf("Email/query: %w", err)
	}

	return &result, nil
}

// EmailGet fetches email objects by their IDs.
func (c *Client) EmailGet(ctx context.Context, accountID string, ids []string, properties []string) (*EmailGetResponse, error) {
	args := map[string]any{
		"accountId": accountID,
		"ids":       ids,
	}
	if properties != nil {
		args["properties"] = properties
	}

	methodCalls := []any{
		[]any{"Email/get", args, "eg0"},
	}

	var result EmailGetResponse
	if err := c.callJMAPFirst(ctx, jmapMailUsing, methodCalls, &result); err != nil {
		return nil, fmt.Errorf("Email/get: %w", err)
	}

	return &result, nil
}

// EmailGetWithBody fetches email objects with body content included.
func (c *Client) EmailGetWithBody(ctx context.Context, accountID string, ids []string) (*EmailGetResponse, error) {
	args := map[string]any{
		"accountId": accountID,
		"ids":       ids,
			"properties": []string{
				"id", "blobId", "threadId", "mailboxIds", "keywords",
				"size", "receivedAt", "from", "to", "cc", "bcc", "replyTo",
				"subject", "preview", "hasAttachment", "attachments",
				"messageId", "inReplyTo", "references", "sentAt",
				"textBody", "htmlBody", "bodyValues",
			},
		"fetchAllBodyValues": true,
	}

	methodCalls := []any{
		[]any{"Email/get", args, "eg0"},
	}

	var result EmailGetResponse
	if err := c.callJMAPFirst(ctx, jmapMailUsing, methodCalls, &result); err != nil {
		return nil, fmt.Errorf("Email/get (with body): %w", err)
	}

	return &result, nil
}

// EmailSet performs create, update, and/or destroy operations on emails.
// This is a generic adapter — callers compose the specific operation maps.
//
// For create: map[string]any where key is a creation ID, value is the email object.
// For update: map[string]any where key is the email ID, value is the patch object.
// For destroy: []string of email IDs to destroy.
func (c *Client) EmailSet(ctx context.Context, accountID string, create map[string]any, update map[string]any, destroy []string) (*EmailSetResponse, error) {
	args := map[string]any{
		"accountId": accountID,
	}
	if create != nil {
		args["create"] = create
	}
	if update != nil {
		args["update"] = update
	}
	if destroy != nil {
		args["destroy"] = destroy
	}

	methodCalls := []any{
		[]any{"Email/set", args, "es0"},
	}

	var result EmailSetResponse
	if err := c.callJMAPFirst(ctx, jmapMailUsing, methodCalls, &result); err != nil {
		return nil, fmt.Errorf("Email/set: %w", err)
	}

	// Check for partial failures in the response
	if len(result.NotCreated) > 0 {
		for key, setErr := range result.NotCreated {
			return &result, fmt.Errorf("Email/set create failed for %s: [%s] %s", key, setErr.Type, setErr.Description)
		}
	}
	if len(result.NotUpdated) > 0 {
		for key, setErr := range result.NotUpdated {
			return &result, fmt.Errorf("Email/set update failed for %s: [%s] %s", key, setErr.Type, setErr.Description)
		}
	}
	if len(result.NotDestroyed) > 0 {
		for key, setErr := range result.NotDestroyed {
			return &result, fmt.Errorf("Email/set destroy failed for %s: [%s] %s", key, setErr.Type, setErr.Description)
		}
	}

	return &result, nil
}
