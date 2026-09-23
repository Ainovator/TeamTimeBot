-- +goose Up
ALTER TABLE event_settlement_payments ADD COLUMN IF NOT EXISTS cash_amount_paid NUMERIC(10,2);
CREATE OR REPLACE VIEW event_effective_payments AS
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
-- Cash amounts are retained to avoid losing payment history on rollback.
SELECT 1;
