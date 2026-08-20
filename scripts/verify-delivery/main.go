package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/norest-mail/server/internal/db"
	"github.com/norest-mail/server/internal/mail"
	"github.com/norest-mail/server/internal/stalwart"
)

func main() {
	ctx := context.Background()

	// 1. Connect to DB
	connString := "postgres://norest:norest@localhost:5433/norest?sslmode=disable"
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		log.Fatalf("Failed to connect to db: %v", err)
	}
	defer pool.Close()

	dbImpl := db.NewMailRepository(pool)

	// 2. Setup Services
	stalwartClient := stalwart.NewClient("http://localhost:8081", "admin", "change-me-development-only")
	mailSvc := mail.NewService(dbImpl, stalwartClient, pool)

	userID := "85c11c80-7e7a-4fd5-9b25-f30321a91070"

	// 3. Send Message
	fmt.Println("--- 1. Sending Message via mail.Service ---")
	req := mail.SendRequest{
		To: []mail.EmailAddressDTO{
			{Email: "external-success@gmail.com"},
		},
		CC: []mail.EmailAddressDTO{
			{Email: "external-cc-success@outlook.com"},
		},
		BCC: []mail.EmailAddressDTO{
			{Email: "fail-domain-dns-error@domain-that-does-not-exist.local"},
		},
		Subject:  "E2E Delivery Status Test",
		TextBody: "Testing delivery status.",
	}
	req.IdempotencyKey = uuid.New().String()

	resp, err := mailSvc.SendMessage(ctx, userID, req)
	if err != nil {
		log.Fatalf("Failed to send message: %v", err)
	}

	msgID := resp.MessageID
	subID := resp.SubmissionID
	fmt.Printf("Message Sent! Message ID: %s, Submission ID: %s\n", msgID, subID)

	// 4. Check initial DB statuses (should be submitted)
	fmt.Println("\n--- 2. Initial Delivery Statuses (Expected: submitted) ---")
	printDeliveryStatuses(ctx, pool, msgID)

	// 5. Wait for worker to run (runs every 30s)
	fmt.Println("\n--- 3. Waiting 35 seconds for DeliveryStatusWorker to poll Stalwart ---")
	for i := 0; i < 35; i++ {
		time.Sleep(1 * time.Second)
		if i%5 == 0 {
			fmt.Printf(".")
		}
	}
	fmt.Println(" Done!")

	// 6. Check updated statuses
	fmt.Println("\n--- 4. Updated Delivery Statuses ---")
	printDeliveryStatuses(ctx, pool, msgID)

	// 7. Get full message via service (API Response)
	fmt.Println("\n--- 5. Final GetMessage API Response ---")
	msgResp, err := mailSvc.GetMessage(ctx, userID, msgID)
	if err != nil {
		log.Fatalf("Failed to get message: %v", err)
	}
	for _, status := range msgResp.DeliveryStatuses {
		fmt.Printf("  Recipient: %s, Status: %s, FailedAt: %v, Error: %s, IsPermanent: %v\n",
			status.RecipientEmail, status.Status, status.FailedAt, status.ErrorMessage, status.IsPermanent)
	}
}

func printDeliveryStatuses(ctx context.Context, pool *pgxpool.Pool, messageID string) {
	rows, err := pool.Query(ctx, "SELECT recipient_email, status, error_type, error_message, is_permanent, retry_count FROM message_delivery_status WHERE message_id = $1", messageID)
	if err != nil {
		log.Fatalf("Failed to query statuses: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var email, status string
		var errType, errMsg *string
		var isPerm bool
		var retryCount int

		if err := rows.Scan(&email, &status, &errType, &errMsg, &isPerm, &retryCount); err != nil {
			log.Fatalf("Failed to scan status: %v", err)
		}

		fmt.Printf("  -> Recipient: %-45s | Status: %-15s | Perm: %-5v | Attempts: %d | Error: %v\n",
			email, status, isPerm, retryCount, ptrString(errMsg))
	}
}

func ptrString(s *string) string {
	if s == nil {
		return "<none>"
	}
	return *s
}
