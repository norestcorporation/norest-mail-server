package mail

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/norest-mail/server/internal/realtime"
)

// OutboxProcessor polls the mail_events_outbox table and publishes events to the realtime broker.
type OutboxProcessor struct {
	pool         *pgxpool.Pool
	pollInterval time.Duration
	batchSize    int
}

func NewOutboxProcessor(pool *pgxpool.Pool) *OutboxProcessor {
	return &OutboxProcessor{
		pool:         pool,
		pollInterval: 1 * time.Second,
		batchSize:    100,
	}
}

func (p *OutboxProcessor) Run(ctx context.Context) error {
	slog.Info("starting outbox processor")
	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := p.processBatch(ctx); err != nil && err != context.Canceled {
				slog.Error("outbox processing error", "error", err)
			}
		}
	}
}

func (p *OutboxProcessor) processBatch(ctx context.Context) error {
	// Use SKIP LOCKED to safely grab a batch across multiple worker replicas
	query := `
		WITH batch AS (
			SELECT id
			FROM mail_events_outbox
			WHERE status = 'pending'
			ORDER BY created_at ASC
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE mail_events_outbox o
		SET status = 'processing'
		FROM batch
		WHERE o.id = batch.id
		RETURNING o.id, o.user_id, o.event_type, o.payload
	`

	rows, err := p.pool.Query(ctx, query, p.batchSize)
	if err != nil {
		return err
	}
	defer rows.Close()

	type outboxEvent struct {
		ID        string
		UserID    string
		EventType string
		Payload   string
	}
	var events []outboxEvent

	for rows.Next() {
		var e outboxEvent
		if err := rows.Scan(&e.ID, &e.UserID, &e.EventType, &e.Payload); err != nil {
			return err
		}
		events = append(events, e)
	}
	rows.Close()

	if len(events) == 0 {
		return nil
	}

	for _, e := range events {
		var payload any
		if err := json.Unmarshal([]byte(e.Payload), &payload); err != nil {
			slog.Error("failed to unmarshal event payload", "error", err, "event_id", e.ID)
			p.markFailed(ctx, e.ID, err.Error())
			continue
		}

		rtEvent := realtime.Event{
			UserID:    e.UserID,
			EventType: e.EventType,
			Payload:   payload,
		}

		if err := realtime.Publish(ctx, p.pool, rtEvent); err != nil {
			slog.Error("failed to publish event", "error", err, "event_id", e.ID)
			p.markFailed(ctx, e.ID, err.Error())
			continue
		}

		p.markPublished(ctx, e.ID)
	}

	return nil
}

func (p *OutboxProcessor) markPublished(ctx context.Context, id string) {
	_, err := p.pool.Exec(ctx, "UPDATE mail_events_outbox SET status = 'published', published_at = NOW() WHERE id = $1", id)
	if err != nil {
		slog.Error("failed to mark outbox event published", "error", err, "event_id", id)
	}
}

func (p *OutboxProcessor) markFailed(ctx context.Context, id string, errorMsg string) {
	_, err := p.pool.Exec(ctx, "UPDATE mail_events_outbox SET status = 'failed', error_message = $1 WHERE id = $2", errorMsg, id)
	if err != nil {
		slog.Error("failed to mark outbox event failed", "error", err, "event_id", id)
	}
}
