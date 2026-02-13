-- +goose Up
CREATE TABLE IF NOT EXISTS group_admins (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES telegram_groups(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL,
    username TEXT,
    first_name TEXT,
    last_name TEXT,
    role TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT group_admins_unique_group_user UNIQUE (group_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_group_admins_user_active
    ON group_admins (user_id, is_active);

CREATE INDEX IF NOT EXISTS idx_group_admins_group_active
    ON group_admins (group_id, is_active);

-- +goose Down
DROP TABLE IF EXISTS group_admins;
