-- +goose Up
ALTER TABLE group_events
    ADD COLUMN IF NOT EXISTS poll_publish_time TIME;

-- +goose Down
ALTER TABLE group_events
    DROP COLUMN IF EXISTS poll_publish_time;
