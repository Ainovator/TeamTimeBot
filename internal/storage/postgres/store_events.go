package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

func clockMinutes(value string) (int, error) {
	parsed, err := time.Parse("15:04", strings.TrimSpace(value))
	if err != nil {
		return 0, err
	}
	return parsed.Hour()*60 + parsed.Minute(), nil
}

func (s *Store) CreateEvent(
	ctx context.Context,
	chatID int64,
	name string,
	eventType string,
	pollTemplateName string,
	startWeekday int,
	pollPublishWeekday int,
	pollPublishTime, startTime, endTime string,
	announcementText string,
	announcementEnabled bool,
	announcementLeadMinutes int,
	teamsAutoSplit bool,
	teamsPublishList bool,
	teamSize int,
	maxPlaces int,
	minVotesToHold int,
	cancelLeadMinutes int,
	cancelNotifyEnabled bool,
	settlementEnabled bool,
	settlementPublishBefore bool,
	settlementPublishAfter bool,
	costAmount *float64,
) (*GroupEvent, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(name) == "" {
		return nil, errors.New("event name is required")
	}
	eventType = normalizeEventType(eventType)
	if eventType == "" {
		return nil, errors.New("event type must be training or activity")
	}
	pollTemplateName = strings.TrimSpace(pollTemplateName)
	if startWeekday < 1 || startWeekday > 7 {
		return nil, errors.New("weekday must be between 1 and 7")
	}
	if pollPublishWeekday == 0 {
		pollPublishWeekday = startWeekday
	}
	if pollPublishWeekday < 1 || pollPublishWeekday > 7 {
		return nil, errors.New("publish weekday must be between 1 and 7")
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

	startMin, _ := clockMinutes(startTime)
	endMin, _ := clockMinutes(endTime)
	if endMin <= startMin {
		return nil, errors.New("end time must be after start time")
	}

	// Publish must not be later than the start of the event.
	// If publish weekday differs, interpret it as a day before the event within the previous 6 days.
	if pollPublishWeekday == startWeekday {
		publishMin, _ := clockMinutes(pollPublishTime)
		if publishMin > startMin {
			return nil, errors.New("poll publish time must not be later than event start time")
		}
	}
	announcementText = strings.TrimSpace(announcementText)
	announcementLeadMinutes = normalizeAnnouncementLeadMinutes(announcementLeadMinutes)
	if announcementLeadMinutes == -1 {
		return nil, errors.New("announcement lead must be 60, 120 or 1440 minutes")
	}
	if announcementEnabled && announcementText == "" {
		return nil, errors.New("announcement text is required when announcement publishing is enabled")
	}
	if teamSize < 2 {
		return nil, errors.New("team size must be >= 2")
	}
	maxPlaces = normalizePollMaxPlaces(maxPlaces)
	if minVotesToHold < 0 {
		return nil, errors.New("min votes must be >= 0")
	}
	cancelLeadMinutes = normalizeCancelLeadMinutes(cancelLeadMinutes)
	if settlementEnabled && !settlementPublishBefore && !settlementPublishAfter {
		return nil, errors.New("choose at least one settlement publish mode: before or after")
	}
	if costAmount != nil && *costAmount < 0 {
		return nil, errors.New("cost amount must be >= 0")
	}

	var pollTemplateID *uint64
	if pollTemplateName != "" {
		var pollTemplate PollTemplate
		if err := s.db.WithContext(ctx).
			Where("group_id = ? AND name = ? AND is_active = TRUE", group.ID, pollTemplateName).
			First(&pollTemplate).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.New("poll template not found")
			}
			return nil, err
		}
		pollTemplateID = &pollTemplate.ID
	}

	event := GroupEvent{
		GroupID:                 group.ID,
		PollTemplateID:          pollTemplateID,
		Name:                    strings.TrimSpace(name),
		EventType:               eventType,
		StartWeekday:            int16(startWeekday),
		PollPublishWeekday:      int16(pollPublishWeekday),
		PollPublishTime:         pollPublishTime,
		StartTime:               startTime,
		EndTime:                 endTime,
		AnnouncementText:        announcementText,
		AnnouncementEnabled:     announcementEnabled,
		AnnouncementLeadMinutes: int16(announcementLeadMinutes),
		PublishEnabled:          true,
		TeamsAutoSplit:          teamsAutoSplit,
		TeamsPublishList:        teamsPublishList,
		TeamSize:                int16(teamSize),
		MaxPlaces:               int32(maxPlaces),
		MinVotesToHold:          int32(minVotesToHold),
		CancelLeadMinutes:       int32(cancelLeadMinutes),
		CancelNotifyEnabled:     cancelNotifyEnabled,
		SettlementEnabled:       settlementEnabled,
		SettlementPublishBefore: settlementPublishBefore,
		SettlementPublishAfter:  settlementPublishAfter,
		CostAmount:              costAmount,
		IsActive:                true,
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
		// Be defensive: older rows (or partially-migrated DBs) may have NULLs in some columns.
		// Scanning NULL into Go string/int fields will error and break the whole page.
		Select("ge.id, ge.name, COALESCE(ge.event_type, 'training') AS event_type, ge.start_weekday, COALESCE(ge.poll_publish_weekday, ge.start_weekday) AS poll_publish_weekday, COALESCE(ge.poll_publish_time, ge.start_time) AS poll_publish_time, ge.start_time, ge.end_time, COALESCE(ge.announcement_text, '') AS announcement_text, COALESCE(ge.announcement_enabled, FALSE) AS announcement_enabled, COALESCE(ge.announcement_lead_minutes, 60) AS announcement_lead_minutes, COALESCE(ge.publish_enabled, TRUE) AS publish_enabled, COALESCE(ge.teams_auto_split, FALSE) AS teams_auto_split, COALESCE(ge.teams_publish_list, FALSE) AS teams_publish_list, COALESCE(ge.team_size, 6) AS team_size, COALESCE(ge.max_places, 18) AS max_places, COALESCE(ge.min_votes_to_hold, 0) AS min_votes_to_hold, COALESCE(ge.cancel_lead_minutes, 180) AS cancel_lead_minutes, COALESCE(ge.cancel_notify_enabled, FALSE) AS cancel_notify_enabled, COALESCE(ge.settlement_enabled, TRUE) AS settlement_enabled, COALESCE(ge.settlement_publish_before, FALSE) AS settlement_publish_before, COALESCE(ge.settlement_publish_after, TRUE) AS settlement_publish_after, ge.cost_amount, COALESCE(ge.is_active, TRUE) AS is_active, COALESCE(pt.name, '') AS poll_template").
		Joins("LEFT JOIN poll_templates pt ON pt.id = ge.poll_template_id").
		Where("ge.group_id = ? AND ge.is_active = TRUE", group.ID).
		Order("id DESC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Store) ListArchivedEventsByChatID(ctx context.Context, chatID int64) ([]EventView, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	var rows []EventView
	if err := s.db.WithContext(ctx).
		Table("group_events ge").
		Select("ge.id, ge.name, COALESCE(ge.event_type, 'training') AS event_type, ge.start_weekday, COALESCE(ge.poll_publish_weekday, ge.start_weekday) AS poll_publish_weekday, COALESCE(ge.poll_publish_time, ge.start_time) AS poll_publish_time, ge.start_time, ge.end_time, COALESCE(ge.announcement_text, '') AS announcement_text, COALESCE(ge.announcement_enabled, FALSE) AS announcement_enabled, COALESCE(ge.announcement_lead_minutes, 60) AS announcement_lead_minutes, COALESCE(ge.publish_enabled, TRUE) AS publish_enabled, COALESCE(ge.teams_auto_split, FALSE) AS teams_auto_split, COALESCE(ge.teams_publish_list, FALSE) AS teams_publish_list, COALESCE(ge.team_size, 6) AS team_size, COALESCE(ge.max_places, 18) AS max_places, COALESCE(ge.min_votes_to_hold, 0) AS min_votes_to_hold, COALESCE(ge.cancel_lead_minutes, 180) AS cancel_lead_minutes, COALESCE(ge.cancel_notify_enabled, FALSE) AS cancel_notify_enabled, COALESCE(ge.settlement_enabled, TRUE) AS settlement_enabled, COALESCE(ge.settlement_publish_before, FALSE) AS settlement_publish_before, COALESCE(ge.settlement_publish_after, TRUE) AS settlement_publish_after, ge.cost_amount, COALESCE(ge.is_active, TRUE) AS is_active, COALESCE(pt.name, '') AS poll_template").
		Joins("LEFT JOIN poll_templates pt ON pt.id = ge.poll_template_id").
		Where("ge.group_id = ? AND ge.is_active = FALSE", group.ID).
		Order("id DESC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Store) GetEventByID(ctx context.Context, chatID int64, eventID uint64) (*EventView, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	var row EventView
	if err := s.db.WithContext(ctx).
		Table("group_events ge").
		Select("ge.id, ge.name, COALESCE(ge.event_type, 'training') AS event_type, ge.start_weekday, COALESCE(ge.poll_publish_weekday, ge.start_weekday) AS poll_publish_weekday, COALESCE(ge.poll_publish_time, ge.start_time) AS poll_publish_time, ge.start_time, ge.end_time, COALESCE(ge.announcement_text, '') AS announcement_text, COALESCE(ge.announcement_enabled, FALSE) AS announcement_enabled, COALESCE(ge.announcement_lead_minutes, 60) AS announcement_lead_minutes, COALESCE(ge.publish_enabled, TRUE) AS publish_enabled, COALESCE(ge.teams_auto_split, FALSE) AS teams_auto_split, COALESCE(ge.teams_publish_list, FALSE) AS teams_publish_list, COALESCE(ge.team_size, 6) AS team_size, COALESCE(ge.max_places, 18) AS max_places, COALESCE(ge.min_votes_to_hold, 0) AS min_votes_to_hold, COALESCE(ge.cancel_lead_minutes, 180) AS cancel_lead_minutes, COALESCE(ge.cancel_notify_enabled, FALSE) AS cancel_notify_enabled, COALESCE(ge.settlement_enabled, TRUE) AS settlement_enabled, COALESCE(ge.settlement_publish_before, FALSE) AS settlement_publish_before, COALESCE(ge.settlement_publish_after, TRUE) AS settlement_publish_after, ge.cost_amount, COALESCE(ge.is_active, TRUE) AS is_active, COALESCE(pt.name, '') AS poll_template").
		Joins("LEFT JOIN poll_templates pt ON pt.id = ge.poll_template_id").
		Where("ge.id = ? AND ge.group_id = ? AND ge.is_active = TRUE", eventID, group.ID).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("event not found")
		}
		return nil, err
	}

	return &row, nil
}

func (s *Store) UpdateEventName(ctx context.Context, chatID int64, eventID uint64, name string) error {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("event name is required")
	}

	result := s.db.WithContext(ctx).
		Model(&GroupEvent{}).
		Where("id = ? AND group_id = ? AND is_active = TRUE", eventID, group.ID).
		Updates(map[string]interface{}{
			"name":       name,
			"updated_at": gorm.Expr("NOW()"),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("event not found")
	}

	return nil
}

func (s *Store) SetEventArchived(ctx context.Context, chatID int64, eventID uint64, archived bool) error {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return err
	}

	result := s.db.WithContext(ctx).
		Model(&GroupEvent{}).
		Where("id = ? AND group_id = ?", eventID, group.ID).
		Updates(map[string]interface{}{
			"is_active":  !archived,
			"updated_at": gorm.Expr("NOW()"),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("event not found")
	}
	return nil
}

func (s *Store) UpdateEventDetails(
	ctx context.Context,
	chatID int64,
	eventID uint64,
	name string,
	eventType string,
	startWeekday int,
	pollPublishWeekday int,
	pollPublishTime, startTime, endTime string,
	announcementText string,
	announcementEnabled bool,
	announcementLeadMinutes int,
	teamsAutoSplit bool,
	teamsPublishList bool,
	teamSize int,
	maxPlaces int,
	minVotesToHold int,
	cancelLeadMinutes int,
	cancelNotifyEnabled bool,
	settlementEnabled bool,
	settlementPublishBefore bool,
	settlementPublishAfter bool,
) error {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return err
	}

	name = strings.TrimSpace(name)
	pollPublishTime = strings.TrimSpace(pollPublishTime)
	startTime = strings.TrimSpace(startTime)
	endTime = strings.TrimSpace(endTime)
	announcementText = strings.TrimSpace(announcementText)

	if name == "" {
		return errors.New("event name is required")
	}
	eventType = normalizeEventType(eventType)
	if eventType == "" {
		return errors.New("event type must be training or activity")
	}
	if startWeekday < 1 || startWeekday > 7 {
		return errors.New("weekday must be between 1 and 7")
	}
	if pollPublishWeekday < 1 || pollPublishWeekday > 7 {
		return errors.New("publish weekday must be between 1 and 7")
	}
	if _, err := time.Parse("15:04", pollPublishTime); err != nil {
		return errors.New("invalid publish time format, use HH:MM")
	}
	if _, err := time.Parse("15:04", startTime); err != nil {
		return errors.New("invalid start time format, use HH:MM")
	}
	if _, err := time.Parse("15:04", endTime); err != nil {
		return errors.New("invalid end time format, use HH:MM")
	}

	startMin, _ := clockMinutes(startTime)
	endMin, _ := clockMinutes(endTime)
	if endMin <= startMin {
		return errors.New("end time must be after start time")
	}
	if pollPublishWeekday == startWeekday {
		publishMin, _ := clockMinutes(pollPublishTime)
		if publishMin > startMin {
			return errors.New("poll publish time must not be later than event start time")
		}
	}
	announcementLeadMinutes = normalizeAnnouncementLeadMinutes(announcementLeadMinutes)
	if announcementLeadMinutes == -1 {
		return errors.New("announcement lead must be 60, 120 or 1440 minutes")
	}
	if announcementEnabled && announcementText == "" {
		return errors.New("announcement text is required when announcement publishing is enabled")
	}
	if teamSize < 2 {
		return errors.New("team size must be >= 2")
	}
	maxPlaces = normalizePollMaxPlaces(maxPlaces)
	if minVotesToHold < 0 {
		return errors.New("min votes must be >= 0")
	}
	cancelLeadMinutes = normalizeCancelLeadMinutes(cancelLeadMinutes)
	if settlementEnabled && !settlementPublishBefore && !settlementPublishAfter {
		return errors.New("choose at least one settlement publish mode: before or after")
	}

	result := s.db.WithContext(ctx).
		Model(&GroupEvent{}).
		Where("id = ? AND group_id = ? AND is_active = TRUE", eventID, group.ID).
		Updates(map[string]interface{}{
			"name":                      name,
			"event_type":                eventType,
			"start_weekday":             startWeekday,
			"poll_publish_weekday":      pollPublishWeekday,
			"poll_publish_time":         pollPublishTime,
			"start_time":                startTime,
			"end_time":                  endTime,
			"announcement_text":         announcementText,
			"announcement_enabled":      announcementEnabled,
			"announcement_lead_minutes": announcementLeadMinutes,
			"teams_auto_split":          teamsAutoSplit,
			"teams_publish_list":        teamsPublishList,
			"team_size":                 teamSize,
			"max_places":                maxPlaces,
			"min_votes_to_hold":         minVotesToHold,
			"cancel_lead_minutes":       cancelLeadMinutes,
			"cancel_notify_enabled":     cancelNotifyEnabled,
			"settlement_enabled":        settlementEnabled,
			"settlement_publish_before": settlementPublishBefore,
			"settlement_publish_after":  settlementPublishAfter,
			"updated_at":                gorm.Expr("NOW()"),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("event not found")
	}

	return nil
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
