package stalwart

import (
	"context"
	"encoding/json"
	"fmt"
)

// Thread represents a JMAP Thread.
type Thread struct {
	ID       string   `json:"id"`
	EmailIDs []string `json:"emailIds"`
}

// ThreadGetResponse represents the response to a Thread/get method.
type ThreadGetResponse struct {
	AccountID string   `json:"accountId"`
	State     string   `json:"state"`
	List      []Thread `json:"list"`
	NotFound  []string `json:"notFound"`
}

// ThreadGet fetches thread by ID.
func (c *Client) ThreadGet(ctx context.Context, accountID string, threadIDs []string) (*ThreadGetResponse, error) {
	methodCalls := []any{
		[]any{
			"Thread/get",
			map[string]any{
				"accountId": accountID,
				"ids":       threadIDs,
			},
			"0",
		},
	}

	responses, err := c.callJMAP(ctx, jmapMailUsing, methodCalls)
	if err != nil {
		return nil, err
	}

	mr := responses[0]
	if mr.MethodName == "error" {
		var jmapErr JMAPError
		_ = json.Unmarshal(mr.Data, &jmapErr)
		return nil, &jmapErr
	}

	var tr ThreadGetResponse
	if err := json.Unmarshal(mr.Data, &tr); err != nil {
		return nil, fmt.Errorf("parsing Thread/get response: %w", err)
	}

	return &tr, nil
}

// ThreadQueryResponse represents the response to a Thread/query method.
type ThreadQueryResponse struct {
	AccountID        string   `json:"accountId"`
	QueryState       string   `json:"queryState"`
	CanCalculateSort bool     `json:"canCalculateSort"`
	Position         int      `json:"position"`
	Total            int      `json:"total"`
	IDs              []string `json:"ids"`
}

// ThreadQuery queries threads by mailbox or filters.
func (c *Client) ThreadQuery(ctx context.Context, accountID string, filter map[string]any, position, limit int) (*ThreadQueryResponse, error) {
	args := map[string]any{
		"accountId": accountID,
	}

	if filter != nil {
		args["filter"] = filter
	}
	if position > 0 {
		args["position"] = position
	}
	if limit > 0 {
		args["limit"] = limit
	}

	methodCalls := []any{
		[]any{
			"Thread/query",
			args,
			"0",
		},
	}

	responses, err := c.callJMAP(ctx, jmapMailUsing, methodCalls)
	if err != nil {
		return nil, err
	}

	mr := responses[0]
	if mr.MethodName == "error" {
		var jmapErr JMAPError
		_ = json.Unmarshal(mr.Data, &jmapErr)
		return nil, &jmapErr
	}

	var tr ThreadQueryResponse
	if err := json.Unmarshal(mr.Data, &tr); err != nil {
		return nil, fmt.Errorf("parsing Thread/query response: %w", err)
	}

	return &tr, nil
}
