ALTER TABLE configuration_entries
    ADD COLUMN IF NOT EXISTS value_type text NOT NULL DEFAULT 'string',
    ADD COLUMN IF NOT EXISTS description text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS created_at timestamptz NOT NULL DEFAULT now();

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'configuration_entries_key_valid'
          AND conrelid = 'configuration_entries'::regclass
    ) THEN
        ALTER TABLE configuration_entries
            ADD CONSTRAINT configuration_entries_key_valid
            CHECK (key = btrim(key) AND length(key) BETWEEN 1 AND 255);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'configuration_entries_value_type_valid'
          AND conrelid = 'configuration_entries'::regclass
    ) THEN
        ALTER TABLE configuration_entries
            ADD CONSTRAINT configuration_entries_value_type_valid
            CHECK (value_type IN ('string', 'integer', 'number', 'boolean', 'json'));
    END IF;
END $$;

COMMENT ON TABLE configuration_entries IS
    'Globally keyed configuration values shared by services; use namespaced keys such as service.setting.';
COMMENT ON COLUMN configuration_entries.value IS
    'Configuration payload stored as text; value_type describes its interpretation.';
