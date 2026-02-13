-- +goose Up
ALTER TABLE group_events
    ADD COLUMN IF NOT EXISTS event_type TEXT NOT NULL DEFAULT 'training',
    ADD COLUMN IF NOT EXISTS teams_auto_split BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS teams_publish_list BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS team_size SMALLINT NOT NULL DEFAULT 6,
    ADD COLUMN IF NOT EXISTS min_votes_to_hold INT NOT NULL DEFAULT 0;

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'group_events_event_type_check'
    ) THEN
        ALTER TABLE group_events
            ADD CONSTRAINT group_events_event_type_check
            CHECK (event_type IN ('training', 'activity'));
    END IF;
END$$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'group_events_team_size_check'
    ) THEN
        ALTER TABLE group_events
            ADD CONSTRAINT group_events_team_size_check
            CHECK (team_size >= 2);
    END IF;
END$$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'group_events_min_votes_to_hold_check'
    ) THEN
        ALTER TABLE group_events
            ADD CONSTRAINT group_events_min_votes_to_hold_check
            CHECK (min_votes_to_hold >= 0);
    END IF;
END$$;
-- +goose StatementEnd

-- +goose Down
ALTER TABLE group_events
    DROP CONSTRAINT IF EXISTS group_events_min_votes_to_hold_check,
    DROP CONSTRAINT IF EXISTS group_events_team_size_check,
    DROP CONSTRAINT IF EXISTS group_events_event_type_check,
    DROP COLUMN IF EXISTS min_votes_to_hold,
    DROP COLUMN IF EXISTS team_size,
    DROP COLUMN IF EXISTS teams_publish_list,
    DROP COLUMN IF EXISTS teams_auto_split,
    DROP COLUMN IF EXISTS event_type;
