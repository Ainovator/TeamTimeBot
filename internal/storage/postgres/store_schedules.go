package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func (s *Store) UpsertSchedule(
	ctx context.Context,
	chatID int64,
	templateName, scheduleExpr string,
) (*PollSchedule, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	nextRunAt, err := ComputeNextRunAt(group.Timezone, scheduleExpr, time.Now().UTC())
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
			CronExpr:   scheduleExpr,
			NextRunAt:  &nextRunAt,
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

	schedule.CronExpr = scheduleExpr
	schedule.NextRunAt = &nextRunAt
	schedule.IsActive = true
	if err := s.db.WithContext(ctx).Save(&schedule).Error; err != nil {
		return nil, err
	}

	return &schedule, nil
}

func (s *Store) CreateSchedule(
	ctx context.Context,
	chatID int64,
	templateName, scheduleExpr string,
) (*PollSchedule, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	nextRunAt, err := ComputeNextRunAt(group.Timezone, scheduleExpr, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	var template PollTemplate
	if err := s.db.WithContext(ctx).
		Where("group_id = ? AND name = ? AND is_active = TRUE", group.ID, strings.TrimSpace(templateName)).
		First(&template).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("poll template not found")
		}
		return nil, err
	}

	schedule := PollSchedule{
		GroupID:    group.ID,
		TemplateID: template.ID,
		CronExpr:   scheduleExpr,
		NextRunAt:  &nextRunAt,
		IsActive:   true,
	}
	if err := s.db.WithContext(ctx).Create(&schedule).Error; err != nil {
		return nil, err
	}

	return &schedule, nil
}

func (s *Store) ListDueSchedules(ctx context.Context, nowUTC time.Time, limit int) ([]DueSchedule, error) {
	if limit <= 0 {
		limit = 100
	}

	type scheduleRow struct {
		ScheduleID   uint64
		ChatID       int64
		Timezone     string
		TemplateName string
		ScheduleExpr string
		Question     string
		Options      datatypes.JSON
	}

	var rows []scheduleRow
	if err := s.db.WithContext(ctx).
		Table("poll_schedules ps").
		Select("ps.id AS schedule_id, g.chat_id, g.timezone, t.name AS template_name, ps.cron_expr AS schedule_expr, t.question, t.options").
		Joins("JOIN telegram_groups g ON g.id = ps.group_id").
		Joins("JOIN poll_templates t ON t.id = ps.template_id").
		Where("ps.is_active = TRUE AND g.is_active = TRUE AND t.is_active = TRUE AND ps.next_run_at IS NOT NULL AND ps.next_run_at <= ?", nowUTC).
		Order("ps.next_run_at ASC").
		Limit(limit).
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]DueSchedule, 0, len(rows))
	for _, row := range rows {
		var options []string
		if err := json.Unmarshal(row.Options, &options); err != nil {
			return nil, err
		}

		result = append(result, DueSchedule{
			ScheduleID:   row.ScheduleID,
			ChatID:       row.ChatID,
			Timezone:     row.Timezone,
			TemplateName: row.TemplateName,
			ScheduleExpr: row.ScheduleExpr,
			Question:     row.Question,
			Options:      options,
		})
	}

	return result, nil
}

func (s *Store) SetNextRunAt(ctx context.Context, scheduleID uint64, nextRunAt time.Time) error {
	return s.db.WithContext(ctx).
		Model(&PollSchedule{}).
		Where("id = ?", scheduleID).
		Updates(map[string]interface{}{
			"next_run_at": nextRunAt,
			"updated_at":  gorm.Expr("NOW()"),
		}).Error
}

func (s *Store) ListSchedulesByTemplate(ctx context.Context, chatID int64, templateName string) ([]ScheduleView, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	var schedules []ScheduleView
	if err := s.db.WithContext(ctx).
		Table("poll_schedules ps").
		Select("ps.id, t.name AS template_name, ps.cron_expr AS send_at, ps.is_active").
		Joins("JOIN poll_templates t ON t.id = ps.template_id").
		Where("ps.group_id = ? AND t.name = ? AND ps.is_active = TRUE AND t.is_active = TRUE", group.ID, templateName).
		Order("ps.id DESC").
		Scan(&schedules).Error; err != nil {
		return nil, err
	}

	for i := range schedules {
		schedules[i].SendAt = FormatScheduleExprForDisplay(schedules[i].SendAt)
	}
	return schedules, nil
}

type ScheduleDetails struct {
	ID         uint64
	GroupID    uint64
	TemplateID uint64
	Template   string
	Schedule   string
	IsActive   bool
}

func (s *Store) GetScheduleDetails(ctx context.Context, chatID int64, scheduleID uint64) (*ScheduleDetails, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	var row ScheduleDetails
	if err := s.db.WithContext(ctx).
		Table("poll_schedules ps").
		Select("ps.id, ps.group_id, ps.template_id, t.name AS template, ps.cron_expr AS schedule, ps.is_active").
		Joins("JOIN poll_templates t ON t.id = ps.template_id").
		Where("ps.id = ? AND ps.group_id = ?", scheduleID, group.ID).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("schedule not found")
		}
		return nil, err
	}

	return &row, nil
}

func (s *Store) UpdateScheduleExpr(ctx context.Context, chatID int64, scheduleID uint64, scheduleExpr string) error {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return err
	}

	nextRunAt, err := ComputeNextRunAt(group.Timezone, scheduleExpr, time.Now().UTC())
	if err != nil {
		return err
	}

	return s.db.WithContext(ctx).
		Model(&PollSchedule{}).
		Where("id = ? AND group_id = ?", scheduleID, group.ID).
		Updates(map[string]interface{}{
			"cron_expr":   scheduleExpr,
			"next_run_at": nextRunAt,
			"is_active":   true,
			"updated_at":  gorm.Expr("NOW()"),
		}).Error
}

func (s *Store) DeleteSchedule(ctx context.Context, chatID int64, scheduleID uint64) error {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return err
	}

	return s.db.WithContext(ctx).
		Model(&PollSchedule{}).
		Where("id = ? AND group_id = ?", scheduleID, group.ID).
		Updates(map[string]interface{}{
			"is_active":  false,
			"updated_at": gorm.Expr("NOW()"),
		}).Error
}

func (s *Store) SetScheduleActive(ctx context.Context, chatID int64, scheduleID uint64, isActive bool) error {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return err
	}

	updates := map[string]interface{}{
		"is_active":  isActive,
		"updated_at": gorm.Expr("NOW()"),
	}
	if isActive {
		var schedule PollSchedule
		if err := s.db.WithContext(ctx).
			Where("id = ? AND group_id = ?", scheduleID, group.ID).
			First(&schedule).Error; err != nil {
			return err
		}
		nextRunAt, err := ComputeNextRunAt(group.Timezone, schedule.CronExpr, time.Now().UTC())
		if err != nil {
			return err
		}
		updates["next_run_at"] = nextRunAt
	}

	return s.db.WithContext(ctx).
		Model(&PollSchedule{}).
		Where("id = ? AND group_id = ?", scheduleID, group.ID).
		Updates(updates).Error
}
