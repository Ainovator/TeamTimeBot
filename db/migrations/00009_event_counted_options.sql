-- +goose Up
CREATE TABLE IF NOT EXISTS event_counted_options (
    id BIGSERIAL PRIMARY KEY,
    event_id BIGINT NOT NULL REFERENCES group_events(id) ON DELETE CASCADE,
    option_index INT NOT NULL CHECK (option_index >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT event_counted_options_unique UNIQUE (event_id, option_index)
);

CREATE INDEX IF NOT EXISTS idx_event_counted_options_event
    ON event_counted_options (event_id);

-- +goose Down
DROP TABLE IF EXISTS event_counted_options;
