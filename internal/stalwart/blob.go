package stalwart

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// BlobUploadResponse represents the JMAP upload response.
type BlobUploadResponse struct {
	AccountID string `json:"accountId"`
	BlobID    string `json:"blobId"`
	Type      string `json:"type"`
	Size      int    `json:"size"`
}

// UploadBlob streams a file to the JMAP upload endpoint and returns the blob info.
func (c *Client) UploadBlob(ctx context.Context, accountID, contentType string, body io.Reader) (*BlobUploadResponse, error) {
	// First get the JMAP session to find the UploadURL
	session, err := c.GetJMAPSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting JMAP session for upload: %w", err)
	}

	uploadURL := strings.ReplaceAll(session.UploadURL, "{accountId}", accountID)
	if uploadURL == "" {
		return nil, fmt.Errorf("no uploadUrl found in JMAP session")
	}

	importLog := "import(\"log/slog\")" // Just a dummy string, we already import log in client.go, wait blob.go doesn't import slog. I will use fmt.Printf
	_ = importLog
	fmt.Printf("BEFORE REPLACEMENT: %s\n", uploadURL)

	// Quick fix for docker networking where Stalwart returns localhost instead of internal hostname
	if strings.Contains(uploadURL, "localhost") {
		// e.g. https://localhost/jmap/... -> http://stalwart:8080/jmap/...
		idx := strings.Index(uploadURL, "/jmap")
		if idx != -1 {
			uploadURL = c.BaseURL + uploadURL[idx:]
		}
	}

	fmt.Printf("AFTER REPLACEMENT: %s\n", uploadURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, body)
	if err != nil {
		return nil, fmt.Errorf("creating upload request: %w", err)
	}

	req.SetBasicAuth(c.Username, c.Password)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	// Do NOT use c.doRequest because we want to rely on the parent ctx timeout for streaming uploads,
	// not the hardcoded 30s timeout in doRequest, which might be too short for slow uploads.
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing upload request: %w", err)
	}

	data, err := readAndClose(resp)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upload returned status %d: %s", resp.StatusCode, string(data))
	}

	var blobResp BlobUploadResponse
	if err := parseJSON(data, &blobResp); err != nil {
		return nil, fmt.Errorf("parsing upload response: %w", err)
	}

	return &blobResp, nil
}

// DownloadBlob streams a file from the JMAP download endpoint.
// The caller is responsible for closing the returned io.ReadCloser.
// Returns the response body, the resolved content type, and an error.
func (c *Client) DownloadBlob(ctx context.Context, accountID, blobID, expectedType, name string) (io.ReadCloser, string, error) {
	session, err := c.GetJMAPSession(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("getting JMAP session for download: %w", err)
	}

	downloadURL := session.DownloadURL
	downloadURL = strings.ReplaceAll(downloadURL, "{accountId}", accountID)
	downloadURL = strings.ReplaceAll(downloadURL, "{blobId}", blobID)
	downloadURL = strings.ReplaceAll(downloadURL, "{type}", expectedType)
	downloadURL = strings.ReplaceAll(downloadURL, "{name}", name)

	if strings.Contains(downloadURL, "localhost") {
		idx := strings.Index(downloadURL, "/jmap")
		if idx != -1 {
			downloadURL = c.BaseURL + downloadURL[idx:]
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("creating download request: %w", err)
	}

	req.SetBasicAuth(c.Username, c.Password)

	// Since we are returning the response body for streaming, we cannot use doRequest's internal context/cancel
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("executing download request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		data, _ := readAndClose(resp)
		return nil, "", fmt.Errorf("download returned status %d: %s", resp.StatusCode, string(data))
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	return resp.Body, contentType, nil
}
