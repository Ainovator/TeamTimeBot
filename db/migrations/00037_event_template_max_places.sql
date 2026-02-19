-- +goose Up
ALTER TABLE group_events
    ADD COLUMN IF NOT EXISTS max_places INTEGER NOT NULL DEFAULT 18;

ALTER TABLE group_events
    DROP CONSTRAINT IF EXISTS group_events_max_places_positive;

ALTER TABLE group_events
    ADD CONSTRAINT group_events_max_places_positive CHECK (max_places > 0);

ALTER TABLE event_instances
    ADD COLUMN IF NOT EXISTS poll_max_places INTEGER NOT NULL DEFAULT 18;

ALTER TABLE event_instances
    DROP CONSTRAINT IF EXISTS event_instances_poll_max_places_positive;

ALTER TABLE event_instances
    ADD CONSTRAINT event_instances_poll_max_places_positive CHECK (poll_max_places > 0);

UPDATE event_instances ei
SET poll_max_places = COALESCE(ge.max_places, 18)
FROM group_events ge
WHERE ge.id = ei.event_id;

-- +goose Down
ALTER TABLE event_instances
    DROP CONSTRAINT IF EXISTS event_instances_poll_max_places_positive;

ALTER TABLE event_instances
    DROP COLUMN IF EXISTS poll_max_places;

ALTER TABLE group_events
    DROP CONSTRAINT IF EXISTS group_events_max_places_positive;

ALTER TABLE group_events
    DROP COLUMN IF EXISTS max_places;
