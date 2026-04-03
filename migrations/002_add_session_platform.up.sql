-- Migration: add_session_platform
-- Adds platform field to auth_sessions to support per-platform single-session enforcement

ALTER TABLE auth_sessions
ADD COLUMN platform TEXT NOT NULL DEFAULT 'web';

CREATE INDEX IF NOT EXISTS idx_auth_sessions_user_platform
ON auth_sessions(user_id, platform);
