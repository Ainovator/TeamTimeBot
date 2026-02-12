-- +goose Up
CREATE TABLE IF NOT EXISTS telegram_groups (
    id BIGSERIAL PRIMARY KEY,
    chat_id BIGINT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    timezone TEXT NOT NULL DEFAULT 'UTC',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS poll_templates (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES telegram_groups(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    question TEXT NOT NULL,
    options JSONB NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT poll_templates_options_array CHECK (jsonb_typeof(options) = 'array'),
    CONSTRAINT poll_templates_unique_name_per_group UNIQUE (group_id, name)
);

CREATE TABLE IF NOT EXISTS poll_schedules (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES telegram_groups(id) ON DELETE CASCADE,
    template_id BIGINT NOT NULL REFERENCES poll_templates(id) ON DELETE CASCADE,
    cron_expr TEXT NOT NULL,
    next_run_at TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS training_slots (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES telegram_groups(id) ON DELETE CASCADE,
    weekday SMALLINT NOT NULL CHECK (weekday BETWEEN 0 AND 6),
    start_time TIME NOT NULL,
    duration_min INT NOT NULL DEFAULT 120 CHECK (duration_min > 0),
    location TEXT,
    note TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_poll_schedules_group_active
    ON poll_schedules (group_id, is_active);

CREATE INDEX IF NOT EXISTS idx_training_slots_group_weekday
    ON training_slots (group_id, weekday);

-- +goose Down
DROP TABLE IF EXISTS training_slots;
DROP TABLE IF EXISTS poll_schedules;
DROP TABLE IF EXISTS poll_templates;
DROP TABLE IF EXISTS telegram_groups;
