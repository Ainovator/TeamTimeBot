-- +goose Up
ALTER TABLE group_members
    ADD COLUMN IF NOT EXISTS player_type TEXT NOT NULL DEFAULT '';

-- +goose StatementBegin
DO $m$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'group_members_player_type_check'
    ) THEN
        ALTER TABLE group_members
            ADD CONSTRAINT group_members_player_type_check
            CHECK (player_type IN ('', 'attacker', 'setter', 'libero'));
    END IF;
END;
$m$;
-- +goose StatementEnd

-- +goose Down
ALTER TABLE group_members
    DROP CONSTRAINT IF EXISTS group_members_player_type_check;

ALTER TABLE group_members
    DROP COLUMN IF EXISTS player_type;
