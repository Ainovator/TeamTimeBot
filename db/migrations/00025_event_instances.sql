-- +goose Up
CREATE TABLE IF NOT EXISTS event_instances (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES telegram_groups(id) ON DELETE CASCADE,
    event_id BIGINT NOT NULL REFERENCES group_events(id) ON DELETE CASCADE,
    local_date DATE NOT NULL,
    planned_start_at TIMESTAMPTZ NOT NULL,
    planned_end_at TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL DEFAULT 'in_voting',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT event_instances_status_check CHECK (status IN ('in_voting', 'on_distribution', 'on_review', 'completed', 'not_held')),
    CONSTRAINT event_instances_unique UNIQUE (event_id, local_date)
);

CREATE INDEX IF NOT EXISTS idx_event_instances_group_date
    ON event_instances (group_id, local_date DESC);

ALTER TABLE event_poll_posts
    ADD COLUMN IF NOT EXISTS instance_id BIGINT REFERENCES event_instances(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_event_poll_posts_instance
    ON event_poll_posts (instance_id);

ALTER TABLE event_settlements
    ADD COLUMN IF NOT EXISTS instance_id BIGINT REFERENCES event_instances(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_event_settlements_instance
    ON event_settlements (instance_id);

-- Backfill instances from existing poll posts.
INSERT INTO event_instances (group_id, event_id, local_date, planned_start_at, planned_end_at, status, created_at, updated_at)
SELECT
    epp.group_id,
    epp.event_id,
    (epp.published_at AT TIME ZONE 'UTC')::date AS local_date,
    date_trunc('day', epp.published_at) AS planned_start_at,
    date_trunc('day', epp.published_at) AS planned_end_at,
    'completed',
    NOW(),
    NOW()
FROM event_poll_posts epp
WHERE epp.event_id IS NOT NULL
ON CONFLICT (event_id, local_date) DO NOTHING;

UPDATE event_poll_posts epp
SET instance_id = ei.id
FROM event_instances ei
WHERE epp.event_id = ei.event_id
  AND epp.event_id IS NOT NULL
  AND ei.local_date = (epp.published_at AT TIME ZONE 'UTC')::date
  AND epp.instance_id IS NULL;

UPDATE event_settlements es
SET instance_id = ei.id
FROM event_instances ei
WHERE es.event_id = ei.event_id
  AND es.local_date = ei.local_date
  AND es.instance_id IS NULL;

-- +goose Down
ALTER TABLE event_settlements DROP COLUMN IF EXISTS instance_id;
ALTER TABLE event_poll_posts DROP COLUMN IF EXISTS instance_id;
DROP TABLE IF EXISTS event_instances;
