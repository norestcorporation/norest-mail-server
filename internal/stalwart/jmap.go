package stalwart

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// JMAPRequest represents a standard JMAP request envelope.
type JMAPRequest struct {
	Using       []string `json:"using"`
	MethodCalls []any    `json:"methodCalls"`
}

// JMAPResponse represents the outer JMAP response envelope.
type JMAPResponse struct {
	MethodResponses []json.RawMessage `json:"methodResponses"`
	SessionState    string            `json:"sessionState,omitempty"`
}

// JMAPMethodResponse represents a single parsed JMAP method response triple:
// [methodName, responseObject, callId]
type JMAPMethodResponse struct {
	MethodName string
	Data       json.RawMessage
	CallID     string
}

// JMAPError represents a JMAP-level error returned in a method response.
type JMAPError struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

func (e *JMAPError) Error() string {
	if e.Description != "" {
		return fmt.Sprintf("JMAP error [%s]: %s", e.Type, e.Description)
	}
	return fmt.Sprintf("JMAP error [%s]", e.Type)
}

// Standard JMAP using URNs for mail operations.
var jmapMailUsing = []string{
	"urn:ietf:params:jmap:core",
	"urn:ietf:params:jmap:mail",
}

// JMAP using URNs for mail operations including submission.
var jmapMailSubmissionUsing = []string{
	"urn:ietf:params:jmap:core",
	"urn:ietf:params:jmap:mail",
	"urn:ietf:params:jmap:submission",
}

// callJMAP sends a JMAP request and parses the response into method response triples.
// It returns the first method response's data or an error if the response indicates failure.
func (c *Client) callJMAP(ctx context.Context, using []string, methodCalls []any) ([]JMAPMethodResponse, error) {
	request := JMAPRequest{
		Using:       using,
		MethodCalls: methodCalls,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshaling JMAP request: %w", err)
	}

	url := c.BaseURL + "/jmap"
	resp, err := c.doRequest(ctx, http.MethodPost, url, bytes.NewReader(body), "application/json")
	if err != nil {
		return nil, fmt.Errorf("JMAP request: %w", err)
	}

	data, err := readAndClose(resp)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JMAP returned status %d: %s", resp.StatusCode, string(data))
	}

	var jmapResp JMAPResponse
	if err := parseJSON(data, &jmapResp); err != nil {
		return nil, err
	}

	if len(jmapResp.MethodResponses) == 0 {
		return nil, fmt.Errorf("no method responses in JMAP response")
	}

	var results []JMAPMethodResponse
	for _, raw := range jmapResp.MethodResponses {
		var triple []json.RawMessage
		if err := json.Unmarshal(raw, &triple); err != nil {
			return nil, fmt.Errorf("parsing method response triple: %w", err)
		}
		if len(triple) < 2 {
			return nil, fmt.Errorf("invalid method response format: expected at least 2 elements")
		}

		var methodName string
		if err := json.Unmarshal(triple[0], &methodName); err != nil {
			return nil, fmt.Errorf("parsing method name: %w", err)
		}

		// Check for JMAP-level error responses
		if methodName == "error" {
			var jmapErr JMAPError
			if err := json.Unmarshal(triple[1], &jmapErr); err != nil {
				return nil, fmt.Errorf("JMAP error (unparseable): %s", string(triple[1]))
			}
			return nil, &jmapErr
		}

		mr := JMAPMethodResponse{
			MethodName: methodName,
			Data:       triple[1],
		}
		if len(triple) >= 3 {
			json.Unmarshal(triple[2], &mr.CallID)
		}
		results = append(results, mr)
	}

	return results, nil
}

// callJMAPFirst is a convenience wrapper that calls callJMAP and returns only the
// first method response's data, unmarshaled into the target.
func (c *Client) callJMAPFirst(ctx context.Context, using []string, methodCalls []any, target any) error {
	results, err := c.callJMAP(ctx, using, methodCalls)
	if err != nil {
		return err
	}
	if len(results) == 0 {
		return fmt.Errorf("no method responses")
	}
	if err := json.Unmarshal(results[0].Data, target); err != nil {
		return fmt.Errorf("parsing method response data: %w", err)
	}
	return nil
}
