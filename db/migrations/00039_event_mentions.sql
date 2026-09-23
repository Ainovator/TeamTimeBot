-- +goose Up
ALTER TABLE group_events
    ADD COLUMN mention_user_ids BIGINT[] NOT NULL DEFAULT '{}',
    ADD COLUMN mention_on_poll BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN mention_on_announcement BOOLEAN NOT NULL DEFAULT TRUE;

-- +goose Down
ALTER TABLE group_events
    DROP COLUMN mention_user_ids,
    DROP COLUMN mention_on_poll,
    DROP COLUMN mention_on_announcement;
