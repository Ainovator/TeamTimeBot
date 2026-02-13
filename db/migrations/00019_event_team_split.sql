-- +goose Up
CREATE TABLE IF NOT EXISTS event_team_sessions (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES telegram_groups(id) ON DELETE CASCADE,
    event_id BIGINT NOT NULL REFERENCES group_events(id) ON DELETE CASCADE,
    post_id BIGINT NOT NULL REFERENCES event_poll_posts(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT event_team_sessions_post_unique UNIQUE (post_id)
);

CREATE INDEX IF NOT EXISTS idx_event_team_sessions_event
    ON event_team_sessions (event_id);

CREATE TABLE IF NOT EXISTS event_team_assignments (
    id BIGSERIAL PRIMARY KEY,
    session_id BIGINT NOT NULL REFERENCES event_team_sessions(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL,
    team TEXT NOT NULL,
    position INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT event_team_assignments_unique UNIQUE (session_id, user_id),
    CONSTRAINT event_team_assignments_team_check CHECK (team IN ('unassigned', 'A', 'B'))
);

CREATE INDEX IF NOT EXISTS idx_event_team_assignments_session_team
    ON event_team_assignments (session_id, team, position, id);

-- +goose Down
DROP TABLE IF EXISTS event_team_assignments;
DROP TABLE IF EXISTS event_team_sessions;
