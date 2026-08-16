package domains

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

// DNSChecker provides DNS verification capabilities for domain validation.
type DNSChecker struct{}

// NewDNSChecker creates a new DNS checker instance.
func NewDNSChecker() *DNSChecker {
	return &DNSChecker{}
}

// DNSVerificationResult represents the result of DNS verification checks.
type DNSVerificationResult struct {
	TXTRecordVerified bool      `json:"txt_record_verified"`
	MXRecordVerified  bool      `json:"mx_record_verified"`
	MXRecords         []string  `json:"mx_records,omitempty"`
	CheckedAt         time.Time `json:"checked_at"`
	Error             string    `json:"error,omitempty"`
}

// VerificationCheckStatus represents the status of verification checks.
type VerificationCheckStatus string

const (
	CheckStatusNotChecked VerificationCheckStatus = "not_checked"
	CheckStatusPending    VerificationCheckStatus = "pending"
	CheckStatusFailed     VerificationCheckStatus = "failed"
	CheckStatusVerified   VerificationCheckStatus = "verified"
)

// VerifyTXTRecord verifies the TXT record for domain ownership.
func (d *DNSChecker) VerifyTXTRecord(ctx context.Context, domainName, expectedToken string) (bool, error) {
	recordName := "_norest-verification." + domainName
	
	// Use context with timeout for DNS lookup
	lookupCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	
	txts, err := net.DefaultResolver.LookupTXT(lookupCtx, recordName)
	if err != nil {
		return false, fmt.Errorf("DNS TXT lookup failed: %w", err)
	}

	expectedPrefix := "norest-verification="
	for _, txt := range txts {
		if strings.HasPrefix(txt, expectedPrefix) {
			val := strings.TrimPrefix(txt, expectedPrefix)
			if val == expectedToken {
				return true, nil
			}
		}
	}

	return false, nil
}

// VerifyMXRecords verifies MX records for the domain.
func (d *DNSChecker) VerifyMXRecords(ctx context.Context, domainName string) (bool, []string, error) {
	// Use context with timeout for DNS lookup
	lookupCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	
	mxRecords, err := net.DefaultResolver.LookupMX(lookupCtx, domainName)
	if err != nil {
		return false, nil, fmt.Errorf("DNS MX lookup failed: %w", err)
	}

	if len(mxRecords) == 0 {
		return false, nil, nil
	}

	// Extract MX record hostnames
	var mxHostnames []string
	for _, mx := range mxRecords {
		mxHostnames = append(mxHostnames, mx.Host)
	}

	// For now, we consider any MX record as valid
	// In production, you might want to validate against specific mail servers
	return true, mxHostnames, nil
}

// PerformFullVerification performs both TXT and MX record verification.
func (d *DNSChecker) PerformFullVerification(ctx context.Context, domainName, expectedToken string) (*DNSVerificationResult, error) {
	result := &DNSVerificationResult{
		CheckedAt: time.Now(),
	}

	// Verify TXT record
	txtVerified, err := d.VerifyTXTRecord(ctx, domainName, expectedToken)
	if err != nil {
		result.Error = err.Error()
		return result, err
	}
	result.TXTRecordVerified = txtVerified

	// Verify MX records
	mxVerified, mxRecords, err := d.VerifyMXRecords(ctx, domainName)
	if err != nil {
		// MX record failure is not critical for ownership verification
		// but we should record it
		result.Error = fmt.Sprintf("TXT verified but MX check failed: %v", err)
		result.MXRecordVerified = false
	} else {
		result.MXRecordVerified = mxVerified
		result.MXRecords = mxRecords
	}

	return result, nil
}

// GetExpectedMXRecords returns the expected MX records for a domain.
// This is informational for the user to configure their DNS.
func (d *DNSChecker) GetExpectedMXRecords(domainName string) []string {
	// In a real deployment, these would be your actual mail server hostnames
	// For now, we return placeholder values that should be configured
	return []string{
		"mx1.norestmail.com",
		"mx2.norestmail.com",
	}
}