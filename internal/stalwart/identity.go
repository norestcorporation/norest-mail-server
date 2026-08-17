package stalwart

import (
	"context"
	"fmt"
)

// Identity represents a JMAP Identity object used for sending.
type Identity struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Email         string         `json:"email"`
	ReplyTo       []EmailAddress `json:"replyTo,omitempty"`
	BCC           []EmailAddress `json:"bcc,omitempty"`
	TextSignature string         `json:"textSignature,omitempty"`
	HTMLSignature string         `json:"htmlSignature,omitempty"`
	MayDelete     bool           `json:"mayDelete"`
}

// IdentityGetResponse represents the response to an Identity/get JMAP call.
type IdentityGetResponse struct {
	AccountID string     `json:"accountId"`
	State     string     `json:"state"`
	List      []Identity `json:"list"`
	NotFound  []string   `json:"notFound"`
}

// IdentityGet retrieves the sending identities for an account.
// Each identity represents a "from" address the account is allowed to use.
func (c *Client) IdentityGet(ctx context.Context, accountID string) (*IdentityGetResponse, error) {
	args := map[string]any{
		"accountId": accountID,
	}

	methodCalls := []any{
		[]any{"Identity/get", args, "ig0"},
	}

	var result IdentityGetResponse
	if err := c.callJMAPFirst(ctx, jmapMailSubmissionUsing, methodCalls, &result); err != nil {
		return nil, fmt.Errorf("Identity/get: %w", err)
	}

	return &result, nil
}

// FindPrimaryIdentity returns the first identity for the account, or an error if none exist.
func (c *Client) FindPrimaryIdentity(ctx context.Context, accountID string) (*Identity, error) {
	resp, err := c.IdentityGet(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if len(resp.List) == 0 {
		return nil, fmt.Errorf("no sending identities found for account %s", accountID)
	}
	return &resp.List[0], nil
}
