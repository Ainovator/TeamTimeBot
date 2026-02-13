-- +goose Up
CREATE TABLE IF NOT EXISTS event_settlement_payments (
    id BIGSERIAL PRIMARY KEY,
    settlement_id BIGINT NOT NULL REFERENCES event_settlements(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL,
    username TEXT NOT NULL DEFAULT '',
    first_name TEXT NOT NULL DEFAULT '',
    last_name TEXT NOT NULL DEFAULT '',
    amount_due NUMERIC(10,2) NOT NULL DEFAULT 0.00,
    is_paid BOOLEAN NOT NULL DEFAULT FALSE,
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT event_settlement_payments_unique UNIQUE (settlement_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_event_settlement_payments_settlement
    ON event_settlement_payments (settlement_id);

CREATE INDEX IF NOT EXISTS idx_event_settlement_payments_paid
    ON event_settlement_payments (settlement_id, is_paid);

-- +goose Down
DROP TABLE IF EXISTS event_settlement_payments;
