package postgres

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *Store) ListGroupMembersByChatID(ctx context.Context, chatID int64) ([]GroupMemberView, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	_ = s.EnsureDefaultGroupRoles(ctx, chatID)

	var rows []GroupMemberView
	if err := s.db.WithContext(ctx).
		Table("group_members gm").
		Select("gm.user_telegram_id, COALESCE(tu.username, '') AS username, COALESCE(tu.first_name, '') AS first_name, COALESCE(tu.last_name, '') AS last_name, COALESCE(gm.player_type, '') AS player_type, gm.role, gm.status, gm.last_seen_at, COALESCE(gra.role_code, '') AS app_role_code, COALESCE(gr.title, '') AS app_role_title").
		Joins("LEFT JOIN telegram_users tu ON tu.telegram_id = gm.user_telegram_id").
		Joins("LEFT JOIN group_role_assignments gra ON gra.group_id = gm.group_id AND gra.user_telegram_id = gm.user_telegram_id AND gra.is_active = TRUE").
		Joins("LEFT JOIN group_roles gr ON gr.group_id = gm.group_id AND gr.code = gra.role_code AND gr.is_active = TRUE").
		Where("gm.group_id = ? AND gm.is_active = TRUE", group.ID).
		Order("gm.role DESC, gm.last_seen_at DESC, gm.user_telegram_id ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Store) ListSkillsCatalog(ctx context.Context) ([]SkillCatalogItem, error) {
	var rows []SkillCatalogItem
	if err := s.db.WithContext(ctx).
		Table("skills_catalog").
		Select("code, name").
		Where("is_active = TRUE").
		Order("id ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Store) GetMemberSkillProfile(ctx context.Context, chatID int64, userTelegramID int64) (*MemberSkillProfile, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	var member struct {
		UserTelegramID int64
		Username       string
		FirstName      string
		LastName       string
		PlayerType     string
	}
	if err := s.db.WithContext(ctx).
		Table("group_members gm").
		Select("gm.user_telegram_id, COALESCE(tu.username, '') AS username, COALESCE(tu.first_name, '') AS first_name, COALESCE(tu.last_name, '') AS last_name, COALESCE(gm.player_type, '') AS player_type").
		Joins("LEFT JOIN telegram_users tu ON tu.telegram_id = gm.user_telegram_id").
		Where("gm.group_id = ? AND gm.user_telegram_id = ? AND gm.is_active = TRUE", group.ID, userTelegramID).
		Take(&member).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("member not found")
		}
		return nil, err
	}

	type row struct {
		SkillCode string
		SkillName string
		Score     *int
	}
	var rows []row
	if err := s.db.WithContext(ctx).
		Table("skills_catalog sc").
		Select("sc.code AS skill_code, sc.name AS skill_name, gms.score").
		Joins("LEFT JOIN group_member_skills gms ON gms.skill_id = sc.id AND gms.group_id = ? AND gms.user_telegram_id = ?", group.ID, userTelegramID).
		Where("sc.is_active = TRUE").
		Order("sc.id ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	skills := make([]MemberSkillValue, 0, len(rows))
	for _, r := range rows {
		skills = append(skills, MemberSkillValue{
			SkillCode: r.SkillCode,
			SkillName: r.SkillName,
			Score:     r.Score,
		})
	}

	return &MemberSkillProfile{
		UserTelegramID: member.UserTelegramID,
		Username:       member.Username,
		FirstName:      member.FirstName,
		LastName:       member.LastName,
		PlayerType:     member.PlayerType,
		Skills:         skills,
	}, nil
}

func (s *Store) UpdateMemberPlayerType(ctx context.Context, chatID int64, userTelegramID int64, playerType string) error {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return err
	}

	normalized, ok := normalizePlayerType(playerType)
	if !ok {
		return errors.New("invalid player type")
	}

	result := s.db.WithContext(ctx).
		Model(&GroupMember{}).
		Where("group_id = ? AND user_telegram_id = ? AND is_active = TRUE", group.ID, userTelegramID).
		Updates(map[string]interface{}{
			"player_type": normalized,
			"updated_at":  gorm.Expr("NOW()"),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("member not found")
	}
	return nil
}

func (s *Store) ListPlayerRelationsByUser(ctx context.Context, chatID int64, userTelegramID int64) ([]PlayerRelationView, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	type row struct {
		UserAID          int64
		UserBID          int64
		RelationType     string
		Weight           int
		RelatedUserID    int64
		RelatedUsername  string
		RelatedFirstName string
		RelatedLastName  string
	}
	var rows []row
	if err := s.db.WithContext(ctx).
		Table("player_relations pr").
		Select(`
			pr.user_a_id AS user_a_id,
			pr.user_b_id AS user_b_id,
			pr.relation_type,
			pr.weight,
			CASE WHEN pr.user_a_id = @uid THEN pr.user_b_id ELSE pr.user_a_id END AS related_user_id,
			COALESCE(tu.username, '') AS related_username,
			COALESCE(tu.first_name, '') AS related_first_name,
			COALESCE(tu.last_name, '') AS related_last_name
		`, map[string]interface{}{"uid": userTelegramID}).
		Joins("LEFT JOIN telegram_users tu ON tu.telegram_id = CASE WHEN pr.user_a_id = @uid THEN pr.user_b_id ELSE pr.user_a_id END", map[string]interface{}{"uid": userTelegramID}).
		Where("pr.group_id = ? AND pr.is_active = TRUE AND (pr.user_a_id = ? OR pr.user_b_id = ?)", group.ID, userTelegramID, userTelegramID).
		Order("pr.relation_type ASC, pr.weight DESC, related_first_name ASC, related_username ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]PlayerRelationView, 0, len(rows))
	for _, r := range rows {
		out = append(out, PlayerRelationView{
			UserAID:          r.UserAID,
			UserBID:          r.UserBID,
			RelationType:     r.RelationType,
			Weight:           r.Weight,
			RelatedUserID:    r.RelatedUserID,
			RelatedUsername:  r.RelatedUsername,
			RelatedFirstName: r.RelatedFirstName,
			RelatedLastName:  r.RelatedLastName,
		})
	}
	return out, nil
}

func (s *Store) UpsertPlayerRelation(
	ctx context.Context,
	chatID int64,
	userAID int64,
	userBID int64,
	relationType string,
	weight int,
) error {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return err
	}
	if userAID == 0 || userBID == 0 || userAID == userBID {
		return errors.New("invalid user pair")
	}
	relationType, ok := normalizeRelationType(relationType)
	if !ok {
		return errors.New("invalid relation type")
	}
	if weight < 1 || weight > 10 {
		return errors.New("weight must be between 1 and 10")
	}
	if userAID > userBID {
		userAID, userBID = userBID, userAID
	}

	record := map[string]interface{}{
		"group_id":      group.ID,
		"user_a_id":     userAID,
		"user_b_id":     userBID,
		"relation_type": relationType,
		"weight":        weight,
		"is_active":     true,
		"updated_at":    gorm.Expr("NOW()"),
	}
	return s.db.WithContext(ctx).
		Table("player_relations").
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "group_id"},
				{Name: "user_a_id"},
				{Name: "user_b_id"},
				{Name: "relation_type"},
			},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"weight":     weight,
				"is_active":  true,
				"updated_at": gorm.Expr("NOW()"),
			}),
		}).
		Create(record).Error
}

func (s *Store) DeletePlayerRelation(
	ctx context.Context,
	chatID int64,
	userAID int64,
	userBID int64,
	relationType string,
) error {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return err
	}
	if userAID == 0 || userBID == 0 || userAID == userBID {
		return errors.New("invalid user pair")
	}
	relationType, ok := normalizeRelationType(relationType)
	if !ok {
		return errors.New("invalid relation type")
	}
	if userAID > userBID {
		userAID, userBID = userBID, userAID
	}

	result := s.db.WithContext(ctx).
		Table("player_relations").
		Where("group_id = ? AND user_a_id = ? AND user_b_id = ? AND relation_type = ? AND is_active = TRUE", group.ID, userAID, userBID, relationType).
		Updates(map[string]interface{}{
			"is_active":  false,
			"updated_at": gorm.Expr("NOW()"),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("relation not found")
	}
	return nil
}

func (s *Store) UpsertMemberSkills(ctx context.Context, chatID int64, userTelegramID int64, scores map[string]int) error {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return err
	}

	var memberExists int64
	if err := s.db.WithContext(ctx).
		Table("group_members").
		Where("group_id = ? AND user_telegram_id = ? AND is_active = TRUE", group.ID, userTelegramID).
		Count(&memberExists).Error; err != nil {
		return err
	}
	if memberExists == 0 {
		return errors.New("member not found")
	}

	type skillRow struct {
		ID   uint64
		Code string
	}
	var skills []skillRow
	if err := s.db.WithContext(ctx).
		Table("skills_catalog").
		Select("id, code").
		Where("is_active = TRUE").
		Scan(&skills).Error; err != nil {
		return err
	}
	skillIDByCode := make(map[string]uint64, len(skills))
	for _, sk := range skills {
		skillIDByCode[sk.Code] = sk.ID
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for code, score := range scores {
			skillID, ok := skillIDByCode[strings.TrimSpace(code)]
			if !ok {
				return errors.New("unknown skill code: " + code)
			}
			if score < 1 || score > 10 {
				return errors.New("skill score must be between 1 and 10")
			}

			row := map[string]interface{}{
				"group_id":         group.ID,
				"user_telegram_id": userTelegramID,
				"skill_id":         skillID,
				"score":            score,
			}
			if err := tx.Table("group_member_skills").
				Clauses(clause.OnConflict{
					Columns: []clause.Column{
						{Name: "group_id"},
						{Name: "user_telegram_id"},
						{Name: "skill_id"},
					},
					DoUpdates: clause.Assignments(map[string]interface{}{
						"score":      score,
						"updated_at": gorm.Expr("NOW()"),
					}),
				}).
				Create(row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
