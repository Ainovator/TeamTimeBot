package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"sort"
	"strconv"
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
	Name                string `json:"name"`
	Question            string `json:"question"`
	CountedOptionsCount int    `json:"countedOptionsCount"`
}

type GroupView struct {
	ChatID   int64  `json:"chatID"`
	Title    string `json:"title"`
	Timezone string `json:"timezone"`
}

type ScheduleView struct {
	ID           uint64 `json:"id"`
	TemplateName string `json:"templateName"`
	SendAt       string `json:"sendAt"`
	IsActive     bool   `json:"isActive"`
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
	Name           string   `json:"name"`
	Question       string   `json:"question"`
	Options        []string `json:"options"`
	CountedOptions []int    `json:"countedOptions"`
}

type EventView struct {
	ID                      uint64   `json:"id"`
	Name                    string   `json:"name"`
	EventType               string   `json:"eventType"`
	StartWeekday            int      `json:"startWeekday"`
	PollPublishWeekday      int      `json:"pollPublishWeekday"`
	PollPublishTime         string   `json:"pollPublishTime"`
	StartTime               string   `json:"startTime"`
	EndTime                 string   `json:"endTime"`
	AnnouncementText        string   `json:"announcementText"`
	AnnouncementEnabled     bool     `json:"announcementEnabled"`
	AnnouncementLeadMinutes int      `json:"announcementLeadMinutes"`
	PublishEnabled          bool     `json:"publishEnabled"`
	TeamsAutoSplit          bool     `json:"teamsAutoSplit"`
	TeamsPublishList        bool     `json:"teamsPublishList"`
	TeamSize                int      `json:"teamSize"`
	MinVotesToHold          int      `json:"minVotesToHold"`
	CancelLeadMinutes       int      `json:"cancelLeadMinutes"`
	CancelNotifyEnabled     bool     `json:"cancelNotifyEnabled"`
	SettlementEnabled       bool     `json:"settlementEnabled"`
	SettlementPublishBefore bool     `json:"settlementPublishBefore"`
	SettlementPublishAfter  bool     `json:"settlementPublishAfter"`
	CostAmount              *float64 `json:"costAmount"`
	IsActive                bool     `json:"isActive"`
	PollTemplate            string   `json:"pollTemplate"`
}

type EventActivitySummary struct {
	EventID            uint64 `json:"eventID"`
	PollsTotal         int64  `json:"pollsTotal"`
	VotesTotal         int64  `json:"votesTotal"`
	SettlementsTotal   int64  `json:"settlementsTotal"`
	AnnouncementsTotal int64  `json:"announcementsTotal"`
}

type EventHistoryStatus string

const (
	EventHistoryStatusCompleted      EventHistoryStatus = "completed"
	EventHistoryStatusInVoting       EventHistoryStatus = "in_voting"
	EventHistoryStatusOnDistribution EventHistoryStatus = "on_distribution"
	EventHistoryStatusOnReview       EventHistoryStatus = "on_review"
	EventHistoryStatusNotHeld        EventHistoryStatus = "not_held"
)

type EventHistoryItem struct {
	InstanceID     uint64             `json:"instanceID"`
	EventID        uint64             `json:"eventID"`
	Name           string             `json:"name"`
	EventType      string             `json:"eventType"`
	LocalDate      time.Time          `json:"localDate"`
	StartWeekday   int                `json:"startWeekday"`
	StartTime      string             `json:"startTime"`
	PollTemplate   string             `json:"pollTemplate"`
	LatestPostID   *uint64            `json:"latestPostID"`
	LatestPollAt   *time.Time         `json:"latestPollAt"`
	NextStartAt    time.Time          `json:"nextStartAt"`
	EndAt          time.Time          `json:"endAt"`
	Status         EventHistoryStatus `json:"status"`
	CanDistribute  bool               `json:"canDistribute"`
	PublishEnabled bool               `json:"publishEnabled"`
	DebtAmount     float64            `json:"debtAmount"`
}

type EventBillingParticipant struct {
	UserID    int64      `json:"userID"`
	Username  string     `json:"username"`
	FirstName string     `json:"firstName"`
	LastName  string     `json:"lastName"`
	AmountDue float64    `json:"amountDue"`
	IsPaid    bool       `json:"isPaid"`
	PaidAt    *time.Time `json:"paidAt,omitempty"`
}

type EventBillingView struct {
	EventID            uint64                    `json:"eventID"`
	SettlementID       uint64                    `json:"settlementID"`
	LocalDate          time.Time                 `json:"localDate"`
	TotalAmount        float64                   `json:"totalAmount"`
	AmountPerPerson    float64                   `json:"amountPerPerson"`
	ParticipantsCount  int                       `json:"participantsCount"`
	PaidCount          int                       `json:"paidCount"`
	UnpaidCount        int                       `json:"unpaidCount"`
	DebtAmount         float64                   `json:"debtAmount"`
	Players            []EventBillingParticipant `json:"players"`
	AllPaymentsChecked bool                      `json:"allPaymentsChecked"`
}

type GroupDebtSummary struct {
	TotalDebt  float64 `json:"totalDebt"`
	UnpaidRows int64   `json:"unpaidRows"`
}

type EventPollHistoryItem struct {
	PostID            uint64    `json:"postID"`
	TelegramMessageID int64     `json:"telegramMessageID"`
	TelegramPollID    string    `json:"telegramPollID"`
	Status            string    `json:"status"`
	PublishedAt       time.Time `json:"publishedAt"`
	Question          string    `json:"question"`
	CountedVotes      int       `json:"countedVotes"`
	TotalVotes        int       `json:"totalVotes"`
	TeamsConfigured   bool      `json:"teamsConfigured"`
}

type GroupPollItem struct {
	PostID            uint64     `json:"postID"`
	InstanceID        *uint64    `json:"instanceID,omitempty"`
	EventID           *uint64    `json:"eventID,omitempty"`
	EventName         string     `json:"eventName"`
	EventLocalDate    *time.Time `json:"eventLocalDate,omitempty"`
	TemplateName      string     `json:"templateName"`
	Question          string     `json:"question"`
	TelegramMessageID int64      `json:"telegramMessageID"`
	TelegramPollID    string     `json:"telegramPollID"`
	Status            string     `json:"status"`
	PublishedAt       time.Time  `json:"publishedAt"`
	CountedVotes      int        `json:"countedVotes"`
	TotalVotes        int        `json:"totalVotes"`
}

type GroupPollVoteItem struct {
	UserID      int64     `json:"userID"`
	Username    string    `json:"username"`
	FirstName   string    `json:"firstName"`
	LastName    string    `json:"lastName"`
	Choice      string    `json:"choice"`
	ChoiceIndex *int      `json:"choiceIndex,omitempty"`
	ChoiceLabel string    `json:"choiceLabel"`
	Counted     bool      `json:"counted"`
	Source      string    `json:"source"`
	VotedAt     time.Time `json:"votedAt"`
}

type TeamSplitPlayer struct {
	UserID      int64   `json:"userID"`
	Username    string  `json:"username"`
	FirstName   string  `json:"firstName"`
	LastName    string  `json:"lastName"`
	Choice      string  `json:"choice"`
	ChoiceIndex int     `json:"choiceIndex"`
	ChoiceLabel string  `json:"choiceLabel"`
	Rating      float64 `json:"rating"`
	Team        string  `json:"team"`
	Position    int     `json:"position"`
}

type TeamWinChance struct {
	TeamAScore float64 `json:"teamAScore"`
	TeamBScore float64 `json:"teamBScore"`
	TeamAProb  float64 `json:"teamAProb"`
	TeamBProb  float64 `json:"teamBProb"`
}

type EventTeamSplitState struct {
	EventID uint64            `json:"eventID"`
	PostID  uint64            `json:"postID"`
	Players []TeamSplitPlayer `json:"players"`
	Chance  TeamWinChance     `json:"chance"`
}

type GroupMemberView struct {
	UserTelegramID int64     `json:"userTelegramID"`
	Username       string    `json:"username"`
	FirstName      string    `json:"firstName"`
	LastName       string    `json:"lastName"`
	PlayerType     string    `json:"playerType"`
	Role           string    `json:"role"`
	Status         string    `json:"status"`
	LastSeenAt     time.Time `json:"lastSeenAt"`
}

type SkillCatalogItem struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type MemberSkillValue struct {
	SkillCode string `json:"skillCode"`
	SkillName string `json:"skillName"`
	Score     *int   `json:"score"`
}

type MemberSkillProfile struct {
	UserTelegramID int64              `json:"userTelegramID"`
	Username       string             `json:"username"`
	FirstName      string             `json:"firstName"`
	LastName       string             `json:"lastName"`
	PlayerType     string             `json:"playerType"`
	Skills         []MemberSkillValue `json:"skills"`
}

type PlayerRelationView struct {
	UserAID          int64  `json:"userAID"`
	UserBID          int64  `json:"userBID"`
	RelationType     string `json:"relationType"`
	Weight           int    `json:"weight"`
	RelatedUserID    int64  `json:"relatedUserID"`
	RelatedUsername  string `json:"relatedUsername"`
	RelatedFirstName string `json:"relatedFirstName"`
	RelatedLastName  string `json:"relatedLastName"`
}

type EventPollPostView struct {
	ID                uint64
	GroupID           uint64
	EventID           *uint64
	InstanceID        *uint64
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
	EventID                 uint64
	GroupID                 uint64
	ChatID                  int64
	Timezone                string
	Name                    string
	EventType               string
	StartWeekday            int
	PollPublishWeekday      int
	PollPublishTime         string
	StartTime               string
	EndTime                 string
	AnnouncementText        string
	AnnouncementEnabled     bool
	AnnouncementLeadMinutes int
	PublishEnabled          bool
	TeamsAutoSplit          bool
	TeamsPublishList        bool
	TeamSize                int
	MinVotesToHold          int
	CancelLeadMinutes       int
	CancelNotifyEnabled     bool
	SettlementEnabled       bool
	SettlementPublishBefore bool
	SettlementPublishAfter  bool
	CostAmount              *float64
	PollTemplate            string
}

func normalizeAnnouncementLeadMinutes(value int) int {
	switch value {
	case 60, 120, 1440:
		return value
	case 0:
		return 60
	default:
		return -1
	}
}

func normalizeCancelLeadMinutes(value int) int {
	if value <= 0 {
		return 180
	}
	return value
}

func normalizeEventType(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "", "training":
		return "training"
	case "activity":
		return "activity"
	default:
		return ""
	}
}

func normalizePlayerType(value string) (string, bool) {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "", "none":
		return "", true
	case "attacker":
		return "attacker", true
	case "setter":
		return "setter", true
	case "libero":
		return "libero", true
	default:
		return "", false
	}
}

func normalizeRelationType(value string) (string, bool) {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "prefer_together":
		return "prefer_together", true
	case "avoid_together":
		return "avoid_together", true
	default:
		return "", false
	}
}

func normalizeCountedOptionIndexes(optionCount int, indexes []int) []int {
	if optionCount <= 0 {
		return []int{}
	}
	unique := make(map[int]struct{}, len(indexes))
	for _, idx := range indexes {
		if idx < 0 || idx >= optionCount {
			continue
		}
		unique[idx] = struct{}{}
	}
	result := make([]int, 0, len(unique))
	for idx := range unique {
		result = append(result, idx)
	}
	sort.Ints(result)
	return result
}

func parsePollOptionChoice(choice string) (int, bool) {
	choice = strings.TrimSpace(choice)
	if !strings.HasPrefix(choice, "option_") {
		return 0, false
	}
	idx, err := strconv.Atoi(strings.TrimPrefix(choice, "option_"))
	if err != nil || idx < 0 {
		return 0, false
	}
	return idx, true
}

func parseClockTime(value string) (int, int, error) {
	layouts := []string{"15:04:05", "15:04"}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, strings.TrimSpace(value))
		if err == nil {
			return parsed.Hour(), parsed.Minute(), nil
		}
	}
	return 0, 0, errors.New("invalid time format")
}

func nextWeekdayTime(now time.Time, weekday int, hour int, minute int) time.Time {
	if weekday < 1 || weekday > 7 {
		return now
	}
	currentWeekday := int(now.Weekday())
	if currentWeekday == 0 {
		currentWeekday = 7
	}
	delta := weekday - currentWeekday
	if delta < 0 {
		delta += 7
	}
	candidateDate := now.AddDate(0, 0, delta)
	candidate := time.Date(candidateDate.Year(), candidateDate.Month(), candidateDate.Day(), hour, minute, 0, 0, now.Location())
	if !candidate.After(now) {
		candidate = candidate.AddDate(0, 0, 7)
	}
	return candidate
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
	defaultCounted := []int{}
	if len(options) > 0 {
		defaultCounted = []int{0}
	}
	return s.UpsertPollTemplateWithCounted(ctx, chatID, name, question, options, defaultCounted)
}

func (s *Store) UpsertPollTemplateWithCounted(
	ctx context.Context,
	chatID int64,
	name, question string,
	options []string,
	countedOptions []int,
) (*PollTemplate, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	sanitizedOptions := make([]string, 0, len(options))
	for _, option := range options {
		option = strings.TrimSpace(option)
		if option != "" {
			sanitizedOptions = append(sanitizedOptions, option)
		}
	}
	if len(sanitizedOptions) < 2 {
		return nil, errors.New("at least 2 options are required")
	}

	name = strings.TrimSpace(name)
	question = strings.TrimSpace(question)
	if name == "" {
		return nil, errors.New("template name is required")
	}
	if question == "" {
		return nil, errors.New("question is required")
	}

	payload, err := json.Marshal(sanitizedOptions)
	if err != nil {
		return nil, err
	}

	normalizedCounted := normalizeCountedOptionIndexes(len(sanitizedOptions), countedOptions)
	countedPayload, err := json.Marshal(normalizedCounted)
	if err != nil {
		return nil, err
	}

	template := PollTemplate{
		GroupID:        group.ID,
		Name:           name,
		Question:       question,
		Options:        datatypes.JSON(payload),
		CountedOptions: datatypes.JSON(countedPayload),
		IsActive:       true,
	}

	if err := s.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "group_id"},
				{Name: "name"},
			},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"question":        question,
				"options":         datatypes.JSON(payload),
				"counted_options": datatypes.JSON(countedPayload),
				"is_active":       true,
				"updated_at":      gorm.Expr("NOW()"),
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

func (s *Store) EnsureDefaultRegistrationTemplate(ctx context.Context, chatID int64) (*TemplateDetails, error) {
	const name = "Регистрация"
	const question = "Регистрация: выбери вариант"
	options := []string{"Зарегистрироваться", "Не буду"}
	counted := []int{0}

	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	// Do not overwrite an existing template if the admin edited it.
	var existing PollTemplate
	if err := s.db.WithContext(ctx).
		Where("group_id = ? AND name = ? AND is_active = TRUE", group.ID, name).
		First(&existing).Error; err == nil {
		return s.GetTemplateByName(ctx, chatID, name)
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if _, err := s.UpsertPollTemplateWithCounted(ctx, chatID, name, question, options, counted); err != nil {
		return nil, err
	}
	return s.GetTemplateByName(ctx, chatID, name)
}

func isSystemTemplateName(templateName string) bool {
	// Keep this simple and explicit for now.
	// If we later add more system templates, they can be listed here.
	return strings.EqualFold(strings.TrimSpace(templateName), "Регистрация")
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

func clampInstanceStatus(status string) string {
	switch strings.TrimSpace(status) {
	case string(EventHistoryStatusInVoting):
		return string(EventHistoryStatusInVoting)
	case string(EventHistoryStatusOnDistribution):
		return string(EventHistoryStatusOnDistribution)
	case string(EventHistoryStatusOnReview):
		return string(EventHistoryStatusOnReview)
	case string(EventHistoryStatusCompleted):
		return string(EventHistoryStatusCompleted)
	case string(EventHistoryStatusNotHeld):
		return string(EventHistoryStatusNotHeld)
	default:
		return string(EventHistoryStatusInVoting)
	}
}

func ensureEventInstanceTx(
	ctx context.Context,
	tx *gorm.DB,
	groupID uint64,
	eventID uint64,
	localDate time.Time,
	plannedStartAt time.Time,
	plannedEndAt time.Time,
	status string,
	pollTemplateOverride *PollTemplate,
) (*EventInstance, error) {
	status = clampInstanceStatus(status)
	record := map[string]interface{}{
		"group_id":         groupID,
		"event_id":         eventID,
		"local_date":       localDate.Format("2006-01-02"),
		"planned_start_at": plannedStartAt.UTC(),
		"planned_end_at":   plannedEndAt.UTC(),
		"status":           status,
	}

	// Snapshot settings at creation time. On conflict, we intentionally do NOT update these fields,
	// so that later template edits do not affect already-created instances.
	type snapshotRow struct {
		Name                    string
		EventType               string
		StartWeekday            int
		StartTime               string
		EndTime                 string
		PublishEnabled          bool
		CostAmount              *float64
		MinVotesToHold          int
		CancelLeadMinutes       int
		CancelNotifyEnabled     bool
		SettlementEnabled       bool
		SettlementPublishBefore bool
		SettlementPublishAfter  bool
		AnnouncementText        string
		AnnouncementEnabled     bool
		AnnouncementLeadMinutes int
		TeamsAutoSplit          bool
		TeamsPublishList        bool
		TeamSize                int
		PollTemplateID          *uint64
		PollTemplateName        string
		PollQuestion            string
		PollOptions             datatypes.JSON
		PollCountedOptions      datatypes.JSON
	}
	var snap snapshotRow
	if err := tx.WithContext(ctx).
		Table("group_events ge").
		Select(`
			ge.name,
			ge.event_type,
			ge.start_weekday,
			ge.start_time,
			ge.end_time,
			ge.publish_enabled,
			ge.cost_amount,
			ge.min_votes_to_hold,
			ge.cancel_lead_minutes,
			ge.cancel_notify_enabled,
			ge.settlement_enabled,
			ge.settlement_publish_before,
			ge.settlement_publish_after,
			COALESCE(ge.announcement_text, '') AS announcement_text,
			ge.announcement_enabled,
			ge.announcement_lead_minutes,
			ge.teams_auto_split,
			ge.teams_publish_list,
			ge.team_size,
			ge.poll_template_id,
			COALESCE(pt.name, '') AS poll_template_name,
			COALESCE(pt.question, '') AS poll_question,
			COALESCE(pt.options, '[]'::jsonb) AS poll_options,
			COALESCE(pt.counted_options, '[]'::jsonb) AS poll_counted_options
		`).
		Joins("LEFT JOIN poll_templates pt ON pt.id = ge.poll_template_id").
		Where("ge.id = ? AND ge.group_id = ? AND ge.is_active = TRUE", eventID, groupID).
		Take(&snap).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("event not found")
		}
		return nil, err
	}

	// Poll snapshot comes from the template actually used for this poll (if provided),
	// otherwise from the currently bound template on the event.
	if pollTemplateOverride != nil {
		snap.PollTemplateID = &pollTemplateOverride.ID
		snap.PollTemplateName = pollTemplateOverride.Name
		snap.PollQuestion = pollTemplateOverride.Question
		snap.PollOptions = pollTemplateOverride.Options
		snap.PollCountedOptions = pollTemplateOverride.CountedOptions
	}

	record["event_name"] = snap.Name
	record["event_type"] = snap.EventType
	record["start_weekday"] = snap.StartWeekday
	record["start_time"] = snap.StartTime
	record["end_time"] = snap.EndTime
	record["publish_enabled"] = snap.PublishEnabled
	record["cost_amount"] = snap.CostAmount
	record["min_votes_to_hold"] = snap.MinVotesToHold
	record["cancel_lead_minutes"] = snap.CancelLeadMinutes
	record["cancel_notify_enabled"] = snap.CancelNotifyEnabled
	record["settlement_enabled"] = snap.SettlementEnabled
	record["settlement_publish_before"] = snap.SettlementPublishBefore
	record["settlement_publish_after"] = snap.SettlementPublishAfter
	record["announcement_text"] = snap.AnnouncementText
	record["announcement_enabled"] = snap.AnnouncementEnabled
	record["announcement_lead_minutes"] = snap.AnnouncementLeadMinutes
	record["teams_auto_split"] = snap.TeamsAutoSplit
	record["teams_publish_list"] = snap.TeamsPublishList
	record["team_size"] = snap.TeamSize
	record["poll_template_id"] = snap.PollTemplateID
	record["poll_template_name"] = snap.PollTemplateName
	record["poll_question"] = snap.PollQuestion
	record["poll_options"] = snap.PollOptions
	record["poll_counted_options"] = snap.PollCountedOptions

	if err := tx.WithContext(ctx).Table("event_instances").
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "event_id"},
				{Name: "local_date"},
			},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"planned_start_at": record["planned_start_at"],
				"planned_end_at":   record["planned_end_at"],
				"updated_at":       gorm.Expr("NOW()"),
			}),
		}).
		Create(record).Error; err != nil {
		return nil, err
	}
	var instance EventInstance
	if err := tx.WithContext(ctx).
		Table("event_instances").
		Where("event_id = ? AND local_date = ?", eventID, localDate.Format("2006-01-02")).
		Take(&instance).Error; err != nil {
		return nil, err
	}
	return &instance, nil
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
	if isSystemTemplateName(templateName) {
		return errors.New("нельзя удалить системный шаблон \"Регистрация\"")
	}

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
		Select("name, question, COALESCE(jsonb_array_length(counted_options), 0) AS counted_options_count").
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
		Name           string
		Question       string
		Options        datatypes.JSON
		CountedOptions datatypes.JSON
	}
	if err := s.db.WithContext(ctx).
		Table("poll_templates").
		Select("name, question, options, counted_options").
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
	var countedOptions []int
	if err := json.Unmarshal(row.CountedOptions, &countedOptions); err != nil {
		return nil, err
	}

	return &TemplateDetails{
		Name:           row.Name,
		Question:       row.Question,
		Options:        options,
		CountedOptions: normalizeCountedOptionIndexes(len(options), countedOptions),
	}, nil
}

func (s *Store) UpdateTemplateByName(
	ctx context.Context,
	chatID int64,
	currentName, newName, question string,
	options []string,
) (*TemplateDetails, error) {
	defaultCounted := []int{}
	if len(options) > 0 {
		defaultCounted = []int{0}
	}
	return s.UpdateTemplateByNameWithCounted(ctx, chatID, currentName, newName, question, options, defaultCounted)
}

func (s *Store) UpdateTemplateByNameWithCounted(
	ctx context.Context,
	chatID int64,
	currentName, newName, question string,
	options []string,
	countedOptions []int,
) (*TemplateDetails, error) {
	// Prevent renaming of system templates to avoid breaking integrations
	// that rely on a stable name.
	if isSystemTemplateName(currentName) && strings.TrimSpace(newName) != "" && !strings.EqualFold(strings.TrimSpace(newName), strings.TrimSpace(currentName)) {
		return nil, errors.New("нельзя переименовать системный шаблон \"Регистрация\"")
	}

	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	currentName = strings.TrimSpace(currentName)
	newName = strings.TrimSpace(newName)
	question = strings.TrimSpace(question)
	if currentName == "" {
		return nil, errors.New("current template name is required")
	}
	if newName == "" {
		newName = currentName
	}
	if question == "" {
		return nil, errors.New("question is required")
	}

	sanitizedOptions := make([]string, 0, len(options))
	for _, option := range options {
		option = strings.TrimSpace(option)
		if option != "" {
			sanitizedOptions = append(sanitizedOptions, option)
		}
	}
	if len(sanitizedOptions) < 2 {
		return nil, errors.New("at least 2 options are required")
	}

	payload, err := json.Marshal(sanitizedOptions)
	if err != nil {
		return nil, err
	}
	normalizedCounted := normalizeCountedOptionIndexes(len(sanitizedOptions), countedOptions)
	countedPayload, err := json.Marshal(normalizedCounted)
	if err != nil {
		return nil, err
	}

	var result TemplateDetails
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var template PollTemplate
		if err := tx.
			Where("group_id = ? AND name = ? AND is_active = TRUE", group.ID, currentName).
			First(&template).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("template not found")
			}
			return err
		}

		var conflicts int64
		if err := tx.Model(&PollTemplate{}).
			Where("group_id = ? AND name = ? AND is_active = TRUE AND id <> ?", group.ID, newName, template.ID).
			Count(&conflicts).Error; err != nil {
			return err
		}
		if conflicts > 0 {
			return errors.New("template name already exists")
		}

		if err := tx.Model(&PollTemplate{}).
			Where("id = ?", template.ID).
			Updates(map[string]interface{}{
				"name":            newName,
				"question":        question,
				"options":         datatypes.JSON(payload),
				"counted_options": datatypes.JSON(countedPayload),
				"is_active":       true,
				"updated_at":      gorm.Expr("NOW()"),
			}).Error; err != nil {
			return err
		}

		var row struct {
			Name           string
			Question       string
			Options        datatypes.JSON
			CountedOptions datatypes.JSON
		}
		if err := tx.
			Table("poll_templates").
			Select("name, question, options, counted_options").
			Where("id = ?", template.ID).
			First(&row).Error; err != nil {
			return err
		}

		var parsedOptions []string
		if err := json.Unmarshal(row.Options, &parsedOptions); err != nil {
			return err
		}
		var parsedCounted []int
		if err := json.Unmarshal(row.CountedOptions, &parsedCounted); err != nil {
			return err
		}

		result = TemplateDetails{
			Name:           row.Name,
			Question:       row.Question,
			Options:        parsedOptions,
			CountedOptions: normalizeCountedOptionIndexes(len(parsedOptions), parsedCounted),
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &result, nil
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

func (s *Store) CreateEvent(
	ctx context.Context,
	chatID int64,
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

	event := GroupEvent{
		GroupID:                 group.ID,
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
		Select("ge.id, ge.name, COALESCE(ge.event_type, 'training') AS event_type, ge.start_weekday, COALESCE(ge.poll_publish_weekday, ge.start_weekday) AS poll_publish_weekday, COALESCE(ge.poll_publish_time, ge.start_time) AS poll_publish_time, ge.start_time, ge.end_time, COALESCE(ge.announcement_text, '') AS announcement_text, COALESCE(ge.announcement_enabled, FALSE) AS announcement_enabled, COALESCE(ge.announcement_lead_minutes, 60) AS announcement_lead_minutes, COALESCE(ge.publish_enabled, TRUE) AS publish_enabled, COALESCE(ge.teams_auto_split, FALSE) AS teams_auto_split, COALESCE(ge.teams_publish_list, FALSE) AS teams_publish_list, COALESCE(ge.team_size, 6) AS team_size, COALESCE(ge.min_votes_to_hold, 0) AS min_votes_to_hold, COALESCE(ge.cancel_lead_minutes, 180) AS cancel_lead_minutes, COALESCE(ge.cancel_notify_enabled, FALSE) AS cancel_notify_enabled, COALESCE(ge.settlement_enabled, TRUE) AS settlement_enabled, COALESCE(ge.settlement_publish_before, FALSE) AS settlement_publish_before, COALESCE(ge.settlement_publish_after, TRUE) AS settlement_publish_after, ge.cost_amount, COALESCE(ge.is_active, TRUE) AS is_active, COALESCE(pt.name, '') AS poll_template").
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
		Select("ge.id, ge.name, COALESCE(ge.event_type, 'training') AS event_type, ge.start_weekday, COALESCE(ge.poll_publish_weekday, ge.start_weekday) AS poll_publish_weekday, COALESCE(ge.poll_publish_time, ge.start_time) AS poll_publish_time, ge.start_time, ge.end_time, COALESCE(ge.announcement_text, '') AS announcement_text, COALESCE(ge.announcement_enabled, FALSE) AS announcement_enabled, COALESCE(ge.announcement_lead_minutes, 60) AS announcement_lead_minutes, COALESCE(ge.publish_enabled, TRUE) AS publish_enabled, COALESCE(ge.teams_auto_split, FALSE) AS teams_auto_split, COALESCE(ge.teams_publish_list, FALSE) AS teams_publish_list, COALESCE(ge.team_size, 6) AS team_size, COALESCE(ge.min_votes_to_hold, 0) AS min_votes_to_hold, COALESCE(ge.cancel_lead_minutes, 180) AS cancel_lead_minutes, COALESCE(ge.cancel_notify_enabled, FALSE) AS cancel_notify_enabled, COALESCE(ge.settlement_enabled, TRUE) AS settlement_enabled, COALESCE(ge.settlement_publish_before, FALSE) AS settlement_publish_before, COALESCE(ge.settlement_publish_after, TRUE) AS settlement_publish_after, ge.cost_amount, COALESCE(ge.is_active, TRUE) AS is_active, COALESCE(pt.name, '') AS poll_template").
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
		Select("ge.id, ge.name, COALESCE(ge.event_type, 'training') AS event_type, ge.start_weekday, COALESCE(ge.poll_publish_weekday, ge.start_weekday) AS poll_publish_weekday, COALESCE(ge.poll_publish_time, ge.start_time) AS poll_publish_time, ge.start_time, ge.end_time, COALESCE(ge.announcement_text, '') AS announcement_text, COALESCE(ge.announcement_enabled, FALSE) AS announcement_enabled, COALESCE(ge.announcement_lead_minutes, 60) AS announcement_lead_minutes, COALESCE(ge.publish_enabled, TRUE) AS publish_enabled, COALESCE(ge.teams_auto_split, FALSE) AS teams_auto_split, COALESCE(ge.teams_publish_list, FALSE) AS teams_publish_list, COALESCE(ge.team_size, 6) AS team_size, COALESCE(ge.min_votes_to_hold, 0) AS min_votes_to_hold, COALESCE(ge.cancel_lead_minutes, 180) AS cancel_lead_minutes, COALESCE(ge.cancel_notify_enabled, FALSE) AS cancel_notify_enabled, COALESCE(ge.settlement_enabled, TRUE) AS settlement_enabled, COALESCE(ge.settlement_publish_before, FALSE) AS settlement_publish_before, COALESCE(ge.settlement_publish_after, TRUE) AS settlement_publish_after, ge.cost_amount, COALESCE(ge.is_active, TRUE) AS is_active, COALESCE(pt.name, '') AS poll_template").
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

func (s *Store) ListGroupMembersByChatID(ctx context.Context, chatID int64) ([]GroupMemberView, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	var rows []GroupMemberView
	if err := s.db.WithContext(ctx).
		Table("group_members gm").
		Select("gm.user_telegram_id, COALESCE(tu.username, '') AS username, COALESCE(tu.first_name, '') AS first_name, COALESCE(tu.last_name, '') AS last_name, COALESCE(gm.player_type, '') AS player_type, gm.role, gm.status, gm.last_seen_at").
		Joins("LEFT JOIN telegram_users tu ON tu.telegram_id = gm.user_telegram_id").
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

	post := EventPollPost{
		GroupID:           group.ID,
		EventID:           eventID,
		TemplateID:        template.ID,
		TelegramMessageID: telegramMessageID,
		TelegramPollID:    strings.TrimSpace(telegramPollID),
		Status:            "open",
		PublishedAt:       time.Now().UTC(),
	}

	if eventID == nil {
		if err := s.db.WithContext(ctx).Create(&post).Error; err != nil {
			return nil, err
		}
		return &post, nil
	}

	var event GroupEvent
	if err := s.db.WithContext(ctx).
		Where("id = ? AND group_id = ? AND is_active = TRUE", *eventID, group.ID).
		First(&event).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("event not found")
		}
		return nil, err
	}

	loc, err := time.LoadLocation(group.Timezone)
	if err != nil {
		loc = time.UTC
	}
	startHour, startMinute, err := parseClockTime(event.StartTime)
	if err != nil {
		return nil, err
	}
	endHour, endMinute, err := parseClockTime(event.EndTime)
	if err != nil {
		return nil, err
	}

	nowLocal := post.PublishedAt.In(loc)
	plannedStartLocal := nextWeekdayTime(nowLocal, int(event.StartWeekday), startHour, startMinute)
	plannedEndLocal := time.Date(
		plannedStartLocal.Year(),
		plannedStartLocal.Month(),
		plannedStartLocal.Day(),
		endHour,
		endMinute,
		0,
		0,
		loc,
	)
	if !plannedEndLocal.After(plannedStartLocal) {
		plannedEndLocal = plannedEndLocal.Add(24 * time.Hour)
	}
	localDate := time.Date(plannedStartLocal.Year(), plannedStartLocal.Month(), plannedStartLocal.Day(), 0, 0, 0, 0, loc)

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		instance, err := ensureEventInstanceTx(
			ctx,
			tx,
			group.ID,
			event.ID,
			localDate,
			plannedStartLocal,
			plannedEndLocal,
			string(EventHistoryStatusInVoting),
			&template,
		)
		if err != nil {
			return err
		}
		var existingPostID uint64
		if err := tx.WithContext(ctx).
			Table("event_poll_posts").
			Select("id").
			Where("instance_id = ?", instance.ID).
			Order("id DESC").
			Limit(1).
			Scan(&existingPostID).Error; err != nil {
			return err
		}
		if existingPostID != 0 {
			return errors.New("poll already exists for this event instance")
		}
		post.InstanceID = &instance.ID
		if err := tx.Create(&post).Error; err != nil {
			return err
		}
		return tx.Table("event_instances").
			Where("id = ?", instance.ID).
			Updates(map[string]interface{}{
				"poll_post_id": post.ID,
				"updated_at":   gorm.Expr("NOW()"),
			}).Error
	}); err != nil {
		return nil, err
	}
	return &post, nil
}

func (s *Store) CreateEventPollPostForInstance(
	ctx context.Context,
	chatID int64,
	eventID uint64,
	instanceID uint64,
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

	var event GroupEvent
	if err := s.db.WithContext(ctx).
		Where("id = ? AND group_id = ? AND is_active = TRUE", eventID, group.ID).
		First(&event).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("event not found")
		}
		return nil, err
	}

	var instance EventInstance
	if err := s.db.WithContext(ctx).
		Where("id = ? AND group_id = ? AND event_id = ? AND is_active = TRUE", instanceID, group.ID, event.ID).
		First(&instance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("event instance not found")
		}
		return nil, err
	}
	if instance.PollPostID != nil {
		return nil, errors.New("poll already exists for this event instance")
	}

	post := EventPollPost{
		GroupID:           group.ID,
		EventID:           &event.ID,
		InstanceID:        &instance.ID,
		TemplateID:        template.ID,
		TelegramMessageID: telegramMessageID,
		TelegramPollID:    strings.TrimSpace(telegramPollID),
		Status:            "open",
		PublishedAt:       time.Now().UTC(),
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&post).Error; err != nil {
			return err
		}
		return tx.Table("event_instances").
			Where("id = ?", instance.ID).
			Updates(map[string]interface{}{
				"poll_post_id": post.ID,
				"status":       string(EventHistoryStatusInVoting),
				"updated_at":   gorm.Expr("NOW()"),
			}).Error
	}); err != nil {
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
		if err := ensureDefaultMemberSkillsTx(tx, post.GroupID, userID); err != nil {
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
		Select("epp.id, epp.group_id, epp.event_id, epp.instance_id, epp.template_id, pt.name AS template_name, epp.telegram_message_id, epp.telegram_poll_id, epp.status, epp.published_at").
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
		Where("ge.group_id = ? AND ge.is_active = TRUE AND ge.poll_publish_weekday = ? AND pt.name = ? AND pt.is_active = TRUE", group.ID, weekday, strings.TrimSpace(templateName)).
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
		Select("ge.id AS event_id, ge.group_id, g.chat_id, g.timezone, ge.name, COALESCE(ge.event_type, 'training') AS event_type, ge.start_weekday, COALESCE(ge.poll_publish_weekday, ge.start_weekday) AS poll_publish_weekday, COALESCE(ge.poll_publish_time, ge.start_time) AS poll_publish_time, ge.start_time, ge.end_time, COALESCE(ge.announcement_text, '') AS announcement_text, COALESCE(ge.announcement_enabled, FALSE) AS announcement_enabled, COALESCE(ge.announcement_lead_minutes, 60) AS announcement_lead_minutes, COALESCE(ge.publish_enabled, TRUE) AS publish_enabled, COALESCE(ge.teams_auto_split, FALSE) AS teams_auto_split, COALESCE(ge.teams_publish_list, FALSE) AS teams_publish_list, COALESCE(ge.team_size, 6) AS team_size, COALESCE(ge.min_votes_to_hold, 0) AS min_votes_to_hold, COALESCE(ge.cancel_lead_minutes, 180) AS cancel_lead_minutes, COALESCE(ge.cancel_notify_enabled, FALSE) AS cancel_notify_enabled, COALESCE(ge.settlement_enabled, TRUE) AS settlement_enabled, COALESCE(ge.settlement_publish_before, FALSE) AS settlement_publish_before, COALESCE(ge.settlement_publish_after, TRUE) AS settlement_publish_after, ge.cost_amount, COALESCE(pt.name, '') AS poll_template").
		Joins("JOIN telegram_groups g ON g.id = ge.group_id").
		Joins("LEFT JOIN poll_templates pt ON pt.id = ge.poll_template_id").
		Where("ge.is_active = TRUE AND ge.publish_enabled = TRUE AND g.is_active = TRUE").
		Order("ge.id ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Store) SetEventPublishEnabled(ctx context.Context, chatID int64, eventID uint64, enabled bool) error {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return err
	}

	result := s.db.WithContext(ctx).
		Model(&GroupEvent{}).
		Where("id = ? AND group_id = ? AND is_active = TRUE", eventID, group.ID).
		Updates(map[string]interface{}{
			"publish_enabled": enabled,
			"updated_at":      gorm.Expr("NOW()"),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("event not found")
	}
	return nil
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

func (s *Store) HasEventAnnouncement(ctx context.Context, eventID uint64, eventDate time.Time) (bool, error) {
	var count int64
	if err := s.db.WithContext(ctx).
		Table("event_announcements").
		Where("event_id = ? AND event_date = ?", eventID, eventDate.Format("2006-01-02")).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *Store) CreateEventAnnouncement(ctx context.Context, groupID uint64, eventID uint64, eventDate time.Time) error {
	record := map[string]interface{}{
		"group_id":   groupID,
		"event_id":   eventID,
		"event_date": eventDate.Format("2006-01-02"),
		"sent_at":    gorm.Expr("NOW()"),
		"created_at": gorm.Expr("NOW()"),
		"updated_at": gorm.Expr("NOW()"),
	}
	return s.db.WithContext(ctx).Table("event_announcements").Create(record).Error
}

func (s *Store) HasEventCancellation(ctx context.Context, eventID uint64, eventDate time.Time) (bool, error) {
	var count int64
	if err := s.db.WithContext(ctx).
		Table("event_cancellations").
		Where("event_id = ? AND event_date = ?", eventID, eventDate.Format("2006-01-02")).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *Store) CreateEventCancellation(ctx context.Context, groupID uint64, eventID uint64, eventDate time.Time) error {
	record := map[string]interface{}{
		"group_id":   groupID,
		"event_id":   eventID,
		"event_date": eventDate.Format("2006-01-02"),
		"sent_at":    gorm.Expr("NOW()"),
		"created_at": gorm.Expr("NOW()"),
		"updated_at": gorm.Expr("NOW()"),
	}
	return s.db.WithContext(ctx).Table("event_cancellations").Create(record).Error
}

func (s *Store) HasEventSettlementNotice(ctx context.Context, eventID uint64, localDate time.Time, noticeType string) (bool, error) {
	var count int64
	if err := s.db.WithContext(ctx).
		Table("event_settlement_notices").
		Where("event_id = ? AND local_date = ? AND notice_type = ?", eventID, localDate.Format("2006-01-02"), strings.TrimSpace(noticeType)).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *Store) CreateEventSettlementNotice(ctx context.Context, groupID uint64, eventID uint64, localDate time.Time, noticeType string) error {
	record := map[string]interface{}{
		"group_id":    groupID,
		"event_id":    eventID,
		"local_date":  localDate.Format("2006-01-02"),
		"notice_type": strings.TrimSpace(noticeType),
		"sent_at":     gorm.Expr("NOW()"),
		"created_at":  gorm.Expr("NOW()"),
		"updated_at":  gorm.Expr("NOW()"),
	}
	return s.db.WithContext(ctx).Table("event_settlement_notices").Create(record).Error
}

func (s *Store) GetLatestEventPollPostForRange(ctx context.Context, eventID uint64, fromUTC, toUTC time.Time) (*EventPollPostView, error) {
	var row EventPollPostView
	if err := s.db.WithContext(ctx).
		Table("event_poll_posts epp").
		Select("epp.id, epp.group_id, epp.event_id, epp.instance_id, epp.template_id, pt.name AS template_name, epp.telegram_message_id, epp.telegram_poll_id, epp.status, epp.published_at").
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
		Select("epp.id, epp.group_id, epp.event_id, epp.instance_id, epp.template_id, pt.name AS template_name, epp.telegram_message_id, epp.telegram_poll_id, epp.status, epp.published_at").
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

func (s *Store) EnsureEventInstanceForDate(
	ctx context.Context,
	groupID uint64,
	eventID uint64,
	localDate time.Time,
	startTime string,
	endTime string,
	status string,
	timezone string,
) (*EventInstance, error) {
	loc, err := time.LoadLocation(strings.TrimSpace(timezone))
	if err != nil {
		loc = time.UTC
	}
	startHour, startMinute, err := parseClockTime(startTime)
	if err != nil {
		return nil, err
	}
	endHour, endMinute, err := parseClockTime(endTime)
	if err != nil {
		return nil, err
	}
	startLocal := time.Date(localDate.Year(), localDate.Month(), localDate.Day(), startHour, startMinute, 0, 0, loc)
	endLocal := time.Date(localDate.Year(), localDate.Month(), localDate.Day(), endHour, endMinute, 0, 0, loc)
	if !endLocal.After(startLocal) {
		endLocal = endLocal.Add(24 * time.Hour)
	}

	var instance *EventInstance
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		row, err := ensureEventInstanceTx(ctx, tx, groupID, eventID, localDate, startLocal, endLocal, status, nil)
		if err != nil {
			return err
		}
		instance = row
		return nil
	}); err != nil {
		return nil, err
	}
	return instance, nil
}

type EventInstanceSnapshotView struct {
	InstanceID     uint64
	GroupID        uint64
	EventID        uint64
	EventName      string
	Timezone       string
	ChatID         int64
	LocalDate      time.Time
	PlannedStartAt time.Time
	PlannedEndAt   time.Time
	PollPostID     *uint64

	CostAmount              *float64
	MinVotesToHold          int
	CancelLeadMinutes       int
	CancelNotifyEnabled     bool
	SettlementEnabled       bool
	SettlementPublishBefore bool
	SettlementPublishAfter  bool

	PollCountedOptions datatypes.JSON
}

func (s *Store) GetEventInstanceSnapshotByEventDate(ctx context.Context, groupID uint64, eventID uint64, localDate time.Time) (*EventInstanceSnapshotView, error) {
	type row struct {
		InstanceID              uint64
		GroupID                 uint64
		EventID                 uint64
		EventName               string
		Timezone                string
		ChatID                  int64
		LocalDate               time.Time
		PlannedStartAt          time.Time
		PlannedEndAt            time.Time
		PollPostID              *uint64
		CostAmount              *float64
		MinVotesToHold          int
		CancelLeadMinutes       int
		CancelNotifyEnabled     bool
		SettlementEnabled       bool
		SettlementPublishBefore bool
		SettlementPublishAfter  bool
		PollCountedOptions      datatypes.JSON
	}
	var r row
	err := s.db.WithContext(ctx).
		Table("event_instances ei").
		Select(`
			ei.id AS instance_id,
			ei.group_id,
			ei.event_id,
			ei.event_name,
			g.timezone,
			g.chat_id,
			ei.local_date,
			ei.planned_start_at,
			ei.planned_end_at,
			ei.poll_post_id,
			ei.cost_amount,
			ei.min_votes_to_hold,
			ei.cancel_lead_minutes,
			ei.cancel_notify_enabled,
			ei.settlement_enabled,
			ei.settlement_publish_before,
			ei.settlement_publish_after,
			COALESCE(ei.poll_counted_options, '[]'::jsonb) AS poll_counted_options
		`).
		Joins("JOIN telegram_groups g ON g.id = ei.group_id").
		Where("ei.group_id = ? AND ei.event_id = ? AND ei.local_date = ? AND ei.is_active = TRUE AND g.is_active = TRUE", groupID, eventID, localDate.Format("2006-01-02")).
		Take(&r).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &EventInstanceSnapshotView{
		InstanceID:              r.InstanceID,
		GroupID:                 r.GroupID,
		EventID:                 r.EventID,
		EventName:               r.EventName,
		Timezone:                r.Timezone,
		ChatID:                  r.ChatID,
		LocalDate:               r.LocalDate,
		PlannedStartAt:          r.PlannedStartAt,
		PlannedEndAt:            r.PlannedEndAt,
		PollPostID:              r.PollPostID,
		CostAmount:              r.CostAmount,
		MinVotesToHold:          r.MinVotesToHold,
		CancelLeadMinutes:       r.CancelLeadMinutes,
		CancelNotifyEnabled:     r.CancelNotifyEnabled,
		SettlementEnabled:       r.SettlementEnabled,
		SettlementPublishBefore: r.SettlementPublishBefore,
		SettlementPublishAfter:  r.SettlementPublishAfter,
		PollCountedOptions:      r.PollCountedOptions,
	}, nil
}

func (s *Store) ListEventInstancesForCancellation(ctx context.Context, nowUTC time.Time) ([]EventInstanceSnapshotView, error) {
	untilUTC := nowUTC.Add(24 * time.Hour)
	type row struct {
		InstanceID     uint64
		GroupID        uint64
		EventID        uint64
		EventName      string
		Timezone       string
		ChatID         int64
		LocalDate      time.Time
		PlannedStartAt time.Time
		PlannedEndAt   time.Time
		PollPostID     *uint64

		MinVotesToHold      int
		CancelLeadMinutes   int
		CancelNotifyEnabled bool
		PollCountedOptions  datatypes.JSON
	}
	var rows []row
	if err := s.db.WithContext(ctx).
		Table("event_instances ei").
		Select(`
			ei.id AS instance_id,
			ei.group_id,
			ei.event_id,
			ei.event_name,
			g.timezone,
			g.chat_id,
			ei.local_date,
			ei.planned_start_at,
			ei.planned_end_at,
			ei.poll_post_id,
			ei.min_votes_to_hold,
			ei.cancel_lead_minutes,
			ei.cancel_notify_enabled,
			COALESCE(ei.poll_counted_options, '[]'::jsonb) AS poll_counted_options
		`).
		Joins("JOIN telegram_groups g ON g.id = ei.group_id").
		Where(`
			ei.is_active = TRUE
			AND g.is_active = TRUE
			AND ei.cancel_notify_enabled = TRUE
			AND ei.min_votes_to_hold > 0
			AND ei.planned_start_at > ?
			AND ei.planned_start_at <= ?
			AND ei.status IN ('in_voting', 'on_distribution', 'on_review')
		`, nowUTC, untilUTC).
		Order("ei.planned_start_at ASC, ei.id ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]EventInstanceSnapshotView, 0, len(rows))
	for _, r := range rows {
		out = append(out, EventInstanceSnapshotView{
			InstanceID:          r.InstanceID,
			GroupID:             r.GroupID,
			EventID:             r.EventID,
			EventName:           r.EventName,
			Timezone:            r.Timezone,
			ChatID:              r.ChatID,
			LocalDate:           r.LocalDate,
			PlannedStartAt:      r.PlannedStartAt,
			PlannedEndAt:        r.PlannedEndAt,
			PollPostID:          r.PollPostID,
			MinVotesToHold:      r.MinVotesToHold,
			CancelLeadMinutes:   r.CancelLeadMinutes,
			CancelNotifyEnabled: r.CancelNotifyEnabled,
			PollCountedOptions:  r.PollCountedOptions,
		})
	}
	return out, nil
}

func (s *Store) SetEventInstanceStatusByEventDate(ctx context.Context, groupID uint64, eventID uint64, localDate time.Time, status string) error {
	status = clampInstanceStatus(status)
	result := s.db.WithContext(ctx).
		Table("event_instances").
		Where("group_id = ? AND event_id = ? AND local_date = ? AND is_active = TRUE", groupID, eventID, localDate.Format("2006-01-02")).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": gorm.Expr("NOW()"),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("event instance not found")
	}
	return nil
}

func (s *Store) CreateEventInstanceFromTemplate(ctx context.Context, chatID int64, eventID uint64, localDateRaw string) (*EventInstance, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}
	var event GroupEvent
	if err := s.db.WithContext(ctx).
		Where("id = ? AND group_id = ? AND is_active = TRUE", eventID, group.ID).
		First(&event).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("event not found")
		}
		return nil, err
	}

	loc, err := time.LoadLocation(group.Timezone)
	if err != nil {
		loc = time.UTC
	}
	startHour, startMinute, err := parseClockTime(event.StartTime)
	if err != nil {
		return nil, err
	}
	nowLocal := time.Now().In(loc)

	var localDate time.Time
	trimmed := strings.TrimSpace(localDateRaw)
	if trimmed == "" {
		nextStartLocal := nextWeekdayTime(nowLocal, int(event.StartWeekday), startHour, startMinute)
		localDate = time.Date(nextStartLocal.Year(), nextStartLocal.Month(), nextStartLocal.Day(), 0, 0, 0, 0, loc)
		for i := 0; i < 104; i++ {
			var existing struct {
				ID         uint64
				PollPostID *uint64
			}
			err := s.db.WithContext(ctx).
				Table("event_instances").
				Select("id, poll_post_id").
				Where("group_id = ? AND event_id = ? AND local_date = ? AND is_active = TRUE", group.ID, event.ID, localDate.Format("2006-01-02")).
				Take(&existing).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				break
			}
			if err != nil {
				return nil, err
			}
			if existing.PollPostID == nil {
				break
			}
			localDate = localDate.AddDate(0, 0, 7)
		}
	} else {
		parsed, err := time.ParseInLocation("2006-01-02", trimmed, loc)
		if err != nil {
			return nil, errors.New("invalid localDate format, expected YYYY-MM-DD")
		}
		localDate = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, loc)
	}

	return s.EnsureEventInstanceForDate(
		ctx,
		group.ID,
		event.ID,
		localDate,
		event.StartTime,
		event.EndTime,
		string(EventHistoryStatusInVoting),
		group.Timezone,
	)
}

func (s *Store) CreateEventSettlement(
	ctx context.Context,
	groupID uint64,
	eventID uint64,
	instanceID *uint64,
	postID *uint64,
	localDate time.Time,
	totalAmount float64,
	participants int,
	amountPerPerson float64,
) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		record := map[string]interface{}{
			"group_id":           groupID,
			"event_id":           eventID,
			"instance_id":        instanceID,
			"post_id":            postID,
			"local_date":         localDate.Format("2006-01-02"),
			"total_amount":       totalAmount,
			"participants_count": participants,
			"amount_per_person":  amountPerPerson,
			"sent_at":            gorm.Expr("NOW()"),
		}
		if err := tx.Table("event_settlements").Create(record).Error; err != nil {
			return err
		}

		var created struct {
			ID uint64
		}
		if err := tx.Table("event_settlements").
			Select("id").
			Where("event_id = ? AND local_date = ?", eventID, localDate.Format("2006-01-02")).
			Order("id DESC").
			Take(&created).Error; err != nil {
			return err
		}
		return s.ensureSettlementPaymentsTx(ctx, tx, created.ID)
	})
}

func (s *Store) ensureSettlementPaymentsTx(ctx context.Context, tx *gorm.DB, settlementID uint64) error {
	if settlementID == 0 {
		return errors.New("settlement id is required")
	}
	var settlement struct {
		ID              uint64
		PostID          *uint64
		InstanceID      *uint64
		AmountPerPerson float64
	}
	if err := tx.WithContext(ctx).
		Table("event_settlements").
		Select("id, post_id, instance_id, amount_per_person").
		Where("id = ?", settlementID).
		Take(&settlement).Error; err != nil {
		return err
	}
	if settlement.PostID == nil {
		return nil
	}

	var options []string
	var counted []int
	if settlement.InstanceID != nil {
		var snap struct {
			Options        datatypes.JSON
			CountedOptions datatypes.JSON
		}
		if err := tx.WithContext(ctx).
			Table("event_instances").
			Select("COALESCE(poll_options, '[]'::jsonb) AS options, COALESCE(poll_counted_options, '[]'::jsonb) AS counted_options").
			Where("id = ?", *settlement.InstanceID).
			Take(&snap).Error; err != nil {
			return err
		}
		_ = json.Unmarshal(snap.Options, &options)
		_ = json.Unmarshal(snap.CountedOptions, &counted)
	} else {
		var template struct {
			Options        datatypes.JSON
			CountedOptions datatypes.JSON
		}
		if err := tx.WithContext(ctx).
			Table("event_poll_posts epp").
			Select("COALESCE(pt.options, '[]'::jsonb) AS options, COALESCE(pt.counted_options, '[]'::jsonb) AS counted_options").
			Joins("JOIN poll_templates pt ON pt.id = epp.template_id").
			Where("epp.id = ?", *settlement.PostID).
			Take(&template).Error; err != nil {
			return err
		}
		_ = json.Unmarshal(template.Options, &options)
		_ = json.Unmarshal(template.CountedOptions, &counted)
	}
	counted = normalizeCountedOptionIndexes(len(options), counted)
	if len(counted) == 0 {
		return nil
	}

	choices := make([]string, 0, len(counted))
	for _, idx := range counted {
		choices = append(choices, "option_"+strconv.Itoa(idx))
	}

	type voteRow struct {
		UserID    int64
		Username  string
		FirstName string
		LastName  string
	}
	var rows []voteRow
	if err := tx.WithContext(ctx).
		Table("event_poll_votes").
		Select("DISTINCT user_id, COALESCE(username, '') AS username, COALESCE(first_name, '') AS first_name, COALESCE(last_name, '') AS last_name").
		Where("post_id = ? AND choice IN ?", *settlement.PostID, choices).
		Scan(&rows).Error; err != nil {
		return err
	}

	for _, row := range rows {
		rec := map[string]interface{}{
			"settlement_id": settlementID,
			"user_id":       row.UserID,
			"username":      strings.TrimSpace(row.Username),
			"first_name":    strings.TrimSpace(row.FirstName),
			"last_name":     strings.TrimSpace(row.LastName),
			"amount_due":    settlement.AmountPerPerson,
			"is_paid":       false,
		}
		if err := tx.WithContext(ctx).
			Table("event_settlement_payments").
			Clauses(clause.OnConflict{
				Columns: []clause.Column{
					{Name: "settlement_id"},
					{Name: "user_id"},
				},
				DoNothing: true,
			}).
			Create(rec).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) GetEventBilling(ctx context.Context, chatID int64, eventID uint64) (*EventBillingView, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}
	var settlement struct {
		ID                uint64
		EventID           uint64
		LocalDate         time.Time
		TotalAmount       float64
		ParticipantsCount int
		AmountPerPerson   float64
	}
	if err := s.db.WithContext(ctx).
		Table("event_settlements").
		Select("id, event_id, local_date, total_amount, participants_count, amount_per_person").
		Where("group_id = ? AND event_id = ?", group.ID, eventID).
		Order("local_date DESC, id DESC").
		Take(&settlement).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.ensureSettlementPaymentsTx(ctx, tx, settlement.ID)
	}); err != nil {
		return nil, err
	}

	type paymentRow struct {
		UserID    int64
		Username  string
		FirstName string
		LastName  string
		AmountDue float64
		IsPaid    bool
		PaidAt    *time.Time
	}
	var rows []paymentRow
	if err := s.db.WithContext(ctx).
		Table("event_settlement_payments").
		Select("user_id, COALESCE(username, '') AS username, COALESCE(first_name, '') AS first_name, COALESCE(last_name, '') AS last_name, amount_due, is_paid, paid_at").
		Where("settlement_id = ?", settlement.ID).
		Order("first_name ASC, last_name ASC, username ASC, user_id ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	players := make([]EventBillingParticipant, 0, len(rows))
	paidCount := 0
	unpaidCount := 0
	debt := 0.0
	for _, row := range rows {
		players = append(players, EventBillingParticipant{
			UserID:    row.UserID,
			Username:  row.Username,
			FirstName: row.FirstName,
			LastName:  row.LastName,
			AmountDue: row.AmountDue,
			IsPaid:    row.IsPaid,
			PaidAt:    row.PaidAt,
		})
		if row.IsPaid {
			paidCount++
		} else {
			unpaidCount++
			debt += row.AmountDue
		}
	}

	return &EventBillingView{
		EventID:            eventID,
		SettlementID:       settlement.ID,
		LocalDate:          settlement.LocalDate,
		TotalAmount:        settlement.TotalAmount,
		AmountPerPerson:    settlement.AmountPerPerson,
		ParticipantsCount:  settlement.ParticipantsCount,
		PaidCount:          paidCount,
		UnpaidCount:        unpaidCount,
		DebtAmount:         debt,
		Players:            players,
		AllPaymentsChecked: len(players) > 0 && unpaidCount == 0,
	}, nil
}

func (s *Store) GetEventBillingByInstance(ctx context.Context, chatID int64, instanceID uint64) (*EventBillingView, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}
	var instance struct {
		ID      uint64
		EventID uint64
	}
	if err := s.db.WithContext(ctx).
		Table("event_instances").
		Select("id, event_id").
		Where("id = ? AND group_id = ? AND is_active = TRUE", instanceID, group.ID).
		Take(&instance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	var settlement struct {
		ID                uint64
		EventID           uint64
		LocalDate         time.Time
		TotalAmount       float64
		ParticipantsCount int
		AmountPerPerson   float64
	}
	if err := s.db.WithContext(ctx).
		Table("event_settlements").
		Select("id, event_id, local_date, total_amount, participants_count, amount_per_person").
		Where("group_id = ? AND instance_id = ?", group.ID, instance.ID).
		Order("id DESC").
		Take(&settlement).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.ensureSettlementPaymentsTx(ctx, tx, settlement.ID)
	}); err != nil {
		return nil, err
	}

	type paymentRow struct {
		UserID    int64
		Username  string
		FirstName string
		LastName  string
		AmountDue float64
		IsPaid    bool
		PaidAt    *time.Time
	}
	var rows []paymentRow
	if err := s.db.WithContext(ctx).
		Table("event_settlement_payments").
		Select("user_id, COALESCE(username, '') AS username, COALESCE(first_name, '') AS first_name, COALESCE(last_name, '') AS last_name, amount_due, is_paid, paid_at").
		Where("settlement_id = ?", settlement.ID).
		Order("first_name ASC, last_name ASC, username ASC, user_id ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	players := make([]EventBillingParticipant, 0, len(rows))
	paidCount := 0
	unpaidCount := 0
	debt := 0.0
	for _, row := range rows {
		players = append(players, EventBillingParticipant{
			UserID:    row.UserID,
			Username:  row.Username,
			FirstName: row.FirstName,
			LastName:  row.LastName,
			AmountDue: row.AmountDue,
			IsPaid:    row.IsPaid,
			PaidAt:    row.PaidAt,
		})
		if row.IsPaid {
			paidCount++
		} else {
			unpaidCount++
			debt += row.AmountDue
		}
	}

	return &EventBillingView{
		EventID:            settlement.EventID,
		SettlementID:       settlement.ID,
		LocalDate:          settlement.LocalDate,
		TotalAmount:        settlement.TotalAmount,
		AmountPerPerson:    settlement.AmountPerPerson,
		ParticipantsCount:  settlement.ParticipantsCount,
		PaidCount:          paidCount,
		UnpaidCount:        unpaidCount,
		DebtAmount:         debt,
		Players:            players,
		AllPaymentsChecked: len(players) > 0 && unpaidCount == 0,
	}, nil
}

func (s *Store) EnsureEventBillingByInstance(ctx context.Context, chatID int64, instanceID uint64) (*EventBillingView, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	var inst struct {
		ID             uint64
		GroupID        uint64
		EventID        uint64
		LocalDate      time.Time
		PollPostID     *uint64
		CostAmount     *float64
		PollOptions    datatypes.JSON
		CountedOptions datatypes.JSON
		Status         string
	}
	if err := s.db.WithContext(ctx).
		Table("event_instances").
		Select("id, group_id, event_id, local_date, poll_post_id, cost_amount, COALESCE(poll_options, '[]'::jsonb) AS poll_options, COALESCE(poll_counted_options, '[]'::jsonb) AS counted_options, status").
		Where("id = ? AND group_id = ? AND is_active = TRUE", instanceID, group.ID).
		Take(&inst).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("event instance not found")
		}
		return nil, err
	}
	if strings.TrimSpace(inst.Status) == string(EventHistoryStatusNotHeld) {
		return nil, errors.New("event instance is not held")
	}
	if inst.PollPostID == nil {
		return nil, errors.New("no published poll for this event instance")
	}

	var options []string
	_ = json.Unmarshal(inst.PollOptions, &options)
	var counted []int
	_ = json.Unmarshal(inst.CountedOptions, &counted)
	counted = normalizeCountedOptionIndexes(len(options), counted)
	if len(counted) == 0 {
		return nil, errors.New("template has no options marked with accounting flag")
	}
	choices := make([]string, 0, len(counted))
	for _, idx := range counted {
		choices = append(choices, "option_"+strconv.Itoa(idx))
	}
	participants, err := s.CountVotesForPostChoices(ctx, *inst.PollPostID, choices)
	if err != nil {
		return nil, err
	}

	totalAmount := 4000.0
	if inst.CostAmount != nil {
		totalAmount = *inst.CostAmount
	}
	perPerson := 0.0
	if participants > 0 {
		perPerson = totalAmount / float64(participants)
	}

	// If settlement already exists, return it (and ensure missing payment rows exist).
	existing, err := s.GetEventBillingByInstance(ctx, chatID, instanceID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	if err := s.CreateEventSettlement(ctx, group.ID, inst.EventID, &inst.ID, inst.PollPostID, inst.LocalDate, totalAmount, participants, perPerson); err != nil {
		return nil, err
	}
	return s.GetEventBillingByInstance(ctx, chatID, instanceID)
}

func (s *Store) SaveEventBillingPayments(ctx context.Context, chatID int64, eventID uint64, statuses map[int64]bool) error {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return err
	}
	var settlement struct {
		ID uint64
	}
	if err := s.db.WithContext(ctx).
		Table("event_settlements").
		Select("id").
		Where("group_id = ? AND event_id = ?", group.ID, eventID).
		Order("local_date DESC, id DESC").
		Take(&settlement).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("settlement not found")
		}
		return err
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.ensureSettlementPaymentsTx(ctx, tx, settlement.ID); err != nil {
			return err
		}

		for userID, isPaid := range statuses {
			updates := map[string]interface{}{
				"is_paid":    isPaid,
				"updated_at": gorm.Expr("NOW()"),
			}
			if isPaid {
				updates["paid_at"] = gorm.Expr("NOW()")
			} else {
				updates["paid_at"] = gorm.Expr("NULL")
			}
			if err := tx.Table("event_settlement_payments").
				Where("settlement_id = ? AND user_id = ?", settlement.ID, userID).
				Updates(updates).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) SaveEventBillingPaymentsByInstance(ctx context.Context, chatID int64, instanceID uint64, statuses map[int64]bool) error {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return err
	}
	var settlement struct {
		ID      uint64
		EventID uint64
	}
	if err := s.db.WithContext(ctx).
		Table("event_settlements").
		Select("id, event_id").
		Where("group_id = ? AND instance_id = ?", group.ID, instanceID).
		Order("id DESC").
		Take(&settlement).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("settlement not found")
		}
		return err
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.ensureSettlementPaymentsTx(ctx, tx, settlement.ID); err != nil {
			return err
		}
		for userID, isPaid := range statuses {
			updates := map[string]interface{}{
				"is_paid":    isPaid,
				"updated_at": gorm.Expr("NOW()"),
			}
			if isPaid {
				updates["paid_at"] = gorm.Expr("NOW()")
			} else {
				updates["paid_at"] = gorm.Expr("NULL")
			}
			if err := tx.Table("event_settlement_payments").
				Where("settlement_id = ? AND user_id = ?", settlement.ID, userID).
				Updates(updates).Error; err != nil {
				return err
			}
		}
		var unpaid int64
		if err := tx.Table("event_settlement_payments").
			Where("settlement_id = ? AND is_paid = FALSE", settlement.ID).
			Count(&unpaid).Error; err != nil {
			return err
		}
		nextStatus := string(EventHistoryStatusOnReview)
		if unpaid == 0 {
			nextStatus = string(EventHistoryStatusCompleted)
		}
		return tx.Table("event_instances").
			Where("id = ? AND group_id = ?", instanceID, group.ID).
			Updates(map[string]interface{}{
				"status":     nextStatus,
				"updated_at": gorm.Expr("NOW()"),
			}).Error
	})
}

func (s *Store) GetGroupDebtSummary(ctx context.Context, chatID int64) (*GroupDebtSummary, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}
	var out struct {
		TotalDebt  float64
		UnpaidRows int64
	}
	if err := s.db.WithContext(ctx).
		Table("event_settlement_payments esp").
		Select("COALESCE(SUM(esp.amount_due), 0) AS total_debt, COUNT(*) AS unpaid_rows").
		Joins("JOIN event_settlements es ON es.id = esp.settlement_id").
		Where("es.group_id = ? AND esp.is_paid = FALSE", group.ID).
		Scan(&out).Error; err != nil {
		return nil, err
	}
	return &GroupDebtSummary{
		TotalDebt:  out.TotalDebt,
		UnpaidRows: out.UnpaidRows,
	}, nil
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
		Take(&row).Error; err != nil {
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

func (s *Store) GetEventTemplateCountedOptions(ctx context.Context, eventID uint64) ([]int, error) {
	var row struct {
		TemplateOptions datatypes.JSON
		CountedOptions  datatypes.JSON
	}
	if err := s.db.WithContext(ctx).
		Table("group_events ge").
		Select("pt.options AS template_options, pt.counted_options").
		Joins("JOIN poll_templates pt ON pt.id = ge.poll_template_id").
		Where("ge.id = ? AND ge.is_active = TRUE AND pt.is_active = TRUE", eventID).
		Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []int{}, nil
		}
		return nil, err
	}

	var options []string
	if err := json.Unmarshal(row.TemplateOptions, &options); err != nil {
		return nil, err
	}
	var counted []int
	if err := json.Unmarshal(row.CountedOptions, &counted); err != nil {
		return nil, err
	}
	return normalizeCountedOptionIndexes(len(options), counted), nil
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

func (s *Store) GetEventActivitySummary(ctx context.Context, chatID int64, eventID uint64) (*EventActivitySummary, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	var exists int64
	if err := s.db.WithContext(ctx).
		Table("group_events").
		Where("id = ? AND group_id = ?", eventID, group.ID).
		Count(&exists).Error; err != nil {
		return nil, err
	}
	if exists == 0 {
		return nil, errors.New("event not found")
	}

	var pollsTotal int64
	if err := s.db.WithContext(ctx).
		Table("event_poll_posts").
		Where("event_id = ?", eventID).
		Count(&pollsTotal).Error; err != nil {
		return nil, err
	}

	var votesTotal int64
	if err := s.db.WithContext(ctx).
		Table("event_poll_votes ev").
		Joins("JOIN event_poll_posts ep ON ep.id = ev.post_id").
		Where("ep.event_id = ?", eventID).
		Count(&votesTotal).Error; err != nil {
		return nil, err
	}

	var settlementsTotal int64
	if err := s.db.WithContext(ctx).
		Table("event_settlements").
		Where("event_id = ?", eventID).
		Count(&settlementsTotal).Error; err != nil {
		return nil, err
	}

	var announcementsTotal int64
	if err := s.db.WithContext(ctx).
		Table("event_announcements").
		Where("event_id = ?", eventID).
		Count(&announcementsTotal).Error; err != nil {
		return nil, err
	}

	return &EventActivitySummary{
		EventID:            eventID,
		PollsTotal:         pollsTotal,
		VotesTotal:         votesTotal,
		SettlementsTotal:   settlementsTotal,
		AnnouncementsTotal: announcementsTotal,
	}, nil
}

type TeamSplitAssignmentInput struct {
	UserID   int64  `json:"userID"`
	Team     string `json:"team"`
	Position int    `json:"position"`
}

func normalizeTeamValue(team string) string {
	switch strings.TrimSpace(strings.ToUpper(team)) {
	case "A":
		return "A"
	case "B":
		return "B"
	case "C":
		return "C"
	default:
		return "unassigned"
	}
}

func calculateTeamChance(players []TeamSplitPlayer) TeamWinChance {
	var a float64
	var b float64
	for _, player := range players {
		switch player.Team {
		case "A":
			a += player.Rating
		case "B":
			b += player.Rating
		}
	}
	total := a + b
	if total <= 0 {
		return TeamWinChance{
			TeamAScore: a,
			TeamBScore: b,
			TeamAProb:  0.5,
			TeamBProb:  0.5,
		}
	}
	return TeamWinChance{
		TeamAScore: a,
		TeamBScore: b,
		TeamAProb:  a / total,
		TeamBProb:  b / total,
	}
}

func (s *Store) ListEventPollHistory(ctx context.Context, chatID int64, eventID uint64) ([]EventPollHistoryItem, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	var eventExists int64
	if err := s.db.WithContext(ctx).
		Table("group_events").
		Where("id = ? AND group_id = ? AND is_active = TRUE", eventID, group.ID).
		Count(&eventExists).Error; err != nil {
		return nil, err
	}
	if eventExists == 0 {
		return nil, errors.New("event not found")
	}

	type row struct {
		PostID            uint64
		TelegramMessageID int64
		TelegramPollID    string
		Status            string
		PublishedAt       time.Time
		Question          string
		TemplateOptions   datatypes.JSON
		CountedOptions    datatypes.JSON
	}
	var rows []row
	if err := s.db.WithContext(ctx).
		Table("event_poll_posts epp").
		Select("epp.id AS post_id, epp.telegram_message_id, COALESCE(epp.telegram_poll_id, '') AS telegram_poll_id, epp.status, epp.published_at, COALESCE(pt.question, '') AS question, COALESCE(pt.options, '[]'::jsonb) AS template_options, COALESCE(pt.counted_options, '[]'::jsonb) AS counted_options").
		Joins("JOIN poll_templates pt ON pt.id = epp.template_id").
		Where("epp.group_id = ? AND epp.event_id = ?", group.ID, eventID).
		Order("epp.published_at DESC, epp.id DESC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]EventPollHistoryItem, 0, len(rows))
	for _, r := range rows {
		var options []string
		_ = json.Unmarshal(r.TemplateOptions, &options)
		var counted []int
		_ = json.Unmarshal(r.CountedOptions, &counted)
		counted = normalizeCountedOptionIndexes(len(options), counted)

		var totalVotes int64
		if err := s.db.WithContext(ctx).
			Table("event_poll_votes").
			Where("post_id = ?", r.PostID).
			Count(&totalVotes).Error; err != nil {
			return nil, err
		}

		countedVotes := 0
		if len(counted) > 0 {
			choices := make([]string, 0, len(counted))
			for _, idx := range counted {
				choices = append(choices, "option_"+strconv.Itoa(idx))
			}
			countedVotes, err = s.CountVotesForPostChoices(ctx, r.PostID, choices)
			if err != nil {
				return nil, err
			}
		}

		var sessionCount int64
		if err := s.db.WithContext(ctx).
			Table("event_team_sessions").
			Where("post_id = ?", r.PostID).
			Count(&sessionCount).Error; err != nil {
			return nil, err
		}

		items = append(items, EventPollHistoryItem{
			PostID:            r.PostID,
			TelegramMessageID: r.TelegramMessageID,
			TelegramPollID:    r.TelegramPollID,
			Status:            r.Status,
			PublishedAt:       r.PublishedAt,
			Question:          r.Question,
			CountedVotes:      countedVotes,
			TotalVotes:        int(totalVotes),
			TeamsConfigured:   sessionCount > 0,
		})
	}
	return items, nil
}

func (s *Store) ListEventPollHistoryByInstance(ctx context.Context, chatID int64, instanceID uint64) ([]EventPollHistoryItem, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}
	var exists int64
	if err := s.db.WithContext(ctx).
		Table("event_instances").
		Where("id = ? AND group_id = ? AND is_active = TRUE", instanceID, group.ID).
		Count(&exists).Error; err != nil {
		return nil, err
	}
	if exists == 0 {
		return nil, errors.New("event instance not found")
	}

	type row struct {
		PostID            uint64
		TelegramMessageID int64
		TelegramPollID    string
		Status            string
		PublishedAt       time.Time
		Question          string
		TemplateOptions   datatypes.JSON
		CountedOptions    datatypes.JSON
	}
	var rows []row
	if err := s.db.WithContext(ctx).
		Table("event_poll_posts epp").
		Select("epp.id AS post_id, epp.telegram_message_id, COALESCE(epp.telegram_poll_id, '') AS telegram_poll_id, epp.status, epp.published_at, COALESCE(pt.question, '') AS question, COALESCE(pt.options, '[]'::jsonb) AS template_options, COALESCE(pt.counted_options, '[]'::jsonb) AS counted_options").
		Joins("JOIN poll_templates pt ON pt.id = epp.template_id").
		Where("epp.group_id = ? AND epp.instance_id = ?", group.ID, instanceID).
		Order("epp.published_at DESC, epp.id DESC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]EventPollHistoryItem, 0, len(rows))
	for _, r := range rows {
		var options []string
		_ = json.Unmarshal(r.TemplateOptions, &options)
		var counted []int
		_ = json.Unmarshal(r.CountedOptions, &counted)
		counted = normalizeCountedOptionIndexes(len(options), counted)

		var totalVotes int64
		if err := s.db.WithContext(ctx).
			Table("event_poll_votes").
			Where("post_id = ?", r.PostID).
			Count(&totalVotes).Error; err != nil {
			return nil, err
		}

		countedVotes := 0
		if len(counted) > 0 {
			choices := make([]string, 0, len(counted))
			for _, idx := range counted {
				choices = append(choices, "option_"+strconv.Itoa(idx))
			}
			countedVotes, err = s.CountVotesForPostChoices(ctx, r.PostID, choices)
			if err != nil {
				return nil, err
			}
		}

		var sessionCount int64
		if err := s.db.WithContext(ctx).
			Table("event_team_sessions").
			Where("post_id = ?", r.PostID).
			Count(&sessionCount).Error; err != nil {
			return nil, err
		}

		items = append(items, EventPollHistoryItem{
			PostID:            r.PostID,
			TelegramMessageID: r.TelegramMessageID,
			TelegramPollID:    r.TelegramPollID,
			Status:            r.Status,
			PublishedAt:       r.PublishedAt,
			Question:          r.Question,
			CountedVotes:      countedVotes,
			TotalVotes:        int(totalVotes),
			TeamsConfigured:   sessionCount > 0,
		})
	}
	return items, nil
}

func (s *Store) ListGroupPollsByChatID(ctx context.Context, chatID int64) ([]GroupPollItem, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}
	type row struct {
		PostID            uint64
		InstanceID        *uint64
		EventID           *uint64
		EventName         string
		EventLocalDate    *time.Time
		TemplateName      string
		Question          string
		TelegramMessageID int64
		TelegramPollID    string
		Status            string
		PublishedAt       time.Time
		TemplateOptions   datatypes.JSON
		CountedOptions    datatypes.JSON
	}
	var rows []row
	if err := s.db.WithContext(ctx).
		Table("event_poll_posts epp").
		Select(`
			epp.id AS post_id,
			epp.instance_id,
			epp.event_id,
			COALESCE(ge.name, '') AS event_name,
			ei.local_date AS event_local_date,
			COALESCE(pt.name, '') AS template_name,
			COALESCE(pt.question, '') AS question,
			epp.telegram_message_id,
			COALESCE(epp.telegram_poll_id, '') AS telegram_poll_id,
			epp.status,
			epp.published_at,
			COALESCE(pt.options, '[]'::jsonb) AS template_options,
			COALESCE(pt.counted_options, '[]'::jsonb) AS counted_options
		`).
		Joins("JOIN poll_templates pt ON pt.id = epp.template_id").
		Joins("LEFT JOIN group_events ge ON ge.id = epp.event_id").
		Joins("LEFT JOIN event_instances ei ON ei.id = epp.instance_id").
		Where("epp.group_id = ?", group.ID).
		Order("epp.published_at DESC, epp.id DESC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]GroupPollItem, 0, len(rows))
	for _, r := range rows {
		var options []string
		_ = json.Unmarshal(r.TemplateOptions, &options)
		var counted []int
		_ = json.Unmarshal(r.CountedOptions, &counted)
		counted = normalizeCountedOptionIndexes(len(options), counted)

		var totalVotes int64
		if err := s.db.WithContext(ctx).
			Table("event_poll_votes").
			Where("post_id = ?", r.PostID).
			Count(&totalVotes).Error; err != nil {
			return nil, err
		}

		countedVotes := 0
		if len(counted) > 0 {
			choices := make([]string, 0, len(counted))
			for _, idx := range counted {
				choices = append(choices, "option_"+strconv.Itoa(idx))
			}
			countedVotes, err = s.CountVotesForPostChoices(ctx, r.PostID, choices)
			if err != nil {
				return nil, err
			}
		}

		out = append(out, GroupPollItem{
			PostID:            r.PostID,
			InstanceID:        r.InstanceID,
			EventID:           r.EventID,
			EventName:         r.EventName,
			EventLocalDate:    r.EventLocalDate,
			TemplateName:      r.TemplateName,
			Question:          r.Question,
			TelegramMessageID: r.TelegramMessageID,
			TelegramPollID:    r.TelegramPollID,
			Status:            r.Status,
			PublishedAt:       r.PublishedAt,
			CountedVotes:      countedVotes,
			TotalVotes:        int(totalVotes),
		})
	}
	return out, nil
}

func (s *Store) ListGroupPollVotesByPostID(ctx context.Context, chatID int64, postID uint64) ([]GroupPollVoteItem, error) {
	if postID == 0 {
		return nil, errors.New("post_id is required")
	}
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	type postRow struct {
		TemplateOptions datatypes.JSON
		CountedOptions  datatypes.JSON
	}
	var post postRow
	if err := s.db.WithContext(ctx).
		Table("event_poll_posts epp").
		Select("COALESCE(pt.options, '[]'::jsonb) AS template_options, COALESCE(pt.counted_options, '[]'::jsonb) AS counted_options").
		Joins("JOIN poll_templates pt ON pt.id = epp.template_id").
		Where("epp.id = ? AND epp.group_id = ?", postID, group.ID).
		Take(&post).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("poll post not found")
		}
		return nil, err
	}

	var options []string
	_ = json.Unmarshal(post.TemplateOptions, &options)
	var counted []int
	_ = json.Unmarshal(post.CountedOptions, &counted)
	counted = normalizeCountedOptionIndexes(len(options), counted)
	countedChoices := make(map[string]struct{}, len(counted))
	for _, idx := range counted {
		countedChoices["option_"+strconv.Itoa(idx)] = struct{}{}
	}

	type voteRow struct {
		UserID    int64
		Username  string
		FirstName string
		LastName  string
		Choice    string
		Source    string
		VotedAt   time.Time
	}
	var rows []voteRow
	if err := s.db.WithContext(ctx).
		Table("event_poll_votes ev").
		Select("ev.user_id, COALESCE(tu.username, ev.username, '') AS username, COALESCE(tu.first_name, ev.first_name, '') AS first_name, COALESCE(tu.last_name, ev.last_name, '') AS last_name, ev.choice, ev.source, ev.voted_at").
		Joins("LEFT JOIN telegram_users tu ON tu.telegram_id = ev.user_id").
		Where("ev.post_id = ?", postID).
		Order("ev.voted_at ASC, ev.user_id ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]GroupPollVoteItem, 0, len(rows))
	for _, row := range rows {
		choiceLabel := row.Choice
		var choiceIndex *int
		if idx, ok := parsePollOptionChoice(row.Choice); ok {
			idxCopy := idx
			choiceIndex = &idxCopy
			if idx >= 0 && idx < len(options) {
				choiceLabel = options[idx]
			}
		}
		_, countedChoice := countedChoices[row.Choice]
		items = append(items, GroupPollVoteItem{
			UserID:      row.UserID,
			Username:    row.Username,
			FirstName:   row.FirstName,
			LastName:    row.LastName,
			Choice:      row.Choice,
			ChoiceIndex: choiceIndex,
			ChoiceLabel: choiceLabel,
			Counted:     countedChoice,
			Source:      row.Source,
			VotedAt:     row.VotedAt,
		})
	}
	return items, nil
}

func (s *Store) GetEventTeamSplitState(ctx context.Context, chatID int64, eventID, postID uint64) (*EventTeamSplitState, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	type postRow struct {
		PostID          uint64
		TemplateOptions datatypes.JSON
		CountedOptions  datatypes.JSON
	}
	var post postRow
	if err := s.db.WithContext(ctx).
		Table("event_poll_posts epp").
		Select("epp.id AS post_id, COALESCE(pt.options, '[]'::jsonb) AS template_options, COALESCE(pt.counted_options, '[]'::jsonb) AS counted_options").
		Joins("JOIN poll_templates pt ON pt.id = epp.template_id").
		Where("epp.id = ? AND epp.group_id = ? AND epp.event_id = ?", postID, group.ID, eventID).
		Take(&post).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("poll post not found")
		}
		return nil, err
	}

	var options []string
	if err := json.Unmarshal(post.TemplateOptions, &options); err != nil {
		return nil, err
	}
	var counted []int
	if err := json.Unmarshal(post.CountedOptions, &counted); err != nil {
		return nil, err
	}
	counted = normalizeCountedOptionIndexes(len(options), counted)
	if len(counted) == 0 {
		return &EventTeamSplitState{
			EventID: eventID,
			PostID:  postID,
			Players: []TeamSplitPlayer{},
			Chance:  calculateTeamChance(nil),
		}, nil
	}
	choiceSet := make(map[string]struct{}, len(counted))
	for _, idx := range counted {
		choiceSet["option_"+strconv.Itoa(idx)] = struct{}{}
	}

	type voteRow struct {
		UserID    int64
		Username  string
		FirstName string
		LastName  string
		Choice    string
		Rating    *float64
		Team      string
		Position  *int
	}
	var rows []voteRow
	if err := s.db.WithContext(ctx).
		Table("event_poll_votes ev").
		Select("ev.user_id, COALESCE(tu.username, ev.username, '') AS username, COALESCE(tu.first_name, ev.first_name, '') AS first_name, COALESCE(tu.last_name, ev.last_name, '') AS last_name, ev.choice, rs.rating, COALESCE(eta.team, 'unassigned') AS team, eta.position").
		Joins("JOIN event_poll_posts epp ON epp.id = ev.post_id").
		Joins("LEFT JOIN telegram_users tu ON tu.telegram_id = ev.user_id").
		Joins("LEFT JOIN event_team_sessions ets ON ets.post_id = ev.post_id").
		Joins("LEFT JOIN event_team_assignments eta ON eta.session_id = ets.id AND eta.user_id = ev.user_id").
		Joins("LEFT JOIN (SELECT user_telegram_id, AVG(score)::float8 AS rating FROM group_member_skills WHERE group_id = ? GROUP BY user_telegram_id) rs ON rs.user_telegram_id = ev.user_id", group.ID).
		Where("ev.post_id = ? AND ev.choice IN ?", postID, keysOfMap(choiceSet)).
		Order("COALESCE(eta.team, 'unassigned') ASC, COALESCE(eta.position, 0) ASC, ev.voted_at ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	players := make([]TeamSplitPlayer, 0, len(rows))
	for idx, row := range rows {
		choiceIdx, _ := parsePollOptionChoice(row.Choice)
		choiceLabel := row.Choice
		if choiceIdx >= 0 && choiceIdx < len(options) {
			choiceLabel = options[choiceIdx]
		}
		rating := 5.0
		if row.Rating != nil {
			rating = *row.Rating
		}
		pos := idx
		if row.Position != nil {
			pos = *row.Position
		}
		players = append(players, TeamSplitPlayer{
			UserID:      row.UserID,
			Username:    row.Username,
			FirstName:   row.FirstName,
			LastName:    row.LastName,
			Choice:      row.Choice,
			ChoiceIndex: choiceIdx,
			ChoiceLabel: choiceLabel,
			Rating:      rating,
			Team:        normalizeTeamValue(row.Team),
			Position:    pos,
		})
	}

	return &EventTeamSplitState{
		EventID: eventID,
		PostID:  postID,
		Players: players,
		Chance:  calculateTeamChance(players),
	}, nil
}

func (s *Store) SaveEventTeamSplit(ctx context.Context, chatID int64, eventID, postID uint64, updates []TeamSplitAssignmentInput) error {
	state, err := s.GetEventTeamSplitState(ctx, chatID, eventID, postID)
	if err != nil {
		return err
	}
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return err
	}

	byUser := make(map[int64]TeamSplitAssignmentInput, len(updates))
	for _, item := range updates {
		if item.UserID == 0 {
			continue
		}
		byUser[item.UserID] = TeamSplitAssignmentInput{
			UserID:   item.UserID,
			Team:     normalizeTeamValue(item.Team),
			Position: item.Position,
		}
	}

	nextPlayers := make([]TeamSplitPlayer, 0, len(state.Players))
	for idx, player := range state.Players {
		if upd, ok := byUser[player.UserID]; ok {
			player.Team = upd.Team
			player.Position = upd.Position
		} else {
			player.Team = "unassigned"
			player.Position = idx
		}
		nextPlayers = append(nextPlayers, player)
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var session EventTeamSession
		if err := tx.Where("post_id = ?", postID).First(&session).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			session = EventTeamSession{
				GroupID: group.ID,
				EventID: eventID,
				PostID:  postID,
			}
			if err := tx.Create(&session).Error; err != nil {
				return err
			}
		}

		userIDs := make([]int64, 0, len(nextPlayers))
		for _, player := range nextPlayers {
			userIDs = append(userIDs, player.UserID)
			row := map[string]interface{}{
				"session_id": session.ID,
				"user_id":    player.UserID,
				"team":       normalizeTeamValue(player.Team),
				"position":   player.Position,
			}
			if err := tx.Table("event_team_assignments").
				Clauses(clause.OnConflict{
					Columns: []clause.Column{
						{Name: "session_id"},
						{Name: "user_id"},
					},
					DoUpdates: clause.Assignments(map[string]interface{}{
						"team":       row["team"],
						"position":   row["position"],
						"updated_at": gorm.Expr("NOW()"),
					}),
				}).
				Create(row).Error; err != nil {
				return err
			}
		}
		if len(userIDs) == 0 {
			return tx.Table("event_team_assignments").Where("session_id = ?", session.ID).Delete(nil).Error
		}
		return tx.Table("event_team_assignments").Where("session_id = ? AND user_id NOT IN ?", session.ID, userIDs).Delete(nil).Error
	})
}

func (s *Store) AutoSplitEventTeams(ctx context.Context, chatID int64, eventID, postID uint64) (*EventTeamSplitState, error) {
	state, err := s.GetEventTeamSplitState(ctx, chatID, eventID, postID)
	if err != nil {
		return nil, err
	}
	if state == nil || len(state.Players) == 0 {
		return state, nil
	}
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	userIDs := make([]int64, 0, len(state.Players))
	for _, p := range state.Players {
		userIDs = append(userIDs, p.UserID)
	}
	type roleRow struct {
		UserID     int64
		PlayerType string
	}
	var roleRows []roleRow
	if err := s.db.WithContext(ctx).
		Table("group_members").
		Select("user_telegram_id AS user_id, COALESCE(player_type, '') AS player_type").
		Where("group_id = ? AND user_telegram_id IN ? AND is_active = TRUE", group.ID, userIDs).
		Scan(&roleRows).Error; err != nil {
		return nil, err
	}
	roleByUser := make(map[int64]string, len(roleRows))
	for _, row := range roleRows {
		roleByUser[row.UserID] = strings.TrimSpace(strings.ToLower(row.PlayerType))
	}

	type relationRow struct {
		UserAID      int64
		UserBID      int64
		RelationType string
		Weight       int
	}
	var relationRows []relationRow
	if err := s.db.WithContext(ctx).
		Table("player_relations").
		Select("user_a_id AS user_a_id, user_b_id AS user_b_id, relation_type, weight").
		Where("group_id = ? AND is_active = TRUE AND user_a_id IN ? AND user_b_id IN ?", group.ID, userIDs, userIDs).
		Scan(&relationRows).Error; err != nil {
		return nil, err
	}
	type relEdge struct {
		Other  int64
		Type   string
		Weight int
	}
	relations := make(map[int64][]relEdge, len(userIDs))
	for _, rel := range relationRows {
		relations[rel.UserAID] = append(relations[rel.UserAID], relEdge{
			Other:  rel.UserBID,
			Type:   rel.RelationType,
			Weight: rel.Weight,
		})
		relations[rel.UserBID] = append(relations[rel.UserBID], relEdge{
			Other:  rel.UserAID,
			Type:   rel.RelationType,
			Weight: rel.Weight,
		})
	}

	teamCodes := []string{"A", "B"}
	hasTeamC := false
	for _, p := range state.Players {
		if p.Team == "C" {
			hasTeamC = true
			break
		}
	}
	if hasTeamC || len(state.Players) > 14 {
		teamCodes = append(teamCodes, "C")
	}

	capacity := make(map[string]int, len(teamCodes))
	base := len(state.Players) / len(teamCodes)
	rest := len(state.Players) % len(teamCodes)
	for idx, code := range teamCodes {
		capacity[code] = base
		if idx < rest {
			capacity[code]++
		}
	}

	type bucket struct {
		Code    string
		Players []TeamSplitPlayer
		Score   float64
		Setters int
		Liberos int
	}
	buckets := make([]*bucket, 0, len(teamCodes))
	byCode := make(map[string]*bucket, len(teamCodes))
	for _, code := range teamCodes {
		b := &bucket{Code: code}
		buckets = append(buckets, b)
		byCode[code] = b
	}
	assignedTeam := make(map[int64]string, len(state.Players))

	assignToBest := func(player TeamSplitPlayer, role string, preferRole bool) {
		var chosen *bucket
		best := math.MaxFloat64
		for _, b := range buckets {
			if len(b.Players) >= capacity[b.Code] {
				continue
			}
			rolePenalty := 0.0
			if preferRole {
				if role == "setter" {
					rolePenalty = float64(b.Setters) * 3
				} else if role == "libero" {
					rolePenalty = float64(b.Liberos) * 3
				}
			}
			relationPenalty := 0.0
			for _, edge := range relations[player.UserID] {
				otherTeam, ok := assignedTeam[edge.Other]
				if !ok {
					continue
				}
				switch edge.Type {
				case "prefer_together":
					if otherTeam != b.Code {
						relationPenalty += float64(edge.Weight) * 4
					} else {
						relationPenalty -= float64(edge.Weight) * 0.75
					}
				case "avoid_together":
					if otherTeam == b.Code {
						relationPenalty += float64(edge.Weight) * 6
					}
				}
			}
			metric := b.Score + rolePenalty + relationPenalty + float64(len(b.Players))*0.25
			if chosen == nil || metric < best {
				chosen = b
				best = metric
			}
		}
		if chosen == nil {
			chosen = buckets[0]
		}
		chosen.Players = append(chosen.Players, player)
		chosen.Score += player.Rating
		assignedTeam[player.UserID] = chosen.Code
		if role == "setter" {
			chosen.Setters++
		}
		if role == "libero" {
			chosen.Liberos++
		}
	}

	setters := make([]TeamSplitPlayer, 0)
	liberos := make([]TeamSplitPlayer, 0)
	restPlayers := make([]TeamSplitPlayer, 0)
	for _, p := range state.Players {
		switch roleByUser[p.UserID] {
		case "setter":
			setters = append(setters, p)
		case "libero":
			liberos = append(liberos, p)
		default:
			restPlayers = append(restPlayers, p)
		}
	}
	sort.SliceStable(setters, func(i, j int) bool { return setters[i].Rating > setters[j].Rating })
	sort.SliceStable(liberos, func(i, j int) bool { return liberos[i].Rating > liberos[j].Rating })
	sort.SliceStable(restPlayers, func(i, j int) bool { return restPlayers[i].Rating > restPlayers[j].Rating })

	for _, p := range setters {
		assignToBest(p, "setter", true)
	}
	for _, p := range liberos {
		assignToBest(p, "libero", true)
	}
	for _, p := range restPlayers {
		assignToBest(p, "", false)
	}

	assignments := make([]TeamSplitAssignmentInput, 0, len(state.Players))
	for _, code := range teamCodes {
		list := byCode[code].Players
		sort.SliceStable(list, func(i, j int) bool {
			return list[i].Rating > list[j].Rating
		})
		for pos, p := range list {
			assignments = append(assignments, TeamSplitAssignmentInput{
				UserID:   p.UserID,
				Team:     code,
				Position: pos,
			})
		}
	}

	if err := s.SaveEventTeamSplit(ctx, chatID, eventID, postID, assignments); err != nil {
		return nil, err
	}
	return s.GetEventTeamSplitState(ctx, chatID, eventID, postID)
}

func keysOfMap(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}

func (s *Store) ListEventHistory(ctx context.Context, chatID int64) ([]EventHistoryItem, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	loc, err := time.LoadLocation(group.Timezone)
	if err != nil {
		loc = time.UTC
	}
	nowLocal := time.Now().In(loc)

	type row struct {
		InstanceID     uint64
		EventID        uint64
		Name           string
		EventType      string
		PollPostID     *uint64
		LocalDate      time.Time
		StartWeekday   int
		StartTime      string
		EndTime        string
		PollTemplate   string
		PublishEnabled bool
		Status         string
		PlannedStartAt time.Time
		PlannedEndAt   time.Time
	}
	var rows []row
	if err := s.db.WithContext(ctx).
		Table("event_instances ei").
		Select("ei.id AS instance_id, ei.event_id, ei.poll_post_id, ei.event_name AS name, ei.event_type, ei.local_date, ei.start_weekday, ei.start_time, ei.end_time, COALESCE(ei.poll_template_name, '') AS poll_template, ei.publish_enabled, ei.status, ei.planned_start_at, ei.planned_end_at").
		Joins("JOIN group_events ge ON ge.id = ei.event_id").
		Where("ei.group_id = ? AND ei.is_active = TRUE AND ge.is_active = TRUE", group.ID).
		Order("ei.local_date DESC, ei.id DESC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]EventHistoryItem, 0, len(rows))
	for _, row := range rows {
		startLocal := row.PlannedStartAt.In(loc)
		endLocal := row.PlannedEndAt.In(loc)
		distributionStart := startLocal.Add(-30 * time.Minute)

		status := EventHistoryStatus(clampInstanceStatus(row.Status))
		if status != EventHistoryStatusNotHeld {
			if nowLocal.Before(distributionStart) {
				status = EventHistoryStatusInVoting
			} else if nowLocal.Before(endLocal) {
				status = EventHistoryStatusOnDistribution
			}
		}

		var latestPostID *uint64
		var latestPollAt *time.Time
		if row.PollPostID != nil {
			var poll struct {
				PublishedAt time.Time
			}
			if err := s.db.WithContext(ctx).
				Table("event_poll_posts").
				Select("published_at").
				Where("id = ?", *row.PollPostID).
				Take(&poll).Error; err == nil {
				latestPostID = row.PollPostID
				publishedAtLocal := poll.PublishedAt.In(loc)
				latestPollAt = &publishedAtLocal
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, err
			} else {
				latestPostID = row.PollPostID
			}
		}

		debtAmount := 0.0
		var billing struct {
			DebtAmount  float64
			UnpaidCount int64
		}
		if err := s.db.WithContext(ctx).
			Table("event_settlements es").
			Select("COALESCE(SUM(CASE WHEN esp.is_paid = FALSE THEN esp.amount_due ELSE 0 END), 0) AS debt_amount, COUNT(CASE WHEN esp.is_paid = FALSE THEN 1 END) AS unpaid_count").
			Joins("LEFT JOIN event_settlement_payments esp ON esp.settlement_id = es.id").
			Where("es.group_id = ? AND es.instance_id = ?", group.ID, row.InstanceID).
			Scan(&billing).Error; err == nil {
			debtAmount = billing.DebtAmount
			if nowLocal.After(endLocal) && status != EventHistoryStatusNotHeld {
				if billing.UnpaidCount > 0 {
					status = EventHistoryStatusOnReview
				} else {
					status = EventHistoryStatusCompleted
				}
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}

		items = append(items, EventHistoryItem{
			InstanceID:     row.InstanceID,
			EventID:        row.EventID,
			Name:           row.Name,
			EventType:      row.EventType,
			LocalDate:      row.LocalDate,
			StartWeekday:   row.StartWeekday,
			StartTime:      row.StartTime,
			PollTemplate:   row.PollTemplate,
			LatestPostID:   latestPostID,
			LatestPollAt:   latestPollAt,
			NextStartAt:    startLocal,
			EndAt:          endLocal,
			Status:         status,
			CanDistribute:  status == EventHistoryStatusOnDistribution && latestPostID != nil,
			PublishEnabled: row.PublishEnabled,
			DebtAmount:     debtAmount,
		})
	}
	return items, nil
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

func ensureDefaultMemberSkillsTx(tx *gorm.DB, groupID uint64, userTelegramID int64) error {
	const defaultScore = 5
	return tx.Exec(`
		INSERT INTO group_member_skills (group_id, user_telegram_id, skill_id, score, created_at, updated_at)
		SELECT ?, ?, sc.id, ?, NOW(), NOW()
		FROM skills_catalog sc
		WHERE sc.is_active = TRUE
		ON CONFLICT (group_id, user_telegram_id, skill_id) DO NOTHING
	`, groupID, userTelegramID, defaultScore).Error
}
