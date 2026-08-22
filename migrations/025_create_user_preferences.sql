-- 025_create_user_preferences.sql

CREATE TABLE user_preferences (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    welcome_experience_completed BOOLEAN NOT NULL DEFAULT FALSE,
    welcome_experience_completed_at TIMESTAMPTZ NULL
);
