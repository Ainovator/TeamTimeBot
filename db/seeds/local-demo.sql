-- Development fixture only. Run explicitly with db/seed-local.py, never as a migration.
-- Synthetic Telegram identifiers; no schedules or external publications are enabled.
BEGIN;
SET LOCAL TIME ZONE 'Europe/Moscow';
SELECT pg_advisory_xact_lock(-900001);

DO $demo$
DECLARE
    demo_group bigint;
    demo_template bigint;
    demo_event bigint;
    demo_instance bigint;
    demo_post bigint;
    demo_session bigint;
    demo_settlement bigint;
    day_offset integer;
    meeting_date date;
    meeting_status text;
    attendee_count integer;
    fixture record;
BEGIN
    -- Re-running preserves every edit made while testing, including renaming the team.
    IF EXISTS (SELECT 1 FROM telegram_groups WHERE chat_id = -900001) THEN
        RAISE NOTICE 'Demo chat -900001 already exists; nothing was changed.';
        RETURN;
    END IF;
    IF EXISTS (SELECT 1 FROM telegram_users WHERE telegram_id BETWEEN 900000001 AND 900000016) THEN
        RAISE EXCEPTION 'Demo user ID range is already occupied; no data was changed.';
    END IF;

    INSERT INTO telegram_groups (chat_id, title, timezone)
    VALUES (-900001, 'Орбита · тестовая команда', 'Europe/Moscow')
    RETURNING id INTO demo_group;

    FOR fixture IN SELECT * FROM (VALUES
        (1, 'Александр', 'Морозов', 'setter'),
        (2, 'Дмитрий', 'Соколов', 'attacker'),
        (3, 'Иван', 'Петров', 'central'),
        (4, 'Анна', 'Волкова', 'libero'),
        (5, 'Максим', 'Орлов', 'attacker'),
        (6, 'Мария', 'Лебедева', 'central'),
        (7, 'Артём', 'Кузнецов', 'setter'),
        (8, 'Елена', 'Новикова', 'attacker'),
        (9, 'Никита', 'Фролов', 'central'),
        (10, 'Ольга', 'Белова', 'libero'),
        (11, 'Сергей', 'Романов', 'attacker'),
        (12, 'Дарья', 'Смирнова', 'central'),
        (13, 'Павел', 'Громов', 'setter'),
        (14, 'Софья', 'Миронова', 'libero'),
        (15, 'Кирилл', 'Зайцев', 'attacker'),
        (16, 'Полина', 'Соколова', 'central')
    ) AS player(number, first_name, last_name, player_type)
    LOOP
        INSERT INTO telegram_users (telegram_id, first_name, last_name, language)
        VALUES (900000000 + fixture.number, fixture.first_name, fixture.last_name, 'ru');
        INSERT INTO group_members (group_id, user_telegram_id, role, player_type, real_name, joined_at)
        VALUES (demo_group, 900000000 + fixture.number,
            CASE WHEN fixture.number = 1 THEN 'admin' ELSE 'member' END,
            fixture.player_type, fixture.first_name || ' ' || fixture.last_name,
            now() - (fixture.number * 3 + 10) * interval '1 day');
        INSERT INTO group_member_skills (group_id, user_telegram_id, skill_id, score)
        SELECT demo_group, 900000000 + fixture.number, id,
            LEAST(10, 4 + ((fixture.number + id::integer) % 4) +
                CASE WHEN (fixture.player_type = 'setter' AND code = 'set')
                    OR (fixture.player_type = 'attacker' AND code = 'attack')
                    OR (fixture.player_type = 'central' AND code = 'block')
                    OR (fixture.player_type = 'libero' AND code IN ('receive', 'defense'))
                THEN 2 ELSE 0 END)
        FROM skills_catalog WHERE is_active;
    END LOOP;

    INSERT INTO group_admins (group_id, user_id, first_name, last_name, role)
    VALUES (demo_group, 900000001, 'Александр', 'Морозов', 'creator');
    INSERT INTO group_roles (group_id, code, title, permissions) VALUES
        (demo_group, 'trainer', 'Тренер', '{"members_read":true,"members_write":true,"templates_manage":true,"event_templates_manage":true,"events_read":true,"events_manage":true,"polls_read":true,"roles_manage":true,"profile_read":true}'),
        (demo_group, 'captain', 'Капитан', '{"events_read":true,"events_manage":true,"polls_read":true,"profile_read":true}');
    INSERT INTO group_role_assignments (group_id, user_telegram_id, role_code) VALUES
        (demo_group, 900000007, 'trainer'), (demo_group, 900000002, 'captain');
    INSERT INTO player_relations (group_id, user_a_id, user_b_id, relation_type, weight) VALUES
        (demo_group, 900000001, 900000002, 'prefer_together', 7),
        (demo_group, 900000003, 900000009, 'avoid_together', 5);

    INSERT INTO poll_templates (group_id, name, question, options, counted_options, option_weights)
    VALUES (demo_group, 'Запись на волейбол', 'Кто придёт на тренировку?',
        '["Буду", "Буду с другом", "Не смогу"]', '[0, 1]', '[1, 2, 1]')
    RETURNING id INTO demo_template;
    INSERT INTO poll_templates (group_id, name, question, options, counted_options, option_weights) VALUES
        (demo_group, 'Выбираем время', 'Когда удобнее встретиться?',
            '["В субботу утром", "В субботу вечером", "В воскресенье"]', '[]', '[1, 1, 1]'),
        (demo_group, 'Открытая игра', 'Играем в выходные?',
            '["Играю", "Пока под вопросом", "Пропускаю"]', '[0]', '[1, 1, 1]');

    INSERT INTO group_events (group_id, name, start_weekday, start_time, end_time,
        poll_template_id, poll_publish_weekday, poll_publish_time, cost_amount,
        publish_enabled, announcement_enabled, settlement_publish_before, settlement_publish_after,
        teams_publish_list, cancel_notify_enabled, min_votes_to_hold, max_places, announcement_lead_minutes)
    VALUES (demo_group, 'Вечерний волейбол', 3, '19:00', '21:00', demo_template, 1, '10:00', 7200,
        false, false, false, false, false, false, 12, 18, 60)
    RETURNING id INTO demo_event;
    INSERT INTO group_events (group_id, name, start_weekday, start_time, end_time,
        poll_template_id, poll_publish_weekday, poll_publish_time, cost_amount,
        publish_enabled, announcement_enabled, settlement_publish_before, settlement_publish_after,
        teams_publish_list, cancel_notify_enabled, max_places, announcement_lead_minutes) VALUES
        (demo_group, 'Открытая игра в выходные', 7, '11:00', '13:00',
            (SELECT id FROM poll_templates WHERE group_id = demo_group AND name = 'Открытая игра'),
            5, '10:00', 9000, false, false, false, false, false, false, 18, 60),
        (demo_group, 'Отработка подачи и приёма', 5, '19:00', '20:30', demo_template,
            4, '10:00', 6000, false, false, false, false, false, false, 12, 60);
    INSERT INTO event_counted_options (event_id, option_index)
    SELECT ge.id, idx.value::integer
    FROM group_events ge JOIN poll_templates pt ON pt.id = ge.poll_template_id
    CROSS JOIN LATERAL jsonb_array_elements_text(pt.counted_options) idx(value)
    WHERE ge.group_id = demo_group;

    -- Relative dates keep a fresh installation useful in any week.
    FOREACH day_offset IN ARRAY ARRAY[-10, -3, 1, 5]
    LOOP
        meeting_date := current_date + day_offset;
        meeting_status := CASE WHEN day_offset = -10 THEN 'completed'
            WHEN day_offset = -3 THEN 'on_review'
            WHEN day_offset = 1 THEN 'on_distribution' ELSE 'in_voting' END;
        attendee_count := CASE WHEN day_offset = 5 THEN 9 ELSE 12 END;
        INSERT INTO event_instances (group_id, event_id, local_date, planned_start_at, planned_end_at,
            status, event_name, event_type, start_weekday, start_time, end_time,
            publish_enabled, cost_amount, min_votes_to_hold, settlement_enabled,
            settlement_publish_before, settlement_publish_after, teams_publish_list,
            poll_template_id, poll_template_name, poll_question, poll_options,
            poll_counted_options, poll_option_weights, poll_max_places)
        SELECT demo_group, demo_event, meeting_date,
            (meeting_date + time '19:00') AT TIME ZONE 'Europe/Moscow',
            (meeting_date + time '21:00') AT TIME ZONE 'Europe/Moscow', meeting_status,
            ge.name, ge.event_type, extract(isodow from meeting_date)::smallint, '19:00', '21:00',
            false, ge.cost_amount, ge.min_votes_to_hold, true, false, false, false,
            pt.id, pt.name, pt.question, pt.options, pt.counted_options, pt.option_weights, ge.max_places
        FROM group_events ge JOIN poll_templates pt ON pt.id = ge.poll_template_id
        WHERE ge.id = demo_event
        RETURNING id INTO demo_instance;
        INSERT INTO event_poll_posts (group_id, event_id, template_id, instance_id,
            telegram_message_id, telegram_poll_id, status, published_at)
        VALUES (demo_group, demo_event, demo_template, demo_instance, 0, NULL,
            CASE WHEN day_offset < 0 THEN 'closed' ELSE 'open' END,
            LEAST(now() - interval '1 day', (meeting_date - 2)::timestamptz + interval '10 hours'))
        RETURNING id INTO demo_post;
        UPDATE event_instances SET poll_post_id = demo_post WHERE id = demo_instance;
        INSERT INTO event_poll_votes (post_id, user_id, first_name, last_name, choice, source, voted_at)
        SELECT demo_post, telegram_id, first_name, last_name,
            CASE WHEN telegram_id - 900000000 <= attendee_count THEN 'option_0' ELSE 'option_2' END,
            'inline', (SELECT published_at FROM event_poll_posts WHERE id = demo_post)
                + (telegram_id - 900000000) * interval '10 minutes'
        FROM telegram_users WHERE telegram_id BETWEEN 900000001 AND 900000016;

        IF day_offset <= 1 THEN
            INSERT INTO event_team_sessions (group_id, event_id, post_id)
            VALUES (demo_group, demo_event, demo_post) RETURNING id INTO demo_session;
            INSERT INTO event_team_assignments (session_id, user_id, team, position)
            SELECT demo_session, 900000000 + n, CASE WHEN n <= 6 THEN 'A' ELSE 'B' END,
                (n - 1) % 6 FROM generate_series(1, 12) n;
        END IF;
        IF day_offset < 0 THEN
            INSERT INTO event_settlements (group_id, event_id, post_id, instance_id, local_date,
                total_amount, participants_count, amount_per_person)
            VALUES (demo_group, demo_event, demo_post, demo_instance, meeting_date, 7200, 12, 600)
            RETURNING id INTO demo_settlement;
            INSERT INTO event_settlement_payments (settlement_id, user_id, first_name, last_name,
                amount_due, is_paid, paid_at)
            SELECT demo_settlement, telegram_id, first_name, last_name, 600,
                NOT (telegram_id = 900000002 OR (day_offset = -3 AND telegram_id IN (900000003, 900000008))),
                CASE WHEN telegram_id = 900000002 OR (day_offset = -3 AND telegram_id IN (900000003, 900000008))
                    THEN NULL ELSE meeting_date::timestamptz + interval '22 hours' END
            FROM telegram_users WHERE telegram_id BETWEEN 900000001 AND 900000012;
            INSERT INTO event_instance_set_rows (group_id, event_id, instance_id, ordinal, team1, score1, team2, score2) VALUES
                (demo_group, demo_event, demo_instance, 1, 'A', 25, 'B', 21),
                (demo_group, demo_event, demo_instance, 2, 'A', 23, 'B', 25),
                (demo_group, demo_event, demo_instance, 3, 'A', 15, 'B', 12);
        END IF;
    END LOOP;
    RAISE NOTICE 'Created local demo team: 16 players, 4 meetings, 3 event templates, 3 poll templates.';
END
$demo$;
COMMIT;

SELECT chat_id, title FROM telegram_groups WHERE chat_id = -900001;
