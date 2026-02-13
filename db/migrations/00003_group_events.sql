-- +goose Up
CREATE TABLE IF NOT EXISTS group_events (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES telegram_groups(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    start_weekday SMALLINT NOT NULL CHECK (start_weekday BETWEEN 1 AND 7),
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_group_events_group_active
    ON group_events (group_id, is_active);

-- +goose Down
DROP TABLE IF EXISTS group_events;
