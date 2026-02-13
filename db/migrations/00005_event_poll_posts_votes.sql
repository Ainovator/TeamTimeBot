-- +goose Up
CREATE TABLE IF NOT EXISTS event_poll_posts (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES telegram_groups(id) ON DELETE CASCADE,
    event_id BIGINT REFERENCES group_events(id) ON DELETE SET NULL,
    template_id BIGINT NOT NULL REFERENCES poll_templates(id) ON DELETE RESTRICT,
    telegram_message_id BIGINT NOT NULL,
    telegram_poll_id TEXT,
    status TEXT NOT NULL DEFAULT 'open',
    published_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT event_poll_posts_status_check CHECK (status IN ('open', 'closed', 'canceled'))
);

CREATE INDEX IF NOT EXISTS idx_event_poll_posts_group_published
    ON event_poll_posts (group_id, published_at DESC);

CREATE INDEX IF NOT EXISTS idx_event_poll_posts_event
    ON event_poll_posts (event_id);

CREATE INDEX IF NOT EXISTS idx_event_poll_posts_poll_id
    ON event_poll_posts (telegram_poll_id);

CREATE TABLE IF NOT EXISTS event_poll_votes (
    id BIGSERIAL PRIMARY KEY,
    post_id BIGINT NOT NULL REFERENCES event_poll_posts(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL,
    username TEXT,
    first_name TEXT,
    last_name TEXT,
    choice TEXT NOT NULL,
    source TEXT NOT NULL DEFAULT 'inline',
    voted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT event_poll_votes_source_check CHECK (source IN ('inline', 'poll')),
    CONSTRAINT event_poll_votes_unique_post_user UNIQUE (post_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_event_poll_votes_post
    ON event_poll_votes (post_id);

-- +goose Down
DROP TABLE IF EXISTS event_poll_votes;
DROP TABLE IF EXISTS event_poll_posts;
