-- +goose Up
ALTER TABLE group_events
    ADD COLUMN IF NOT EXISTS cost_amount NUMERIC(10,2);

-- +goose Down
ALTER TABLE group_events
    DROP COLUMN IF EXISTS cost_amount;
