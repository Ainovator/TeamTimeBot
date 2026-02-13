-- +goose Up
ALTER TABLE group_events
    ADD COLUMN IF NOT EXISTS poll_publish_weekday SMALLINT;

UPDATE group_events
SET poll_publish_weekday = start_weekday
WHERE poll_publish_weekday IS NULL;

ALTER TABLE group_events
    ALTER COLUMN poll_publish_weekday SET NOT NULL;

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'group_events_poll_publish_weekday_check'
    ) THEN
        ALTER TABLE group_events
            ADD CONSTRAINT group_events_poll_publish_weekday_check
            CHECK (poll_publish_weekday BETWEEN 1 AND 7);
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
ALTER TABLE group_events
    DROP CONSTRAINT IF EXISTS group_events_poll_publish_weekday_check;

ALTER TABLE group_events
    DROP COLUMN IF EXISTS poll_publish_weekday;
