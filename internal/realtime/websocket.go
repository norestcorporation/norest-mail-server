package realtime

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
)

// TicketValidator defines the interface for validating WebSocket tickets
type TicketValidator interface {
	ValidateWebSocketTicket(ctx context.Context, ticket string) (uuid.UUID, error)
}

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

type Client struct {
	broker *Broker
	conn   *websocket.Conn
	UserID string
	send   chan Event
}

// Handler upgrades HTTP requests to WebSocket connections and registers them with the broker.
func Handler(broker *Broker, ticketValidator TicketValidator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check for ticket in query parameter
		ticket := r.URL.Query().Get("ticket")
		if ticket == "" {
			// No ticket provided - this endpoint now requires ticket authentication
			http.Error(w, "ticket required", http.StatusUnauthorized)
			return
		}

		// Validate ticket
		userID, err := ticketValidator.ValidateWebSocketTicket(r.Context(), ticket)
		if err != nil {
			slog.Error("websocket ticket validation failed", "error", err)
			http.Error(w, "invalid ticket", http.StatusUnauthorized)
			return
		}

		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			OriginPatterns: []string{"*"}, // allow all origins for now
		})
		if err != nil {
			slog.Error("websocket accept error", "error", err)
			return
		}

		client := &Client{
			broker: broker,
			conn:   c,
			UserID: userID.String(),
			send:   make(chan Event, 256),
		}

		broker.Register(client)

		// Start pump goroutines
		go client.writePump(r.Context())
		client.readPump(r.Context())
	}
}

func (c *Client) readPump(ctx context.Context) {
	defer func() {
		c.broker.Unregister(c)
		c.conn.Close(websocket.StatusNormalClosure, "")
	}()

	c.conn.SetReadLimit(512)
	// Not using gorilla/websocket, using coder/websocket. Ping/Pong is handled automatically by coder/websocket!
	for {
		_, _, err := c.conn.Read(ctx)
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure ||
				websocket.CloseStatus(err) == websocket.StatusGoingAway {
				break
			}
			slog.Debug("websocket read error", "error", err)
			break
		}
	}
}

func (c *Client) writePump(ctx context.Context) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close(websocket.StatusNormalClosure, "")
	}()

	for {
		select {
		case event, ok := <-c.send:
			if !ok {
				c.conn.Close(websocket.StatusNormalClosure, "")
				return
			}

			ctxWrite, cancel := context.WithTimeout(ctx, writeWait)
			w, err := c.conn.Writer(ctxWrite, websocket.MessageText)
			if err == nil {
				_, err = w.Write([]byte(mustMarshal(event)))
				w.Close()
			}
			cancel()

			if err != nil {
				return
			}
		case <-ticker.C:
			ctxWrite, cancel := context.WithTimeout(ctx, writeWait)
			err := c.conn.Ping(ctxWrite)
			cancel()
			if err != nil {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func mustMarshal(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
