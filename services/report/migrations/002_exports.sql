ALTER TABLE report_jobs ADD COLUMN IF NOT EXISTS tournament_id uuid;
ALTER TABLE report_jobs ADD COLUMN IF NOT EXISTS format text NOT NULL DEFAULT 'csv';
ALTER TABLE report_jobs ADD COLUMN IF NOT EXISTS file_name text NOT NULL DEFAULT '';
ALTER TABLE report_jobs ADD COLUMN IF NOT EXISTS content_type text NOT NULL DEFAULT 'application/octet-stream';
ALTER TABLE report_jobs ADD COLUMN IF NOT EXISTS content bytea NOT NULL DEFAULT decode('', 'hex');

CREATE INDEX IF NOT EXISTS report_jobs_requester_created_idx ON report_jobs(requester_id, created_at DESC);
