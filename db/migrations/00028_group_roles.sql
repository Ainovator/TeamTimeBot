-- +goose Up
CREATE TABLE IF NOT EXISTS group_roles (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES telegram_groups(id) ON DELETE CASCADE,
    code TEXT NOT NULL,
    title TEXT NOT NULL,
    permissions JSONB NOT NULL DEFAULT '{}'::jsonb,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT group_roles_unique UNIQUE (group_id, code)
);

CREATE INDEX IF NOT EXISTS idx_group_roles_group
    ON group_roles (group_id, is_active);

CREATE TABLE IF NOT EXISTS group_role_assignments (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES telegram_groups(id) ON DELETE CASCADE,
    user_telegram_id BIGINT NOT NULL,
    role_code TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT group_role_assignments_unique UNIQUE (group_id, user_telegram_id),
    CONSTRAINT group_role_assignments_role_fk
        FOREIGN KEY (group_id, role_code) REFERENCES group_roles(group_id, code) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_group_role_assignments_group
    ON group_role_assignments (group_id, is_active);

CREATE INDEX IF NOT EXISTS idx_group_role_assignments_user
    ON group_role_assignments (user_telegram_id, is_active);

-- +goose Down
DROP TABLE IF EXISTS group_role_assignments;
DROP TABLE IF EXISTS group_roles;

