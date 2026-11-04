-- Who submitted and who reviewed a registration, so team managers only see their own
-- registrations and every decision can be traced to an admin.
ALTER TABLE registrations ADD COLUMN IF NOT EXISTS registered_by uuid;
ALTER TABLE registrations ADD COLUMN IF NOT EXISTS reviewed_by uuid;
CREATE INDEX IF NOT EXISTS registrations_tournament_status_idx ON registrations(tournament_id, status);
CREATE INDEX IF NOT EXISTS registrations_registered_by_idx ON registrations(registered_by);
