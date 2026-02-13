-- +goose Up
ALTER TABLE event_instances
    ADD COLUMN IF NOT EXISTS event_name TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS event_type TEXT NOT NULL DEFAULT 'training',
    ADD COLUMN IF NOT EXISTS start_weekday SMALLINT NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS start_time TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS end_time TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS publish_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS cost_amount DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS min_votes_to_hold INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS cancel_lead_minutes INT NOT NULL DEFAULT 180,
    ADD COLUMN IF NOT EXISTS cancel_notify_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS settlement_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS settlement_publish_before BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS settlement_publish_after BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS announcement_text TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS announcement_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS announcement_lead_minutes INT NOT NULL DEFAULT 60,
    ADD COLUMN IF NOT EXISTS teams_auto_split BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS teams_publish_list BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS team_size INT NOT NULL DEFAULT 6,
    ADD COLUMN IF NOT EXISTS poll_template_id BIGINT REFERENCES poll_templates(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS poll_template_name TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS poll_question TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS poll_options JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS poll_counted_options JSONB NOT NULL DEFAULT '[]'::jsonb;

-- Backfill snapshots from the current template state.
UPDATE event_instances ei
SET
    event_name = ge.name,
    event_type = ge.event_type,
    start_weekday = ge.start_weekday,
    start_time = ge.start_time,
    end_time = ge.end_time,
    publish_enabled = ge.publish_enabled,
    cost_amount = ge.cost_amount,
    min_votes_to_hold = ge.min_votes_to_hold,
    cancel_lead_minutes = ge.cancel_lead_minutes,
    cancel_notify_enabled = ge.cancel_notify_enabled,
    settlement_enabled = ge.settlement_enabled,
    settlement_publish_before = ge.settlement_publish_before,
    settlement_publish_after = ge.settlement_publish_after,
    announcement_text = COALESCE(ge.announcement_text, ''),
    announcement_enabled = ge.announcement_enabled,
    announcement_lead_minutes = ge.announcement_lead_minutes,
    teams_auto_split = ge.teams_auto_split,
    teams_publish_list = ge.teams_publish_list,
    team_size = ge.team_size,
    poll_template_id = ge.poll_template_id,
    poll_template_name = COALESCE(pt.name, ''),
    poll_question = COALESCE(pt.question, ''),
    poll_options = COALESCE(pt.options, '[]'::jsonb),
    poll_counted_options = COALESCE(pt.counted_options, '[]'::jsonb)
FROM group_events ge
LEFT JOIN poll_templates pt ON pt.id = ge.poll_template_id
WHERE ei.event_id = ge.id
  AND ei.event_name = '';

-- +goose Down
ALTER TABLE event_instances
    DROP COLUMN IF EXISTS poll_counted_options,
    DROP COLUMN IF EXISTS poll_options,
    DROP COLUMN IF EXISTS poll_question,
    DROP COLUMN IF EXISTS poll_template_name,
    DROP COLUMN IF EXISTS poll_template_id,
    DROP COLUMN IF EXISTS team_size,
    DROP COLUMN IF EXISTS teams_publish_list,
    DROP COLUMN IF EXISTS teams_auto_split,
    DROP COLUMN IF EXISTS announcement_lead_minutes,
    DROP COLUMN IF EXISTS announcement_enabled,
    DROP COLUMN IF EXISTS announcement_text,
    DROP COLUMN IF EXISTS settlement_publish_after,
    DROP COLUMN IF EXISTS settlement_publish_before,
    DROP COLUMN IF EXISTS settlement_enabled,
    DROP COLUMN IF EXISTS cancel_notify_enabled,
    DROP COLUMN IF EXISTS cancel_lead_minutes,
    DROP COLUMN IF EXISTS min_votes_to_hold,
    DROP COLUMN IF EXISTS cost_amount,
    DROP COLUMN IF EXISTS publish_enabled,
    DROP COLUMN IF EXISTS end_time,
    DROP COLUMN IF EXISTS start_time,
    DROP COLUMN IF EXISTS start_weekday,
    DROP COLUMN IF EXISTS event_type,
    DROP COLUMN IF EXISTS event_name;

