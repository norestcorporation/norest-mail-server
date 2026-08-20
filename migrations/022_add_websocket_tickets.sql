-- 022_add_websocket_tickets.sql
-- Create table for WebSocket authentication tickets

CREATE TABLE IF NOT EXISTS websocket_tickets (
    ticket TEXT PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for efficient cleanup of expired tickets
CREATE INDEX idx_websocket_tickets_expires_at ON websocket_tickets(expires_at);

-- Index for user lookups
CREATE INDEX idx_websocket_tickets_user_id ON websocket_tickets(user_id);
