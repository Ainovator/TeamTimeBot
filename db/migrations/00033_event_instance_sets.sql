-- +goose Up
CREATE TABLE IF NOT EXISTS event_instance_sets (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES telegram_groups(id) ON DELETE CASCADE,
    event_id BIGINT NOT NULL REFERENCES group_events(id) ON DELETE CASCADE,
    instance_id BIGINT NOT NULL REFERENCES event_instances(id) ON DELETE CASCADE,
    data JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT event_instance_sets_unique_instance UNIQUE (instance_id),
    CONSTRAINT event_instance_sets_data_array CHECK (jsonb_typeof(data) = 'array')
);

CREATE INDEX IF NOT EXISTS idx_event_instance_sets_group_instance
    ON event_instance_sets (group_id, instance_id);

CREATE INDEX IF NOT EXISTS idx_event_instance_sets_group_event
    ON event_instance_sets (group_id, event_id, instance_id);

-- +goose Down
DROP TABLE IF EXISTS event_instance_sets;

