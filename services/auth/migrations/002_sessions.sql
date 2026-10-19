-- Refresh-token rotation, session lookup by user, and failed-login tracking for lockout.
ALTER TABLE auth_sessions ADD COLUMN IF NOT EXISTS last_used_at timestamptz;
CREATE INDEX IF NOT EXISTS auth_sessions_user_idx ON auth_sessions(user_id);
CREATE TABLE IF NOT EXISTS login_attempts (
  username text PRIMARY KEY,
  failed integer NOT NULL DEFAULT 0,
  window_start timestamptz,
  locked_until timestamptz
);
