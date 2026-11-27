ALTER TABLE media_assets
    ADD COLUMN IF NOT EXISTS match_id uuid,
    ADD COLUMN IF NOT EXISTS media_kind text NOT NULL DEFAULT 'photo',
    ADD COLUMN IF NOT EXISTS caption text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS duration_seconds bigint;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'media_assets_match_required'
          AND conrelid = 'media_assets'::regclass
    ) THEN
        -- Keep pre-existing rows migratable; new inserts must identify their match.
        ALTER TABLE media_assets
            ADD CONSTRAINT media_assets_match_required
            CHECK (match_id IS NOT NULL) NOT VALID;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'media_assets_kind_valid'
          AND conrelid = 'media_assets'::regclass
    ) THEN
        ALTER TABLE media_assets
            ADD CONSTRAINT media_assets_kind_valid
            CHECK (media_kind IN ('photo', 'highlight'));
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'media_assets_caption_length_valid'
          AND conrelid = 'media_assets'::regclass
    ) THEN
        ALTER TABLE media_assets
            ADD CONSTRAINT media_assets_caption_length_valid
            CHECK (length(caption) <= 1000);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'media_assets_duration_nonnegative'
          AND conrelid = 'media_assets'::regclass
    ) THEN
        ALTER TABLE media_assets
            ADD CONSTRAINT media_assets_duration_nonnegative
            CHECK (duration_seconds IS NULL OR duration_seconds >= 0);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS media_assets_match_created_idx
    ON media_assets (match_id, created_at DESC);

COMMENT ON TABLE media_assets IS
    'Photo and highlight metadata for matches; binary content is stored outside PostgreSQL.';
COMMENT ON COLUMN media_assets.match_id IS
    'Opaque UUID of the Match service entity; existence is validated by the service because Match uses a separate database.';
COMMENT ON COLUMN media_assets.object_key IS
    'Object-storage key for the binary asset; do not store the binary in PostgreSQL.';
COMMENT ON COLUMN media_assets.media_kind IS
    'Logical asset type: photo or highlight.';
COMMENT ON COLUMN media_assets.duration_seconds IS
    'Optional duration for video highlights, in whole seconds.';
