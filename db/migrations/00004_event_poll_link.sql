-- +goose Up
ALTER TABLE group_events
    ADD COLUMN IF NOT EXISTS poll_template_id BIGINT REFERENCES poll_templates(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_group_events_poll_template
    ON group_events (poll_template_id);

-- +goose Down
ALTER TABLE group_events
    DROP COLUMN IF EXISTS poll_template_id;
