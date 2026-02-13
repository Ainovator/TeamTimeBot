-- +goose Up
CREATE TABLE IF NOT EXISTS group_members (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES telegram_groups(id) ON DELETE CASCADE,
    user_telegram_id BIGINT NOT NULL REFERENCES telegram_users(telegram_id) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'member',
    status TEXT NOT NULL DEFAULT 'active',
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT group_members_unique_group_user UNIQUE (group_id, user_telegram_id),
    CONSTRAINT group_members_role_check CHECK (role IN ('member', 'admin')),
    CONSTRAINT group_members_status_check CHECK (status IN ('active', 'left', 'kicked'))
);

CREATE INDEX IF NOT EXISTS idx_group_members_user_active
    ON group_members (user_telegram_id, is_active);

CREATE INDEX IF NOT EXISTS idx_group_members_group_active
    ON group_members (group_id, is_active);

-- +goose Down
DROP TABLE IF EXISTS group_members;
