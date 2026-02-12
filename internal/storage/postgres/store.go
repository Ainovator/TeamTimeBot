package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Store struct {
	db *gorm.DB
}

func New(dsn string) (*Store, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	return &Store{db: db}, nil
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

func (s *Store) UpsertPollTemplate(
	ctx context.Context,
	chatID int64,
	name, question string,
	options []string,
) (*PollTemplate, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	payload, err := json.Marshal(options)
	if err != nil {
		return nil, err
	}

	template := PollTemplate{
		GroupID:  group.ID,
		Name:     name,
		Question: question,
		Options:  datatypes.JSON(payload),
		IsActive: true,
	}

	if err := s.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "group_id"},
				{Name: "name"},
			},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"question":   question,
				"options":    datatypes.JSON(payload),
				"is_active":  true,
				"updated_at": gorm.Expr("NOW()"),
			}),
		}).
		Create(&template).Error; err != nil {
		return nil, err
	}

	if err := s.db.WithContext(ctx).
		Where("group_id = ? AND name = ?", group.ID, name).
		First(&template).Error; err != nil {
		return nil, err
	}

	return &template, nil
}

func (s *Store) UpsertSchedule(
	ctx context.Context,
	chatID int64,
	templateName, cronExpr string,
) (*PollSchedule, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	var template PollTemplate
	if err := s.db.WithContext(ctx).
		Where("group_id = ? AND name = ? AND is_active = TRUE", group.ID, templateName).
		First(&template).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("poll template not found")
		}
		return nil, err
	}

	var schedule PollSchedule
	err = s.db.WithContext(ctx).
		Where("group_id = ? AND template_id = ? AND is_active = TRUE", group.ID, template.ID).
		Order("id DESC").
		First(&schedule).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		schedule = PollSchedule{
			GroupID:    group.ID,
			TemplateID: template.ID,
			CronExpr:   cronExpr,
			IsActive:   true,
		}

		if createErr := s.db.WithContext(ctx).Create(&schedule).Error; createErr != nil {
			return nil, createErr
		}
		return &schedule, nil
	}
	if err != nil {
		return nil, err
	}

	schedule.CronExpr = cronExpr
	schedule.IsActive = true
	if err := s.db.WithContext(ctx).Save(&schedule).Error; err != nil {
		return nil, err
	}

	return &schedule, nil
}

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

func ParsePollSpec(payload string) (name, question string, options []string, err error) {
	parts := strings.Split(payload, "|")
	if len(parts) != 3 {
		return "", "", nil, errors.New("format: /setpoll name|question|option1,option2")
	}

	name = strings.TrimSpace(parts[0])
	question = strings.TrimSpace(parts[1])
	rawOptions := strings.Split(parts[2], ",")
	for _, option := range rawOptions {
		option = strings.TrimSpace(option)
		if option != "" {
			options = append(options, option)
		}
	}

	if name == "" || question == "" || len(options) < 2 {
		return "", "", nil, errors.New("name, question and at least 2 options are required")
	}

	return name, question, options, nil
}

func ParseScheduleSpec(payload string) (templateName, cronExpr string, err error) {
	parts := strings.Split(payload, "|")
	if len(parts) != 2 {
		return "", "", errors.New("format: /setschedule template_name|cron_expr")
	}

	templateName = strings.TrimSpace(parts[0])
	cronExpr = strings.TrimSpace(parts[1])
	if templateName == "" || cronExpr == "" {
		return "", "", errors.New("template_name and cron_expr are required")
	}

	return templateName, cronExpr, nil
}
