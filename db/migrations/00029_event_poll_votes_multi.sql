-- +goose Up
-- Allow multiple selected answers per user by storing one row per (post_id, user_id, choice).

ALTER TABLE event_poll_votes
    DROP CONSTRAINT IF EXISTS event_poll_votes_unique_post_user;

-- If we ever persisted comma-separated choices (legacy bug), split them into multiple rows.
WITH to_split AS (
    SELECT
        id,
        post_id,
        user_id,
        username,
        first_name,
        last_name,
        source,
        voted_at,
        created_at,
        updated_at,
        unnest(string_to_array(choice, ',')) AS choice_item
    FROM event_poll_votes
    WHERE position(',' in choice) > 0
),
ins AS (
    INSERT INTO event_poll_votes (post_id, user_id, username, first_name, last_name, choice, source, voted_at, created_at, updated_at)
    SELECT
        post_id,
        user_id,
        username,
        first_name,
        last_name,
        trim(choice_item),
        source,
        voted_at,
        created_at,
        updated_at
    FROM to_split
    WHERE trim(choice_item) <> ''
    ON CONFLICT DO NOTHING
)
DELETE FROM event_poll_votes
WHERE id IN (SELECT id FROM to_split);

ALTER TABLE event_poll_votes
    ADD CONSTRAINT event_poll_votes_unique_post_user_choice UNIQUE (post_id, user_id, choice);

CREATE INDEX IF NOT EXISTS idx_event_poll_votes_post_user
    ON event_poll_votes (post_id, user_id);

CREATE INDEX IF NOT EXISTS idx_event_poll_votes_post_choice
    ON event_poll_votes (post_id, choice);

-- +goose Down
DROP INDEX IF EXISTS idx_event_poll_votes_post_choice;
DROP INDEX IF EXISTS idx_event_poll_votes_post_user;

ALTER TABLE event_poll_votes
    DROP CONSTRAINT IF EXISTS event_poll_votes_unique_post_user_choice;

ALTER TABLE event_poll_votes
    ADD CONSTRAINT event_poll_votes_unique_post_user UNIQUE (post_id, user_id);

