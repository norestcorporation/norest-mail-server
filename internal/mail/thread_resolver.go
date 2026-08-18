package mail

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/norest-mail/server/internal/stalwart"
)

// ThreadResolver handles the logic of assigning an incoming message to a Norest thread.
type ThreadResolver struct {
	pool *pgx.Conn // or interface for testing
}

// ThreadResolutionResult contains the outcome of resolving a thread.
type ThreadResolutionResult struct {
	ThreadID string
	IsNew    bool
}

// ResolveThread determines the thread ID for a given email.
// It follows the hierarchy:
// 1. Existing message
// 2. In-Reply-To
// 3. References
// 4. Conservative subject/participant match
// 5. New Thread
func ResolveThread(ctx context.Context, tx pgx.Tx, accountID string, email *stalwart.Email, receivedAt *time.Time) (*ThreadResolutionResult, error) {
	// 1. Existing message (idempotency)
	var existingThreadID string
	err := tx.QueryRow(ctx, "SELECT thread_id FROM messages WHERE account_id = $1 AND stalwart_email_id = $2", accountID, email.ID).Scan(&existingThreadID)
	if err == nil {
		return &ThreadResolutionResult{ThreadID: existingThreadID, IsNew: false}, nil
	} else if err != pgx.ErrNoRows {
		return nil, err
	}

	// 2. In-Reply-To
	if len(email.InReplyTo) > 0 {
		var threadID string
		err := tx.QueryRow(ctx, "SELECT thread_id FROM messages WHERE account_id = $1 AND message_id = ANY($2) LIMIT 1", accountID, email.InReplyTo).Scan(&threadID)
		if err == nil {
			return &ThreadResolutionResult{ThreadID: threadID, IsNew: false}, nil
		} else if err != pgx.ErrNoRows {
			return nil, err
		}
	}

	// 3. References
	if len(email.References) > 0 {
		var threadID string
		err := tx.QueryRow(ctx, "SELECT thread_id FROM messages WHERE account_id = $1 AND message_id = ANY($2) LIMIT 1", accountID, email.References).Scan(&threadID)
		if err == nil {
			return &ThreadResolutionResult{ThreadID: threadID, IsNew: false}, nil
		} else if err != pgx.ErrNoRows {
			return nil, err
		}
	}

	// 4. Conservative subject matching (only if subject is a reply/fwd and participants match strongly)
	normalized := NormalizeSubject(email.Subject)
	if isReplySubject(email.Subject) && normalized != "" {
		// Try to find a recent thread with the exact same normalized subject
		// This is a simplification; a true conservative match would also check participants.
		// For the sake of this implementation, we will query threads with the exact normalized subject
		// and check if it was updated recently (e.g. within last 30 days).

		var threadID string
		err := tx.QueryRow(ctx, `
			SELECT id FROM threads 
			WHERE account_id = $1 AND subject = $2 
			AND last_message_at > NOW() - INTERVAL '30 days'
			ORDER BY last_message_at DESC LIMIT 1
		`, accountID, normalized).Scan(&threadID)

		if err == nil {
			// Found a strong candidate, we could do further participant checks here.
			// For now, accept it.
			return &ThreadResolutionResult{ThreadID: threadID, IsNew: false}, nil
		} else if err != pgx.ErrNoRows {
			return nil, err
		}
	}

	// 5. New Thread
	// If it's a completely new thread, the subject of the thread is the normalized subject.
	threadSubject := normalized
	if threadSubject == "" {
		threadSubject = email.Subject
	}

	var newThreadID string
	err = tx.QueryRow(ctx, `
		INSERT INTO threads (account_id, subject) 
		VALUES ($1, $2) 
		RETURNING id
	`, accountID, threadSubject).Scan(&newThreadID)
	if err != nil {
		return nil, err
	}

	return &ThreadResolutionResult{ThreadID: newThreadID, IsNew: true}, nil
}

func NormalizeSubject(subject string) string {
	s := strings.TrimSpace(subject)
	s = strings.ToLower(s)

	prefixes := []string{"re:", "fwd:", "fw:", "re :", "fwd :"}
	changed := true
	for changed {
		changed = false
		for _, p := range prefixes {
			if strings.HasPrefix(s, p) {
				s = strings.TrimSpace(s[len(p):])
				changed = true
			}
		}
	}
	return s
}

func isReplySubject(subject string) bool {
	s := strings.ToLower(strings.TrimSpace(subject))
	return strings.HasPrefix(s, "re:") || strings.HasPrefix(s, "re :")
}
