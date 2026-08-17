package realtime

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Broker handles multi-replica realtime fan-out using PostgreSQL LISTEN/NOTIFY.
type Broker struct {
	pool        *pgxpool.Pool
	clients     map[string]map[*Client]struct{} // userID -> set of clients
	clientsLock sync.RWMutex
}

type Event struct {
	UserID    string `json:"user_id"`
	EventType string `json:"event_type"`
	Payload   any    `json:"payload"`
}

func NewBroker(pool *pgxpool.Pool) *Broker {
	return &Broker{
		pool:    pool,
		clients: make(map[string]map[*Client]struct{}),
	}
}

// Start begins listening for Postgres notifications
func (b *Broker) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			err := b.listen(ctx)
			if err != nil && err != context.Canceled {
				slog.Error("realtime broker listen error", "error", err)
				time.Sleep(2 * time.Second) // backoff
			}
		}
	}
}

func (b *Broker) listen(ctx context.Context) error {
	conn, err := b.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	_, err = conn.Exec(ctx, "LISTEN ws_events")
	if err != nil {
		return err
	}
	slog.Info("realtime broker listening on ws_events")

	for {
		notification, err := conn.Conn().WaitForNotification(ctx)
		if err != nil {
			return err
		}

		var event Event
		if err := json.Unmarshal([]byte(notification.Payload), &event); err != nil {
			slog.Error("realtime broker unmarshal error", "error", err)
			continue
		}

		b.broadcast(event)
	}
}

// Register adds a new client
func (b *Broker) Register(c *Client) {
	b.clientsLock.Lock()
	defer b.clientsLock.Unlock()

	if b.clients[c.UserID] == nil {
		b.clients[c.UserID] = make(map[*Client]struct{})
	}
	b.clients[c.UserID][c] = struct{}{}
	slog.Info("websocket client registered", "user_id", c.UserID)
}

// Unregister removes a client
func (b *Broker) Unregister(c *Client) {
	b.clientsLock.Lock()
	defer b.clientsLock.Unlock()

	if clients, ok := b.clients[c.UserID]; ok {
		delete(clients, c)
		if len(clients) == 0 {
			delete(b.clients, c.UserID)
		}
	}
	slog.Info("websocket client unregistered", "user_id", c.UserID)
}

func (b *Broker) broadcast(event Event) {
	b.clientsLock.RLock()
	defer b.clientsLock.RUnlock()

	clients, ok := b.clients[event.UserID]
	if !ok {
		return // No active clients for this user on this replica
	}

	for c := range clients {
		select {
		case c.send <- event:
		default:
			// If buffer is full, we close the client (too slow)
			close(c.send)
			delete(clients, c)
		}
	}
}

// Publish is used by the Outbox Worker to fan-out events across replicas.
func Publish(ctx context.Context, pool *pgxpool.Pool, event Event) error {
	payloadBytes, err := json.Marshal(event)
	if err != nil {
		return err
	}
	
	// pg_notify limits payload size to 8000 bytes. If payload is too large, we shouldn't send the full body.
	// For now, we assume events are small (just metadata).
	_, err = pool.Exec(ctx, "SELECT pg_notify('ws_events', $1)", string(payloadBytes))
	return err
}
