-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'event_team_assignments_team_check'
    ) THEN
        ALTER TABLE event_team_assignments
            DROP CONSTRAINT event_team_assignments_team_check;
    END IF;
END $$;
-- +goose StatementEnd

ALTER TABLE event_team_assignments
    ADD CONSTRAINT event_team_assignments_team_check
    CHECK (team IN ('unassigned', 'A', 'B', 'C'));

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'event_team_assignments_team_check'
    ) THEN
        ALTER TABLE event_team_assignments
            DROP CONSTRAINT event_team_assignments_team_check;
    END IF;
END $$;
-- +goose StatementEnd

ALTER TABLE event_team_assignments
    ADD CONSTRAINT event_team_assignments_team_check
    CHECK (team IN ('unassigned', 'A', 'B'));
