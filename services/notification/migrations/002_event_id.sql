ALTER TABLE notifications ADD COLUMN IF NOT EXISTS event_id uuid;
CREATE UNIQUE INDEX IF NOT EXISTS notifications_event_recipient_channel_unique
  ON notifications(event_id,user_id,channel) WHERE event_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS notifications_user_created_idx ON notifications(user_id,created_at DESC,id DESC);
