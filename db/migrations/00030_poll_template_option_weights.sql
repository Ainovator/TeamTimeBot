-- +goose Up
ALTER TABLE poll_templates
    ADD COLUMN IF NOT EXISTS option_weights JSONB NOT NULL DEFAULT '[]'::jsonb;

ALTER TABLE poll_templates
    DROP CONSTRAINT IF EXISTS poll_templates_option_weights_array;

ALTER TABLE poll_templates
    ADD CONSTRAINT poll_templates_option_weights_array CHECK (jsonb_typeof(option_weights) = 'array');

-- Backfill: default all weights to 1 per option (if empty).
UPDATE poll_templates pt
SET option_weights = COALESCE((
    SELECT jsonb_agg(1)
    FROM generate_series(1, COALESCE(jsonb_array_length(pt.options), 0))
), '[]'::jsonb)
WHERE pt.option_weights = '[]'::jsonb;

-- +goose Down
ALTER TABLE poll_templates
    DROP CONSTRAINT IF EXISTS poll_templates_option_weights_array;

ALTER TABLE poll_templates
    DROP COLUMN IF EXISTS option_weights;

