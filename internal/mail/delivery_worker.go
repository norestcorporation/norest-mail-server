package mail

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/norest-mail/server/internal/stalwart"
)

// DeliveryStatusWorker polls Stalwart for delivery status updates on sent messages.
type DeliveryStatusWorker struct {
	pool           *pgxpool.Pool
	stalwartClient *stalwart.Client
	db             DB
	pollInterval   time.Duration
	batchSize      int
	bounceGenerator *BounceGenerator
}

func NewDeliveryStatusWorker(pool *pgxpool.Pool, stalwartClient *stalwart.Client, db DB, mailerEmail, mailerName string) *DeliveryStatusWorker {
	return &DeliveryStatusWorker{
		pool:           pool,
		stalwartClient: stalwartClient,
		db:             db,
		pollInterval:   30 * time.Second, // Poll every 30 seconds
		batchSize:      50,               // Process 50 submissions at a time
		bounceGenerator: NewBounceGenerator(pool, stalwartClient, db, mailerEmail, mailerName),
	}
}

func (w *DeliveryStatusWorker) Run(ctx context.Context) error {
	slog.Info("starting delivery status worker")
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			w.processPendingDeliveries(ctx)
		}
	}
}

func (w *DeliveryStatusWorker) processPendingDeliveries(ctx context.Context) {
	// Get pending delivery status records
	pending, err := w.db.GetPendingDeliveryStatus(ctx, w.batchSize)
	if err != nil {
		slog.Error("failed to get pending delivery status", "error", err)
		return
	}

	if len(pending) == 0 {
		return
	}

	slog.Info("processing pending deliveries", "count", len(pending))

	// Group by user account for efficient JMAP queries
	accountGroups := make(map[string][]map[string]any)
	for _, p := range pending {
		mailboxID := p["mailbox_id"].(string)
		// Get the Stalwart account ID for this mailbox
		var stalwartAccountID string
		err := w.pool.QueryRow(ctx,
			"SELECT stalwart_account_id FROM mailboxes WHERE id = $1",
			mailboxID).Scan(&stalwartAccountID)
		if err != nil {
			slog.Error("failed to get stalwart account id", "error", err, "mailbox_id", mailboxID)
			continue
		}

		accountGroups[stalwartAccountID] = append(accountGroups[stalwartAccountID], p)
	}

	// Process each account's submissions
	for accountID, submissions := range accountGroups {
		w.processAccountSubmissions(ctx, accountID, submissions)
	}
}

func (w *DeliveryStatusWorker) processAccountSubmissions(ctx context.Context, accountID string, submissions []map[string]any) {
	// Collect submission IDs
	submissionIDs := make([]string, len(submissions))
	for i, s := range submissions {
		submissionIDs[i] = s["submission_id"].(string)
	}

	// Query Stalwart for EmailSubmission objects
	submissionsResp, err := w.stalwartClient.EmailSubmissionGet(ctx, accountID, submissionIDs)
	if err != nil {
		slog.Error("failed to get email submissions from stalwart", "error", err, "account_id", accountID)
		return
	}

	// Create a map of submission ID to delivery status
	statusMap := make(map[string]*stalwart.EmailSubmission)
	for _, sub := range submissionsResp.List {
		statusMap[sub.ID] = &sub
	}

	// Update delivery status for each submission
	for _, s := range submissions {
		submissionID := s["submission_id"].(string)
		recipientEmail := s["recipient_email"].(string)
		messageID := s["message_id"].(string)
		userID := s["user_id"].(string)
		currentStatus := s["status"].(string)
		retryCount := s["retry_count"].(int)

		submission, exists := statusMap[submissionID]
		if !exists {
			slog.Warn("submission not found in stalwart response", "submission_id", submissionID)
			continue
		}

		// Extract delivery status from Stalwart response
		newStatus, errorMessage, errorType, smtpReply, isPermanent := w.parseDeliveryStatus(submission, recipientEmail)

		// Only update if status has changed
		if newStatus != currentStatus {
			err := w.db.UpdateDeliveryStatusWithOutbox(ctx, submissionID, recipientEmail, userID, messageID, newStatus, errorMessage, errorType, smtpReply, isPermanent)
			if err != nil {
				slog.Error("failed to update delivery status with outbox", "error", err, "submission_id", submissionID)
			} else {
				slog.Info("updated delivery status and published event",
					"submission_id", submissionID,
					"message_id", messageID,
					"recipient", recipientEmail,
					"old_status", currentStatus,
					"new_status", newStatus)
			}

			// Generate bounce notification for permanent failures
			if newStatus == "failed" && isPermanent {
				w.generateBounceNotification(ctx, submissionID, recipientEmail, userID, messageID, errorMessage, errorType, smtpReply)
			}
		}

		// Handle retry logic for temporary failures
		if newStatus == "temporary_failure" && retryCount < 3 {
			w.scheduleRetry(ctx, submissionID, retryCount)
		}
	}
}

func (w *DeliveryStatusWorker) parseDeliveryStatus(submission *stalwart.EmailSubmission, recipientEmail string) (string, string, string, string, bool) {
	// Check deliveryStatus for this specific recipient
	if deliveryStatus, ok := submission.DeliveryStatus[recipientEmail]; ok {
		switch deliveryStatus.Delivered {
		case "yes":
			return "delivered", "", "", "", false
		case "no":
			// Determine if permanent or temporary based on SMTP reply
			errorMessage := w.translateSMTPReply(deliveryStatus.SMTPReply)
			errorType := w.classifyError(deliveryStatus.SMTPReply)
			isPermanent := w.isPermanentError(deliveryStatus.SMTPReply)

			if isPermanent {
				return "failed", errorMessage, errorType, deliveryStatus.SMTPReply, true
			}
			return "temporary_failure", errorMessage, errorType, deliveryStatus.SMTPReply, false
		case "queued":
			return "queued", "", "", "", false
		default:
			return "sending", "", "", "", false
		}
	}

	// If no specific recipient status, check overall status
	// Default to sending if we can't determine status
	return "sending", "", "", "", false
}

func (w *DeliveryStatusWorker) translateSMTPReply(smtpReply string) string {
	if smtpReply == "" {
		return "Delivery failed"
	}

	// Common SMTP error codes and their user-friendly translations
	switch {
	case strings.Contains(smtpReply, "550"), strings.Contains(smtpReply, "5.1.1"):
		return "The recipient's email address doesn't exist"
	case strings.Contains(smtpReply, "552"), strings.Contains(smtpReply, "5.2.2"):
		return "The recipient's mailbox is full"
	case strings.Contains(smtpReply, "554"), strings.Contains(smtpReply, "5.7.1"):
		return "The message was rejected by the recipient's server"
	case strings.Contains(smtpReply, "450"), strings.Contains(smtpReply, "4.2.1"):
		return "The recipient's mailbox is temporarily unavailable"
	case strings.Contains(smtpReply, "451"), strings.Contains(smtpReply, "4.4.0"):
		return "Temporary delivery failure - will retry"
	default:
		return "Delivery failed"
	}
}

func (w *DeliveryStatusWorker) classifyError(smtpReply string) string {
	if smtpReply == "" {
		return "unknown"
	}

	switch {
	case strings.Contains(smtpReply, "550"), strings.Contains(smtpReply, "5.1.1"):
		return "mailbox_not_found"
	case strings.Contains(smtpReply, "552"), strings.Contains(smtpReply, "5.2.2"):
		return "mailbox_full"
	case strings.Contains(smtpReply, "554"), strings.Contains(smtpReply, "5.7.1"):
		return "message_rejected"
	case strings.Contains(smtpReply, "450"), strings.Contains(smtpReply, "4.2.1"):
		return "mailbox_unavailable"
	case strings.Contains(smtpReply, "451"), strings.Contains(smtpReply, "4.4.0"):
		return "temporary_failure"
	default:
		return "unknown"
	}
}

func (w *DeliveryStatusWorker) isPermanentError(smtpReply string) bool {
	if smtpReply == "" {
		return false
	}

	// 5.x.x errors are permanent, 4.x.x are temporary
	// Check if the first digit of the SMTP code is 5
	for _, char := range smtpReply {
		if char >= '0' && char <= '9' {
			return char == '5'
		}
	}
	return false
}

func (w *DeliveryStatusWorker) scheduleRetry(ctx context.Context, submissionID string, retryCount int) {
	// Exponential backoff: 1min, 5min, 15min
	retryDelays := []time.Duration{1 * time.Minute, 5 * time.Minute, 15 * time.Minute}
	if retryCount >= len(retryDelays) {
		return
	}

	nextRetry := time.Now().Add(retryDelays[retryCount])
	_, err := w.pool.Exec(ctx,
		"UPDATE message_delivery_status SET next_retry_at = $1, retry_count = retry_count + 1 WHERE submission_id = $2",
		nextRetry, submissionID)
	if err != nil {
		slog.Error("failed to schedule retry", "error", err, "submission_id", submissionID)
	}
}

// generateBounceNotification creates a bounce notification for a permanently failed delivery
func (w *DeliveryStatusWorker) generateBounceNotification(ctx context.Context, submissionID, recipientEmail, userID, messageID, errorMessage, errorType, smtpReply string) {
	// Get additional information needed for the bounce
	var subject, originalSender string
	err := w.pool.QueryRow(ctx,
		`SELECT subject, 
		 (SELECT a.local_part || '@' || d.name FROM addresses a JOIN domains d ON a.domain_id = d.id WHERE a.id = (SELECT address_id FROM mailboxes WHERE id = mds.mailbox_id))
		 FROM message_delivery_status mds
		 WHERE mds.submission_id = $1 AND mds.recipient_email = $2`,
		submissionID, recipientEmail).Scan(&subject, &originalSender)
	if err != nil {
		slog.Error("failed to get bounce information", "error", err, "submission_id", submissionID)
		return
	}

	// Create bounce delivery info
	bounceInfo := DeliveryFailureInfo{
		SubmissionID:   submissionID,
		RecipientEmail: recipientEmail,
		UserID:         userID,
		MessageID:      messageID,
		Subject:        subject,
		ErrorMessage:   errorMessage,
		ErrorType:      errorType,
		SMTPReply:      smtpReply,
		OriginalSender: originalSender,
		OriginalTo:     recipientEmail,
	}

	// Generate the bounce
	if err := w.bounceGenerator.GenerateBounce(ctx, bounceInfo); err != nil {
		slog.Error("failed to generate bounce notification", 
			"error", err, 
			"submission_id", submissionID, 
			"recipient", recipientEmail)
	}
}
