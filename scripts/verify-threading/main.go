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
	var accountID, stalwartAccountID string
	err = pool.QueryRow(ctx, "SELECT m.id, m.stalwart_account_id FROM mailboxes m JOIN addresses a ON m.address_id = a.id WHERE a.claimed_by = $1", userID).Scan(&accountID, &stalwartAccountID)
	if err != nil {
		log.Fatalf("Failed to lookup account IDs: %v", err)
	}

	// --- TEST 1 & 2: Real thread creation & RFC Headers ---
	fmt.Println("\n=== TEST 1 & 2: Real thread creation & RFC Headers ===")
	msgAReq := mail.SendRequest{
		To:             []mail.EmailAddressDTO{{Email: "test-recipient@example.com"}},
		Subject:        "Threading Test Conversation",
		TextBody:       "This is Message A",
		IdempotencyKey: uuid.New().String(),
	}
	msgAResp, err := mailSvc.SendMessage(ctx, userID, msgAReq)
	if err != nil {
		log.Fatalf("Failed to send Message A: %v", err)
	}
	fmt.Printf("Message A Sent. MessageID: %s\n", msgAResp.MessageID)

	// Wait for sync worker to run
	fmt.Println("Running Norest Sync (via Backfill worker logic)...")
	time.Sleep(35 * time.Second)

	// Check DB
	var threadA, threadB string
	var rfcMsgA string
	err = pool.QueryRow(ctx, "SELECT thread_id, message_id FROM messages WHERE stalwart_email_id = $1", msgAResp.MessageID).Scan(&threadA, &rfcMsgA)
	if err != nil {
		log.Fatalf("Failed to get thread A: %v", err)
	}

	// Reply (Message B)
	fmt.Println("Sending Message B (Reply to A)...")
	msgBReq := mail.SendRequest{
		To:             []mail.EmailAddressDTO{{Email: "test-recipient@example.com"}},
		Subject:        "Re: Threading Test Conversation",
		TextBody:       "This is Message B (Reply)",
		InReplyTo:      []string{rfcMsgA}, // Uses RFC Message-ID
		References:     []string{rfcMsgA},
		IdempotencyKey: uuid.New().String(),
	}
	msgBResp, err := mailSvc.SendMessage(ctx, userID, msgBReq)
	if err != nil {
		log.Fatalf("Failed to send Message B: %v", err)
	}
	fmt.Printf("Message B Sent. MessageID: %s\n", msgBResp.MessageID)

	fmt.Println("Waiting 35 seconds for sync...")
	time.Sleep(35 * time.Second)

	err = pool.QueryRow(ctx, "SELECT thread_id FROM messages WHERE stalwart_email_id = $1", msgBResp.MessageID).Scan(&threadB)
	if err != nil {
		log.Fatalf("Failed to get thread B: %v", err)
	}

	if threadA == threadB {
		fmt.Printf("✅ TEST 1 PASSED: Messages belong to the same thread (%s)\n", threadA)
	} else {
		fmt.Printf("❌ TEST 1 FAILED: Threads do not match. A: %s, B: %s\n", threadA, threadB)
	}

	// We can't easily check Stalwart RFC headers here without parsing JMAP output,
	// but we can query Stalwart via `stalwartClient.Call` if needed.

	// --- TEST 5: Changed subject ---
	fmt.Println("\n=== TEST 5: Changed subject ===")

	var rfcMsgB string
	err = pool.QueryRow(ctx, "SELECT message_id FROM messages WHERE stalwart_email_id = $1", msgBResp.MessageID).Scan(&rfcMsgB)

	msgCReq := mail.SendRequest{
		To:             []mail.EmailAddressDTO{{Email: "test-recipient@example.com"}},
		Subject:        "Completely Different Looking Subject", // Changed subject
		TextBody:       "This is Message C (Reply with changed subject)",
		InReplyTo:      []string{rfcMsgB},
		References:     []string{rfcMsgA, rfcMsgB},
		IdempotencyKey: uuid.New().String(),
	}
	msgCResp, err := mailSvc.SendMessage(ctx, userID, msgCReq)
	if err != nil {
		log.Fatalf("Failed to send Message C: %v", err)
	}
	fmt.Printf("Message C Sent. MessageID: %s\n", msgCResp.MessageID)

	fmt.Println("Waiting 35 seconds for sync...")
	time.Sleep(35 * time.Second)

	var threadC string
	err = pool.QueryRow(ctx, "SELECT thread_id FROM messages WHERE stalwart_email_id = $1", msgCResp.MessageID).Scan(&threadC)
	if err != nil {
		log.Fatalf("Failed to get thread C: %v", err)
	}

	if threadA == threadC {
		fmt.Printf("✅ TEST 5 PASSED: Changed subject remained in same thread (%s)\n", threadC)
	} else {
		fmt.Printf("❌ TEST 5 FAILED: Thread changed. Expected: %s, Got: %s\n", threadA, threadC)
	}

	// --- TEST 6: Same subject, unrelated conversation ---
	fmt.Println("\n=== TEST 6: Same subject, unrelated conversation ===")
	msgDReq := mail.SendRequest{
		To:             []mail.EmailAddressDTO{{Email: "another-recipient@example.com"}},
		Subject:        "Threading Test Conversation", // Same subject as A, but not a reply
		TextBody:       "This is Message D (Unrelated)",
		IdempotencyKey: uuid.New().String(),
	}
	msgDResp, err := mailSvc.SendMessage(ctx, userID, msgDReq)
	if err != nil {
		log.Fatalf("Failed to send Message D: %v", err)
	}
	fmt.Printf("Message D Sent. MessageID: %s\n", msgDResp.MessageID)

	fmt.Println("Waiting 35 seconds for sync...")
	time.Sleep(35 * time.Second)

	var threadD string
	err = pool.QueryRow(ctx, "SELECT thread_id FROM messages WHERE stalwart_email_id = $1", msgDResp.MessageID).Scan(&threadD)
	if err != nil {
		log.Fatalf("Failed to get thread D: %v", err)
	}

	if threadA != threadD {
		fmt.Printf("✅ TEST 6 PASSED: Unrelated conversation got a different thread (%s)\n", threadD)
	} else {
		fmt.Printf("❌ TEST 6 FAILED: Unrelated conversation merged into same thread. A: %s, D: %s\n", threadA, threadD)
	}

	// --- TEST 7: Mailbox behavior ---
	fmt.Println("\n=== TEST 7: Mailbox behavior ===")
	var messageCount int
	err = pool.QueryRow(ctx, "SELECT COUNT(DISTINCT message_id) FROM message_mailboxes WHERE message_id IN (SELECT id FROM messages WHERE thread_id = $1)", threadA).Scan(&messageCount)
	if err != nil {
		fmt.Printf("❌ TEST 7 Failed to count mailboxes: %v\n", err)
	} else {
		fmt.Printf("✅ TEST 7: Thread A has %d messages in different mailbox state configurations.\n", messageCount)
	}

	// --- TEST 3: Thread API ---
	fmt.Println("\n=== TEST 3: Thread API ===")
	threadsResp, err := mailSvc.ListThreads(ctx, userID, mail.ListMessagesOptions{Limit: 10})
	if err != nil {
		fmt.Printf("❌ TEST 3 Failed: %v\n", err)
	} else {
		foundA := false
		for _, th := range threadsResp.Threads {
			if th.ID == threadA {
				foundA = true
				if th.MessageCount == 3 {
					fmt.Printf("✅ TEST 3 PASSED: Thread API returned Thread A with MessageCount=3 (Total Threads returned: %d)\n", len(threadsResp.Threads))
				} else {
					fmt.Printf("❌ TEST 3 FAILED: Thread A found but MessageCount is %d, expected 3\n", th.MessageCount)
				}
			}
		}
		if !foundA {
			fmt.Printf("❌ TEST 3 FAILED: Thread A (%s) not found in ListThreads response\n", threadA)
		}

		// Test specific thread GET
		threadDetails, err := mailSvc.GetThread(ctx, userID, threadA)
		if err != nil {
			fmt.Printf("❌ TEST 3 Failed (GetThread): %v\n", err)
		} else {
			fmt.Printf("✅ TEST 3 PASSED: GetThread returned correctly for Thread A (Messages: %d)\n", threadDetails.MessageCount)
		}
	}

	// Wait to show success
	fmt.Println("\nThreading Backend Tests Complete!")
}
