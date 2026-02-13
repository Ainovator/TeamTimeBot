package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Store struct {
	db *gorm.DB
}

type DueSchedule struct {
	ScheduleID   uint64
	ChatID       int64
	Timezone     string
	TemplateName string
	ScheduleExpr string
	Question     string
	Options      []string
}

type TemplateView struct {
	Name     string
	Question string
}

type ScheduleView struct {
	ID           uint64
	TemplateName string
	SendAt       string
	IsActive     bool
}

type AdminMember struct {
	UserID    int64
	Username  string
	FirstName string
	LastName  string
	Role      string
}

type AdminGroup struct {
	ChatID int64
	Title  string
}

type TemplateDetails struct {
	Name     string
	Question string
	Options  []string
}

type EventView struct {
	ID              uint64
	Name            string
	StartWeekday    int
	PollPublishTime string
	StartTime       string
	EndTime         string
	CostAmount      *float64
	IsActive        bool
	PollTemplate    string
}

type EventPollPostView struct {
	ID                uint64
	GroupID           uint64
	EventID           *uint64
	TemplateID        uint64
	TemplateName      string
	TelegramMessageID int64
	TelegramPollID    string
	Status            string
	PublishedAt       time.Time
}

type EventPollVoteView struct {
	ID        uint64
	PostID    uint64
	UserID    int64
	Username  string
	FirstName string
	LastName  string
	Choice    string
	Source    string
	VotedAt   time.Time
}

type EventWithGroupView struct {
	EventID         uint64
	GroupID         uint64
	ChatID          int64
	Timezone        string
	Name            string
	StartWeekday    int
	PollPublishTime string
	StartTime       string
	EndTime         string
	CostAmount      *float64
	PollTemplate    string
}

type EventTemplateDetails struct {
	EventID          uint64
	EventName        string
	TemplateName     string
	TemplateQuestion string
	TemplateOptions  []string
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
		Where("group_id = ? AND name = ? AND is_active = TRUE", group.ID, templateName).
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
	if len(parts) != 2 && len(parts) != 3 {
		return "", "", errors.New("format: /setschedule template_name|HH:MM OR /setschedule template_name|1,3,5|HH:MM")
	}

	templateName = strings.TrimSpace(parts[0])
	if templateName == "" {
		return "", "", errors.New("template_name is required")
	}

	if len(parts) == 2 {
		sendAt := strings.TrimSpace(parts[1])
		if sendAt == "" {
			return "", "", errors.New("HH:MM is required")
		}
		cronExpr, err = BuildScheduleExpr([]int{1, 2, 3, 4, 5, 6, 7}, sendAt)
		return templateName, cronExpr, err
	}

	weekdaysCSV := strings.TrimSpace(parts[1])
	sendAt := strings.TrimSpace(parts[2])
	if weekdaysCSV == "" || sendAt == "" {
		return "", "", errors.New("weekdays and HH:MM are required")
	}

	weekdays, err := ParseWeekdaysCSV(weekdaysCSV)
	if err != nil {
		return "", "", err
	}

	cronExpr, err = BuildScheduleExpr(weekdays, sendAt)
	return templateName, cronExpr, err
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

func (s *Store) DeleteTemplateByName(ctx context.Context, chatID int64, templateName string) error {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return err
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var template PollTemplate
		if err := tx.
			Where("group_id = ? AND name = ? AND is_active = TRUE", group.ID, templateName).
			First(&template).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("template not found")
			}
			return err
		}

		if err := tx.Model(&PollTemplate{}).
			Where("id = ?", template.ID).
			Updates(map[string]interface{}{
				"is_active":  false,
				"updated_at": gorm.Expr("NOW()"),
			}).Error; err != nil {
			return err
		}

		if err := tx.Model(&PollSchedule{}).
			Where("template_id = ?", template.ID).
			Updates(map[string]interface{}{
				"is_active":  false,
				"updated_at": gorm.Expr("NOW()"),
			}).Error; err != nil {
			return err
		}

		return nil
	})
}

func (s *Store) GetGroupSnapshot(ctx context.Context, chatID int64) ([]TemplateView, []ScheduleView, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, nil, err
	}

	var templates []TemplateView
	if err := s.db.WithContext(ctx).
		Table("poll_templates").
		Select("name, question").
		Where("group_id = ? AND is_active = TRUE", group.ID).
		Order("name ASC").
		Scan(&templates).Error; err != nil {
		return nil, nil, err
	}

	var schedules []ScheduleView
	if err := s.db.WithContext(ctx).
		Table("poll_schedules ps").
		Select("t.name AS template_name, ps.cron_expr AS send_at").
		Joins("JOIN poll_templates t ON t.id = ps.template_id").
		Where("ps.group_id = ? AND ps.is_active = TRUE AND t.is_active = TRUE", group.ID).
		Order("t.name ASC").
		Scan(&schedules).Error; err != nil {
		return nil, nil, err
	}

	for i := range schedules {
		schedules[i].SendAt = FormatScheduleExprForDisplay(schedules[i].SendAt)
	}

	return templates, schedules, nil
}

func (s *Store) ListTemplateNamesByChatID(ctx context.Context, chatID int64) ([]string, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	type row struct {
		Name string
	}
	var rows []row
	if err := s.db.WithContext(ctx).
		Table("poll_templates").
		Select("name").
		Where("group_id = ? AND is_active = TRUE", group.ID).
		Order("name ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]string, 0, len(rows))
	for _, r := range rows {
		result = append(result, r.Name)
	}
	return result, nil
}

func (s *Store) GetTemplateByName(ctx context.Context, chatID int64, name string) (*TemplateDetails, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	var row struct {
		Name     string
		Question string
		Options  datatypes.JSON
	}
	if err := s.db.WithContext(ctx).
		Table("poll_templates").
		Select("name, question, options").
		Where("group_id = ? AND name = ? AND is_active = TRUE", group.ID, name).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("template not found")
		}
		return nil, err
	}

	var options []string
	if err := json.Unmarshal(row.Options, &options); err != nil {
		return nil, err
	}

	return &TemplateDetails{
		Name:     row.Name,
		Question: row.Question,
		Options:  options,
	}, nil
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

func (s *Store) CreateEvent(ctx context.Context, chatID int64, name string, startWeekday int, pollPublishTime, startTime, endTime string, costAmount *float64) (*GroupEvent, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(name) == "" {
		return nil, errors.New("event name is required")
	}
	if startWeekday < 1 || startWeekday > 7 {
		return nil, errors.New("weekday must be between 1 and 7")
	}
	if _, err := time.Parse("15:04", startTime); err != nil {
		return nil, errors.New("invalid start time format, use HH:MM")
	}
	if _, err := time.Parse("15:04", endTime); err != nil {
		return nil, errors.New("invalid end time format, use HH:MM")
	}
	if _, err := time.Parse("15:04", pollPublishTime); err != nil {
		return nil, errors.New("invalid publish time format, use HH:MM")
	}
	if costAmount != nil && *costAmount < 0 {
		return nil, errors.New("cost amount must be >= 0")
	}

	event := GroupEvent{
		GroupID:         group.ID,
		Name:            strings.TrimSpace(name),
		StartWeekday:    int16(startWeekday),
		PollPublishTime: pollPublishTime,
		StartTime:       startTime,
		EndTime:         endTime,
		CostAmount:      costAmount,
		IsActive:        true,
	}
	if err := s.db.WithContext(ctx).Create(&event).Error; err != nil {
		return nil, err
	}
	return &event, nil
}

func (s *Store) ListEventsByChatID(ctx context.Context, chatID int64) ([]EventView, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	var rows []EventView
	if err := s.db.WithContext(ctx).
		Table("group_events ge").
		Select("ge.id, ge.name, ge.start_weekday, ge.poll_publish_time, ge.start_time, ge.end_time, ge.cost_amount, ge.is_active, COALESCE(pt.name, '') AS poll_template").
		Joins("LEFT JOIN poll_templates pt ON pt.id = ge.poll_template_id").
		Where("ge.group_id = ? AND ge.is_active = TRUE", group.ID).
		Order("id DESC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Store) BindEventToTemplate(ctx context.Context, chatID int64, eventID uint64, templateName string) error {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(templateName) == "" {
		return errors.New("template name is required")
	}

	var template PollTemplate
	if err := s.db.WithContext(ctx).
		Where("group_id = ? AND name = ? AND is_active = TRUE", group.ID, strings.TrimSpace(templateName)).
		First(&template).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("template not found")
		}
		return err
	}

	result := s.db.WithContext(ctx).
		Model(&GroupEvent{}).
		Where("id = ? AND group_id = ? AND is_active = TRUE", eventID, group.ID).
		Updates(map[string]interface{}{
			"poll_template_id": template.ID,
			"updated_at":       gorm.Expr("NOW()"),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("event not found")
	}
	return nil
}

func (s *Store) CreateEventPollPost(
	ctx context.Context,
	chatID int64,
	eventID *uint64,
	templateName string,
	telegramMessageID int64,
	telegramPollID string,
) (*EventPollPost, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(templateName) == "" {
		return nil, errors.New("template name is required")
	}
	if telegramMessageID == 0 {
		return nil, errors.New("telegram message id is required")
	}

	var template PollTemplate
	if err := s.db.WithContext(ctx).
		Where("group_id = ? AND name = ? AND is_active = TRUE", group.ID, strings.TrimSpace(templateName)).
		First(&template).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("template not found")
		}
		return nil, err
	}

	if eventID != nil {
		var event GroupEvent
		if err := s.db.WithContext(ctx).
			Where("id = ? AND group_id = ? AND is_active = TRUE", *eventID, group.ID).
			First(&event).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.New("event not found")
			}
			return nil, err
		}
	}

	post := EventPollPost{
		GroupID:           group.ID,
		EventID:           eventID,
		TemplateID:        template.ID,
		TelegramMessageID: telegramMessageID,
		TelegramPollID:    strings.TrimSpace(telegramPollID),
		Status:            "open",
		PublishedAt:       time.Now().UTC(),
	}
	if err := s.db.WithContext(ctx).Create(&post).Error; err != nil {
		return nil, err
	}
	return &post, nil
}

func (s *Store) UpsertEventPollVote(
	ctx context.Context,
	postID uint64,
	userID int64,
	username, firstName, lastName, choice, source string,
	votedAt time.Time,
) error {
	if postID == 0 || userID == 0 {
		return errors.New("post_id and user_id are required")
	}
	choice = strings.TrimSpace(choice)
	if choice == "" {
		return errors.New("choice is required")
	}
	source = strings.TrimSpace(source)
	if source == "" {
		source = "inline"
	}
	if votedAt.IsZero() {
		votedAt = time.Now().UTC()
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := upsertTelegramUserTx(tx, userID, username, firstName, lastName); err != nil {
			return err
		}
		var post struct {
			GroupID uint64
		}
		if err := tx.Table("event_poll_posts").
			Select("group_id").
			Where("id = ?", postID).
			First(&post).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("poll post not found")
			}
			return err
		}
		if err := upsertGroupMemberTx(tx, post.GroupID, userID, "member", "active"); err != nil {
			return err
		}

		vote := EventPollVote{
			PostID:    postID,
			UserID:    userID,
			Username:  username,
			FirstName: firstName,
			LastName:  lastName,
			Choice:    choice,
			Source:    source,
			VotedAt:   votedAt.UTC(),
		}
		return tx.
			Clauses(clause.OnConflict{
				Columns: []clause.Column{
					{Name: "post_id"},
					{Name: "user_id"},
				},
				DoUpdates: clause.Assignments(map[string]interface{}{
					"username":   vote.Username,
					"first_name": vote.FirstName,
					"last_name":  vote.LastName,
					"choice":     vote.Choice,
					"source":     vote.Source,
					"voted_at":   vote.VotedAt,
					"updated_at": gorm.Expr("NOW()"),
				}),
			}).
			Create(&vote).Error
	})
}

func (s *Store) ListEventPollVotes(ctx context.Context, postID uint64) ([]EventPollVoteView, error) {
	if postID == 0 {
		return nil, errors.New("post_id is required")
	}
	var rows []EventPollVoteView
	if err := s.db.WithContext(ctx).
		Table("event_poll_votes").
		Select("id, post_id, user_id, username, first_name, last_name, choice, source, voted_at").
		Where("post_id = ?", postID).
		Order("voted_at ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Store) GetEventPollPostByTelegramPollID(ctx context.Context, telegramPollID string) (*EventPollPostView, error) {
	telegramPollID = strings.TrimSpace(telegramPollID)
	if telegramPollID == "" {
		return nil, errors.New("telegram poll id is required")
	}
	var row EventPollPostView
	if err := s.db.WithContext(ctx).
		Table("event_poll_posts epp").
		Select("epp.id, epp.group_id, epp.event_id, epp.template_id, pt.name AS template_name, epp.telegram_message_id, epp.telegram_poll_id, epp.status, epp.published_at").
		Joins("JOIN poll_templates pt ON pt.id = epp.template_id").
		Where("epp.telegram_poll_id = ?", telegramPollID).
		Order("epp.id DESC").
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("poll post not found")
		}
		return nil, err
	}
	return &row, nil
}

func (s *Store) FindBoundEventIDByTemplateAndWeekday(ctx context.Context, chatID int64, templateName string, weekday int) (*uint64, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(templateName) == "" {
		return nil, errors.New("template name is required")
	}

	type row struct {
		ID uint64
	}
	var rows []row
	if err := s.db.WithContext(ctx).
		Table("group_events ge").
		Select("ge.id").
		Joins("JOIN poll_templates pt ON pt.id = ge.poll_template_id").
		Where("ge.group_id = ? AND ge.is_active = TRUE AND ge.start_weekday = ? AND pt.name = ? AND pt.is_active = TRUE", group.ID, weekday, strings.TrimSpace(templateName)).
		Order("ge.id DESC").
		Limit(1).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	id := rows[0].ID
	return &id, nil
}

func (s *Store) ListActiveEventsWithGroups(ctx context.Context) ([]EventWithGroupView, error) {
	var rows []EventWithGroupView
	if err := s.db.WithContext(ctx).
		Table("group_events ge").
		Select("ge.id AS event_id, ge.group_id, g.chat_id, g.timezone, ge.name, ge.start_weekday, ge.poll_publish_time, ge.start_time, ge.end_time, ge.cost_amount, COALESCE(pt.name, '') AS poll_template").
		Joins("JOIN telegram_groups g ON g.id = ge.group_id").
		Joins("LEFT JOIN poll_templates pt ON pt.id = ge.poll_template_id").
		Where("ge.is_active = TRUE AND g.is_active = TRUE").
		Order("ge.id ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Store) HasEventSettlement(ctx context.Context, eventID uint64, localDate time.Time) (bool, error) {
	var count int64
	if err := s.db.WithContext(ctx).
		Table("event_settlements").
		Where("event_id = ? AND local_date = ?", eventID, localDate.Format("2006-01-02")).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *Store) GetLatestEventPollPostForRange(ctx context.Context, eventID uint64, fromUTC, toUTC time.Time) (*EventPollPostView, error) {
	var row EventPollPostView
	if err := s.db.WithContext(ctx).
		Table("event_poll_posts epp").
		Select("epp.id, epp.group_id, epp.event_id, epp.template_id, pt.name AS template_name, epp.telegram_message_id, epp.telegram_poll_id, epp.status, epp.published_at").
		Joins("JOIN poll_templates pt ON pt.id = epp.template_id").
		Where("epp.event_id = ? AND epp.published_at >= ? AND epp.published_at < ?", eventID, fromUTC, toUTC).
		Order("epp.id DESC").
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

func (s *Store) GetLatestEventPollPost(ctx context.Context, eventID uint64) (*EventPollPostView, error) {
	var row EventPollPostView
	if err := s.db.WithContext(ctx).
		Table("event_poll_posts epp").
		Select("epp.id, epp.group_id, epp.event_id, epp.template_id, pt.name AS template_name, epp.telegram_message_id, epp.telegram_poll_id, epp.status, epp.published_at").
		Joins("JOIN poll_templates pt ON pt.id = epp.template_id").
		Where("epp.event_id = ?", eventID).
		Order("epp.id DESC").
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

func (s *Store) CountVotesForPostChoice(ctx context.Context, postID uint64, choice string) (int, error) {
	var count int64
	if err := s.db.WithContext(ctx).
		Table("event_poll_votes").
		Where("post_id = ? AND choice = ?", postID, strings.TrimSpace(choice)).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (s *Store) CreateEventSettlement(
	ctx context.Context,
	groupID uint64,
	eventID uint64,
	postID *uint64,
	localDate time.Time,
	totalAmount float64,
	participants int,
	amountPerPerson float64,
) error {
	record := map[string]interface{}{
		"group_id":           groupID,
		"event_id":           eventID,
		"post_id":            postID,
		"local_date":         localDate.Format("2006-01-02"),
		"total_amount":       totalAmount,
		"participants_count": participants,
		"amount_per_person":  amountPerPerson,
		"sent_at":            gorm.Expr("NOW()"),
	}
	return s.db.WithContext(ctx).Table("event_settlements").Create(record).Error
}

func (s *Store) GetEventTemplateDetails(ctx context.Context, chatID int64, eventID uint64) (*EventTemplateDetails, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}
	var row struct {
		EventID          uint64
		EventName        string
		TemplateName     string
		TemplateQuestion string
		TemplateOptions  datatypes.JSON
	}
	if err := s.db.WithContext(ctx).
		Table("group_events ge").
		Select("ge.id AS event_id, ge.name AS event_name, pt.name AS template_name, pt.question AS template_question, pt.options AS template_options").
		Joins("JOIN poll_templates pt ON pt.id = ge.poll_template_id").
		Where("ge.id = ? AND ge.group_id = ? AND ge.is_active = TRUE AND pt.is_active = TRUE", eventID, group.ID).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("event or bound template not found")
		}
		return nil, err
	}
	var options []string
	if err := json.Unmarshal(row.TemplateOptions, &options); err != nil {
		return nil, err
	}
	return &EventTemplateDetails{
		EventID:          row.EventID,
		EventName:        row.EventName,
		TemplateName:     row.TemplateName,
		TemplateQuestion: row.TemplateQuestion,
		TemplateOptions:  options,
	}, nil
}

func (s *Store) ReplaceEventCountedOptions(ctx context.Context, chatID int64, eventID uint64, optionIndexes []int) error {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var exists int64
		if err := tx.Table("group_events").
			Where("id = ? AND group_id = ? AND is_active = TRUE", eventID, group.ID).
			Count(&exists).Error; err != nil {
			return err
		}
		if exists == 0 {
			return errors.New("event not found")
		}
		if err := tx.Exec("DELETE FROM event_counted_options WHERE event_id = ?", eventID).Error; err != nil {
			return err
		}
		for _, idx := range optionIndexes {
			if idx < 0 {
				continue
			}
			row := map[string]interface{}{
				"event_id":     eventID,
				"option_index": idx,
				"created_at":   gorm.Expr("NOW()"),
				"updated_at":   gorm.Expr("NOW()"),
			}
			if err := tx.Table("event_counted_options").Create(row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) GetEventCountedOptions(ctx context.Context, eventID uint64) ([]int, error) {
	type row struct {
		OptionIndex int
	}
	var rows []row
	if err := s.db.WithContext(ctx).
		Table("event_counted_options").
		Select("option_index").
		Where("event_id = ?", eventID).
		Order("option_index ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	res := make([]int, 0, len(rows))
	for _, r := range rows {
		res = append(res, r.OptionIndex)
	}
	return res, nil
}

func (s *Store) CountVotesForPostChoices(ctx context.Context, postID uint64, choices []string) (int, error) {
	if postID == 0 {
		return 0, errors.New("post_id is required")
	}
	filtered := make([]string, 0, len(choices))
	for _, c := range choices {
		c = strings.TrimSpace(c)
		if c != "" {
			filtered = append(filtered, c)
		}
	}
	if len(filtered) == 0 {
		return 0, nil
	}
	var count int64
	if err := s.db.WithContext(ctx).
		Table("event_poll_votes").
		Where("post_id = ? AND choice IN ?", postID, filtered).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (s *Store) UpdateEventCostAmount(ctx context.Context, chatID int64, eventID uint64, costAmount *float64) error {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return err
	}
	if costAmount != nil && *costAmount < 0 {
		return errors.New("cost amount must be >= 0")
	}

	updates := map[string]interface{}{
		"updated_at": gorm.Expr("NOW()"),
	}
	if costAmount == nil {
		updates["cost_amount"] = gorm.Expr("NULL")
	} else {
		updates["cost_amount"] = *costAmount
	}

	result := s.db.WithContext(ctx).
		Model(&GroupEvent{}).
		Where("id = ? AND group_id = ? AND is_active = TRUE", eventID, group.ID).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("event not found")
	}
	return nil
}

func upsertTelegramUserTx(tx *gorm.DB, telegramID int64, username, firstName, lastName string) error {
	record := map[string]interface{}{
		"telegram_id":   telegramID,
		"username":      strings.TrimSpace(username),
		"first_name":    strings.TrimSpace(firstName),
		"last_name":     strings.TrimSpace(lastName),
		"is_active":     true,
		"last_seen_at":  gorm.Expr("NOW()"),
		"updated_at":    gorm.Expr("NOW()"),
		"first_seen_at": gorm.Expr("NOW()"),
	}
	return tx.Table("telegram_users").
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "telegram_id"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"username":     record["username"],
				"first_name":   record["first_name"],
				"last_name":    record["last_name"],
				"is_active":    true,
				"last_seen_at": gorm.Expr("NOW()"),
				"updated_at":   gorm.Expr("NOW()"),
			}),
		}).
		Create(record).Error
}

func upsertGroupMemberTx(tx *gorm.DB, groupID uint64, userTelegramID int64, role, status string) error {
	role = strings.TrimSpace(strings.ToLower(role))
	if role == "" {
		role = "member"
	}
	status = strings.TrimSpace(strings.ToLower(status))
	if status == "" {
		status = "active"
	}

	record := map[string]interface{}{
		"group_id":         groupID,
		"user_telegram_id": userTelegramID,
		"role":             role,
		"status":           status,
		"is_active":        true,
		"last_seen_at":     gorm.Expr("NOW()"),
		"updated_at":       gorm.Expr("NOW()"),
		"joined_at":        gorm.Expr("NOW()"),
	}
	return tx.Table("group_members").
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "group_id"},
				{Name: "user_telegram_id"},
			},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"role":         record["role"],
				"status":       record["status"],
				"is_active":    true,
				"last_seen_at": gorm.Expr("NOW()"),
				"updated_at":   gorm.Expr("NOW()"),
			}),
		}).
		Create(record).Error
}
