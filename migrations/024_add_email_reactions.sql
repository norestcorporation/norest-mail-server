-- +goose Up
-- +goose StatementBegin
CREATE TABLE email_reactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id TEXT NOT NULL,
    user_email TEXT NOT NULL,
    emoji TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(message_id, user_email, emoji)
);
CREATE INDEX idx_email_reactions_message_id ON email_reactions(message_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE email_reactions;
-- +goose StatementEnd
