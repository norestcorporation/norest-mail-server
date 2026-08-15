-- 008_add_user_roles.sql
-- Add user role system for admin functionality
ALTER TABLE users ADD COLUMN role TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('user', 'admin'));
