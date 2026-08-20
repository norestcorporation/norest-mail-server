package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/norest-mail/server/internal/mail"
)

// ToggleReaction adds or removes an emoji reaction for a user on a message.
// Returns true if added, false if removed.
func ToggleReaction(ctx context.Context, pool *pgxpool.Pool, messageID, userEmail, emoji string) (bool, error) {
	ctx, cancel := WithDefaultTimeout(ctx)
	defer cancel()

	// First try to insert, if conflict on unique constraint, then delete
	res, err := pool.Exec(ctx, `
		INSERT INTO email_reactions (message_id, user_email, emoji)
		VALUES ($1, $2, $3)
		ON CONFLICT (message_id, user_email, emoji) DO NOTHING
	`, messageID, userEmail, emoji)
	if err != nil {
		return false, fmt.Errorf("toggling reaction: %w", err)
	}

	if res.RowsAffected() > 0 {
		return true, nil // Added
	}

	// It already existed, so remove it
	_, err = pool.Exec(ctx, `
		DELETE FROM email_reactions
		WHERE message_id = $1 AND user_email = $2 AND emoji = $3
	`, messageID, userEmail, emoji)
	if err != nil {
		return false, fmt.Errorf("removing reaction: %w", err)
	}

	return false, nil // Removed
}

// GetReactionsForMessage returns all reactions for a specific message.
func GetReactionsForMessage(ctx context.Context, pool *pgxpool.Pool, messageID string) ([]mail.EmailReaction, error) {
	ctx, cancel := WithDefaultTimeout(ctx)
	defer cancel()

	rows, err := pool.Query(ctx, `
		SELECT id, message_id, user_email, emoji, created_at
		FROM email_reactions
		WHERE message_id = $1
		ORDER BY created_at ASC
	`, messageID)
	if err != nil {
		return nil, fmt.Errorf("querying reactions: %w", err)
	}
	defer rows.Close()

	var reactions []mail.EmailReaction
	for rows.Next() {
		var r mail.EmailReaction
		if err := rows.Scan(&r.ID, &r.MessageID, &r.UserEmail, &r.Emoji, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning reaction: %w", err)
		}
		reactions = append(reactions, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating reactions: %w", err)
	}

	return reactions, nil
}

// ToggleReaction adds or removes an emoji reaction for a user on a message.
func (s *MailRepository) ToggleReaction(ctx context.Context, messageID, userEmail, emoji string) (bool, error) {
	return ToggleReaction(ctx, s.pool, messageID, userEmail, emoji)
}

// GetReactionsForMessage returns all reactions for a specific message.
func (s *MailRepository) GetReactionsForMessage(ctx context.Context, messageID string) ([]mail.EmailReaction, error) {
	return GetReactionsForMessage(ctx, s.pool, messageID)
}
