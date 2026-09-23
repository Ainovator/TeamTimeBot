package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *Store) EnsureDefaultGroupRoles(ctx context.Context, chatID int64) error {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return err
	}

	type roleSeed struct {
		Code        string
		Title       string
		Permissions map[string]bool
	}
	seeds := []roleSeed{
		{
			Code:  "trainer",
			Title: "Тренер",
			Permissions: map[string]bool{
				"members_read":           true,
				"members_write":          true,
				"templates_manage":       true,
				"event_templates_manage": true,
				"events_read":            true,
				"events_manage":          true,
				"polls_read":             true,
				"roles_manage":           true,
				"profile_read":           true,
			},
		},
		{
			Code:  "captain",
			Title: "Капитан",
			Permissions: map[string]bool{
				"events_read":   true,
				"events_manage": true,
				"polls_read":    true,
				"profile_read":  true,
			},
		},
	}

	for _, seed := range seeds {
		payload, err := json.Marshal(seed.Permissions)
		if err != nil {
			return err
		}
		record := GroupRole{
			GroupID:     group.ID,
			Code:        seed.Code,
			Title:       seed.Title,
			Permissions: datatypes.JSON(payload),
			IsActive:    true,
		}
		if err := s.db.WithContext(ctx).
			Clauses(clause.OnConflict{
				Columns: []clause.Column{
					{Name: "group_id"},
					{Name: "code"},
				},
				DoUpdates: clause.Assignments(map[string]interface{}{
					"title":       seed.Title,
					"permissions": datatypes.JSON(payload),
					"is_active":   true,
					"updated_at":  gorm.Expr("NOW()"),
				}),
			}).
			Create(&record).Error; err != nil {
			return err
		}
	}

	return nil
}

func (s *Store) ListGroupRoles(ctx context.Context, chatID int64) ([]GroupRoleView, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}
	if err := s.EnsureDefaultGroupRoles(ctx, chatID); err != nil {
		return nil, err
	}

	type row struct {
		Code        string
		Title       string
		Permissions datatypes.JSON
	}
	var rows []row
	if err := s.db.WithContext(ctx).
		Table("group_roles").
		Select("code, title, permissions").
		Where("group_id = ? AND is_active = TRUE", group.ID).
		Order("code ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]GroupRoleView, 0, len(rows))
	for _, r := range rows {
		perms := map[string]bool{}
		_ = json.Unmarshal(r.Permissions, &perms)
		out = append(out, GroupRoleView{
			Code:        strings.TrimSpace(r.Code),
			Title:       strings.TrimSpace(r.Title),
			Permissions: perms,
		})
	}
	return out, nil
}

func defaultMemberPermissions() map[string]bool {
	return map[string]bool{
		"events_read":  true,
		"polls_read":   true,
		"profile_read": true,
	}
}

func mergePermissions(dst map[string]bool, src map[string]bool) map[string]bool {
	if dst == nil {
		dst = map[string]bool{}
	}
	for k, v := range src {
		if v {
			dst[k] = true
		}
	}
	return dst
}

func (s *Store) GetGroupPermissionsForUser(ctx context.Context, chatID int64, userTelegramID int64) (*GroupPermissionsView, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}
	if err := s.EnsureDefaultGroupRoles(ctx, chatID); err != nil {
		return nil, err
	}

	telegramRole, err := s.GetGroupRoleForUser(ctx, chatID, userTelegramID)
	if err != nil {
		return nil, err
	}
	if telegramRole == "" {
		return nil, errors.New("user is not a member of this group")
	}

	// Telegram admins always have full access.
	if strings.EqualFold(strings.TrimSpace(telegramRole), "admin") {
		return &GroupPermissionsView{
			RoleCode:  "admin",
			RoleTitle: "Администратор",
			Permissions: map[string]bool{
				"members_read":           true,
				"members_write":          true,
				"templates_manage":       true,
				"event_templates_manage": true,
				"events_read":            true,
				"events_manage":          true,
				"billing_manage":         true,
				"polls_read":             true,
				"roles_manage":           true,
				"profile_read":           true,
			},
		}, nil
	}

	type row struct {
		RoleCode    string
		RoleTitle   string
		Permissions datatypes.JSON
	}
	var r row
	err = s.db.WithContext(ctx).
		Table("group_role_assignments gra").
		Select("gra.role_code, COALESCE(gr.title, '') AS role_title, COALESCE(gr.permissions, '{}'::jsonb) AS permissions").
		Joins("JOIN group_roles gr ON gr.group_id = gra.group_id AND gr.code = gra.role_code AND gr.is_active = TRUE").
		Where("gra.group_id = ? AND gra.user_telegram_id = ? AND gra.is_active = TRUE", group.ID, userTelegramID).
		Limit(1).
		Scan(&r).Error
	if err != nil {
		return nil, err
	}

	perms := defaultMemberPermissions()
	roleCode := "member"
	roleTitle := "Участник"
	if strings.TrimSpace(r.RoleCode) != "" {
		roleCode = strings.TrimSpace(r.RoleCode)
		roleTitle = strings.TrimSpace(r.RoleTitle)
		custom := map[string]bool{}
		_ = json.Unmarshal(r.Permissions, &custom)
		perms = mergePermissions(perms, custom)
	}

	return &GroupPermissionsView{
		RoleCode:    roleCode,
		RoleTitle:   roleTitle,
		Permissions: perms,
	}, nil
}

func (s *Store) AssignGroupRole(ctx context.Context, chatID int64, userTelegramID int64, roleCode string) error {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return err
	}
	if err := s.EnsureDefaultGroupRoles(ctx, chatID); err != nil {
		return err
	}
	roleCode = strings.TrimSpace(strings.ToLower(roleCode))

	// prevent overriding telegram admins with app role
	telegramRole, err := s.GetGroupRoleForUser(ctx, chatID, userTelegramID)
	if err != nil {
		return err
	}
	if strings.EqualFold(strings.TrimSpace(telegramRole), "admin") {
		return errors.New("telegram admins do not need an additional role")
	}

	// Allow clearing assignment back to default "member".
	if roleCode == "" || roleCode == "member" || roleCode == "none" {
		return s.db.WithContext(ctx).
			Model(&GroupRoleAssignment{}).
			Where("group_id = ? AND user_telegram_id = ? AND is_active = TRUE", group.ID, userTelegramID).
			Updates(map[string]interface{}{
				"is_active":  false,
				"updated_at": gorm.Expr("NOW()"),
			}).Error
	}

	// ensure role exists
	var role GroupRole
	if err := s.db.WithContext(ctx).
		Where("group_id = ? AND code = ? AND is_active = TRUE", group.ID, roleCode).
		First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("role not found")
		}
		return err
	}

	record := GroupRoleAssignment{
		GroupID:        group.ID,
		UserTelegramID: userTelegramID,
		RoleCode:       roleCode,
		IsActive:       true,
	}

	return s.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "group_id"},
				{Name: "user_telegram_id"},
			},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"role_code":  roleCode,
				"is_active":  true,
				"updated_at": gorm.Expr("NOW()"),
			}),
		}).
		Create(&record).Error
}
