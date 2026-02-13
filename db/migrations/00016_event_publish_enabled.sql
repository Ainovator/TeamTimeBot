-- +goose Up
ALTER TABLE group_events
    ADD COLUMN IF NOT EXISTS publish_enabled BOOLEAN NOT NULL DEFAULT TRUE;

-- +goose Down
ALTER TABLE group_events
    DROP COLUMN IF EXISTS publish_enabled;
