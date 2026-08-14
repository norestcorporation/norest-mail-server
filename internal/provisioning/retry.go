package provisioning

import (
	"math/rand"
	"time"
)

// ErrorType classifies errors for retry decisions.
type ErrorType string

const (
	ErrorTypeTemporary    ErrorType = "TEMPORARY"
	ErrorTypeTimeout      ErrorType = "TIMEOUT"
	ErrorTypeRateLimited  ErrorType = "RATE_LIMITED"
	ErrorTypeConflict     ErrorType = "CONFLICT"
	ErrorTypeAlreadyExists ErrorType = "ALREADY_EXISTS"
	ErrorTypeNotFound     ErrorType = "NOT_FOUND"
	ErrorTypeUnauthorized ErrorType = "UNAUTHORIZED"
	ErrorTypeForbidden    ErrorType = "FORBIDDEN"
	ErrorTypeInvalid      ErrorType = "INVALID_REQUEST"
	ErrorTypePermanent    ErrorType = "PERMANENT"
)

// RetryPolicy defines retry behavior.
type RetryPolicy struct {
	MaxAttempts      int
	InitialBackoff   time.Duration
	MaxBackoff       time.Duration
	JitterFraction   float64
}

// DefaultRetryPolicy returns a sensible default retry policy.
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxAttempts:      10,
		InitialBackoff:   2 * time.Second,
		MaxBackoff:       5 * time.Minute,
		JitterFraction:   0.25,
	}
}

// IsRetryable determines if an error type should be retried.
func (p *RetryPolicy) IsRetryable(errorType ErrorType) bool {
	switch errorType {
	case ErrorTypeTemporary, ErrorTypeTimeout, ErrorTypeRateLimited:
		return true
	case ErrorTypeConflict, ErrorTypeAlreadyExists:
		// These may be retried with idempotency checks
		return true
	case ErrorTypeNotFound:
		// May be retried for idempotency
		return true
	default:
		return false
	}
}

// CalculateBackoff computes the next attempt time with exponential backoff and jitter.
func (p *RetryPolicy) CalculateBackoff(attempt int) time.Duration {
	if attempt < 1 {
		return p.InitialBackoff
	}

	// Exponential backoff: 2^(attempt-1) * initial
	power := 1 << uint(attempt-1)
	exponential := float64(power) * p.InitialBackoff.Seconds()

	// Cap at max backoff
	if exponential > p.MaxBackoff.Seconds() {
		exponential = p.MaxBackoff.Seconds()
	}

	// Add jitter: ±jitterFraction
	jitter := (rand.Float64() - 0.5) * 2 * p.JitterFraction * exponential
	backoff := exponential + jitter

	if backoff < 0 {
		backoff = exponential * 0.5
	}

	return time.Duration(backoff * float64(time.Second))
}

// ClassifyHTTPError converts HTTP status codes to error types.
func ClassifyHTTPError(statusCode int) ErrorType {
	switch {
	case statusCode >= 500:
		return ErrorTypeTemporary
	case statusCode == 429:
		return ErrorTypeRateLimited
	case statusCode == 409:
		return ErrorTypeConflict
	case statusCode == 404:
		return ErrorTypeNotFound
	case statusCode == 401:
		return ErrorTypeUnauthorized
	case statusCode == 403:
		return ErrorTypeForbidden
	case statusCode >= 400 && statusCode < 500:
		return ErrorTypeInvalid
	default:
		return ErrorTypePermanent
	}
}
