package postgres

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *Store) getGroupByChatID(ctx context.Context, chatID int64) (*TelegramGroup, error) {
	var group TelegramGroup
	if err := s.db.WithContext(ctx).Where("chat_id = ? AND is_active = TRUE", chatID).First(&group).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("group not found, run /setgroup first")
		}
		return nil, err
	}
	return &group, nil
}

func (s *Store) GetGroupByChatID(ctx context.Context, chatID int64) (*TelegramGroup, error) {
	return s.getGroupByChatID(ctx, chatID)
}

func (s *Store) getGroupByID(ctx context.Context, groupID uint64) (*TelegramGroup, error) {
	var group TelegramGroup
	if err := s.db.WithContext(ctx).Where("id = ? AND is_active = TRUE", groupID).First(&group).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("group not found")
		}
		return nil, err
	}
	return &group, nil
}

func (s *Store) ListActiveGroups(ctx context.Context) ([]GroupView, error) {
	var groups []GroupView
	if err := s.db.WithContext(ctx).
		Table("telegram_groups").
		Select("chat_id, title, timezone").
		Where("is_active = TRUE").
		Order("title ASC, chat_id ASC").
		Scan(&groups).Error; err != nil {
		return nil, err
	}
	return groups, nil
}

func (s *Store) ListActiveGroupsForAdmin(ctx context.Context, userTelegramID int64) ([]GroupView, error) {
	var groups []GroupView
	if err := s.db.WithContext(ctx).
		Table("telegram_groups g").
		Select("g.chat_id, g.title, g.timezone, gm.role").
		Joins("JOIN group_members gm ON gm.group_id = g.id").
		Where("g.is_active = TRUE AND gm.is_active = TRUE AND gm.user_telegram_id = ? AND gm.role = 'admin'", userTelegramID).
		Order("g.title ASC, g.chat_id ASC").
		Scan(&groups).Error; err != nil {
		return nil, err
	}
	return groups, nil
}

func (s *Store) ListActiveGroupsForUser(ctx context.Context, userTelegramID int64) ([]GroupView, error) {
	var groups []GroupView
	if err := s.db.WithContext(ctx).
		Table("telegram_groups g").
		Select("g.chat_id, g.title, g.timezone, gm.role").
		Joins("JOIN group_members gm ON gm.group_id = g.id").
		Where("g.is_active = TRUE AND gm.is_active = TRUE AND gm.status = 'active' AND gm.user_telegram_id = ?", userTelegramID).
		Order("g.title ASC, g.chat_id ASC").
		Scan(&groups).Error; err != nil {
		return nil, err
	}
	return groups, nil
}

func (s *Store) GetGroupRoleForUser(ctx context.Context, chatID int64, userTelegramID int64) (string, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return "", err
	}
	type row struct {
		Role string
	}
	var out row
	err = s.db.WithContext(ctx).
		Table("group_members").
		Select("role").
		Where("group_id = ? AND is_active = TRUE AND status = 'active' AND user_telegram_id = ?", group.ID, userTelegramID).
		Limit(1).
		Scan(&out).Error
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out.Role), nil
}

func (s *Store) UpsertGroup(ctx context.Context, chatID int64, title, timezone string) (*TelegramGroup, error) {
	group := TelegramGroup{
		ChatID:   chatID,
		Title:    title,
		Timezone: timezone,
		IsActive: true,
	}

	if err := s.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "chat_id"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"title":      title,
				"timezone":   timezone,
				"is_active":  true,
				"updated_at": gorm.Expr("NOW()"),
			}),
		}).
		Create(&group).Error; err != nil {
		return nil, err
	}

	if err := s.db.WithContext(ctx).Where("chat_id = ?", chatID).First(&group).Error; err != nil {
		return nil, err
	}

	return &group, nil
}

func (s *Store) SyncGroupAdmins(ctx context.Context, chatID int64, admins []AdminMember) error {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return err
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("UPDATE group_admins SET is_active = FALSE, updated_at = NOW() WHERE group_id = ?", group.ID).Error; err != nil {
			return err
		}

		for _, admin := range admins {
			record := map[string]interface{}{
				"group_id":   group.ID,
				"user_id":    admin.UserID,
				"username":   admin.Username,
				"first_name": admin.FirstName,
				"last_name":  admin.LastName,
				"role":       admin.Role,
				"is_active":  true,
				"synced_at":  gorm.Expr("NOW()"),
			}

			if err := tx.Table("group_admins").
				Clauses(clause.OnConflict{
					Columns: []clause.Column{
						{Name: "group_id"},
						{Name: "user_id"},
					},
					DoUpdates: clause.Assignments(map[string]interface{}{
						"username":   admin.Username,
						"first_name": admin.FirstName,
						"last_name":  admin.LastName,
						"role":       admin.Role,
						"is_active":  true,
						"synced_at":  gorm.Expr("NOW()"),
						"updated_at": gorm.Expr("NOW()"),
					}),
				}).
				Create(record).Error; err != nil {
				return err
			}

			if err := upsertTelegramUserTx(tx, admin.UserID, admin.Username, admin.FirstName, admin.LastName); err != nil {
				return err
			}
			if err := upsertGroupMemberTx(tx, group.ID, admin.UserID, "admin", "active"); err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *Store) ListGroupsForAdmin(ctx context.Context, userID int64) ([]AdminGroup, error) {
	type row struct {
		ChatID int64
		Title  string
	}
	var rows []row
	if err := s.db.WithContext(ctx).
		Table("group_admins ga").
		Select("g.chat_id, g.title").
		Joins("JOIN telegram_groups g ON g.id = ga.group_id").
		Where("ga.user_id = ? AND ga.is_active = TRUE AND g.is_active = TRUE", userID).
		Order("g.title ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	groups := make([]AdminGroup, 0, len(rows))
	for _, row := range rows {
		groups = append(groups, AdminGroup{
			ChatID: row.ChatID,
			Title:  row.Title,
		})
	}
	return groups, nil
}
