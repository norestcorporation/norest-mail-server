package stalwart

import (
	"context"
	"fmt"
)

// EmailSubmission represents a JMAP EmailSubmission object.
type EmailSubmission struct {
	ID             string                    `json:"id,omitempty"`
	IdentityID     string                    `json:"identityId"`
	EmailID        string                    `json:"emailId"`
	ThreadID       string                    `json:"threadId,omitempty"`
	SendAt         *string                   `json:"sendAt,omitempty"`
	UndoStatus     string                    `json:"undoStatus,omitempty"`
	DeliveryStatus map[string]DeliveryStatus `json:"deliveryStatus,omitempty"`
}

// DeliveryStatus represents the delivery status for a recipient.
type DeliveryStatus struct {
	SMTPReply string `json:"smtpReply,omitempty"`
	Delivered string `json:"delivered"` // "queued", "yes", "no", "unknown"
	Displayed string `json:"displayed"` // "yes", "unknown"
}

// EmailSubmissionSetResponse represents the response to an EmailSubmission/set JMAP call.
type EmailSubmissionSetResponse struct {
	AccountID    string                     `json:"accountId"`
	OldState     string                     `json:"oldState"`
	NewState     string                     `json:"newState"`
	Created      map[string]EmailSubmission `json:"created"`
	Updated      map[string]any             `json:"updated"`
	Destroyed    []string                   `json:"destroyed"`
	NotCreated   map[string]EmailSetError   `json:"notCreated"`
	NotUpdated   map[string]EmailSetError   `json:"notUpdated"`
	NotDestroyed map[string]EmailSetError   `json:"notDestroyed"`
}

// EmailSubmissionSet creates an email submission to send a message.
//
// Parameters:
//   - accountID: the user's Stalwart account ID
//   - emailID: the ID of the email to submit (must already exist, e.g., a draft)
//   - identityID: the sending identity ID
//   - onSuccessUpdate: optional Email/set update to apply on successful submission
//     (e.g., move from Drafts to Sent, remove $draft keyword)
func (c *Client) EmailSubmissionSet(
	ctx context.Context,
	accountID string,
	emailID string,
	identityID string,
	onSuccessUpdateEmail map[string]any,
) (*EmailSubmissionSetResponse, error) {
	createKey := "sub0"

	submissionArgs := map[string]any{
		"accountId": accountID,
		"create": map[string]any{
			createKey: map[string]any{
				"identityId": identityID,
				"emailId":    emailID,
			},
		},
	}

	// onSuccessUpdateEmail allows atomic post-send operations:
	// e.g., move the email from Drafts to Sent and remove $draft keyword.
	if onSuccessUpdateEmail != nil {
		submissionArgs["onSuccessUpdateEmail"] = map[string]any{
			"#" + createKey: onSuccessUpdateEmail,
		}
	}

	methodCalls := []any{
		[]any{"EmailSubmission/set", submissionArgs, "esub0"},
	}

	var result EmailSubmissionSetResponse
	if err := c.callJMAPFirst(ctx, jmapMailSubmissionUsing, methodCalls, &result); err != nil {
		return nil, fmt.Errorf("EmailSubmission/set: %w", err)
	}

	// Check for submission-level failures
	if len(result.NotCreated) > 0 {
		for key, setErr := range result.NotCreated {
			return &result, fmt.Errorf("EmailSubmission/set create failed for %s: [%s] %s", key, setErr.Type, setErr.Description)
		}
	}

	return &result, nil
}

// EmailSubmissionGetResponse represents the response to an EmailSubmission/get JMAP call.
type EmailSubmissionGetResponse struct {
	AccountID string            `json:"accountId"`
	State     string            `json:"state"`
	List      []EmailSubmission `json:"list"`
	NotFound  []string          `json:"notFound"`
}

// EmailSubmissionGet retrieves email submissions by their IDs.
func (c *Client) EmailSubmissionGet(ctx context.Context, accountID string, ids []string) (*EmailSubmissionGetResponse, error) {
	methodCalls := []any{
		[]any{"EmailSubmission/get", map[string]any{
			"accountId": accountID,
			"ids":       ids,
		}, "0"},
	}

	var result EmailSubmissionGetResponse
	if err := c.callJMAPFirst(ctx, jmapMailSubmissionUsing, methodCalls, &result); err != nil {
		return nil, fmt.Errorf("EmailSubmission/get: %w", err)
	}

	return &result, nil
}
