-- +goose Up
ALTER TABLE group_events
    ADD COLUMN IF NOT EXISTS announcement_text TEXT;

ALTER TABLE group_events
    ADD COLUMN IF NOT EXISTS announcement_enabled BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE group_events
    ADD COLUMN IF NOT EXISTS announcement_lead_minutes INT;

UPDATE group_events
SET announcement_lead_minutes = 60
WHERE announcement_lead_minutes IS NULL;

ALTER TABLE group_events
    ALTER COLUMN announcement_lead_minutes SET NOT NULL;

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'group_events_announcement_lead_minutes_check'
    ) THEN
        ALTER TABLE group_events
            ADD CONSTRAINT group_events_announcement_lead_minutes_check
            CHECK (announcement_lead_minutes IN (60, 120, 1440));
    END IF;
END
$$;
-- +goose StatementEnd

CREATE TABLE IF NOT EXISTS event_announcements (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES telegram_groups(id) ON DELETE CASCADE,
    event_id BIGINT NOT NULL REFERENCES group_events(id) ON DELETE CASCADE,
    event_date DATE NOT NULL,
    sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT event_announcements_unique_event_date UNIQUE (event_id, event_date)
);

CREATE INDEX IF NOT EXISTS idx_event_announcements_group_date
    ON event_announcements (group_id, event_date DESC);

-- +goose Down
DROP TABLE IF EXISTS event_announcements;

ALTER TABLE group_events
    DROP CONSTRAINT IF EXISTS group_events_announcement_lead_minutes_check;

ALTER TABLE group_events
    DROP COLUMN IF EXISTS announcement_lead_minutes;

ALTER TABLE group_events
    DROP COLUMN IF EXISTS announcement_enabled;

ALTER TABLE group_events
    DROP COLUMN IF EXISTS announcement_text;
