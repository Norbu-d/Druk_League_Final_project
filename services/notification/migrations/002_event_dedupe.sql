ALTER TABLE notifications ADD COLUMN IF NOT EXISTS event_id uuid;
ALTER TABLE notifications ALTER COLUMN status SET DEFAULT 'unread';
DROP INDEX IF EXISTS notifications_event_id_unique;
CREATE UNIQUE INDEX IF NOT EXISTS notifications_event_user_unique ON notifications(event_id,user_id) WHERE event_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS notifications_user_created ON notifications(user_id, created_at DESC, id DESC);