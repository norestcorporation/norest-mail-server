-- 018_add_threads_and_messages.sql
-- Establishes the Norest-owned threading and message projection models.

CREATE TABLE threads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES mailboxes(id) ON DELETE CASCADE,
    subject TEXT NOT NULL,
    -- Denormalized projections
    participants JSONB NOT NULL DEFAULT '[]',
    message_count INT NOT NULL DEFAULT 1,
    unread_count INT NOT NULL DEFAULT 0,
    snippet TEXT,
    last_message_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (id, account_id)
);

CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES mailboxes(id) ON DELETE CASCADE,
    thread_id UUID NOT NULL,
    stalwart_email_id TEXT NOT NULL,
    message_id TEXT, -- RFC Message-ID
    in_reply_to TEXT,
    references_chain TEXT[],
    subject TEXT,
    sender JSONB,
    recipients JSONB,
    received_at TIMESTAMPTZ,
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (thread_id, account_id) REFERENCES threads(id, account_id) ON DELETE CASCADE,
    UNIQUE(account_id, stalwart_email_id)
);

-- Normalized message-to-mailbox relationship (Many-to-Many)
CREATE TABLE message_mailboxes (
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    stalwart_mailbox_id TEXT NOT NULL,
    is_unread BOOLEAN DEFAULT TRUE,
    is_draft BOOLEAN DEFAULT FALSE,
    is_trashed BOOLEAN DEFAULT FALSE,
    is_sent BOOLEAN DEFAULT FALSE,
    keywords TEXT[],
    PRIMARY KEY (message_id, stalwart_mailbox_id)
);

CREATE INDEX idx_messages_message_id ON messages(message_id);
CREATE INDEX idx_messages_thread_id ON messages(thread_id);
CREATE INDEX idx_threads_account_id ON threads(account_id);
