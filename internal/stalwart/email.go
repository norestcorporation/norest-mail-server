package stalwart

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Email represents a JMAP Email object (RFC 8621).
type Email struct {
	ID            string          `json:"id"`
	BlobID        string          `json:"blobId"`
	ThreadID      string          `json:"threadId"`
	MailboxIDs    map[string]bool `json:"mailboxIds"`
	Keywords      map[string]bool `json:"keywords"`
	Size          int             `json:"size"`
	ReceivedAt    string          `json:"receivedAt"`
	From          []EmailAddress  `json:"from"`
	To            []EmailAddress  `json:"to"`
	CC            []EmailAddress  `json:"cc"`
	Subject       string          `json:"subject"`
	Preview       string          `json:"preview"`
	HasAttachment bool            `json:"hasAttachment"`
}

// EmailAddress represents a JMAP email address.
type EmailAddress struct {
	Name  string `json:"name"`
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

	methodCalls := []any{
		[]any{"Email/query", args, "eq0"},
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
		return nil, fmt.Errorf("marshaling Email/query request: %w", err)
	}

	url := c.BaseURL + "/jmap"
	resp, err := c.doRequest(ctx, http.MethodPost, url, bytes.NewReader(body), "application/json")
	if err != nil {
		return nil, fmt.Errorf("Email/query request: %w", err)
	}

	data, err := readAndClose(resp)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Email/query returned status %d: %s", resp.StatusCode, string(data))
	}

	var jmapResp struct {
		MethodResponses []json.RawMessage `json:"methodResponses"`
	}
	if err := parseJSON(data, &jmapResp); err != nil {
		return nil, err
	}

	if len(jmapResp.MethodResponses) == 0 {
		return nil, fmt.Errorf("no method responses in Email/query")
	}

	var methodResp []json.RawMessage
	if err := json.Unmarshal(jmapResp.MethodResponses[0], &methodResp); err != nil {
		return nil, fmt.Errorf("parsing method response: %w", err)
	}

	if len(methodResp) < 2 {
		return nil, fmt.Errorf("invalid method response format")
	}

	var result EmailQueryResponse
	if err := json.Unmarshal(methodResp[1], &result); err != nil {
		return nil, fmt.Errorf("parsing Email/query response: %w", err)
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

	request := map[string]any{
		"using": []string{
			"urn:ietf:params:jmap:core",
			"urn:ietf:params:jmap:mail",
		},
		"methodCalls": methodCalls,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshaling Email/get request: %w", err)
	}

	url := c.BaseURL + "/jmap"
	resp, err := c.doRequest(ctx, http.MethodPost, url, bytes.NewReader(body), "application/json")
	if err != nil {
		return nil, fmt.Errorf("Email/get request: %w", err)
	}

	data, err := readAndClose(resp)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Email/get returned status %d: %s", resp.StatusCode, string(data))
	}

	var jmapResp struct {
		MethodResponses []json.RawMessage `json:"methodResponses"`
	}
	if err := parseJSON(data, &jmapResp); err != nil {
		return nil, err
	}

	if len(jmapResp.MethodResponses) == 0 {
		return nil, fmt.Errorf("no method responses in Email/get")
	}

	var methodResp []json.RawMessage
	if err := json.Unmarshal(jmapResp.MethodResponses[0], &methodResp); err != nil {
		return nil, fmt.Errorf("parsing method response: %w", err)
	}

	if len(methodResp) < 2 {
		return nil, fmt.Errorf("invalid method response format")
	}

	var result EmailGetResponse
	if err := json.Unmarshal(methodResp[1], &result); err != nil {
		return nil, fmt.Errorf("parsing Email/get response: %w", err)
	}

	return &result, nil
}
