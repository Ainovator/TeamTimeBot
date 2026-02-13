-- +goose Up
ALTER TABLE event_instances
    ADD COLUMN IF NOT EXISTS poll_post_id BIGINT REFERENCES event_poll_posts(id) ON DELETE SET NULL;

-- Deduplicate historical data: keep only the latest poll per instance.
WITH ranked AS (
    SELECT
        id,
        instance_id,
        ROW_NUMBER() OVER (PARTITION BY instance_id ORDER BY published_at DESC, id DESC) AS rn
    FROM event_poll_posts
    WHERE instance_id IS NOT NULL
)
UPDATE event_poll_posts epp
SET instance_id = NULL
FROM ranked r
WHERE epp.id = r.id
  AND r.rn > 1;

-- Fill direct link from instance to its single poll.
UPDATE event_instances ei
SET poll_post_id = epp.id
FROM event_poll_posts epp
WHERE epp.instance_id = ei.id
  AND ei.poll_post_id IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS ux_event_instances_poll_post
    ON event_instances (poll_post_id)
    WHERE poll_post_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS ux_event_poll_posts_instance
    ON event_poll_posts (instance_id)
    WHERE instance_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS ux_event_poll_posts_instance;
DROP INDEX IF EXISTS ux_event_instances_poll_post;
ALTER TABLE event_instances DROP COLUMN IF EXISTS poll_post_id;
