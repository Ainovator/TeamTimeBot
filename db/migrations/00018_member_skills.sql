-- +goose Up
CREATE TABLE IF NOT EXISTS skills_catalog (
    id BIGSERIAL PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS group_member_skills (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES telegram_groups(id) ON DELETE CASCADE,
    user_telegram_id BIGINT NOT NULL,
    skill_id BIGINT NOT NULL REFERENCES skills_catalog(id) ON DELETE RESTRICT,
    score SMALLINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT group_member_skills_score_check CHECK (score BETWEEN 1 AND 10),
    CONSTRAINT group_member_skills_unique UNIQUE (group_id, user_telegram_id, skill_id)
);

CREATE INDEX IF NOT EXISTS idx_group_member_skills_group_user
    ON group_member_skills (group_id, user_telegram_id);

INSERT INTO skills_catalog (code, name, is_active)
VALUES
    ('receive', 'Прием', TRUE),
    ('serve', 'Подача', TRUE),
    ('set', 'Пас', TRUE),
    ('defense', 'Защита', TRUE),
    ('attack', 'Атака', TRUE)
ON CONFLICT (code) DO UPDATE
SET
    name = EXCLUDED.name,
    is_active = TRUE,
    updated_at = NOW();

-- +goose Down
DROP TABLE IF EXISTS group_member_skills;
DROP TABLE IF EXISTS skills_catalog;
