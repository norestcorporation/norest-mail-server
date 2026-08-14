package stalwart

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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

	request := map[string]any{
		"using": []string{
			"urn:ietf:params:jmap:core",
			"urn:ietf:params:jmap:mail",
		},
		"methodCalls": methodCalls,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshaling Mailbox/get request: %w", err)
	}

	url := c.BaseURL + "/jmap"
	resp, err := c.doRequest(ctx, http.MethodPost, url, bytes.NewReader(body), "application/json")
	if err != nil {
		return nil, fmt.Errorf("Mailbox/get request: %w", err)
	}

	data, err := readAndClose(resp)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Mailbox/get returned status %d: %s", resp.StatusCode, string(data))
	}

	var jmapResp struct {
		MethodResponses []json.RawMessage `json:"methodResponses"`
	}
	if err := parseJSON(data, &jmapResp); err != nil {
		return nil, err
	}

	if len(jmapResp.MethodResponses) == 0 {
		return nil, fmt.Errorf("no method responses in Mailbox/get")
	}

	// JMAP method responses are arrays: [methodName, responseObject, callId]
	var methodResp []json.RawMessage
	if err := json.Unmarshal(jmapResp.MethodResponses[0], &methodResp); err != nil {
		return nil, fmt.Errorf("parsing method response: %w", err)
	}

	if len(methodResp) < 2 {
		return nil, fmt.Errorf("invalid method response format")
	}

	var result MailboxGetResponse
	if err := json.Unmarshal(methodResp[1], &result); err != nil {
		return nil, fmt.Errorf("parsing Mailbox/get response: %w", err)
	}

	return &result, nil
}
