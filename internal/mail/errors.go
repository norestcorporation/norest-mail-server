package mail

import (
	"errors"
	"strings"
)

// Sentinel errors for mail operations.
// These are translated to HTTP responses by the handler layer.
var (
	// Resource errors
	ErrMessageNotFound     = errors.New("message not found")
	ErrMailboxNotFound     = errors.New("mailbox not found")
	ErrDraftNotFound       = errors.New("draft not found")
	ErrIdentityNotFound    = errors.New("no sending identity found")

	// Idempotency errors
	ErrIdempotencyMismatch   = errors.New("idempotency mismatch")
	ErrIdempotencyInProgress = errors.New("idempotency in progress")

	// Authorization errors
	ErrUnauthorizedResource = errors.New("unauthorized resource access")
	ErrAccountSuspended     = errors.New("account is suspended")
	ErrAccountNotActive     = errors.New("account is not active")
	ErrMailboxNotReady      = errors.New("mailbox is not ready")

	// Validation errors
	ErrInvalidRecipient    = errors.New("invalid recipient")
	ErrMissingRecipient    = errors.New("at least one recipient is required")
	ErrMissingSubject      = errors.New("subject is required")
	ErrMessageRejected     = errors.New("message was rejected")

	// Quota/policy errors
	ErrQuotaExceeded       = errors.New("quota exceeded")
	ErrAttachmentTooLarge  = errors.New("attachment too large")
	ErrRateLimitExceeded   = errors.New("rate limit exceeded")

	// Infrastructure errors
	ErrStalwartUnavailable = errors.New("mail service unavailable")
	ErrJMAPTimeout         = errors.New("mail service timeout")

	// Delivery status errors
	ErrDeliveryFailed      = errors.New("message delivery failed")
	ErrDeliveryAmbiguous   = errors.New("delivery status unknown — message may have been sent")

	// Idempotency
	ErrDuplicateSend       = errors.New("duplicate send request")
)

// IsNotFound returns true if the error is a resource-not-found error.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrMessageNotFound) ||
		errors.Is(err, ErrMailboxNotFound) ||
		errors.Is(err, ErrDraftNotFound) ||
		errors.Is(err, ErrIdentityNotFound)
}

// IsAuthError returns true if the error is an authorization error.
func IsAuthError(err error) bool {
	return errors.Is(err, ErrUnauthorizedResource) ||
		errors.Is(err, ErrAccountSuspended) ||
		errors.Is(err, ErrAccountNotActive) ||
		errors.Is(err, ErrMailboxNotReady)
}

// IsValidationError returns true if the error is a request validation error.
func IsValidationError(err error) bool {
	return errors.Is(err, ErrInvalidRecipient) ||
		errors.Is(err, ErrMissingRecipient) ||
		errors.Is(err, ErrMissingSubject) ||
		errors.Is(err, ErrMessageRejected)
}

// IsStalwartError returns true if the error indicates a Stalwart communication issue.
// This is used to determine whether the error might be transient.
func IsStalwartError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrStalwartUnavailable) ||
		errors.Is(err, ErrJMAPTimeout) ||
		strings.Contains(err.Error(), "JMAP") ||
		strings.Contains(err.Error(), "stalwart")
}
