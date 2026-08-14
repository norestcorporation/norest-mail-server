package stalwart

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// Client provides a first-class abstraction for communicating with Stalwart.
// All Stalwart-specific HTTP details are encapsulated here.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	Username   string
	Password   string
}

// NewClient creates a Stalwart client configured with appropriate timeouts.
func NewClient(baseURL, username, password string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		Username: username,
		Password: password,
	}
}

// doRequest executes an HTTP request with basic auth and proper cleanup.
func (c *Client) doRequest(ctx context.Context, method, url string, body io.Reader, contentType string) (*http.Response, error) {
	// Use a per-request timeout for safety
	reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	req, err := http.NewRequestWithContext(reqCtx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.SetBasicAuth(c.Username, c.Password)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}

	// Enforce safe response size limits (max 10MB for responses)
	if resp.ContentLength > 10*1024*1024 {
		resp.Body.Close()
		return nil, fmt.Errorf("response too large: %d bytes", resp.ContentLength)
	}

	return resp, nil
}

// readAndClose reads the response body and closes it with size limits.
func readAndClose(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	
	// Limit the response body to 10MB for safety
	limitedReader := io.LimitReader(resp.Body, 10*1024*1024)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}
	return data, nil
}

// HealthCheck verifies that Stalwart is reachable by hitting /.well-known/jmap.
func (c *Client) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	url := c.BaseURL + "/.well-known/jmap"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("creating health check request: %w", err)
	}

	req.SetBasicAuth(c.Username, c.Password)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("stalwart unreachable: %w", err)
	}
	defer resp.Body.Close()

	// Stalwart may return 401 if not yet bootstrapped, but the server is reachable.
	// We accept any response that shows the server is responding.
	if resp.StatusCode >= 500 {
		return fmt.Errorf("stalwart returned server error: %d", resp.StatusCode)
	}

	slog.Debug("stalwart health check passed", "status", resp.StatusCode)
	return nil
}

// parseJSON is a helper to decode JSON responses.
func parseJSON(data []byte, target any) error {
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("parsing JSON: %w", err)
	}
	return nil
}
