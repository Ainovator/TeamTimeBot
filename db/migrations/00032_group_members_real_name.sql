-- +goose Up
ALTER TABLE group_members
    ADD COLUMN IF NOT EXISTS real_name TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE group_members
    DROP COLUMN IF EXISTS real_name;

