-- +goose Up
ALTER TABLE group_events
    ADD COLUMN IF NOT EXISTS cancel_lead_minutes INT NOT NULL DEFAULT 180,
    ADD COLUMN IF NOT EXISTS cancel_notify_enabled BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'group_events_cancel_lead_minutes_check'
    ) THEN
        ALTER TABLE group_events
            ADD CONSTRAINT group_events_cancel_lead_minutes_check
            CHECK (cancel_lead_minutes > 0);
    END IF;
END
$$;
-- +goose StatementEnd

CREATE TABLE IF NOT EXISTS event_cancellations (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES telegram_groups(id) ON DELETE CASCADE,
    event_id BIGINT NOT NULL REFERENCES group_events(id) ON DELETE CASCADE,
    event_date DATE NOT NULL,
    sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT event_cancellations_unique_event_date UNIQUE (event_id, event_date)
);

CREATE INDEX IF NOT EXISTS idx_event_cancellations_group_date
    ON event_cancellations (group_id, event_date DESC);

-- +goose Down
DROP TABLE IF EXISTS event_cancellations;

ALTER TABLE group_events
    DROP CONSTRAINT IF EXISTS group_events_cancel_lead_minutes_check;

ALTER TABLE group_events
    DROP COLUMN IF EXISTS cancel_notify_enabled,
    DROP COLUMN IF EXISTS cancel_lead_minutes;
