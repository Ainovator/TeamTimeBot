-- +goose Up
ALTER TABLE group_events
    ADD COLUMN IF NOT EXISTS settlement_enabled BOOLEAN NOT NULL DEFAULT TRUE;

ALTER TABLE group_events
    ADD COLUMN IF NOT EXISTS settlement_publish_before BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE group_events
    ADD COLUMN IF NOT EXISTS settlement_publish_after BOOLEAN NOT NULL DEFAULT TRUE;

CREATE TABLE IF NOT EXISTS event_settlement_notices (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES telegram_groups(id) ON DELETE CASCADE,
    event_id BIGINT NOT NULL REFERENCES group_events(id) ON DELETE CASCADE,
    local_date DATE NOT NULL,
    notice_type TEXT NOT NULL,
    sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT event_settlement_notices_unique UNIQUE (event_id, local_date, notice_type),
    CONSTRAINT event_settlement_notices_type_check CHECK (notice_type IN ('before', 'after'))
);

CREATE INDEX IF NOT EXISTS idx_event_settlement_notices_group_date
    ON event_settlement_notices (group_id, local_date DESC);

-- +goose Down
DROP TABLE IF EXISTS event_settlement_notices;

ALTER TABLE group_events
    DROP COLUMN IF EXISTS settlement_publish_after;

ALTER TABLE group_events
    DROP COLUMN IF EXISTS settlement_publish_before;

ALTER TABLE group_events
    DROP COLUMN IF EXISTS settlement_enabled;
