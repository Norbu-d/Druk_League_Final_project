-- One venue name per dzongkhag (case-insensitive), so the same ground is not registered twice.
CREATE UNIQUE INDEX IF NOT EXISTS venues_name_dzongkhag_unique ON venues (lower(name), dzongkhag);
