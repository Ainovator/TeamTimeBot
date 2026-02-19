-- +goose Up
ALTER TABLE group_events
    ADD COLUMN IF NOT EXISTS max_places INTEGER NOT NULL DEFAULT 18;

ALTER TABLE group_events
    DROP CONSTRAINT IF EXISTS group_events_max_places_positive;

ALTER TABLE group_events
    ADD CONSTRAINT group_events_max_places_positive CHECK (max_places > 0);

-- If old 00037 was applied, it stored limit in poll_templates.max_places.
-- Copy that value to event template level for all bound events.
UPDATE group_events ge
SET max_places = COALESCE(
    NULLIF(to_jsonb(pt)->>'max_places', '')::INTEGER,
    ge.max_places,
    18
)
FROM poll_templates pt
WHERE pt.id = ge.poll_template_id;

ALTER TABLE event_instances
    ADD COLUMN IF NOT EXISTS poll_max_places INTEGER NOT NULL DEFAULT 18;

ALTER TABLE event_instances
    DROP CONSTRAINT IF EXISTS event_instances_poll_max_places_positive;

ALTER TABLE event_instances
    ADD CONSTRAINT event_instances_poll_max_places_positive CHECK (poll_max_places > 0);

-- Backfill only invalid/missing snapshot values.
UPDATE event_instances ei
SET poll_max_places = COALESCE(ge.max_places, 18)
FROM group_events ge
WHERE ge.id = ei.event_id
  AND (ei.poll_max_places IS NULL OR ei.poll_max_places <= 0);

-- Remove legacy location to avoid confusion with current architecture.
ALTER TABLE poll_templates
    DROP CONSTRAINT IF EXISTS poll_templates_max_places_positive;

ALTER TABLE poll_templates
    DROP COLUMN IF EXISTS max_places;

-- +goose Down
ALTER TABLE poll_templates
    ADD COLUMN IF NOT EXISTS max_places INTEGER NOT NULL DEFAULT 18;

ALTER TABLE poll_templates
    DROP CONSTRAINT IF EXISTS poll_templates_max_places_positive;

ALTER TABLE poll_templates
    ADD CONSTRAINT poll_templates_max_places_positive CHECK (max_places > 0);
