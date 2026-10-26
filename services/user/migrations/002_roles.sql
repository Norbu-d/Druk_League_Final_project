-- Roles are fixed; team_manager supports team self-registration. Usernames are unique
-- regardless of case so "Admin" and "admin" cannot both exist.
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_changed_at timestamptz;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('super_admin','match_official','team_manager','viewer'));
CREATE UNIQUE INDEX IF NOT EXISTS users_username_lower_unique ON users (lower(username));
