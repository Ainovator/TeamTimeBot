-- +goose Up
CREATE TABLE IF NOT EXISTS player_relations (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES telegram_groups(id) ON DELETE CASCADE,
    user_a_id BIGINT NOT NULL,
    user_b_id BIGINT NOT NULL,
    relation_type TEXT NOT NULL,
    weight SMALLINT NOT NULL DEFAULT 5,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT player_relations_users_order_check CHECK (user_a_id < user_b_id),
    CONSTRAINT player_relations_relation_type_check CHECK (relation_type IN ('prefer_together', 'avoid_together')),
    CONSTRAINT player_relations_weight_check CHECK (weight BETWEEN 1 AND 10),
    CONSTRAINT player_relations_unique UNIQUE (group_id, user_a_id, user_b_id, relation_type)
);

CREATE INDEX IF NOT EXISTS idx_player_relations_group_user_a
    ON player_relations (group_id, user_a_id)
    WHERE is_active = TRUE;

CREATE INDEX IF NOT EXISTS idx_player_relations_group_user_b
    ON player_relations (group_id, user_b_id)
    WHERE is_active = TRUE;

-- +goose Down
DROP TABLE IF EXISTS player_relations;
