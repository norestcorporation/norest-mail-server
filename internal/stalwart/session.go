package stalwart

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// JMAPSession represents the JMAP Session Resource (RFC 8620 Section 2).
type JMAPSession struct {
	Capabilities    map[string]json.RawMessage `json:"capabilities"`
	Accounts        map[string]JMAPAccount     `json:"accounts"`
	PrimaryAccounts map[string]string          `json:"primaryAccounts"`
	Username        string                     `json:"username"`
	APIURL          string                     `json:"apiUrl"`
	DownloadURL     string                     `json:"downloadUrl"`
	UploadURL       string                     `json:"uploadUrl"`
	EventSourceURL  string                     `json:"eventSourceUrl"`
	State           string                     `json:"state"`
}

// JMAPAccount represents an account in the JMAP Session.
type JMAPAccount struct {
	Name                string                     `json:"name"`
	IsPersonal          bool                       `json:"isPersonal"`
	IsReadOnly          bool                       `json:"isReadOnly"`
	AccountCapabilities map[string]json.RawMessage `json:"accountCapabilities"`
}

// GetJMAPWellKnown retrieves the JMAP well-known discovery document.
// This is the standard JMAP discovery mechanism per RFC 8620.
func (c *Client) GetJMAPWellKnown(ctx context.Context) (*JMAPSession, error) {
	url := c.BaseURL + "/.well-known/jmap"
	resp, err := c.doRequest(ctx, http.MethodGet, url, nil, "")
	if err != nil {
		return nil, fmt.Errorf("JMAP discovery: %w", err)
	}

	data, err := readAndClose(resp)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JMAP discovery returned status %d: %s", resp.StatusCode, string(data))
	}

	var session JMAPSession
	if err := parseJSON(data, &session); err != nil {
		return nil, fmt.Errorf("parsing JMAP session: %w", err)
	}

	return &session, nil
}

// GetJMAPSession retrieves the JMAP Session Resource from the /jmap endpoint.
// This is the primary authenticated session endpoint.
func (c *Client) GetJMAPSession(ctx context.Context) (*JMAPSession, error) {
	url := c.BaseURL + "/jmap/session"
	resp, err := c.doRequest(ctx, http.MethodGet, url, nil, "")
	if err != nil {
		return nil, fmt.Errorf("JMAP session: %w", err)
	}

	data, err := readAndClose(resp)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JMAP session returned status %d: %s", resp.StatusCode, string(data))
	}

	var session JMAPSession
	if err := parseJSON(data, &session); err != nil {
		return nil, fmt.Errorf("parsing JMAP session: %w", err)
	}

	return &session, nil
}
