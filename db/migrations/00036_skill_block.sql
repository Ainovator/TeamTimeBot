-- +goose Up
INSERT INTO skills_catalog (code, name, is_active)
VALUES ('block', 'Блок', TRUE)
ON CONFLICT (code) DO UPDATE
SET
    name = EXCLUDED.name,
    is_active = TRUE,
    updated_at = NOW();

INSERT INTO group_member_skills (group_id, user_telegram_id, skill_id, score, created_at, updated_at)
SELECT gm.group_id, gm.user_telegram_id, sc.id, 5, NOW(), NOW()
FROM group_members gm
JOIN skills_catalog sc ON sc.code = 'block'
WHERE gm.is_active = TRUE
ON CONFLICT (group_id, user_telegram_id, skill_id) DO NOTHING;

-- +goose Down
DELETE FROM group_member_skills gms
USING skills_catalog sc
WHERE sc.code = 'block'
  AND gms.skill_id = sc.id;

DELETE FROM skills_catalog
WHERE code = 'block';
