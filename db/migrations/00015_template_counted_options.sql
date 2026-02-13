-- +goose Up
ALTER TABLE poll_templates
    ADD COLUMN IF NOT EXISTS counted_options JSONB NOT NULL DEFAULT '[]'::jsonb;

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'poll_templates_counted_options_array'
    ) THEN
        ALTER TABLE poll_templates
            ADD CONSTRAINT poll_templates_counted_options_array
            CHECK (jsonb_typeof(counted_options) = 'array');
    END IF;
END$$;
-- +goose StatementEnd

UPDATE poll_templates
SET counted_options = '[0]'::jsonb
WHERE jsonb_typeof(options) = 'array'
  AND jsonb_array_length(options) > 0
  AND (counted_options IS NULL OR counted_options = '[]'::jsonb);

-- +goose Down
ALTER TABLE poll_templates
    DROP CONSTRAINT IF EXISTS poll_templates_counted_options_array;

ALTER TABLE poll_templates
    DROP COLUMN IF EXISTS counted_options;
