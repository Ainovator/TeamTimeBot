-- +goose Up
CREATE TABLE IF NOT EXISTS event_instance_set_rows (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES telegram_groups(id) ON DELETE CASCADE,
    event_id BIGINT NOT NULL REFERENCES group_events(id) ON DELETE CASCADE,
    instance_id BIGINT NOT NULL REFERENCES event_instances(id) ON DELETE CASCADE,
    ordinal INT NOT NULL CHECK (ordinal > 0),
    team1 TEXT NOT NULL CHECK (team1 IN ('A', 'B', 'C')),
    score1 INT NOT NULL DEFAULT 0 CHECK (score1 >= 0),
    team2 TEXT NOT NULL CHECK (team2 IN ('A', 'B', 'C')),
    score2 INT NOT NULL DEFAULT 0 CHECK (score2 >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT event_instance_set_rows_unique_instance_ordinal UNIQUE (instance_id, ordinal),
    CONSTRAINT event_instance_set_rows_team_diff CHECK (team1 <> team2)
);

CREATE INDEX IF NOT EXISTS idx_event_instance_set_rows_group_instance
    ON event_instance_set_rows (group_id, instance_id);

CREATE INDEX IF NOT EXISTS idx_event_instance_set_rows_group_event
    ON event_instance_set_rows (group_id, event_id, instance_id);

-- Backfill from JSONB storage if present.
INSERT INTO event_instance_set_rows (group_id, event_id, instance_id, ordinal, team1, score1, team2, score2, created_at, updated_at)
WITH matches AS (
    SELECT
        eis.group_id,
        eis.event_id,
        eis.instance_id,
        m.match,
        m.match_idx
    FROM event_instance_sets eis
    CROSS JOIN LATERAL jsonb_array_elements(eis.data) WITH ORDINALITY AS m(match, match_idx)
),
sets AS (
    SELECT
        group_id,
        event_id,
        instance_id,
        (match->>'left') AS team1,
        (match->>'right') AS team2,
        s.set,
        match_idx,
        s.set_idx
    FROM matches
    CROSS JOIN LATERAL jsonb_array_elements(match->'sets') WITH ORDINALITY AS s(set, set_idx)
),
ordered AS (
    SELECT
        group_id,
        event_id,
        instance_id,
        ROW_NUMBER() OVER (PARTITION BY instance_id ORDER BY match_idx, set_idx) AS ordinal,
        team1,
        team2,
        COALESCE((set->>'left')::int, 0) AS score1,
        COALESCE((set->>'right')::int, 0) AS score2
    FROM sets
)
SELECT
    group_id,
    event_id,
    instance_id,
    ordinal,
    team1,
    score1,
    team2,
    score2,
    NOW(),
    NOW()
FROM ordered
WHERE team1 IN ('A','B','C') AND team2 IN ('A','B','C') AND team1 <> team2
ON CONFLICT (instance_id, ordinal) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS event_instance_set_rows;
