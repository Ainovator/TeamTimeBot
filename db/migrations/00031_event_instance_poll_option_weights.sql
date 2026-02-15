-- +goose Up
ALTER TABLE event_instances
    ADD COLUMN IF NOT EXISTS poll_option_weights JSONB NOT NULL DEFAULT '[]'::jsonb;

ALTER TABLE event_instances
    DROP CONSTRAINT IF EXISTS event_instances_poll_option_weights_array;

ALTER TABLE event_instances
    ADD CONSTRAINT event_instances_poll_option_weights_array CHECK (jsonb_typeof(poll_option_weights) = 'array');

-- Backfill snapshots from the current template state.
UPDATE event_instances ei
SET poll_option_weights = COALESCE(pt.option_weights, '[]'::jsonb)
FROM group_events ge
LEFT JOIN poll_templates pt ON pt.id = ge.poll_template_id
WHERE ei.event_id = ge.id
  AND ei.poll_option_weights = '[]'::jsonb;

-- +goose Down
ALTER TABLE event_instances
    DROP CONSTRAINT IF EXISTS event_instances_poll_option_weights_array;

ALTER TABLE event_instances
    DROP COLUMN IF EXISTS poll_option_weights;

