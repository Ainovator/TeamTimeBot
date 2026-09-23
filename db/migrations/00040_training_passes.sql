-- +goose Up
CREATE TABLE training_passes (
 id BIGSERIAL PRIMARY KEY,
 group_id BIGINT NOT NULL REFERENCES telegram_groups(id),
 user_telegram_id BIGINT NOT NULL,
 name TEXT NOT NULL CHECK (length(name) BETWEEN 1 AND 100),
 total_visits INTEGER CHECK (total_visits BETWEEN 1 AND 1000),
 starts_on DATE NOT NULL,
 ends_on DATE NOT NULL CHECK (ends_on >= starts_on),
 price_kopecks BIGINT NOT NULL DEFAULT 0 CHECK (price_kopecks >= 0),
 is_paid BOOLEAN NOT NULL DEFAULT FALSE,
 frozen_at TIMESTAMPTZ,
 is_closed BOOLEAN NOT NULL DEFAULT FALSE,
 version INTEGER NOT NULL DEFAULT 1,
 request_id UUID NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 UNIQUE (group_id, request_id),
 FOREIGN KEY (group_id, user_telegram_id) REFERENCES group_members(group_id, user_telegram_id)
);
CREATE INDEX training_passes_member ON training_passes(group_id,user_telegram_id);
CREATE TABLE training_attendance (
 id BIGSERIAL PRIMARY KEY,
 group_id BIGINT NOT NULL REFERENCES telegram_groups(id),
 instance_id BIGINT NOT NULL REFERENCES event_instances(id),
 user_telegram_id BIGINT NOT NULL,
 pass_id BIGINT REFERENCES training_passes(id),
 status TEXT NOT NULL CHECK (status IN ('attended','absent','unmarked','cancelled')),
 actor_id BIGINT NOT NULL DEFAULT 0,
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 UNIQUE (instance_id,user_telegram_id)
);
CREATE INDEX training_attendance_pass ON training_attendance(pass_id) WHERE status = 'attended';
CREATE TABLE training_pass_history (
 id BIGSERIAL PRIMARY KEY,
 pass_id BIGINT NOT NULL REFERENCES training_passes(id),
 instance_id BIGINT REFERENCES event_instances(id),
 actor_id BIGINT NOT NULL DEFAULT 0,
 note TEXT NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- A cancellation from either the API or scheduler returns visits in the same transaction.
-- +goose StatementBegin
CREATE FUNCTION cancel_training_attendance() RETURNS TRIGGER AS $$
BEGIN
 IF (NEW.status = 'not_held' OR NOT NEW.is_active) AND
    (OLD.status IS DISTINCT FROM NEW.status OR OLD.is_active IS DISTINCT FROM NEW.is_active) THEN
  INSERT INTO training_pass_history(pass_id,instance_id,note)
   SELECT pass_id,instance_id,'Занятие возвращено: событие отменено'
   FROM training_attendance WHERE instance_id=NEW.id AND status='attended' AND pass_id IS NOT NULL;
  UPDATE training_attendance SET status='cancelled',updated_at=NOW() WHERE instance_id=NEW.id;
 END IF;
 RETURN NEW;
END; $$ LANGUAGE plpgsql;
-- +goose StatementEnd
CREATE TRIGGER cancel_training_attendance AFTER UPDATE ON event_instances
 FOR EACH ROW EXECUTE FUNCTION cancel_training_attendance();
-- Keep original cash entries intact: a pass covers exactly one seat, including votes with guests.
ALTER TABLE event_settlement_payments ADD COLUMN cash_amount_paid NUMERIC(10,2);
CREATE VIEW event_effective_payments AS
 SELECT p.id,p.settlement_id,p.user_id,p.username,p.first_name,p.last_name,
 CASE WHEN cash.amount > 0 AND cash.amount < net.amount THEN net.amount-cash.amount ELSE net.amount END AS amount_due,
 (cash.amount >= net.amount) AS is_paid,
 p.paid_at,p.created_at,p.updated_at,(a.id IS NOT NULL) AS pass_covered,
 CASE WHEN a.id IS NOT NULL THEN s.amount_per_person ELSE 0 END AS covered_amount
 FROM event_settlement_payments p
 JOIN event_settlements s ON s.id=p.settlement_id
 LEFT JOIN event_instances e ON e.id=s.instance_id
 LEFT JOIN training_attendance a ON a.instance_id=s.instance_id AND a.user_telegram_id=p.user_id
  AND a.group_id=s.group_id AND a.status='attended' AND a.pass_id IS NOT NULL
  AND e.status <> 'not_held' AND e.is_active
 CROSS JOIN LATERAL (SELECT GREATEST(0,p.amount_due-CASE WHEN a.id IS NOT NULL THEN s.amount_per_person ELSE 0 END) AS amount) net
 CROSS JOIN LATERAL (SELECT COALESCE(p.cash_amount_paid,CASE WHEN p.is_paid THEN p.amount_due ELSE 0 END) AS amount) cash;

-- +goose Down
DROP VIEW IF EXISTS event_effective_payments;
ALTER TABLE event_settlement_payments DROP COLUMN IF EXISTS cash_amount_paid;
DROP TRIGGER IF EXISTS cancel_training_attendance ON event_instances;
DROP FUNCTION IF EXISTS cancel_training_attendance();
DROP TABLE IF EXISTS training_pass_history;
DROP TABLE IF EXISTS training_attendance;
DROP TABLE IF EXISTS training_passes;
