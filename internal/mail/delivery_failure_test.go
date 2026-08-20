package mail

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBounceEmailStructure tests that bounce emails have proper structure
func TestBounceEmailStructure(t *testing.T) {
	bounceGenerator := &BounceGenerator{
		mailerEmail: "mailer-daemon@norest.in",
		mailerName:  "Mail Delivery Subsystem",
	}

	info := DeliveryFailureInfo{
		SubmissionID:   "test-submission",
		RecipientEmail: "nonexistent@example.com",
		UserID:         "test-user",
		MessageID:      "test-message",
		Subject:        "Test Subject",
		ErrorMessage:   "The recipient's email address doesn't exist",
		ErrorType:      "mailbox_not_found",
		SMTPReply:      "550 5.1.2 Mailbox does not exist.",
		OriginalSender: "sender@example.com",
		OriginalTo:     "nonexistent@example.com",
	}

	email := bounceGenerator.createBounceEmail(info, "")

	// Verify basic email structure
	assert.NotNil(t, email)
	
	// Verify from field
	from, ok := email["from"].([]map[string]string)
	require.True(t, ok)
	require.Len(t, from, 1)
	assert.Equal(t, "Mail Delivery Subsystem", from[0]["name"])
	assert.Equal(t, "mailer-daemon@norest.in", from[0]["email"])
	
	// Verify to field
	to, ok := email["to"].([]map[string]string)
	require.True(t, ok)
	require.Len(t, to, 1)
	assert.Equal(t, "sender@example.com", to[0]["email"])
	
	// Verify subject
	subject, ok := email["subject"].(string)
	require.True(t, ok)
	assert.Equal(t, "Address not found", subject) // Should be specific for mailbox_not_found
	
	// Verify HTML body structure
	htmlBody, ok := email["htmlBody"].([]map[string]string)
	require.True(t, ok)
	require.Len(t, htmlBody, 1)
	assert.Equal(t, "html", htmlBody[0]["partId"])
	assert.Equal(t, "text/html", htmlBody[0]["type"])
	
	// Verify body values
	bodyValues, ok := email["bodyValues"].(map[string]map[string]string)
	require.True(t, ok)
	assert.Contains(t, bodyValues, "html")
	assert.Contains(t, bodyValues["html"]["value"], "Delivery Status Notification")
	assert.Contains(t, bodyValues["html"]["value"], "nonexistent@example.com")
	assert.Contains(t, bodyValues["html"]["value"], "550 5.1.2 Mailbox does not exist.")
	
	// Verify keywords
	keywords, ok := email["keywords"].(map[string]bool)
	require.True(t, ok)
	assert.False(t, keywords["$seen"])
	assert.False(t, keywords["$flagged"])
	assert.False(t, keywords["$answered"])
}

// TestBounceSubjectGeneration tests different error types generate appropriate subjects
func TestBounceSubjectGeneration(t *testing.T) {
	bounceGenerator := &BounceGenerator{
		mailerEmail: "mailer-daemon@norest.in",
		mailerName:  "Mail Delivery Subsystem",
	}

	testCases := []struct {
		name       string
		errorType  string
		wantSubject string
	}{
		{
			name:       "mailbox_not_found",
			errorType:  "mailbox_not_found",
			wantSubject: "Address not found",
		},
		{
			name:       "mailbox_full",
			errorType:  "mailbox_full",
			wantSubject: "Mailbox full",
		},
		{
			name:       "message_rejected",
			errorType:  "message_rejected",
			wantSubject: "Message rejected",
		},
		{
			name:       "unknown_error",
			errorType:  "unknown",
			wantSubject: "Delivery Status Notification (Failure)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			info := DeliveryFailureInfo{
				ErrorType: tc.errorType,
			}
			email := bounceGenerator.createBounceEmail(info, "")
			subject := email["subject"].(string)
			assert.Equal(t, tc.wantSubject, subject)
		})
	}
}

// TestSMTPErrorClassification tests SMTP error classification logic
func TestSMTPErrorClassification(t *testing.T) {
	worker := &DeliveryStatusWorker{}

	testCases := []struct {
		name           string
		smtpReply      string
		wantStatus     string
		wantErrorType  string
		wantPermanent  bool
	}{
		{
			name:          "permanent_mailbox_not_found",
			smtpReply:     "550 5.1.1 Mailbox does not exist",
			wantStatus:    "failed",
			wantErrorType: "mailbox_not_found",
			wantPermanent: true,
		},
		{
			name:          "permanent_mailbox_full",
			smtpReply:     "552 5.2.2 Mailbox full",
			wantStatus:    "failed",
			wantErrorType: "mailbox_full",
			wantPermanent: true,
		},
		{
			name:          "permanent_message_rejected",
			smtpReply:     "554 5.7.1 Message rejected",
			wantStatus:    "failed",
			wantErrorType: "message_rejected",
			wantPermanent: true,
		},
		{
			name:          "temporary_mailbox_unavailable",
			smtpReply:     "450 4.2.1 Mailbox temporarily unavailable",
			wantStatus:    "temporary_failure",
			wantErrorType: "mailbox_unavailable",
			wantPermanent: false,
		},
		{
			name:          "temporary_generic",
			smtpReply:     "451 4.4.0 Temporary failure",
			wantStatus:    "temporary_failure",
			wantErrorType: "temporary_failure",
			wantPermanent: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// For this test, we'll test the individual classification methods directly
			assert.Equal(t, tc.wantErrorType, worker.classifyError(tc.smtpReply))
			assert.Equal(t, tc.wantPermanent, worker.isPermanentError(tc.smtpReply))
		})
	}
}
