package stalwart

import (
	"context"
	"fmt"
)

// Mailbox represents a JMAP Mailbox object (RFC 8621).
type Mailbox struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	ParentID      string `json:"parentId,omitempty"`
	Role          string `json:"role,omitempty"`
	TotalEmails   int    `json:"totalEmails"`
	UnreadEmails  int    `json:"unreadEmails"`
	TotalThreads  int    `json:"totalThreads"`
	UnreadThreads int    `json:"unreadThreads"`
}

// MailboxGetResponse represents the response to a Mailbox/get JMAP call.
type MailboxGetResponse struct {
	AccountID string    `json:"accountId"`
	State     string    `json:"state"`
	List      []Mailbox `json:"list"`
	NotFound  []string  `json:"notFound"`
}

// DiscoverMailboxes retrieves all mailboxes for an account and returns role-to-ID mappings.
func (c *Client) DiscoverMailboxes(ctx context.Context, accountID string) (map[string]string, error) {
	response, err := c.MailboxGet(ctx, accountID, nil)
	if err != nil {
		return nil, fmt.Errorf("getting mailboxes: %w", err)
	}

	mappings := make(map[string]string)
	for _, mailbox := range response.List {
		if mailbox.Role != "" {
			mappings[mailbox.Role] = mailbox.ID
		}
	}
	return mappings, nil
}

// GetMailboxByName retrieves a specific mailbox by name for an account.
func (c *Client) GetMailboxByName(ctx context.Context, accountID, name string) (*Mailbox, error) {
	response, err := c.MailboxGet(ctx, accountID, nil)
	if err != nil {
		return nil, fmt.Errorf("getting mailboxes: %w", err)
	}

	for _, mailbox := range response.List {
		if mailbox.Name == name {
			return &mailbox, nil
		}
	}
	return nil, fmt.Errorf("mailbox not found: %s", name)
}

// MailboxGet retrieves mailboxes for the given account using the JMAP Mailbox/get method.
func (c *Client) MailboxGet(ctx context.Context, accountID string, ids []string) (*MailboxGetResponse, error) {
	args := map[string]any{
		"accountId": accountID,
	}
	if ids != nil {
		args["ids"] = ids
	}

	methodCalls := []any{
		[]any{"Mailbox/get", args, "mg0"},
	}

	var result MailboxGetResponse
	if err := c.callJMAPFirst(ctx, jmapMailUsing, methodCalls, &result); err != nil {
		return nil, fmt.Errorf("Mailbox/get: %w", err)
	}

	return &result, nil
}

// MailboxSetResponse represents the response to a Mailbox/set JMAP call.
type MailboxSetResponse struct {
	AccountID  string                       `json:"accountId"`
	OldState   string                       `json:"oldState"`
	NewState   string                       `json:"newState"`
	Created    map[string]Mailbox           `json:"created"`
	Updated    map[string]any               `json:"updated"`
	Destroyed  []string                     `json:"destroyed"`
	NotCreated map[string]EmailSetError     `json:"notCreated"`
	NotUpdated map[string]EmailSetError     `json:"notUpdated"`
	NotDestroyed map[string]EmailSetError   `json:"notDestroyed"`
}

// MailboxSet creates, updates, or destroys mailboxes for the given account.
func (c *Client) MailboxSet(ctx context.Context, accountID string, create map[string]any, update map[string]any, destroy []string) (*MailboxSetResponse, error) {
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
		[]any{"Mailbox/set", args, "ms0"},
	}

	var result MailboxSetResponse
	if err := c.callJMAPFirst(ctx, jmapMailUsing, methodCalls, &result); err != nil {
		return nil, fmt.Errorf("Mailbox/set: %w", err)
	}

	return &result, nil
}
