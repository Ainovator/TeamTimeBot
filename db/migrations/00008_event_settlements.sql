-- +goose Up
CREATE TABLE IF NOT EXISTS event_settlements (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES telegram_groups(id) ON DELETE CASCADE,
    event_id BIGINT NOT NULL REFERENCES group_events(id) ON DELETE CASCADE,
    post_id BIGINT REFERENCES event_poll_posts(id) ON DELETE SET NULL,
    local_date DATE NOT NULL,
    total_amount NUMERIC(10,2) NOT NULL DEFAULT 4000.00,
    participants_count INT NOT NULL DEFAULT 0,
    amount_per_person NUMERIC(10,2) NOT NULL DEFAULT 0.00,
    sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT event_settlements_unique_event_date UNIQUE (event_id, local_date)
);

CREATE INDEX IF NOT EXISTS idx_event_settlements_group_date
    ON event_settlements (group_id, local_date DESC);

-- +goose Down
DROP TABLE IF EXISTS event_settlements;
