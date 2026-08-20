package mail

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/norest-mail/server/internal/stalwart"
)

// BounceGenerator handles the creation and delivery of bounce notification emails
type BounceGenerator struct {
	pool           *pgxpool.Pool
	stalwartClient *stalwart.Client
	db             DB
	mailerEmail    string
	mailerName     string
}

// NewBounceGenerator creates a new bounce generator instance
func NewBounceGenerator(pool *pgxpool.Pool, stalwartClient *stalwart.Client, db DB, mailerEmail, mailerName string) *BounceGenerator {
	return &BounceGenerator{
		pool:           pool,
		stalwartClient: stalwartClient,
		db:             db,
		mailerEmail:    mailerEmail,
		mailerName:     mailerName,
	}
}

// GenerateBounce creates and sends a bounce notification email for a permanently failed delivery
func (bg *BounceGenerator) GenerateBounce(ctx context.Context, deliveryInfo DeliveryFailureInfo) error {
	// Check idempotency - has bounce already been generated?
	claimed, err := bg.db.TryClaimBounceGeneration(ctx, deliveryInfo.SubmissionID, deliveryInfo.RecipientEmail)
	if err != nil {
		return fmt.Errorf("failed to claim bounce generation: %w", err)
	}
	if !claimed {
		slog.Info("bounce already generated, skipping",
			"submission_id", deliveryInfo.SubmissionID,
			"recipient", deliveryInfo.RecipientEmail)
		return nil
	}

	// Get the original sender's mailbox information
	senderMailboxID, senderStalwartAccountID, err := bg.getSenderMailboxInfo(ctx, deliveryInfo.UserID)
	if err != nil {
		return fmt.Errorf("failed to get sender mailbox info: %w", err)
	}

	// Get the sender's inbox mailbox ID
	inboxMailboxID, err := bg.db.GetMailboxMappingByRole(ctx, senderMailboxID, "inbox")
	if err != nil {
		return fmt.Errorf("failed to get inbox mailbox ID: %w", err)
	}

	// Try to get the original message's RFC Message-ID for threading
	var rfcMessageID string
	if msgID, err := bg.db.GetRFCMessageID(ctx, deliveryInfo.MessageID); err == nil && msgID != "" {
		rfcMessageID = msgID
	}

	// Generate bounce email content
	bounceEmail := bg.createBounceEmail(deliveryInfo, rfcMessageID)

	// Create the bounce email directly in the sender's inbox
	emailID, err := bg.createBounceInStalwart(ctx, senderStalwartAccountID, inboxMailboxID, bounceEmail)
	if err != nil {
		return fmt.Errorf("failed to create bounce email in Stalwart: %w", err)
	}

	// Save bounce email ID in database
	if err := bg.db.SaveBounceEmailID(ctx, deliveryInfo.SubmissionID, deliveryInfo.RecipientEmail, emailID); err != nil {
		slog.Error("failed to save bounce email ID", "error", err, "email_id", emailID)
		// Continue anyway - the bounce was created in Stalwart
	}

	slog.Info("bounce notification generated successfully",
		"submission_id", deliveryInfo.SubmissionID,
		"recipient", deliveryInfo.RecipientEmail,
		"bounce_email_id", emailID,
		"sender_user_id", deliveryInfo.UserID)

	return nil
}

// DeliveryFailureInfo contains information about a failed delivery
type DeliveryFailureInfo struct {
	SubmissionID   string
	RecipientEmail string
	UserID         string
	MessageID      string
	Subject        string
	ErrorMessage   string
	ErrorType      string
	SMTPReply      string
	OriginalSender string
	OriginalTo     string
}

// getSenderMailboxInfo retrieves the sender's mailbox and Stalwart account information
func (bg *BounceGenerator) getSenderMailboxInfo(ctx context.Context, userID string) (string, string, error) {
	var mailboxID, stalwartAccountID string
	err := bg.pool.QueryRow(ctx,
		`SELECT m.id, m.stalwart_account_id 
		 FROM mailboxes m
		 JOIN addresses a ON m.address_id = a.id
		 WHERE a.claimed_by = $1`,
		userID).Scan(&mailboxID, &stalwartAccountID)
	if err != nil {
		return "", "", err
	}
	return mailboxID, stalwartAccountID, nil
}

// createBounceEmail generates the bounce email content
func (bg *BounceGenerator) createBounceEmail(info DeliveryFailureInfo, rfcMessageID string) map[string]any {
	// Generate human-readable subject based on error type
	subject := "Delivery Status Notification (Failure)"
	switch info.ErrorType {
	case "mailbox_not_found":
		subject = "Address not found"
	case "mailbox_full":
		subject = "Mailbox full"
	case "message_rejected":
		subject = "Message rejected"
	}

	// Create HTML body
	htmlBody := bg.createHTMLBody(info)

	// Build email object
	now := time.Now().Format(time.RFC3339)
	email := map[string]any{
		"from": []map[string]string{
			{
				"name":  bg.mailerName,
				"email": bg.mailerEmail,
			},
		},
		"to": []map[string]string{
			{
				"email": info.OriginalSender,
			},
		},
		"subject": subject,
		"sentAt":  now,
		"bodyValues": map[string]map[string]string{
			"html": {"value": htmlBody},
		},
		"htmlBody": []map[string]string{
			{"partId": "html", "type": "text/html"},
		},
		"keywords": map[string]bool{
			"$seen":     false,
			"$flagged":  false,
			"$answered": false,
		},
	}

	// Add threading headers if we have the original Message-ID
	if rfcMessageID != "" {
		email["inReplyTo"] = []string{rfcMessageID}
		email["references"] = []string{rfcMessageID}
	}

	return email
}

// createTextBody generates the plain text version of the bounce message
func (bg *BounceGenerator) createTextBody(info DeliveryFailureInfo) string {
	var builder strings.Builder

	// Header
	builder.WriteString("Delivery Status Notification (Failure)\n\n")

	// Error description
	builder.WriteString(fmt.Sprintf("Your message wasn't delivered to %s ", info.RecipientEmail))
	switch info.ErrorType {
	case "mailbox_not_found":
		builder.WriteString("because the recipient's email address doesn't exist.\n")
	case "mailbox_full":
		builder.WriteString("because the recipient's mailbox is full.\n")
	case "message_rejected":
		builder.WriteString("because the message was rejected by the recipient's server.\n")
	case "mailbox_unavailable":
		builder.WriteString("because the recipient's mailbox is temporarily unavailable.\n")
	default:
		builder.WriteString("due to a delivery error.\n")
	}

	builder.WriteString("\nThe response was:\n")
	if info.SMTPReply != "" {
		builder.WriteString(info.SMTPReply)
	} else {
		builder.WriteString(info.ErrorMessage)
	}
	builder.WriteString("\n\n")

	// Separator
	builder.WriteString("------------------------------\n\n")

	// Original message details
	builder.WriteString("Original message\n\n")
	builder.WriteString(fmt.Sprintf("From: %s\n", info.OriginalSender))
	builder.WriteString(fmt.Sprintf("To: %s\n", info.OriginalTo))
	if info.Subject != "" {
		builder.WriteString(fmt.Sprintf("Subject: %s\n", info.Subject))
	}

	return builder.String()
}

// createHTMLBody generates the HTML version of the bounce message
func (bg *BounceGenerator) createHTMLBody(info DeliveryFailureInfo) string {
	var builder strings.Builder

	builder.WriteString("<!DOCTYPE html>\n")
	builder.WriteString("<html>\n<head>\n")
	builder.WriteString("<meta charset=\"utf-8\">\n")
	builder.WriteString("<style>\n")
	builder.WriteString("body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; line-height: 1.6; color: #1a1a1a; margin: 0; padding: 20px; background-color: transparent; }\n")
	builder.WriteString(".container { max-width: 600px; margin: 0 auto; }\n")
	builder.WriteString(".header { margin-bottom: 24px; }\n")
	builder.WriteString(".header h2 { margin: 0; font-size: 20px; font-weight: 600; color: #dc2626; }\n")
	builder.WriteString(".content { margin-bottom: 32px; font-size: 15px; color: #374151; }\n")
	builder.WriteString(".details { background-color: #f9fafb; border-radius: 8px; padding: 16px; margin-top: 24px; font-size: 14px; color: #4b5563; border: 1px solid #e5e7eb; }\n")
	builder.WriteString(".details code { display: block; margin-top: 12px; padding: 12px; background-color: #f3f4f6; border-radius: 6px; font-family: ui-monospace, monospace; font-size: 13px; color: #1f2937; word-break: break-all; border: 1px solid #e5e7eb; }\n")
	builder.WriteString(".footer { margin-top: 32px; padding-top: 24px; border-top: 1px solid #e5e7eb; font-size: 13px; color: #6b7280; }\n")
	builder.WriteString("@media (prefers-color-scheme: dark) {\n")
	builder.WriteString("  body { color: #e5e7eb; }\n")
	builder.WriteString("  .header h2 { color: #f87171; }\n")
	builder.WriteString("  .content { color: #d1d5db; }\n")
	builder.WriteString("  .details { background-color: transparent; color: #9ca3af; border-color: #374151; }\n")
	builder.WriteString("  .details code { background-color: transparent; color: #e5e7eb; border-color: #374151; }\n")
	builder.WriteString("  .footer { border-top-color: #374151; color: #9ca3af; }\n")
	builder.WriteString("}\n")
	builder.WriteString("</style>\n")
	builder.WriteString("</head>\n<body>\n")
	builder.WriteString("<div class=\"container\">\n")

	// Header
	builder.WriteString("<div class=\"header\">\n")
	builder.WriteString("<h2>Delivery Status Notification (Failure)</h2>\n")
	builder.WriteString("</div>\n")

	// Content
	builder.WriteString("<div class=\"content\">\n")
	builder.WriteString("<p>We couldn't deliver your message to <strong>")
	builder.WriteString(info.RecipientEmail)
	builder.WriteString("</strong>.</p>\n")

	builder.WriteString("<p>")
	switch info.ErrorType {
	case "mailbox_not_found":
		builder.WriteString("The email address you entered couldn't be found. Please check the recipient's email address for typos or unnecessary spaces and try again.")
	case "mailbox_full":
		builder.WriteString("The recipient's mailbox is full and can't accept any more messages at this time.")
	case "message_rejected":
		builder.WriteString("Your message was rejected by the recipient's email server.")
	case "mailbox_unavailable":
		builder.WriteString("The recipient's mailbox is temporarily unavailable. You may want to try sending your message again later.")
	default:
		builder.WriteString("An unexpected delivery error occurred.")
	}
	builder.WriteString("</p>\n")

	// Details block (SMTP Reply)
	builder.WriteString("<div class=\"details\">\n")
	builder.WriteString("<strong>Diagnostic information for administrators:</strong><br/>\n")
	fmt.Fprintf(&builder, "Attempted delivery to: %s<br/>\n", info.RecipientEmail)

	builder.WriteString("<code>")
	if info.SMTPReply != "" {
		builder.WriteString(info.SMTPReply)
	} else {
		builder.WriteString(info.ErrorMessage)
	}
	builder.WriteString("</code>\n")
	builder.WriteString("</div>\n")

	// Footer (Original message details)
	builder.WriteString("<div class=\"footer\">\n")
	builder.WriteString("<strong>Original Message Details</strong><br/>\n")
	fmt.Fprintf(&builder, "Date: %s<br/>\n", time.Now().Format("Mon, 02 Jan 2006 15:04:05 -0700"))
	fmt.Fprintf(&builder, "From: %s<br/>\n", info.OriginalSender)
	fmt.Fprintf(&builder, "To: %s<br/>\n", info.OriginalTo)
	if info.Subject != "" {
		fmt.Fprintf(&builder, "Subject: %s<br/>\n", info.Subject)
	}
	builder.WriteString("</div>\n")

	builder.WriteString("</div>\n") // close content
	builder.WriteString("</div>\n") // close container
	builder.WriteString("</body>\n</html>")

	return builder.String()
}

// createBounceInStalwart creates the bounce email in the user's Stalwart account
func (bg *BounceGenerator) createBounceInStalwart(ctx context.Context, accountID, inboxMailboxID string, email map[string]any) (string, error) {
	createObj := map[string]any{
		"bounce0": email,
	}

	// Set the mailbox to inbox
	createObj["bounce0"].(map[string]any)["mailboxIds"] = map[string]bool{
		inboxMailboxID: true,
	}

	setResp, err := bg.stalwartClient.EmailSet(ctx, accountID, createObj, nil, nil)
	if err != nil {
		return "", err
	}

	// Extract the created email ID
	if created, ok := setResp.Created["bounce0"]; ok {
		// created is of type Email, which has an ID field
		return created.ID, nil
	}

	return "", fmt.Errorf("failed to extract created email ID from response")
}
