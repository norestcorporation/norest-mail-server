package stalwart

import (
	"context"
	"encoding/json"
	"fmt"
)

// ChangesResponse represents the response for a changes method (e.g., Email/changes).
type ChangesResponse struct {
	AccountID  string   `json:"accountId"`
	OldState   string   `json:"oldState"`
	NewState   string   `json:"newState"`
	HasMore    bool     `json:"hasMoreChanges"`
	Created    []string `json:"created"`
	Updated    []string `json:"updated"`
	Destroyed  []string `json:"destroyed"`
}

// GetChanges fetches changes for a given entity (Email, Mailbox, Thread).
func (c *Client) GetChanges(ctx context.Context, accountID, entity, sinceState string) (*ChangesResponse, error) {
	methodCalls := []any{
		[]any{
			fmt.Sprintf("%s/changes", entity),
			map[string]any{
				"accountId": accountID,
				"sinceState": sinceState,
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
		
		// If cannotCalculateChanges, we must return a specific error
		if jmapErr.Type == "cannotCalculateChanges" {
			return nil, fmt.Errorf("cannotCalculateChanges")
		}
		
		return nil, &jmapErr
	}

	var cr ChangesResponse
	if err := json.Unmarshal(mr.Data, &cr); err != nil {
		return nil, fmt.Errorf("parsing %s/changes response: %w", entity, err)
	}

	return &cr, nil
}
