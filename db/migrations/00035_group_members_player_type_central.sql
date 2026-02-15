-- +goose Up
-- Extend player_type enum-like constraint with "central".
ALTER TABLE group_members
    DROP CONSTRAINT IF EXISTS group_members_player_type_check;

ALTER TABLE group_members
    ADD CONSTRAINT group_members_player_type_check
    CHECK (player_type IN ('', 'attacker', 'setter', 'libero', 'central'));

-- +goose Down
ALTER TABLE group_members
    DROP CONSTRAINT IF EXISTS group_members_player_type_check;

ALTER TABLE group_members
    ADD CONSTRAINT group_members_player_type_check
    CHECK (player_type IN ('', 'attacker', 'setter', 'libero'));

