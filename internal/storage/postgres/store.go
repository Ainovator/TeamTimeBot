package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"math"
	"net/http"
	"os"
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
	Role     string `json:"role,omitempty"`
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
	OptionWeights  []int    `json:"optionWeights"`
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

type UserGroupTrainingItem struct {
	InstanceID uint64    `json:"instanceID"`
	Name       string    `json:"name"`
	StartAt    time.Time `json:"startAt"`
	EndAt      time.Time `json:"endAt"`
	Status     string    `json:"status"`
	AmountDue  float64   `json:"amountDue"`
	IsPaid     bool      `json:"isPaid"`
}

type UserGroupProfile struct {
	RoleCode   string                  `json:"roleCode"`
	RoleTitle  string                  `json:"roleTitle"`
	DebtAmount float64                 `json:"debtAmount"`
	Trainings  []UserGroupTrainingItem `json:"trainings"`
}

type GroupRoleView struct {
	Code        string          `json:"code"`
	Title       string          `json:"title"`
	Permissions map[string]bool `json:"permissions"`
}

type GroupPermissionsView struct {
	RoleCode    string          `json:"roleCode"`
	RoleTitle   string          `json:"roleTitle"`
	Permissions map[string]bool `json:"permissions"`
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
	RealName       string    `json:"realName"`
	PlayerType     string    `json:"playerType"`
	Role           string    `json:"role"`
	Status         string    `json:"status"`
	LastSeenAt     time.Time `json:"lastSeenAt"`
	AppRoleCode    string    `json:"appRoleCode,omitempty"`
	AppRoleTitle   string    `json:"appRoleTitle,omitempty"`
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
	RealName       string             `json:"realName"`
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

func normalizeOptionWeightsLen(optionsLen int, weights []int) []int {
	if optionsLen <= 0 {
		return []int{}
	}
	out := make([]int, optionsLen)
	for i := 0; i < optionsLen; i++ {
		w := 1
		if i < len(weights) {
			w = weights[i]
		}
		if w <= 0 {
			w = 1
		}
		out[i] = w
	}
	return out
}

func guestSlotUserID(postID uint64, userID int64, ordinal int) int64 {
	// Stable synthetic negative IDs so guest slots can be saved in event_team_assignments.
	// Collisions are extremely unlikely and acceptable for this use-case.
	//
	// IMPORTANT: must fit into JS Number safely (<= 2^53-1) because the web UI sends userID as a number.
	h := fnv.New64a()
	_, _ = h.Write([]byte(fmt.Sprintf("%d:%d:%d", postID, userID, ordinal)))
	// Keep within 52 bits so abs(userID) < 2^53 and round-trips safely through JSON/JS.
	const safeMask = (uint64(1) << 52) - 1
	v := int64(h.Sum64()&safeMask) + 1
	if v == 0 {
		v = 1
	}
	return -v
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
		PollOptionWeights       datatypes.JSON
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
			COALESCE(pt.counted_options, '[]'::jsonb) AS poll_counted_options,
			COALESCE(pt.option_weights, '[]'::jsonb) AS poll_option_weights
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
		snap.PollOptionWeights = pollTemplateOverride.OptionWeights
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
	record["poll_option_weights"] = snap.PollOptionWeights

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

func (s *Store) ReplaceEventPollVotes(
	ctx context.Context,
	postID uint64,
	userID int64,
	username, firstName, lastName string,
	choices []string,
	source string,
	votedAt time.Time,
) error {
	if postID == 0 || userID == 0 {
		return errors.New("post_id and user_id are required")
	}

	uniq := make([]string, 0, len(choices))
	seen := make(map[string]struct{}, len(choices))
	for _, c := range choices {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if _, ok := seen[c]; ok {
			continue
		}
		seen[c] = struct{}{}
		uniq = append(uniq, c)
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

		// Telegram can send updates with empty OptionIDs (user removed vote).
		// In this case we just clear all choices for this user.
		if err := tx.Table("event_poll_votes").
			Where("post_id = ? AND user_id = ?", postID, userID).
			Delete(&EventPollVote{}).Error; err != nil {
			return err
		}
		if len(uniq) == 0 {
			return nil
		}

		votes := make([]EventPollVote, 0, len(uniq))
		for _, c := range uniq {
			votes = append(votes, EventPollVote{
				PostID:    postID,
				UserID:    userID,
				Username:  username,
				FirstName: firstName,
				LastName:  lastName,
				Choice:    c,
				Source:    source,
				VotedAt:   votedAt.UTC(),
			})
		}
		return tx.Create(&votes).Error
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
	PollOptionWeights  datatypes.JSON
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
		PollOptionWeights       datatypes.JSON
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
			COALESCE(ei.poll_counted_options, '[]'::jsonb) AS poll_counted_options,
			COALESCE(ei.poll_option_weights, '[]'::jsonb) AS poll_option_weights
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
		PollOptionWeights:       r.PollOptionWeights,
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
		PollOptionWeights   datatypes.JSON
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
			COALESCE(ei.poll_counted_options, '[]'::jsonb) AS poll_counted_options,
			COALESCE(ei.poll_option_weights, '[]'::jsonb) AS poll_option_weights
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
			PollOptionWeights:   r.PollOptionWeights,
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
	var weights []int
	if settlement.InstanceID != nil {
		var snap struct {
			Options        datatypes.JSON
			CountedOptions datatypes.JSON
			OptionWeights  datatypes.JSON
		}
		if err := tx.WithContext(ctx).
			Table("event_instances").
			Select("COALESCE(poll_options, '[]'::jsonb) AS options, COALESCE(poll_counted_options, '[]'::jsonb) AS counted_options, COALESCE(poll_option_weights, '[]'::jsonb) AS option_weights").
			Where("id = ?", *settlement.InstanceID).
			Take(&snap).Error; err != nil {
			return err
		}
		_ = json.Unmarshal(snap.Options, &options)
		_ = json.Unmarshal(snap.CountedOptions, &counted)
		_ = json.Unmarshal(snap.OptionWeights, &weights)
	} else {
		var template struct {
			Options        datatypes.JSON
			CountedOptions datatypes.JSON
			OptionWeights  datatypes.JSON
		}
		if err := tx.WithContext(ctx).
			Table("event_poll_posts epp").
			Select("COALESCE(pt.options, '[]'::jsonb) AS options, COALESCE(pt.counted_options, '[]'::jsonb) AS counted_options, COALESCE(pt.option_weights, '[]'::jsonb) AS option_weights").
			Joins("JOIN poll_templates pt ON pt.id = epp.template_id").
			Where("epp.id = ?", *settlement.PostID).
			Take(&template).Error; err != nil {
			return err
		}
		_ = json.Unmarshal(template.Options, &options)
		_ = json.Unmarshal(template.CountedOptions, &counted)
		_ = json.Unmarshal(template.OptionWeights, &weights)
	}
	counted = normalizeCountedOptionIndexes(len(options), counted)
	weights = normalizeOptionWeightsLen(len(options), weights)
	if len(counted) == 0 {
		return nil
	}

	choices := make([]string, 0, len(counted))
	weightByChoice := make(map[string]int, len(counted))
	for _, idx := range counted {
		choice := "option_" + strconv.Itoa(idx)
		choices = append(choices, choice)
		w := 1
		if idx >= 0 && idx < len(weights) {
			w = weights[idx]
		}
		if w <= 0 {
			w = 1
		}
		weightByChoice[choice] = w
	}

	type voteRow struct {
		UserID    int64
		Username  string
		FirstName string
		LastName  string
		Choice    string
	}
	var rows []voteRow
	if err := tx.WithContext(ctx).
		Table("event_poll_votes").
		Select("user_id, COALESCE(username, '') AS username, COALESCE(first_name, '') AS first_name, COALESCE(last_name, '') AS last_name, choice").
		Where("post_id = ? AND choice IN ?", *settlement.PostID, choices).
		Scan(&rows).Error; err != nil {
		return err
	}

	// Compute seats per payer.
	type payer struct {
		UserID    int64
		Username  string
		FirstName string
		LastName  string
		Seats     int
	}
	payers := make(map[int64]*payer, len(rows))
	for _, row := range rows {
		p, ok := payers[row.UserID]
		if !ok {
			p = &payer{
				UserID:    row.UserID,
				Username:  strings.TrimSpace(row.Username),
				FirstName: strings.TrimSpace(row.FirstName),
				LastName:  strings.TrimSpace(row.LastName),
				Seats:     0,
			}
			payers[row.UserID] = p
		}
		w := weightByChoice[strings.TrimSpace(row.Choice)]
		if w <= 0 {
			w = 1
		}
		p.Seats += w
	}

	for _, p := range payers {
		if p.Seats <= 0 {
			continue
		}
		rec := map[string]interface{}{
			"settlement_id": settlementID,
			"user_id":       p.UserID,
			"username":      p.Username,
			"first_name":    p.FirstName,
			"last_name":     p.LastName,
			"amount_due":    settlement.AmountPerPerson * float64(p.Seats),
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
		OptionWeights  datatypes.JSON
		Status         string
	}
	if err := s.db.WithContext(ctx).
		Table("event_instances").
		Select("id, group_id, event_id, local_date, poll_post_id, cost_amount, COALESCE(poll_options, '[]'::jsonb) AS poll_options, COALESCE(poll_counted_options, '[]'::jsonb) AS counted_options, COALESCE(poll_option_weights, '[]'::jsonb) AS option_weights, status").
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
	var weights []int
	_ = json.Unmarshal(inst.OptionWeights, &weights)
	counted = normalizeCountedOptionIndexes(len(options), counted)
	weights = normalizeOptionWeightsLen(len(options), weights)
	if len(counted) == 0 {
		return nil, errors.New("template has no options marked with accounting flag")
	}
	choices := make([]string, 0, len(counted))
	weightByChoice := make(map[string]int, len(counted))
	for _, idx := range counted {
		choice := "option_" + strconv.Itoa(idx)
		choices = append(choices, choice)
		w := 1
		if idx >= 0 && idx < len(weights) {
			w = weights[idx]
		}
		if w <= 0 {
			w = 1
		}
		weightByChoice[choice] = w
	}
	byChoice, err := s.CountVotesForPostChoicesByChoice(ctx, *inst.PollPostID, choices)
	if err != nil {
		return nil, err
	}
	participants := 0
	for choice, c := range byChoice {
		w := weightByChoice[choice]
		if w <= 0 {
			w = 1
		}
		participants += c * w
	}

	totalAmount := 4000.0
	if inst.CostAmount != nil {
		totalAmount = *inst.CostAmount
	}
	perPerson := 0.0
	if participants > 0 {
		perPerson = math.Ceil(totalAmount / float64(participants))
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

func (s *Store) GetUserGroupProfile(ctx context.Context, chatID int64, userTelegramID int64) (*UserGroupProfile, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	perms, err := s.GetGroupPermissionsForUser(ctx, chatID, userTelegramID)
	if err != nil {
		return nil, err
	}

	var debt float64
	if err := s.db.WithContext(ctx).
		Table("event_settlement_payments esp").
		Select("COALESCE(SUM(esp.amount_due), 0) AS total_debt").
		Joins("JOIN event_settlements es ON es.id = esp.settlement_id").
		Where("es.group_id = ? AND esp.user_id = ? AND esp.is_paid = FALSE", group.ID, userTelegramID).
		Scan(&debt).Error; err != nil {
		return nil, err
	}

	type row struct {
		InstanceID uint64
		Name       string
		StartAt    time.Time
		EndAt      time.Time
		Status     string
		AmountDue  float64
		IsPaid     bool
	}
	var rows []row
	if err := s.db.WithContext(ctx).
		Table("event_poll_votes epv").
		Select("DISTINCT ei.id AS instance_id, COALESCE(ei.event_name, '') AS name, ei.planned_start_at AS start_at, ei.planned_end_at AS end_at, ei.status, COALESCE(esp.amount_due, 0) AS amount_due, COALESCE(esp.is_paid, FALSE) AS is_paid").
		Joins("JOIN event_poll_posts epp ON epp.id = epv.post_id").
		Joins("JOIN event_instances ei ON ei.id = epp.instance_id").
		Joins("LEFT JOIN event_settlements es ON es.instance_id = ei.id").
		Joins("LEFT JOIN event_settlement_payments esp ON esp.settlement_id = es.id AND esp.user_id = epv.user_id").
		Where("ei.group_id = ? AND epv.user_id = ?", group.ID, userTelegramID).
		Order("ei.planned_start_at DESC, ei.id DESC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]UserGroupTrainingItem, 0, len(rows))
	for _, r := range rows {
		name := strings.TrimSpace(r.Name)
		if name == "" {
			name = "Событие"
		}
		items = append(items, UserGroupTrainingItem{
			InstanceID: r.InstanceID,
			Name:       name,
			StartAt:    r.StartAt,
			EndAt:      r.EndAt,
			Status:     r.Status,
			AmountDue:  r.AmountDue,
			IsPaid:     r.IsPaid,
		})
	}

	return &UserGroupProfile{
		RoleCode:   perms.RoleCode,
		RoleTitle:  perms.RoleTitle,
		DebtAmount: debt,
		Trainings:  items,
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
	counted, _, err := s.GetEventTemplateCountedOptionsAndWeights(ctx, eventID)
	return counted, err
}

func (s *Store) GetEventTemplateCountedOptionsAndWeights(ctx context.Context, eventID uint64) ([]int, []int, error) {
	var row struct {
		TemplateOptions datatypes.JSON
		CountedOptions  datatypes.JSON
		OptionWeights   datatypes.JSON
	}
	if err := s.db.WithContext(ctx).
		Table("group_events ge").
		Select("pt.options AS template_options, pt.counted_options, COALESCE(pt.option_weights, '[]'::jsonb) AS option_weights").
		Joins("JOIN poll_templates pt ON pt.id = ge.poll_template_id").
		Where("ge.id = ? AND ge.is_active = TRUE AND pt.is_active = TRUE", eventID).
		Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []int{}, []int{}, nil
		}
		return nil, nil, err
	}

	var options []string
	if err := json.Unmarshal(row.TemplateOptions, &options); err != nil {
		return nil, nil, err
	}
	var counted []int
	if err := json.Unmarshal(row.CountedOptions, &counted); err != nil {
		return nil, nil, err
	}
	var weights []int
	_ = json.Unmarshal(row.OptionWeights, &weights)
	return normalizeCountedOptionIndexes(len(options), counted), normalizeOptionWeightsLen(len(options), weights), nil
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

func (s *Store) CountVotesForPostChoicesByChoice(ctx context.Context, postID uint64, choices []string) (map[string]int, error) {
	if postID == 0 {
		return nil, errors.New("post_id is required")
	}
	filtered := make([]string, 0, len(choices))
	for _, c := range choices {
		c = strings.TrimSpace(c)
		if c != "" {
			filtered = append(filtered, c)
		}
	}
	out := map[string]int{}
	if len(filtered) == 0 {
		return out, nil
	}
	type row struct {
		Choice string
		Count  int64
	}
	var rows []row
	if err := s.db.WithContext(ctx).
		Table("event_poll_votes").
		Select("choice, COUNT(*) AS count").
		Where("post_id = ? AND choice IN ?", postID, filtered).
		Group("choice").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[strings.TrimSpace(r.Choice)] = int(r.Count)
	}
	return out, nil
}

type PollSeatCountItem struct {
	UserID    int64
	Username  string
	FirstName string
	LastName  string
	Seats     int
}

func (s *Store) ListSeatCountsForPostChoices(ctx context.Context, postID uint64, choices []string, weightByChoice map[string]int) ([]PollSeatCountItem, int, error) {
	if postID == 0 {
		return nil, 0, errors.New("post_id is required")
	}
	filtered := make([]string, 0, len(choices))
	for _, c := range choices {
		c = strings.TrimSpace(c)
		if c != "" {
			filtered = append(filtered, c)
		}
	}
	if len(filtered) == 0 {
		return []PollSeatCountItem{}, 0, nil
	}

	type voteRow struct {
		UserID    int64
		Username  string
		FirstName string
		LastName  string
		Choice    string
	}
	var rows []voteRow
	if err := s.db.WithContext(ctx).
		Table("event_poll_votes ev").
		Select("ev.user_id, COALESCE(tu.username, ev.username, '') AS username, COALESCE(tu.first_name, ev.first_name, '') AS first_name, COALESCE(tu.last_name, ev.last_name, '') AS last_name, ev.choice").
		Joins("LEFT JOIN telegram_users tu ON tu.telegram_id = ev.user_id").
		Where("ev.post_id = ? AND ev.choice IN ?", postID, filtered).
		Order("ev.voted_at ASC, ev.user_id ASC").
		Scan(&rows).Error; err != nil {
		return nil, 0, err
	}

	byUser := make(map[int64]*PollSeatCountItem, len(rows))
	totalSeats := 0
	for _, r := range rows {
		item, ok := byUser[r.UserID]
		if !ok {
			item = &PollSeatCountItem{
				UserID:    r.UserID,
				Username:  strings.TrimSpace(r.Username),
				FirstName: strings.TrimSpace(r.FirstName),
				LastName:  strings.TrimSpace(r.LastName),
				Seats:     0,
			}
			byUser[r.UserID] = item
		}
		w := 1
		if weightByChoice != nil {
			if ww, ok := weightByChoice[strings.TrimSpace(r.Choice)]; ok && ww > 0 {
				w = ww
			}
		}
		item.Seats += w
		totalSeats += w
	}

	out := make([]PollSeatCountItem, 0, len(byUser))
	for _, v := range byUser {
		out = append(out, *v)
	}
	sort.Slice(out, func(i, j int) bool {
		li := strings.ToLower(strings.TrimSpace(out[i].FirstName + " " + out[i].LastName + " " + out[i].Username))
		lj := strings.ToLower(strings.TrimSpace(out[j].FirstName + " " + out[j].LastName + " " + out[j].Username))
		if li != lj {
			return li < lj
		}
		return out[i].UserID < out[j].UserID
	})
	return out, totalSeats, nil
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
		OptionWeights     datatypes.JSON
	}
	var rows []row
	if err := s.db.WithContext(ctx).
		Table("event_poll_posts epp").
		Select("epp.id AS post_id, epp.telegram_message_id, COALESCE(epp.telegram_poll_id, '') AS telegram_poll_id, epp.status, epp.published_at, COALESCE(pt.question, '') AS question, COALESCE(pt.options, '[]'::jsonb) AS template_options, COALESCE(pt.counted_options, '[]'::jsonb) AS counted_options, COALESCE(pt.option_weights, '[]'::jsonb) AS option_weights").
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
		var weights []int
		_ = json.Unmarshal(r.OptionWeights, &weights)
		counted = normalizeCountedOptionIndexes(len(options), counted)
		weights = normalizeOptionWeightsLen(len(options), weights)

		var totalVotes int64
		if err := s.db.WithContext(ctx).
			Table("event_poll_votes").
			Select("COUNT(DISTINCT user_id)").
			Where("post_id = ?", r.PostID).
			Scan(&totalVotes).Error; err != nil {
			return nil, err
		}

		countedVotes := 0
		if len(counted) > 0 {
			choices := make([]string, 0, len(counted))
			weightByChoice := make(map[string]int, len(counted))
			for _, idx := range counted {
				choice := "option_" + strconv.Itoa(idx)
				choices = append(choices, choice)
				w := 1
				if idx >= 0 && idx < len(weights) {
					w = weights[idx]
				}
				if w <= 0 {
					w = 1
				}
				weightByChoice[choice] = w
			}
			byChoice, err := s.CountVotesForPostChoicesByChoice(ctx, r.PostID, choices)
			if err != nil {
				return nil, err
			}
			for choice, c := range byChoice {
				w := weightByChoice[choice]
				if w <= 0 {
					w = 1
				}
				countedVotes += c * w
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
		OptionWeights     datatypes.JSON
	}
	var rows []row
	if err := s.db.WithContext(ctx).
		Table("event_poll_posts epp").
		Select("epp.id AS post_id, epp.telegram_message_id, COALESCE(epp.telegram_poll_id, '') AS telegram_poll_id, epp.status, epp.published_at, COALESCE(pt.question, '') AS question, COALESCE(pt.options, '[]'::jsonb) AS template_options, COALESCE(pt.counted_options, '[]'::jsonb) AS counted_options, COALESCE(pt.option_weights, '[]'::jsonb) AS option_weights").
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
		var weights []int
		_ = json.Unmarshal(r.OptionWeights, &weights)
		counted = normalizeCountedOptionIndexes(len(options), counted)
		weights = normalizeOptionWeightsLen(len(options), weights)

		var totalVotes int64
		if err := s.db.WithContext(ctx).
			Table("event_poll_votes").
			Select("COUNT(DISTINCT user_id)").
			Where("post_id = ?", r.PostID).
			Scan(&totalVotes).Error; err != nil {
			return nil, err
		}

		countedVotes := 0
		if len(counted) > 0 {
			choices := make([]string, 0, len(counted))
			weightByChoice := make(map[string]int, len(counted))
			for _, idx := range counted {
				choice := "option_" + strconv.Itoa(idx)
				choices = append(choices, choice)
				w := 1
				if idx >= 0 && idx < len(weights) {
					w = weights[idx]
				}
				if w <= 0 {
					w = 1
				}
				weightByChoice[choice] = w
			}
			byChoice, err := s.CountVotesForPostChoicesByChoice(ctx, r.PostID, choices)
			if err != nil {
				return nil, err
			}
			for choice, c := range byChoice {
				w := weightByChoice[choice]
				if w <= 0 {
					w = 1
				}
				countedVotes += c * w
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
		OptionWeights     datatypes.JSON
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
			COALESCE(pt.counted_options, '[]'::jsonb) AS counted_options,
			COALESCE(pt.option_weights, '[]'::jsonb) AS option_weights
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
		var weights []int
		_ = json.Unmarshal(r.OptionWeights, &weights)
		counted = normalizeCountedOptionIndexes(len(options), counted)
		weights = normalizeOptionWeightsLen(len(options), weights)

		var totalVotes int64
		if err := s.db.WithContext(ctx).
			Table("event_poll_votes").
			Select("COUNT(DISTINCT user_id)").
			Where("post_id = ?", r.PostID).
			Scan(&totalVotes).Error; err != nil {
			return nil, err
		}

		countedVotes := 0
		if len(counted) > 0 {
			choices := make([]string, 0, len(counted))
			weightByChoice := make(map[string]int, len(counted))
			for _, idx := range counted {
				choice := "option_" + strconv.Itoa(idx)
				choices = append(choices, choice)
				w := 1
				if idx >= 0 && idx < len(weights) {
					w = weights[idx]
				}
				if w <= 0 {
					w = 1
				}
				weightByChoice[choice] = w
			}
			byChoice, err := s.CountVotesForPostChoicesByChoice(ctx, r.PostID, choices)
			if err != nil {
				return nil, err
			}
			for choice, c := range byChoice {
				w := weightByChoice[choice]
				if w <= 0 {
					w = 1
				}
				countedVotes += c * w
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
		InstanceID      *uint64
		TemplateOptions datatypes.JSON
		CountedOptions  datatypes.JSON
		OptionWeights   datatypes.JSON
	}
	var post postRow
	if err := s.db.WithContext(ctx).
		Table("event_poll_posts epp").
		Select("epp.id AS post_id, epp.instance_id, COALESCE(pt.options, '[]'::jsonb) AS template_options, COALESCE(pt.counted_options, '[]'::jsonb) AS counted_options, COALESCE(ei.poll_option_weights, pt.option_weights, '[]'::jsonb) AS option_weights").
		Joins("JOIN poll_templates pt ON pt.id = epp.template_id").
		Joins("LEFT JOIN event_instances ei ON ei.id = epp.instance_id").
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
	var weights []int
	_ = json.Unmarshal(post.OptionWeights, &weights)
	counted = normalizeCountedOptionIndexes(len(options), counted)
	weights = normalizeOptionWeightsLen(len(options), weights)
	if len(counted) == 0 {
		return &EventTeamSplitState{
			EventID: eventID,
			PostID:  postID,
			Players: []TeamSplitPlayer{},
			Chance:  calculateTeamChance(nil),
		}, nil
	}
	choiceSet := make(map[string]struct{}, len(counted))
	weightByChoice := make(map[string]int, len(counted))
	for _, idx := range counted {
		key := "option_" + strconv.Itoa(idx)
		choiceSet[key] = struct{}{}
		if idx >= 0 && idx < len(weights) && weights[idx] > 0 {
			weightByChoice[key] = weights[idx]
		} else {
			weightByChoice[key] = 1
		}
	}

	seatItems, _, err := s.ListSeatCountsForPostChoices(ctx, postID, keysOfMap(choiceSet), weightByChoice)
	if err != nil {
		return nil, err
	}
	if len(seatItems) == 0 {
		return &EventTeamSplitState{
			EventID: eventID,
			PostID:  postID,
			Players: []TeamSplitPlayer{},
			Chance:  calculateTeamChance(nil),
		}, nil
	}

	// Load a single representative counted choice per user (for UI label). If the user voted multiple counted options,
	// we just pick the latest one.
	type userChoiceRow struct {
		UserID int64
		Choice string
	}
	var choiceRows []userChoiceRow
	if err := s.db.WithContext(ctx).
		Table("event_poll_votes ev").
		Select("DISTINCT ON (ev.user_id) ev.user_id, ev.choice").
		Where("ev.post_id = ? AND ev.choice IN ?", postID, keysOfMap(choiceSet)).
		Order("ev.user_id ASC, ev.voted_at DESC").
		Scan(&choiceRows).Error; err != nil {
		return nil, err
	}
	choiceByUser := make(map[int64]string, len(choiceRows))
	for _, r := range choiceRows {
		choiceByUser[r.UserID] = strings.TrimSpace(r.Choice)
	}

	// Load saved assignments (including synthetic guest IDs).
	type sessionRow struct {
		ID uint64
	}
	var session sessionRow
	sessionID := uint64(0)
	if err := s.db.WithContext(ctx).
		Table("event_team_sessions").
		Select("id").
		Where("post_id = ?", postID).
		Take(&session).Error; err == nil {
		sessionID = session.ID
	}
	type assignRow struct {
		UserID   int64
		Team     string
		Position int
	}
	assignments := map[int64]assignRow{}
	if sessionID != 0 {
		var as []assignRow
		if err := s.db.WithContext(ctx).
			Table("event_team_assignments").
			Select("user_id, team, position").
			Where("session_id = ?", sessionID).
			Scan(&as).Error; err != nil {
			return nil, err
		}
		for _, a := range as {
			assignments[a.UserID] = assignRow{
				UserID:   a.UserID,
				Team:     normalizeTeamValue(a.Team),
				Position: a.Position,
			}
		}
	}

	// Load ratings for real users. Guest slots are synthetic and use a neutral baseline rating (5.0).
	userIDs := make([]int64, 0, len(seatItems))
	for _, it := range seatItems {
		userIDs = append(userIDs, it.UserID)
	}
	type ratingRow struct {
		UserID int64
		Rating float64
	}
	var ratingRows []ratingRow
	if err := s.db.WithContext(ctx).
		Table("group_member_skills").
		Select("user_telegram_id AS user_id, AVG(score)::float8 AS rating").
		Where("group_id = ? AND user_telegram_id IN ?", group.ID, userIDs).
		Group("user_telegram_id").
		Scan(&ratingRows).Error; err != nil {
		return nil, err
	}
	ratingByUser := make(map[int64]float64, len(ratingRows))
	for _, r := range ratingRows {
		ratingByUser[r.UserID] = r.Rating
	}

	players := make([]TeamSplitPlayer, 0, len(seatItems))
	nextPos := 0
	for _, it := range seatItems {
		choice := choiceByUser[it.UserID]
		choiceIdx, _ := parsePollOptionChoice(choice)
		choiceLabel := choice
		if choiceIdx >= 0 && choiceIdx < len(options) {
			choiceLabel = options[choiceIdx]
		}
		rating := 5.0
		if rr, ok := ratingByUser[it.UserID]; ok && rr > 0 {
			rating = rr
		}

		team := "unassigned"
		pos := nextPos
		if a, ok := assignments[it.UserID]; ok {
			team = normalizeTeamValue(a.Team)
			pos = a.Position
		}
		players = append(players, TeamSplitPlayer{
			UserID:      it.UserID,
			Username:    it.Username,
			FirstName:   it.FirstName,
			LastName:    it.LastName,
			Choice:      choice,
			ChoiceIndex: choiceIdx,
			ChoiceLabel: choiceLabel,
			Rating:      rating,
			Team:        team,
			Position:    pos,
		})
		nextPos++

		// Add guest slots for extra seats.
		guestCount := it.Seats - 1
		if guestCount < 0 {
			guestCount = 0
		}
		for gi := 1; gi <= guestCount; gi++ {
			guestID := guestSlotUserID(postID, it.UserID, gi)
			gTeam := "unassigned"
			gPos := nextPos
			if a, ok := assignments[guestID]; ok {
				gTeam = normalizeTeamValue(a.Team)
				gPos = a.Position
			}

			display := strings.TrimSpace(strings.TrimSpace(it.FirstName + " " + it.LastName))
			if display == "" && strings.TrimSpace(it.Username) != "" {
				display = "@" + strings.TrimSpace(it.Username)
			}
			if display == "" {
				display = "ID " + strconv.FormatInt(it.UserID, 10)
			}

			players = append(players, TeamSplitPlayer{
				UserID:      guestID,
				Username:    "",
				FirstName:   "Гость",
				LastName:    fmt.Sprintf("(+1 от %s)", display),
				Choice:      choice,
				ChoiceIndex: choiceIdx,
				ChoiceLabel: "Гость (+1)",
				Rating:      5.0,
				Team:        gTeam,
				Position:    gPos,
			})
			nextPos++
		}
	}

	// Stable ordering for UI: by team/position, then by user id.
	sort.Slice(players, func(i, j int) bool {
		ti := normalizeTeamValue(players[i].Team)
		tj := normalizeTeamValue(players[j].Team)
		if ti != tj {
			return ti < tj
		}
		if players[i].Position != players[j].Position {
			return players[i].Position < players[j].Position
		}
		return players[i].UserID < players[j].UserID
	})

	return &EventTeamSplitState{
		EventID: eventID,
		PostID:  postID,
		Players: players,
		Chance:  calculateTeamChance(players),
	}, nil
}

func (s *Store) SaveEventTeamSplit(ctx context.Context, chatID int64, eventID, postID uint64, updates []TeamSplitAssignmentInput) error {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return err
	}

	// Validate that the post belongs to this group/event.
	var exists int64
	if err := s.db.WithContext(ctx).
		Table("event_poll_posts").
		Where("id = ? AND group_id = ? AND event_id = ?", postID, group.ID, eventID).
		Count(&exists).Error; err != nil {
		return err
	}
	if exists == 0 {
		return errors.New("poll post not found")
	}

	// Deduplicate by user_id and normalize team values. Negative user IDs are allowed (synthetic guest slots).
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

	userIDs := make([]int64, 0, len(byUser))
	for uid := range byUser {
		userIDs = append(userIDs, uid)
	}
	sort.Slice(userIDs, func(i, j int) bool { return userIDs[i] < userIDs[j] })

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

		for _, uid := range userIDs {
			upd := byUser[uid]
			row := map[string]interface{}{
				"session_id": session.ID,
				"user_id":    upd.UserID,
				"team":       normalizeTeamValue(upd.Team),
				"position":   upd.Position,
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

	var relationRows []teamSplitRelationRow
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

	if url := strings.TrimSpace(os.Getenv("TEAM_SPLIT_SERVICE_URL")); url != "" {
		assignments, err := callTeamSplitService(ctx, url, state.Players, roleByUser, relationRows)
		if err == nil && len(assignments) > 0 {
			if err := s.SaveEventTeamSplit(ctx, chatID, eventID, postID, assignments); err != nil {
				return nil, err
			}
			return s.GetEventTeamSplitState(ctx, chatID, eventID, postID)
		}
		// Fall back to the in-process algorithm if the service is unavailable.
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

type teamSplitServiceReq struct {
	Players []struct {
		UserID     int64   `json:"userID"`
		Rating     float64 `json:"rating"`
		PlayerType string  `json:"playerType"`
	} `json:"players"`
	Relations []struct {
		UserAID      int64  `json:"userAID"`
		UserBID      int64  `json:"userBID"`
		RelationType string `json:"relationType"`
		Weight       int    `json:"weight"`
	} `json:"relations"`
	TeamCodes []string       `json:"teamCodes,omitempty"`
	Capacity  map[string]int `json:"capacity,omitempty"`
}

type teamSplitServiceResp struct {
	Assignments []TeamSplitAssignmentInput `json:"assignments"`
}

type teamSplitRelationRow struct {
	UserAID      int64
	UserBID      int64
	RelationType string
	Weight       int
}

func callTeamSplitService(
	ctx context.Context,
	baseURL string,
	players []TeamSplitPlayer,
	roleByUser map[int64]string,
	relationRows []teamSplitRelationRow,
) ([]TeamSplitAssignmentInput, error) {
	n := len(players)
	teamCodes := []string{"A", "B"}
	if n > 14 {
		teamCodes = append(teamCodes, "C")
	}
	base := n / len(teamCodes)
	rest := n % len(teamCodes)
	capacity := make(map[string]int, len(teamCodes))
	for idx, code := range teamCodes {
		capacity[code] = base
		if idx < rest {
			capacity[code]++
		}
	}

	reqBody := teamSplitServiceReq{
		TeamCodes: teamCodes,
		Capacity:  capacity,
	}
	reqBody.Players = make([]struct {
		UserID     int64   `json:"userID"`
		Rating     float64 `json:"rating"`
		PlayerType string  `json:"playerType"`
	}, 0, len(players))
	for _, p := range players {
		reqBody.Players = append(reqBody.Players, struct {
			UserID     int64   `json:"userID"`
			Rating     float64 `json:"rating"`
			PlayerType string  `json:"playerType"`
		}{
			UserID:     p.UserID,
			Rating:     p.Rating,
			PlayerType: roleByUser[p.UserID],
		})
	}
	reqBody.Relations = make([]struct {
		UserAID      int64  `json:"userAID"`
		UserBID      int64  `json:"userBID"`
		RelationType string `json:"relationType"`
		Weight       int    `json:"weight"`
	}, 0, len(relationRows))
	for _, r := range relationRows {
		reqBody.Relations = append(reqBody.Relations, struct {
			UserAID      int64  `json:"userAID"`
			UserBID      int64  `json:"userBID"`
			RelationType string `json:"relationType"`
			Weight       int    `json:"weight"`
		}{
			UserAID:      r.UserAID,
			UserBID:      r.UserBID,
			RelationType: r.RelationType,
			Weight:       r.Weight,
		})
	}

	raw, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	url := strings.TrimRight(baseURL, "/") + "/split"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 2 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, errors.New("team split service returned non-2xx")
	}
	var out teamSplitServiceResp
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Assignments, nil
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
				// Never downgrade "admin" to "member" on routine upserts (e.g. poll votes).
				// Allow upgrades to "admin" when the caller provides it.
				"role":         gorm.Expr("CASE WHEN group_members.role = 'admin' THEN 'admin' ELSE EXCLUDED.role END"),
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
