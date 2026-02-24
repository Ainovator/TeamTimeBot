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

const defaultPollMaxPlaces = 18

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
	MaxPlaces               int      `json:"maxPlaces"`
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
	RealName  string     `json:"realName"`
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

type EventSetScore struct {
	Left  int `json:"left"`
	Right int `json:"right"`
}

type EventSetsMatch struct {
	Left  string          `json:"left"`
	Right string          `json:"right"`
	Sets  []EventSetScore `json:"sets"`
}

type EventSetRow struct {
	Ordinal int    `json:"ordinal"`
	Team1   string `json:"team1"`
	Score1  int    `json:"score1"`
	Team2   string `json:"team2"`
	Score2  int    `json:"score2"`
}

type GroupGameRow struct {
	InstanceID uint64    `json:"instanceID"`
	Ordinal    int       `json:"ordinal"`
	EventName  string    `json:"eventName"`
	StartAt    time.Time `json:"startAt"`
	Team1      string    `json:"team1"`
	Score1     int       `json:"score1"`
	Team2      string    `json:"team2"`
	Score2     int       `json:"score2"`
}

type GameRosterResponse struct {
	Team1        string            `json:"team1"`
	Team2        string            `json:"team2"`
	Team1Players []TeamSplitPlayer `json:"team1Players"`
	Team2Players []TeamSplitPlayer `json:"team2Players"`
}

type GroupDebtSummary struct {
	TotalDebt  float64 `json:"totalDebt"`
	UnpaidRows int64   `json:"unpaidRows"`
}

type DebtorTrainingDebt struct {
	InstanceID uint64    `json:"instanceID"`
	EventName  string    `json:"eventName"`
	StartAt    time.Time `json:"startAt"`
	AmountDue  float64   `json:"amountDue"`
}

type GroupDebtor struct {
	UserID    int64                `json:"userID"`
	Username  string               `json:"username"`
	FirstName string               `json:"firstName"`
	LastName  string               `json:"lastName"`
	RealName  string               `json:"realName"`
	TotalDebt float64              `json:"totalDebt"`
	Trainings []DebtorTrainingDebt `json:"trainings"`
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
	UserID       int64     `json:"userID"`
	RealName     string    `json:"realName"`
	Username     string    `json:"username"`
	FirstName    string    `json:"firstName"`
	LastName     string    `json:"lastName"`
	Choice       string    `json:"choice"`
	ChoiceIndex  *int      `json:"choiceIndex,omitempty"`
	ChoiceLabel  string    `json:"choiceLabel"`
	ChoiceWeight int       `json:"choiceWeight"`
	Counted      bool      `json:"counted"`
	Source       string    `json:"source"`
	VotedAt      time.Time `json:"votedAt"`
}

type GroupPollOptionItem struct {
	Choice       string `json:"choice"`
	ChoiceIndex  int    `json:"choiceIndex"`
	ChoiceLabel  string `json:"choiceLabel"`
	ChoiceWeight int    `json:"choiceWeight"`
	Counted      bool   `json:"counted"`
}

type TeamSplitPlayer struct {
	UserID       int64   `json:"userID"`
	GuestOwnerID int64   `json:"guestOwnerID,omitempty"`
	RealName     string  `json:"realName"`
	Username     string  `json:"username"`
	FirstName    string  `json:"firstName"`
	LastName     string  `json:"lastName"`
	Choice       string  `json:"choice"`
	ChoiceIndex  int     `json:"choiceIndex"`
	ChoiceLabel  string  `json:"choiceLabel"`
	Rating       float64 `json:"rating"`
	Team         string  `json:"team"`
	Position     int     `json:"position"`
}

type TeamWinChance struct {
	TeamAScore float64 `json:"teamAScore"`
	TeamBScore float64 `json:"teamBScore"`
	TeamAProb  float64 `json:"teamAProb"`
	TeamBProb  float64 `json:"teamBProb"`
}

type TeamFormationView struct {
	Scheme     string                   `json:"scheme"`
	Analysis   string                   `json:"analysis"`
	Indicators []TeamFormationIndicator `json:"indicators,omitempty"`
}

type TeamFormationIndicator struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	Result    string `json:"result"`
	Reference string `json:"reference"`
	Passed    bool   `json:"passed"`
}

type EventTeamSplitState struct {
	EventID    uint64                       `json:"eventID"`
	PostID     uint64                       `json:"postID"`
	Players    []TeamSplitPlayer            `json:"players"`
	Chance     TeamWinChance                `json:"chance"`
	Formations map[string]TeamFormationView `json:"formations,omitempty"`
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
	MaxPlaces               int
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
	case "central":
		return "central", true
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

func normalizePollMaxPlaces(value int) int {
	if value <= 0 {
		return defaultPollMaxPlaces
	}
	return value
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
	MaxPlaces        int
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
		PollMaxPlaces           int
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
			COALESCE(pt.option_weights, '[]'::jsonb) AS poll_option_weights,
			COALESCE(ge.max_places, 18) AS poll_max_places
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
	record["poll_max_places"] = normalizePollMaxPlaces(snap.PollMaxPlaces)

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
			// When set, poll is bound to an event instance and should obey its lifecycle.
			InstanceID *uint64
		}
		if err := tx.Table("event_poll_posts").
			Select("group_id, instance_id").
			Where("id = ?", postID).
			First(&post).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("poll post not found")
			}
			return err
		}

		// Protect settlements from late poll edits: once the event is no longer in voting,
		// ignore Telegram poll updates (admins can still correct votes via console actions).
		if strings.TrimSpace(source) == "poll" && post.InstanceID != nil && *post.InstanceID != 0 {
			var inst struct {
				Status string
			}
			if err := tx.Table("event_instances").
				Select("status").
				Where("id = ? AND group_id = ? AND is_active = TRUE", *post.InstanceID, post.GroupID).
				Take(&inst).Error; err == nil {
				if strings.TrimSpace(inst.Status) != string(EventHistoryStatusInVoting) {
					return nil
				}
			}
		}

		if err := upsertGroupMemberTx(tx, post.GroupID, userID, "member", "active"); err != nil {
			return err
		}
		if err := ensureDefaultMemberSkillsTx(tx, post.GroupID, userID); err != nil {
			return err
		}

		type existingVoteRow struct {
			Choice  string
			VotedAt time.Time
		}
		var existingVotes []existingVoteRow
		if err := tx.Table("event_poll_votes").
			Select("choice, voted_at").
			Where("post_id = ? AND user_id = ?", postID, userID).
			Scan(&existingVotes).Error; err != nil {
			return err
		}
		existingVotedAtByChoice := make(map[string]time.Time, len(existingVotes))
		for _, row := range existingVotes {
			choice := strings.TrimSpace(row.Choice)
			if choice == "" {
				continue
			}
			existingVotedAtByChoice[choice] = row.VotedAt
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
			choiceVotedAt := votedAt.UTC()
			if existing, ok := existingVotedAtByChoice[c]; ok && !existing.IsZero() {
				// Preserve queue position for choices that user keeps selected.
				choiceVotedAt = existing.UTC()
			}
			votes = append(votes, EventPollVote{
				PostID:    postID,
				UserID:    userID,
				Username:  username,
				FirstName: firstName,
				LastName:  lastName,
				Choice:    c,
				Source:    source,
				VotedAt:   choiceVotedAt,
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

// DeleteGroupPollVoteChoiceByPostUser removes a single choice row for a user in a poll post
// and recalculates the settlement/payments for the bound event instance (if any).
func (s *Store) DeleteGroupPollVoteChoiceByPostUser(ctx context.Context, chatID int64, postID uint64, userID int64, choice string) error {
	if postID == 0 || userID == 0 {
		return errors.New("post_id and user_id are required")
	}
	choice = strings.TrimSpace(choice)
	if choice == "" {
		return errors.New("choice is required")
	}
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return err
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		type postRow struct {
			InstanceID *uint64
		}
		var post postRow
		if err := tx.
			Table("event_poll_posts").
			Select("instance_id").
			Where("id = ? AND group_id = ?", postID, group.ID).
			Take(&post).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("poll post not found")
			}
			return err
		}

		if err := tx.Table("event_poll_votes").
			Where("post_id = ? AND user_id = ? AND choice = ?", postID, userID, choice).
			Delete(&EventPollVote{}).Error; err != nil {
			return err
		}

		if post.InstanceID == nil || *post.InstanceID == 0 {
			return nil
		}
		return s.recalculateEventSettlementByInstanceTx(ctx, tx, group.ID, *post.InstanceID)
	})
}

// DeleteGroupPollVotesByPostAndUser removes all vote rows for a user in a poll post (including multi-choice)
// and recalculates the settlement/payments for the bound event instance (if any).
func (s *Store) DeleteGroupPollVotesByPostAndUser(ctx context.Context, chatID int64, postID uint64, userID int64) error {
	if postID == 0 || userID == 0 {
		return errors.New("post_id and user_id are required")
	}
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return err
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		type postRow struct {
			InstanceID *uint64
		}
		var post postRow
		if err := tx.
			Table("event_poll_posts").
			Select("instance_id").
			Where("id = ? AND group_id = ?", postID, group.ID).
			Take(&post).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("poll post not found")
			}
			return err
		}

		if err := tx.Table("event_poll_votes").
			Where("post_id = ? AND user_id = ?", postID, userID).
			Delete(&EventPollVote{}).Error; err != nil {
			return err
		}

		if post.InstanceID == nil || *post.InstanceID == 0 {
			return nil
		}

		// Settlement might not exist yet: recalc creates it if possible.
		return s.recalculateEventSettlementByInstanceTx(ctx, tx, group.ID, *post.InstanceID)
	})
}

// AddGroupPollVoteChoiceByPostUser adds one option choice for a user in a poll post.
// If this poll is bound to an event instance, settlement/payments are recalculated immediately.
func (s *Store) AddGroupPollVoteChoiceByPostUser(ctx context.Context, chatID int64, postID uint64, userID int64, choice string) error {
	if postID == 0 || userID == 0 {
		return errors.New("post_id and user_id are required")
	}
	choice = strings.TrimSpace(choice)
	if choice == "" {
		return errors.New("choice is required")
	}
	choiceIndex, ok := parsePollOptionChoice(choice)
	if !ok {
		return errors.New("choice must be in option_<index> format")
	}

	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return err
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		type postRow struct {
			InstanceID      *uint64
			TemplateOptions datatypes.JSON
		}
		var post postRow
		if err := tx.
			Table("event_poll_posts epp").
			Select("epp.instance_id, COALESCE(ei.poll_options, pt.options, '[]'::jsonb) AS template_options").
			Joins("JOIN poll_templates pt ON pt.id = epp.template_id").
			Joins("LEFT JOIN event_instances ei ON ei.id = epp.instance_id").
			Where("epp.id = ? AND epp.group_id = ?", postID, group.ID).
			Take(&post).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("poll post not found")
			}
			return err
		}

		var options []string
		_ = json.Unmarshal(post.TemplateOptions, &options)
		if choiceIndex < 0 || choiceIndex >= len(options) {
			return errors.New("choice index is out of range for this poll")
		}

		type userRow struct {
			Username  string
			FirstName string
			LastName  string
		}
		var user userRow
		userExists := false
		if err := tx.Table("telegram_users").
			Select("username, first_name, last_name").
			Where("telegram_id = ?", userID).
			Take(&user).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		} else {
			userExists = true
		}

		if userExists {
			if err := upsertTelegramUserTx(tx, userID, user.Username, user.FirstName, user.LastName); err != nil {
				return err
			}
		}
		if err := upsertGroupMemberTx(tx, group.ID, userID, "member", "active"); err != nil {
			return err
		}
		if err := ensureDefaultMemberSkillsTx(tx, group.ID, userID); err != nil {
			return err
		}

		now := time.Now().UTC()
		vote := EventPollVote{
			PostID:    postID,
			UserID:    userID,
			Username:  strings.TrimSpace(user.Username),
			FirstName: strings.TrimSpace(user.FirstName),
			LastName:  strings.TrimSpace(user.LastName),
			Choice:    choice,
			Source:    "inline",
			VotedAt:   now,
		}
		if err := tx.Table("event_poll_votes").
			Clauses(clause.OnConflict{
				Columns: []clause.Column{
					{Name: "post_id"},
					{Name: "user_id"},
					{Name: "choice"},
				},
				DoUpdates: clause.Assignments(map[string]interface{}{
					"username":   vote.Username,
					"first_name": vote.FirstName,
					"last_name":  vote.LastName,
					"source":     vote.Source,
					"voted_at":   vote.VotedAt,
					"updated_at": gorm.Expr("NOW()"),
				}),
			}).
			Create(&vote).Error; err != nil {
			return err
		}

		if post.InstanceID == nil || *post.InstanceID == 0 {
			return nil
		}

		return s.recalculateEventSettlementByInstanceTx(ctx, tx, group.ID, *post.InstanceID)
	})
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

func (s *Store) recalculateEventSettlementByInstanceTx(ctx context.Context, tx *gorm.DB, groupID uint64, instanceID uint64) error {
	if instanceID == 0 || groupID == 0 {
		return errors.New("group_id and instance_id are required")
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
		PollMaxPlaces  int
		Status         string
	}
	if err := tx.WithContext(ctx).
		Table("event_instances").
		Select("id, group_id, event_id, local_date, poll_post_id, cost_amount, COALESCE(poll_options, '[]'::jsonb) AS poll_options, COALESCE(poll_counted_options, '[]'::jsonb) AS counted_options, COALESCE(poll_option_weights, '[]'::jsonb) AS option_weights, COALESCE(poll_max_places, 18) AS poll_max_places, status").
		Where("id = ? AND group_id = ? AND is_active = TRUE", instanceID, groupID).
		Take(&inst).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("event instance not found")
		}
		return err
	}
	if strings.TrimSpace(inst.Status) == string(EventHistoryStatusNotHeld) {
		// Nothing to recalc for not held events.
		return nil
	}
	if inst.PollPostID == nil || *inst.PollPostID == 0 {
		return nil
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
		return nil
	}
	maxPlaces := normalizePollMaxPlaces(inst.PollMaxPlaces)

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
		Where("post_id = ? AND choice IN ?", *inst.PollPostID, choices).
		Order("voted_at ASC, user_id ASC").
		Scan(&rows).Error; err != nil {
		return err
	}

	type payer struct {
		UserID    int64
		Username  string
		FirstName string
		LastName  string
		Seats     int
	}
	payers := make(map[int64]*payer, len(rows))
	participants := 0
	for _, row := range rows {
		if maxPlaces > 0 && participants >= maxPlaces {
			break
		}
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
		admit := w
		if maxPlaces > 0 {
			remaining := maxPlaces - participants
			if remaining <= 0 {
				break
			}
			if admit > remaining {
				admit = remaining
			}
		}
		if admit <= 0 {
			continue
		}
		p.Seats += admit
		participants += admit
	}

	totalAmount := 4000.0
	if inst.CostAmount != nil {
		totalAmount = *inst.CostAmount
	}
	perPerson := 0.0
	if participants > 0 {
		perPerson = math.Ceil(totalAmount / float64(participants))
	}

	var settlement struct {
		ID uint64
	}
	err := tx.WithContext(ctx).
		Table("event_settlements").
		Select("id").
		Where("group_id = ? AND instance_id = ?", groupID, inst.ID).
		Order("id DESC").
		Take(&settlement).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		record := map[string]interface{}{
			"group_id":           groupID,
			"event_id":           inst.EventID,
			"instance_id":        &inst.ID,
			"post_id":            inst.PollPostID,
			"local_date":         inst.LocalDate.Format("2006-01-02"),
			"total_amount":       totalAmount,
			"participants_count": participants,
			"amount_per_person":  perPerson,
			"sent_at":            gorm.Expr("NOW()"),
			"created_at":         gorm.Expr("NOW()"),
			"updated_at":         gorm.Expr("NOW()"),
		}
		if err := tx.WithContext(ctx).Table("event_settlements").Create(record).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).
			Table("event_settlements").
			Select("id").
			Where("group_id = ? AND instance_id = ?", groupID, inst.ID).
			Order("id DESC").
			Take(&settlement).Error; err != nil {
			return err
		}
	} else {
		if err := tx.WithContext(ctx).
			Table("event_settlements").
			Where("id = ?", settlement.ID).
			Updates(map[string]interface{}{
				"total_amount":       totalAmount,
				"participants_count": participants,
				"amount_per_person":  perPerson,
				"updated_at":         gorm.Expr("NOW()"),
			}).Error; err != nil {
			return err
		}
	}

	// Preserve is_paid/paid_at where possible, but always update amount_due and participant set.
	type existingPayment struct {
		UserID int64
		IsPaid bool
		PaidAt *time.Time
	}
	var existing []existingPayment
	if err := tx.WithContext(ctx).
		Table("event_settlement_payments").
		Select("user_id, is_paid, paid_at").
		Where("settlement_id = ?", settlement.ID).
		Scan(&existing).Error; err != nil {
		return err
	}
	existingByUser := make(map[int64]existingPayment, len(existing))
	for _, e := range existing {
		existingByUser[e.UserID] = e
	}

	userIDs := make([]int64, 0, len(payers))
	for uid := range payers {
		userIDs = append(userIDs, uid)
	}
	if len(userIDs) == 0 {
		// No participants: clear all payments.
		return tx.WithContext(ctx).
			Table("event_settlement_payments").
			Where("settlement_id = ?", settlement.ID).
			Delete(&struct{}{}).Error
	}

	// Remove users who are no longer participants.
	if err := tx.WithContext(ctx).
		Table("event_settlement_payments").
		Where("settlement_id = ? AND user_id NOT IN ?", settlement.ID, userIDs).
		Delete(&struct{}{}).Error; err != nil {
		return err
	}

	for _, uid := range userIDs {
		p := payers[uid]
		if p == nil || p.Seats <= 0 {
			continue
		}
		amountDue := perPerson * float64(p.Seats)

		updates := map[string]interface{}{
			"username":   p.Username,
			"first_name": p.FirstName,
			"last_name":  p.LastName,
			"amount_due": amountDue,
			"updated_at": gorm.Expr("NOW()"),
		}

		if _, ok := existingByUser[uid]; ok {
			if err := tx.WithContext(ctx).
				Table("event_settlement_payments").
				Where("settlement_id = ? AND user_id = ?", settlement.ID, uid).
				Updates(updates).Error; err != nil {
				return err
			}
			continue
		}

		rec := map[string]interface{}{
			"settlement_id": settlement.ID,
			"user_id":       uid,
			"username":      p.Username,
			"first_name":    p.FirstName,
			"last_name":     p.LastName,
			"amount_due":    amountDue,
			"is_paid":       false,
			"created_at":    gorm.Expr("NOW()"),
			"updated_at":    gorm.Expr("NOW()"),
		}
		if err := tx.WithContext(ctx).Table("event_settlement_payments").Create(rec).Error; err != nil {
			return err
		}
	}

	return nil
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
		Select("ge.id AS event_id, ge.group_id, g.chat_id, g.timezone, ge.name, COALESCE(ge.event_type, 'training') AS event_type, ge.start_weekday, COALESCE(ge.poll_publish_weekday, ge.start_weekday) AS poll_publish_weekday, COALESCE(ge.poll_publish_time, ge.start_time) AS poll_publish_time, ge.start_time, ge.end_time, COALESCE(ge.announcement_text, '') AS announcement_text, COALESCE(ge.announcement_enabled, FALSE) AS announcement_enabled, COALESCE(ge.announcement_lead_minutes, 60) AS announcement_lead_minutes, COALESCE(ge.publish_enabled, TRUE) AS publish_enabled, COALESCE(ge.teams_auto_split, FALSE) AS teams_auto_split, COALESCE(ge.teams_publish_list, FALSE) AS teams_publish_list, COALESCE(ge.team_size, 6) AS team_size, COALESCE(ge.max_places, 18) AS max_places, COALESCE(ge.min_votes_to_hold, 0) AS min_votes_to_hold, COALESCE(ge.cancel_lead_minutes, 180) AS cancel_lead_minutes, COALESCE(ge.cancel_notify_enabled, FALSE) AS cancel_notify_enabled, COALESCE(ge.settlement_enabled, TRUE) AS settlement_enabled, COALESCE(ge.settlement_publish_before, FALSE) AS settlement_publish_before, COALESCE(ge.settlement_publish_after, TRUE) AS settlement_publish_after, ge.cost_amount, COALESCE(pt.name, '') AS poll_template").
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

func (s *Store) HasEventPollPostForInstance(ctx context.Context, instanceID uint64) (bool, error) {
	if instanceID == 0 {
		return false, errors.New("instance_id is required")
	}
	var count int64
	if err := s.db.WithContext(ctx).
		Table("event_poll_posts").
		Where("instance_id = ?", instanceID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
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
	PollMaxPlaces      int
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
		PollMaxPlaces           int
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
			COALESCE(ei.poll_option_weights, '[]'::jsonb) AS poll_option_weights,
			COALESCE(ei.poll_max_places, 18) AS poll_max_places
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
		PollMaxPlaces:           normalizePollMaxPlaces(r.PollMaxPlaces),
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
		PollMaxPlaces       int
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
				COALESCE(ei.poll_option_weights, '[]'::jsonb) AS poll_option_weights,
				COALESCE(ei.poll_max_places, 18) AS poll_max_places
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
			PollMaxPlaces:       normalizePollMaxPlaces(r.PollMaxPlaces),
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
	maxPlaces := defaultPollMaxPlaces
	if settlement.InstanceID != nil {
		var snap struct {
			Options        datatypes.JSON
			CountedOptions datatypes.JSON
			OptionWeights  datatypes.JSON
			MaxPlaces      int
		}
		if err := tx.WithContext(ctx).
			Table("event_instances").
			Select("COALESCE(poll_options, '[]'::jsonb) AS options, COALESCE(poll_counted_options, '[]'::jsonb) AS counted_options, COALESCE(poll_option_weights, '[]'::jsonb) AS option_weights, COALESCE(poll_max_places, 18) AS max_places").
			Where("id = ?", *settlement.InstanceID).
			Take(&snap).Error; err != nil {
			return err
		}
		_ = json.Unmarshal(snap.Options, &options)
		_ = json.Unmarshal(snap.CountedOptions, &counted)
		_ = json.Unmarshal(snap.OptionWeights, &weights)
		maxPlaces = normalizePollMaxPlaces(snap.MaxPlaces)
	} else {
		var template struct {
			Options        datatypes.JSON
			CountedOptions datatypes.JSON
			OptionWeights  datatypes.JSON
			MaxPlaces      int
		}
		if err := tx.WithContext(ctx).
			Table("event_poll_posts epp").
			Select("COALESCE(pt.options, '[]'::jsonb) AS options, COALESCE(pt.counted_options, '[]'::jsonb) AS counted_options, COALESCE(pt.option_weights, '[]'::jsonb) AS option_weights, COALESCE(ge.max_places, 18) AS max_places").
			Joins("JOIN poll_templates pt ON pt.id = epp.template_id").
			Joins("LEFT JOIN group_events ge ON ge.id = epp.event_id").
			Where("epp.id = ?", *settlement.PostID).
			Take(&template).Error; err != nil {
			return err
		}
		_ = json.Unmarshal(template.Options, &options)
		_ = json.Unmarshal(template.CountedOptions, &counted)
		_ = json.Unmarshal(template.OptionWeights, &weights)
		maxPlaces = normalizePollMaxPlaces(template.MaxPlaces)
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
		Order("voted_at ASC, user_id ASC").
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
	admittedSeats := 0
	for _, row := range rows {
		if maxPlaces > 0 && admittedSeats >= maxPlaces {
			break
		}
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
		admit := w
		if maxPlaces > 0 {
			remaining := maxPlaces - admittedSeats
			if remaining <= 0 {
				break
			}
			if admit > remaining {
				admit = remaining
			}
		}
		if admit <= 0 {
			continue
		}
		p.Seats += admit
		admittedSeats += admit
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
		RealName  string
		Username  string
		FirstName string
		LastName  string
		AmountDue float64
		IsPaid    bool
		PaidAt    *time.Time
	}
	var rows []paymentRow
	if err := s.db.WithContext(ctx).
		Table("event_settlement_payments esp").
		Select(`
			esp.user_id,
			COALESCE(gm.real_name, '') AS real_name,
			COALESCE(esp.username, '') AS username,
			COALESCE(esp.first_name, '') AS first_name,
			COALESCE(esp.last_name, '') AS last_name,
			esp.amount_due,
			esp.is_paid,
			esp.paid_at
		`).
		Joins("LEFT JOIN group_members gm ON gm.group_id = ? AND gm.user_telegram_id = esp.user_id AND gm.is_active = TRUE", group.ID).
		Where("esp.settlement_id = ?", settlement.ID).
		Order("COALESCE(gm.real_name, '') ASC, first_name ASC, last_name ASC, username ASC, esp.user_id ASC").
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
			RealName:  strings.TrimSpace(row.RealName),
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
		RealName  string
		Username  string
		FirstName string
		LastName  string
		AmountDue float64
		IsPaid    bool
		PaidAt    *time.Time
	}
	var rows []paymentRow
	if err := s.db.WithContext(ctx).
		Table("event_settlement_payments esp").
		Select(`
			esp.user_id,
			COALESCE(gm.real_name, '') AS real_name,
			COALESCE(esp.username, '') AS username,
			COALESCE(esp.first_name, '') AS first_name,
			COALESCE(esp.last_name, '') AS last_name,
			esp.amount_due,
			esp.is_paid,
			esp.paid_at
		`).
		Joins("LEFT JOIN group_members gm ON gm.group_id = ? AND gm.user_telegram_id = esp.user_id AND gm.is_active = TRUE", group.ID).
		Where("esp.settlement_id = ?", settlement.ID).
		Order("COALESCE(gm.real_name, '') ASC, first_name ASC, last_name ASC, username ASC, esp.user_id ASC").
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
			RealName:  strings.TrimSpace(row.RealName),
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
	for _, idx := range counted {
		choice := "option_" + strconv.Itoa(idx)
		choices = append(choices, choice)
	}

	// If settlement already exists, return it (and ensure missing payment rows exist).
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.recalculateEventSettlementByInstanceTx(ctx, tx, group.ID, inst.ID)
	}); err != nil {
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

func (s *Store) GetEventSetRowsByInstance(ctx context.Context, chatID int64, instanceID uint64) ([]EventSetRow, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	var instance struct {
		ID uint64
	}
	if err := s.db.WithContext(ctx).
		Table("event_instances").
		Select("id").
		Where("id = ? AND group_id = ? AND is_active = TRUE", instanceID, group.ID).
		Take(&instance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	type row struct {
		Ordinal int
		Team1   string
		Score1  int
		Team2   string
		Score2  int
	}
	var rows []row
	if err := s.db.WithContext(ctx).
		Table("event_instance_set_rows").
		Select("ordinal, team1, score1, team2, score2").
		Where("group_id = ? AND instance_id = ?", group.ID, instance.ID).
		Order("ordinal ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]EventSetRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, EventSetRow{
			Ordinal: r.Ordinal,
			Team1:   strings.TrimSpace(r.Team1),
			Score1:  r.Score1,
			Team2:   strings.TrimSpace(r.Team2),
			Score2:  r.Score2,
		})
	}
	return out, nil
}

func (s *Store) SaveEventSetRowsByInstance(ctx context.Context, chatID int64, instanceID uint64, rows []EventSetRow) error {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return err
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
			return errors.New("event instance not found")
		}
		return err
	}

	if rows == nil {
		rows = []EventSetRow{}
	}

	allowed := map[string]struct{}{"A": {}, "B": {}, "C": {}}
	seenOrd := make(map[int]struct{}, len(rows))
	normalized := make([]EventSetRow, 0, len(rows))
	for _, r := range rows {
		ord := r.Ordinal
		if ord <= 0 {
			return errors.New("invalid ordinal")
		}
		if _, ok := seenOrd[ord]; ok {
			return errors.New("duplicate ordinal")
		}
		seenOrd[ord] = struct{}{}

		t1 := strings.TrimSpace(r.Team1)
		t2 := strings.TrimSpace(r.Team2)
		if _, ok := allowed[t1]; !ok {
			return errors.New("invalid team1")
		}
		if _, ok := allowed[t2]; !ok {
			return errors.New("invalid team2")
		}
		if t1 == t2 {
			return errors.New("teams must be different")
		}
		if r.Score1 < 0 || r.Score2 < 0 {
			return errors.New("score cannot be negative")
		}
		if r.Score1 > 99 || r.Score2 > 99 {
			return errors.New("score is too large")
		}
		normalized = append(normalized, EventSetRow{
			Ordinal: ord,
			Team1:   t1,
			Score1:  r.Score1,
			Team2:   t2,
			Score2:  r.Score2,
		})
	}

	sort.Slice(normalized, func(i, j int) bool { return normalized[i].Ordinal < normalized[j].Ordinal })

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("event_instance_set_rows").
			Where("group_id = ? AND instance_id = ?", group.ID, instance.ID).
			Delete(nil).Error; err != nil {
			return err
		}
		if len(normalized) == 0 {
			return nil
		}

		inserts := make([]map[string]interface{}, 0, len(normalized))
		for _, r := range normalized {
			inserts = append(inserts, map[string]interface{}{
				"group_id":    group.ID,
				"event_id":    instance.EventID,
				"instance_id": instance.ID,
				"ordinal":     r.Ordinal,
				"team1":       r.Team1,
				"score1":      r.Score1,
				"team2":       r.Team2,
				"score2":      r.Score2,
				"created_at":  gorm.Expr("NOW()"),
				"updated_at":  gorm.Expr("NOW()"),
			})
		}
		return tx.Table("event_instance_set_rows").Create(&inserts).Error
	})
}

func (s *Store) GetEventInstanceHeader(ctx context.Context, chatID int64, instanceID uint64) (string, time.Time, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return "", time.Time{}, err
	}
	var row struct {
		EventName string
		LocalDate time.Time
	}
	if err := s.db.WithContext(ctx).
		Table("event_instances ei").
		Select("COALESCE(NULLIF(ei.event_name, ''), ge.name) AS event_name, ei.local_date").
		Joins("JOIN group_events ge ON ge.id = ei.event_id").
		Where("ei.id = ? AND ei.group_id = ? AND ei.is_active = TRUE", instanceID, group.ID).
		Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", time.Time{}, errors.New("event instance not found")
		}
		return "", time.Time{}, err
	}
	return strings.TrimSpace(row.EventName), row.LocalDate, nil
}

func (s *Store) ListGroupGames(ctx context.Context, chatID int64) ([]GroupGameRow, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	type row struct {
		InstanceID uint64
		Ordinal    int
		EventName  string
		StartAt    time.Time
		Team1      string
		Score1     int
		Team2      string
		Score2     int
	}
	var rows []row
	if err := s.db.WithContext(ctx).
		Table("event_instance_set_rows eis").
		Select("eis.instance_id, eis.ordinal, COALESCE(NULLIF(ei.event_name, ''), '') AS event_name, ei.planned_start_at AS start_at, eis.team1, eis.score1, eis.team2, eis.score2").
		Joins("JOIN event_instances ei ON ei.id = eis.instance_id").
		Where("eis.group_id = ? AND ei.group_id = ? AND ei.is_active = TRUE", group.ID, group.ID).
		Order("ei.planned_start_at DESC, eis.instance_id DESC, eis.ordinal ASC").
		Limit(250).
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]GroupGameRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, GroupGameRow{
			InstanceID: r.InstanceID,
			Ordinal:    r.Ordinal,
			EventName:  strings.TrimSpace(r.EventName),
			StartAt:    r.StartAt,
			Team1:      strings.TrimSpace(r.Team1),
			Score1:     r.Score1,
			Team2:      strings.TrimSpace(r.Team2),
			Score2:     r.Score2,
		})
	}
	return out, nil
}

func (s *Store) GetGameRosterByInstance(ctx context.Context, chatID int64, instanceID uint64, team1, team2 string) (*GameRosterResponse, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	type instRow struct {
		EventID    uint64
		PollPostID *uint64
	}
	var inst instRow
	if err := s.db.WithContext(ctx).
		Table("event_instances").
		Select("event_id, poll_post_id").
		Where("id = ? AND group_id = ? AND is_active = TRUE", instanceID, group.ID).
		Take(&inst).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("event instance not found")
		}
		return nil, err
	}

	t1 := strings.TrimSpace(team1)
	t2 := strings.TrimSpace(team2)
	out := &GameRosterResponse{
		Team1:        t1,
		Team2:        t2,
		Team1Players: []TeamSplitPlayer{},
		Team2Players: []TeamSplitPlayer{},
	}
	if inst.PollPostID == nil || *inst.PollPostID == 0 {
		return out, nil
	}

	state, err := s.GetEventTeamSplitState(ctx, chatID, inst.EventID, *inst.PollPostID)
	if err != nil || state == nil {
		return out, nil
	}
	for _, p := range state.Players {
		if strings.TrimSpace(p.Team) == t1 {
			out.Team1Players = append(out.Team1Players, p)
			continue
		}
		if strings.TrimSpace(p.Team) == t2 {
			out.Team2Players = append(out.Team2Players, p)
			continue
		}
	}
	return out, nil
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

func (s *Store) ListGroupDebtors(ctx context.Context, chatID int64) ([]GroupDebtor, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	type row struct {
		UserID     int64
		Username   string
		FirstName  string
		LastName   string
		RealName   string
		InstanceID *uint64
		EventName  string
		StartAt    *time.Time
		AmountDue  float64
	}

	var rows []row
	if err := s.db.WithContext(ctx).
		Table("event_settlement_payments esp").
		Select(`
			esp.user_id,
			COALESCE(NULLIF(tu.username, ''), NULLIF(esp.username, ''), '') AS username,
			COALESCE(NULLIF(tu.first_name, ''), NULLIF(esp.first_name, ''), '') AS first_name,
			COALESCE(NULLIF(tu.last_name, ''), NULLIF(esp.last_name, ''), '') AS last_name,
			COALESCE(gm.real_name, '') AS real_name,
			es.instance_id,
			COALESCE(NULLIF(ei.event_name, ''), NULLIF(ge.name, ''), '') AS event_name,
			ei.planned_start_at AS start_at,
			esp.amount_due::float8 AS amount_due
		`).
		Joins("JOIN event_settlements es ON es.id = esp.settlement_id").
		Joins("LEFT JOIN event_instances ei ON ei.id = es.instance_id").
		Joins("LEFT JOIN group_events ge ON ge.id = es.event_id").
		Joins("LEFT JOIN telegram_users tu ON tu.telegram_id = esp.user_id").
		Joins("LEFT JOIN group_members gm ON gm.group_id = es.group_id AND gm.user_telegram_id = esp.user_id AND gm.is_active = TRUE").
		Where("es.group_id = ? AND esp.is_paid = FALSE", group.ID).
		Order("esp.user_id ASC, ei.planned_start_at DESC, es.id DESC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	byUser := map[int64]*GroupDebtor{}
	order := make([]int64, 0)
	for _, r := range rows {
		d, ok := byUser[r.UserID]
		if !ok {
			d = &GroupDebtor{
				UserID:    r.UserID,
				Username:  strings.TrimSpace(r.Username),
				FirstName: strings.TrimSpace(r.FirstName),
				LastName:  strings.TrimSpace(r.LastName),
				RealName:  strings.TrimSpace(r.RealName),
				TotalDebt: 0,
				Trainings: make([]DebtorTrainingDebt, 0),
			}
			byUser[r.UserID] = d
			order = append(order, r.UserID)
		}
		d.TotalDebt += r.AmountDue
		if r.InstanceID != nil && *r.InstanceID != 0 && r.StartAt != nil {
			d.Trainings = append(d.Trainings, DebtorTrainingDebt{
				InstanceID: *r.InstanceID,
				EventName:  strings.TrimSpace(r.EventName),
				StartAt:    *r.StartAt,
				AmountDue:  r.AmountDue,
			})
		}
	}

	out := make([]GroupDebtor, 0, len(order))
	for _, uid := range order {
		if d := byUser[uid]; d != nil {
			out = append(out, *d)
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].TotalDebt != out[j].TotalDebt {
			return out[i].TotalDebt > out[j].TotalDebt
		}
		return out[i].UserID < out[j].UserID
	})
	return out, nil
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
		MaxPlaces        int
	}
	if err := s.db.WithContext(ctx).
		Table("group_events ge").
		Select("ge.id AS event_id, ge.name AS event_name, pt.name AS template_name, pt.question AS template_question, pt.options AS template_options, COALESCE(ge.max_places, 18) AS max_places").
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
		MaxPlaces:        normalizePollMaxPlaces(row.MaxPlaces),
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
	RealName  string
	Username  string
	FirstName string
	LastName  string
	Seats     int
}

func (s *Store) ListSeatCountsForPostChoices(ctx context.Context, postID uint64, choices []string, weightByChoice map[string]int) ([]PollSeatCountItem, int, error) {
	return s.ListSeatCountsForPostChoicesWithLimit(ctx, postID, choices, weightByChoice, 0)
}

func (s *Store) ListSeatCountsForPostChoicesWithLimit(
	ctx context.Context,
	postID uint64,
	choices []string,
	weightByChoice map[string]int,
	maxSeats int,
) ([]PollSeatCountItem, int, error) {
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
		RealName  string
		Username  string
		FirstName string
		LastName  string
		Choice    string
		VotedAt   time.Time
	}
	var rows []voteRow
	if err := s.db.WithContext(ctx).
		Table("event_poll_votes ev").
		Select(`
			ev.user_id,
			COALESCE(gm.real_name, '') AS real_name,
			COALESCE(tu.username, ev.username, '') AS username,
			COALESCE(tu.first_name, ev.first_name, '') AS first_name,
			COALESCE(tu.last_name, ev.last_name, '') AS last_name,
			ev.choice,
			ev.voted_at
		`).
		Joins("LEFT JOIN telegram_users tu ON tu.telegram_id = ev.user_id").
		Joins("JOIN event_poll_posts epp ON epp.id = ev.post_id").
		Joins("LEFT JOIN group_members gm ON gm.group_id = epp.group_id AND gm.user_telegram_id = ev.user_id AND gm.is_active = TRUE").
		Where("ev.post_id = ? AND ev.choice IN ?", postID, filtered).
		Order("ev.voted_at ASC, ev.user_id ASC").
		Scan(&rows).Error; err != nil {
		return nil, 0, err
	}

	limit := maxSeats
	if limit > 0 {
		limit = normalizePollMaxPlaces(limit)
	}

	byUser := make(map[int64]*PollSeatCountItem, len(rows))
	totalSeats := 0
	for _, r := range rows {
		if limit > 0 && totalSeats >= limit {
			break
		}
		item, ok := byUser[r.UserID]
		if !ok {
			item = &PollSeatCountItem{
				UserID:    r.UserID,
				RealName:  strings.TrimSpace(r.RealName),
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
		admit := w
		if limit > 0 {
			remaining := limit - totalSeats
			if remaining <= 0 {
				break
			}
			if admit > remaining {
				admit = remaining
			}
		}
		if admit <= 0 {
			continue
		}
		item.Seats += admit
		totalSeats += admit
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

func (s *Store) GetLatestGroupPollByChatID(ctx context.Context, chatID int64) (*uint64, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	type row struct {
		PostID uint64
	}
	var r row

	err = s.db.WithContext(ctx).
		Table("event_poll_posts").
		Select("id AS post_id").
		Where("group_id = ?", group.ID).
		Order("published_at DESC, id DESC").
		Take(&r).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	id := r.PostID
	return &id, nil
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

func (s *Store) GetPollSeatConfigByPostID(ctx context.Context, chatID int64, postID uint64) ([]string, map[string]int, int, error) {
	if postID == 0 {
		return nil, nil, 0, errors.New("post_id is required")
	}
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, nil, 0, err
	}

	type row struct {
		TemplateOptions datatypes.JSON
		CountedOptions  datatypes.JSON
		OptionWeights   datatypes.JSON
		MaxPlaces       int
	}
	var r row
	if err := s.db.WithContext(ctx).
		Table("event_poll_posts epp").
		Select("COALESCE(ei.poll_options, pt.options, '[]'::jsonb) AS template_options, COALESCE(ei.poll_counted_options, pt.counted_options, '[]'::jsonb) AS counted_options, COALESCE(ei.poll_option_weights, pt.option_weights, '[]'::jsonb) AS option_weights, COALESCE(ei.poll_max_places, ge.max_places, 18) AS max_places").
		Joins("JOIN poll_templates pt ON pt.id = epp.template_id").
		Joins("LEFT JOIN event_instances ei ON ei.id = epp.instance_id").
		Joins("LEFT JOIN group_events ge ON ge.id = epp.event_id").
		Where("epp.id = ? AND epp.group_id = ?", postID, group.ID).
		Take(&r).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, 0, errors.New("poll post not found")
		}
		return nil, nil, 0, err
	}

	var options []string
	_ = json.Unmarshal(r.TemplateOptions, &options)
	var counted []int
	_ = json.Unmarshal(r.CountedOptions, &counted)
	var weights []int
	_ = json.Unmarshal(r.OptionWeights, &weights)
	counted = normalizeCountedOptionIndexes(len(options), counted)
	weights = normalizeOptionWeightsLen(len(options), weights)

	choices := make([]string, 0, len(counted))
	weightByChoice := make(map[string]int, len(counted))
	for _, idx := range counted {
		choice := "option_" + strconv.Itoa(idx)
		choices = append(choices, choice)
		w := 1
		if idx >= 0 && idx < len(weights) && weights[idx] > 0 {
			w = weights[idx]
		}
		weightByChoice[choice] = w
	}
	return choices, weightByChoice, normalizePollMaxPlaces(r.MaxPlaces), nil
}

func (s *Store) ListGroupPollOptionsByPostID(ctx context.Context, chatID int64, postID uint64) ([]GroupPollOptionItem, error) {
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
		OptionWeights   datatypes.JSON
	}
	var post postRow
	if err := s.db.WithContext(ctx).
		Table("event_poll_posts epp").
		Select("COALESCE(ei.poll_options, pt.options, '[]'::jsonb) AS template_options, COALESCE(ei.poll_counted_options, pt.counted_options, '[]'::jsonb) AS counted_options, COALESCE(ei.poll_option_weights, pt.option_weights, '[]'::jsonb) AS option_weights").
		Joins("JOIN poll_templates pt ON pt.id = epp.template_id").
		Joins("LEFT JOIN event_instances ei ON ei.id = epp.instance_id").
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
	var weights []int
	_ = json.Unmarshal(post.OptionWeights, &weights)
	counted = normalizeCountedOptionIndexes(len(options), counted)
	weights = normalizeOptionWeightsLen(len(options), weights)
	countedIdx := make(map[int]struct{}, len(counted))
	for _, idx := range counted {
		countedIdx[idx] = struct{}{}
	}

	items := make([]GroupPollOptionItem, 0, len(options))
	for idx, label := range options {
		_, isCounted := countedIdx[idx]
		choiceLabel := strings.TrimSpace(label)
		if choiceLabel == "" {
			choiceLabel = "option_" + strconv.Itoa(idx)
		}
		items = append(items, GroupPollOptionItem{
			Choice:       "option_" + strconv.Itoa(idx),
			ChoiceIndex:  idx,
			ChoiceLabel:  choiceLabel,
			ChoiceWeight: weights[idx],
			Counted:      isCounted,
		})
	}
	return items, nil
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
		OptionWeights   datatypes.JSON
	}
	var post postRow
	if err := s.db.WithContext(ctx).
		Table("event_poll_posts epp").
		Select("COALESCE(ei.poll_options, pt.options, '[]'::jsonb) AS template_options, COALESCE(ei.poll_counted_options, pt.counted_options, '[]'::jsonb) AS counted_options, COALESCE(ei.poll_option_weights, pt.option_weights, '[]'::jsonb) AS option_weights").
		Joins("JOIN poll_templates pt ON pt.id = epp.template_id").
		Joins("LEFT JOIN event_instances ei ON ei.id = epp.instance_id").
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
	var weights []int
	_ = json.Unmarshal(post.OptionWeights, &weights)
	counted = normalizeCountedOptionIndexes(len(options), counted)
	weights = normalizeOptionWeightsLen(len(options), weights)
	countedChoices := make(map[string]struct{}, len(counted))
	for _, idx := range counted {
		countedChoices["option_"+strconv.Itoa(idx)] = struct{}{}
	}

	type voteRow struct {
		UserID    int64
		RealName  string
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
		Select("ev.user_id, COALESCE(gm.real_name, '') AS real_name, COALESCE(tu.username, ev.username, '') AS username, COALESCE(tu.first_name, ev.first_name, '') AS first_name, COALESCE(tu.last_name, ev.last_name, '') AS last_name, ev.choice, ev.source, ev.voted_at").
		Joins("LEFT JOIN telegram_users tu ON tu.telegram_id = ev.user_id").
		Joins("LEFT JOIN group_members gm ON gm.group_id = ? AND gm.user_telegram_id = ev.user_id AND gm.is_active = TRUE", group.ID).
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
		choiceWeight := 1
		if choiceIndex != nil {
			idx := *choiceIndex
			if idx >= 0 && idx < len(weights) && weights[idx] > 0 {
				choiceWeight = weights[idx]
			}
		}
		_, countedChoice := countedChoices[row.Choice]
		items = append(items, GroupPollVoteItem{
			UserID:       row.UserID,
			RealName:     strings.TrimSpace(row.RealName),
			Username:     row.Username,
			FirstName:    row.FirstName,
			LastName:     row.LastName,
			Choice:       row.Choice,
			ChoiceIndex:  choiceIndex,
			ChoiceLabel:  choiceLabel,
			ChoiceWeight: choiceWeight,
			Counted:      countedChoice,
			Source:       row.Source,
			VotedAt:      row.VotedAt,
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
		MaxPlaces       int
	}
	var post postRow
	if err := s.db.WithContext(ctx).
		Table("event_poll_posts epp").
		Select("epp.id AS post_id, epp.instance_id, COALESCE(ei.poll_options, pt.options, '[]'::jsonb) AS template_options, COALESCE(ei.poll_counted_options, pt.counted_options, '[]'::jsonb) AS counted_options, COALESCE(ei.poll_option_weights, pt.option_weights, '[]'::jsonb) AS option_weights, COALESCE(ei.poll_max_places, ge.max_places, 18) AS max_places").
		Joins("JOIN poll_templates pt ON pt.id = epp.template_id").
		Joins("LEFT JOIN event_instances ei ON ei.id = epp.instance_id").
		Joins("LEFT JOIN group_events ge ON ge.id = epp.event_id").
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

	seatItems, _, err := s.ListSeatCountsForPostChoicesWithLimit(ctx, postID, keysOfMap(choiceSet), weightByChoice, normalizePollMaxPlaces(post.MaxPlaces))
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
			UserID:       it.UserID,
			GuestOwnerID: 0,
			RealName:     strings.TrimSpace(it.RealName),
			Username:     it.Username,
			FirstName:    it.FirstName,
			LastName:     it.LastName,
			Choice:       choice,
			ChoiceIndex:  choiceIdx,
			ChoiceLabel:  choiceLabel,
			Rating:       rating,
			Team:         team,
			Position:     pos,
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
				UserID:       guestID,
				GuestOwnerID: it.UserID,
				RealName:     "",
				Username:     "",
				FirstName:    "Гость",
				LastName:     fmt.Sprintf("(+1 от %s)", display),
				Choice:       choice,
				ChoiceIndex:  choiceIdx,
				ChoiceLabel:  "Гость (+1)",
				Rating:       5.0,
				Team:         gTeam,
				Position:     gPos,
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

func (s *Store) GetTeamSplitPlayerProfiles(
	ctx context.Context,
	chatID int64,
	players []TeamSplitPlayer,
) (map[int64]string, map[int64]map[string]float64, error) {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, nil, err
	}

	realUserIDSet := make(map[int64]struct{}, len(players))
	for _, p := range players {
		if p.UserID > 0 {
			realUserIDSet[p.UserID] = struct{}{}
		}
		if p.GuestOwnerID > 0 {
			realUserIDSet[p.GuestOwnerID] = struct{}{}
		}
	}
	realUserIDs := make([]int64, 0, len(realUserIDSet))
	for uid := range realUserIDSet {
		realUserIDs = append(realUserIDs, uid)
	}
	sort.Slice(realUserIDs, func(i, j int) bool { return realUserIDs[i] < realUserIDs[j] })

	roleByUser := make(map[int64]string, len(realUserIDs))
	skillByUser := make(map[int64]map[string]float64, len(realUserIDs))
	if len(realUserIDs) == 0 {
		return roleByUser, skillByUser, nil
	}

	type roleRow struct {
		UserID     int64
		PlayerType string
	}
	var roleRows []roleRow
	if err := s.db.WithContext(ctx).
		Table("group_members").
		Select("user_telegram_id AS user_id, COALESCE(player_type, '') AS player_type").
		Where("group_id = ? AND user_telegram_id IN ? AND is_active = TRUE", group.ID, realUserIDs).
		Scan(&roleRows).Error; err != nil {
		return nil, nil, err
	}
	for _, row := range roleRows {
		normalized, ok := normalizePlayerType(row.PlayerType)
		if ok {
			roleByUser[row.UserID] = normalized
		}
	}

	type skillRow struct {
		UserID    int64
		SkillCode string
		Score     float64
	}
	var skillRows []skillRow
	if err := s.db.WithContext(ctx).
		Table("group_member_skills gms").
		Select("gms.user_telegram_id AS user_id, sc.code AS skill_code, gms.score::float8 AS score").
		Joins("JOIN skills_catalog sc ON sc.id = gms.skill_id AND sc.is_active = TRUE").
		Where("gms.group_id = ? AND gms.user_telegram_id IN ?", group.ID, realUserIDs).
		Scan(&skillRows).Error; err != nil {
		return nil, nil, err
	}
	for _, row := range skillRows {
		if _, ok := skillByUser[row.UserID]; !ok {
			skillByUser[row.UserID] = make(map[string]float64)
		}
		skillByUser[row.UserID][strings.TrimSpace(row.SkillCode)] = row.Score
	}

	return roleByUser, skillByUser, nil
}

func (s *Store) SaveAndReanalyzeEventTeamSplit(
	ctx context.Context,
	chatID int64,
	eventID, postID uint64,
	updates []TeamSplitAssignmentInput,
) (*EventTeamSplitState, error) {
	if err := s.SaveEventTeamSplit(ctx, chatID, eventID, postID, updates); err != nil {
		return nil, err
	}
	return s.ReanalyzeEventTeamSplit(ctx, chatID, eventID, postID)
}

func (s *Store) ReanalyzeEventTeamSplit(
	ctx context.Context,
	chatID int64,
	eventID, postID uint64,
) (*EventTeamSplitState, error) {
	state, err := s.GetEventTeamSplitState(ctx, chatID, eventID, postID)
	if err != nil {
		return nil, err
	}
	if state == nil || len(state.Players) == 0 {
		return state, nil
	}

	roleByUser, skillByUser, err := s.GetTeamSplitPlayerProfiles(ctx, chatID, state.Players)
	if err != nil {
		return nil, err
	}

	playersByTeam := map[string][]TeamSplitPlayer{
		"A":          {},
		"B":          {},
		"C":          {},
		"unassigned": {},
	}
	for _, p := range state.Players {
		team := normalizeTeamValue(p.Team)
		playersByTeam[team] = append(playersByTeam[team], p)
	}

	assignments := make([]TeamSplitAssignmentInput, 0, len(state.Players))
	for _, code := range []string{"A", "B", "C"} {
		ordered := orderTeamPlayersForLineup(playersByTeam[code], roleByUser, skillByUser)
		for pos, p := range ordered {
			assignments = append(assignments, TeamSplitAssignmentInput{
				UserID:   p.UserID,
				Team:     code,
				Position: pos,
			})
		}
	}

	unassigned := playersByTeam["unassigned"]
	sort.SliceStable(unassigned, func(i, j int) bool {
		if unassigned[i].Position != unassigned[j].Position {
			return unassigned[i].Position < unassigned[j].Position
		}
		return unassigned[i].UserID < unassigned[j].UserID
	})
	for pos, p := range unassigned {
		assignments = append(assignments, TeamSplitAssignmentInput{
			UserID:   p.UserID,
			Team:     "unassigned",
			Position: pos,
		})
	}

	assignments = enforceGuestOwnerAssignments(state.Players, assignments)
	assignments = normalizeTeamAssignmentPositions(assignments)

	if err := s.SaveEventTeamSplit(ctx, chatID, eventID, postID, assignments); err != nil {
		return nil, err
	}
	return s.GetEventTeamSplitState(ctx, chatID, eventID, postID)
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
	// Enforce invariant: guest slots (+1) must always stay in the same team as their owner.
	// This guards both auto-split and manual UI edits.
	if currentState, err := s.GetEventTeamSplitState(ctx, chatID, eventID, postID); err == nil && currentState != nil {
		for _, p := range currentState.Players {
			if p.UserID >= 0 || p.GuestOwnerID <= 0 {
				continue
			}
			owner, ok := byUser[p.GuestOwnerID]
			if !ok {
				continue
			}
			guest, exists := byUser[p.UserID]
			if !exists {
				guest = TeamSplitAssignmentInput{
					UserID:   p.UserID,
					Position: owner.Position + 1,
				}
			}
			guest.Team = owner.Team
			byUser[p.UserID] = guest
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

	realUserIDSet := make(map[int64]struct{}, len(state.Players))
	for _, p := range state.Players {
		if p.UserID > 0 {
			realUserIDSet[p.UserID] = struct{}{}
		}
		if p.GuestOwnerID > 0 {
			realUserIDSet[p.GuestOwnerID] = struct{}{}
		}
	}
	realUserIDs := make([]int64, 0, len(realUserIDSet))
	for uid := range realUserIDSet {
		realUserIDs = append(realUserIDs, uid)
	}
	sort.Slice(realUserIDs, func(i, j int) bool { return realUserIDs[i] < realUserIDs[j] })

	type roleRow struct {
		UserID     int64
		PlayerType string
	}
	roleByUser := make(map[int64]string, len(realUserIDs))
	if len(realUserIDs) > 0 {
		var roleRows []roleRow
		if err := s.db.WithContext(ctx).
			Table("group_members").
			Select("user_telegram_id AS user_id, COALESCE(player_type, '') AS player_type").
			Where("group_id = ? AND user_telegram_id IN ? AND is_active = TRUE", group.ID, realUserIDs).
			Scan(&roleRows).Error; err != nil {
			return nil, err
		}
		for _, row := range roleRows {
			normalized, ok := normalizePlayerType(row.PlayerType)
			if ok {
				roleByUser[row.UserID] = normalized
			}
		}
	}

	type skillRow struct {
		UserID    int64
		SkillCode string
		Score     float64
	}
	skillByUser := make(map[int64]map[string]float64, len(realUserIDs))
	if len(realUserIDs) > 0 {
		var skillRows []skillRow
		if err := s.db.WithContext(ctx).
			Table("group_member_skills gms").
			Select("gms.user_telegram_id AS user_id, sc.code AS skill_code, gms.score::float8 AS score").
			Joins("JOIN skills_catalog sc ON sc.id = gms.skill_id AND sc.is_active = TRUE").
			Where("gms.group_id = ? AND gms.user_telegram_id IN ?", group.ID, realUserIDs).
			Scan(&skillRows).Error; err != nil {
			return nil, err
		}
		for _, row := range skillRows {
			if _, ok := skillByUser[row.UserID]; !ok {
				skillByUser[row.UserID] = make(map[string]float64)
			}
			skillByUser[row.UserID][strings.TrimSpace(row.SkillCode)] = row.Score
		}
	}

	var relationRows []teamSplitRelationRow
	if len(realUserIDs) > 0 {
		if err := s.db.WithContext(ctx).
			Table("player_relations").
			Select("user_a_id AS user_a_id, user_b_id AS user_b_id, relation_type, weight").
			Where("group_id = ? AND is_active = TRUE AND user_a_id IN ? AND user_b_id IN ?", group.ID, realUserIDs, realUserIDs).
			Scan(&relationRows).Error; err != nil {
			return nil, err
		}
	}
	relations := make(map[int64][]teamSplitRelEdge, len(realUserIDs))
	for _, rel := range relationRows {
		relType, ok := normalizeRelationType(rel.RelationType)
		if !ok {
			continue
		}
		weight := rel.Weight
		if weight < 1 {
			weight = 1
		}
		if weight > 10 {
			weight = 10
		}
		relations[rel.UserAID] = append(relations[rel.UserAID], teamSplitRelEdge{
			Other:  rel.UserBID,
			Type:   relType,
			Weight: weight,
		})
		relations[rel.UserBID] = append(relations[rel.UserBID], teamSplitRelEdge{
			Other:  rel.UserAID,
			Type:   relType,
			Weight: weight,
		})
	}

	teamCodes := []string{"A", "B"}
	hasTeamC := false
	for _, p := range state.Players {
		if normalizeTeamValue(p.Team) == "C" {
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

	units := buildTeamSplitUnits(state.Players, roleByUser, skillByUser, relations)
	if len(units) == 0 {
		return state, nil
	}

	roleTargets := buildTeamSplitRoleTargets(units, teamCodes)
	skillTargets, powerTargets := buildTeamSplitStatTargets(units, teamCodes, capacity)

	if url := strings.TrimSpace(os.Getenv("TEAM_SPLIT_SERVICE_URL")); url != "" {
		assignments, err := callTeamSplitService(
			ctx,
			url,
			state.Players,
			roleByUser,
			skillByUser,
			relationRows,
			teamCodes,
			capacity,
			roleTargets,
		)
		if err == nil && len(assignments) > 0 {
			assignments = enforceGuestOwnerAssignments(state.Players, assignments)
			assignments = normalizeTeamAssignmentPositions(assignments)
			if validateTeamSplitAssignments(state.Players, assignments, roleByUser, teamCodes, capacity, roleTargets) {
				if err := s.SaveEventTeamSplit(ctx, chatID, eventID, postID, assignments); err != nil {
					return nil, err
				}
				return s.GetEventTeamSplitState(ctx, chatID, eventID, postID)
			}
		}
		// Fall back to the in-process algorithm if the service is unavailable
		// or returned a split that does not satisfy volleyball constraints.
	}

	buckets := make(map[string]*teamSplitBucket, len(teamCodes))
	for _, code := range teamCodes {
		buckets[code] = &teamSplitBucket{
			Code:       code,
			Capacity:   capacity[code],
			Units:      make([]*teamSplitUnit, 0),
			RoleCounts: map[string]int{"setter": 0, "libero": 0, "central": 0},
			SkillTotals: map[string]float64{
				"receive": 0,
				"serve":   0,
				"set":     0,
				"defense": 0,
				"attack":  0,
				"block":   0,
			},
		}
	}

	sort.SliceStable(units, func(i, j int) bool {
		pi := teamSplitRolePriority(units[i].Role)
		pj := teamSplitRolePriority(units[j].Role)
		if pi != pj {
			return pi < pj
		}
		if units[i].RelationWeight != units[j].RelationWeight {
			return units[i].RelationWeight > units[j].RelationWeight
		}
		if units[i].PowerTotal != units[j].PowerTotal {
			return units[i].PowerTotal > units[j].PowerTotal
		}
		if units[i].Size != units[j].Size {
			return units[i].Size > units[j].Size
		}
		return units[i].ID < units[j].ID
	})

	assignedTeamByUser := make(map[int64]string, len(state.Players))
	remainingUnits := make([]*teamSplitUnit, len(units))
	copy(remainingUnits, units)

	// Stage 1: setters first by pass skill ("set"), not by overall rating.
	for _, code := range teamCodes {
		target := 0
		if byTeam, ok := roleTargets["setter"]; ok {
			target = byTeam[code]
		}
		for buckets[code].RoleCounts["setter"] < target {
			bestIdx := -1
			bestSet := -math.MaxFloat64
			for idx, unit := range remainingUnits {
				if unit.Role != "setter" {
					continue
				}
				bucket := buckets[code]
				if bucket.PlayerCount+unit.Size > bucket.Capacity {
					continue
				}
				setScore := teamSplitUnitSkillAvg(unit, "set")
				if bestIdx == -1 || setScore > bestSet {
					bestIdx = idx
					bestSet = setScore
				}
			}
			if bestIdx < 0 {
				break
			}
			chosen := remainingUnits[bestIdx]
			assignUnitToBucket(chosen, buckets[code], assignedTeamByUser)
			remainingUnits = append(remainingUnits[:bestIdx], remainingUnits[bestIdx+1:]...)
		}
	}

	// Stage 2: pull prefer_together links to assigned setters.
	for _, code := range teamCodes {
		bucket := buckets[code]
		setterUsers := teamSplitBucketSetterUsers(bucket)
		if len(setterUsers) == 0 {
			continue
		}
		for bucket.PlayerCount < bucket.Capacity {
			bestIdx := -1
			bestMetric := -math.MaxFloat64
			for idx, unit := range remainingUnits {
				if bucket.PlayerCount+unit.Size > bucket.Capacity {
					continue
				}
				prefer := teamSplitUnitPreferToUsers(unit, setterUsers, relations)
				if prefer <= 0 {
					continue
				}
				basePenalty := teamSplitPlacementScore(
					unit,
					bucket,
					roleTargets,
					skillTargets[code],
					powerTargets[code],
					relations,
					assignedTeamByUser,
				)
				metric := prefer*100.0 - basePenalty
				if bestIdx == -1 || metric > bestMetric {
					bestIdx = idx
					bestMetric = metric
				}
			}
			if bestIdx < 0 {
				break
			}
			linked := remainingUnits[bestIdx]
			assignUnitToBucket(linked, bucket, assignedTeamByUser)
			remainingUnits = append(remainingUnits[:bestIdx], remainingUnits[bestIdx+1:]...)
		}
	}

	// Stage 3: strongest attackers go to teams with weaker setters.
	teamsBySetter := append([]string{}, teamCodes...)
	sort.SliceStable(teamsBySetter, func(i, j int) bool {
		qi := teamSplitBucketSetterQuality(buckets[teamsBySetter[i]])
		qj := teamSplitBucketSetterQuality(buckets[teamsBySetter[j]])
		if qi != qj {
			return qi < qj
		}
		return teamsBySetter[i] < teamsBySetter[j]
	})
	for _, code := range teamsBySetter {
		bestIdx := -1
		bestAttack := -math.MaxFloat64
		for idx, unit := range remainingUnits {
			if unit.Role != "attacker" {
				continue
			}
			bucket := buckets[code]
			if bucket.PlayerCount+unit.Size > bucket.Capacity {
				continue
			}
			attack := teamSplitUnitSkillAvg(unit, "attack")
			if bestIdx == -1 || attack > bestAttack {
				bestIdx = idx
				bestAttack = attack
			}
		}
		if bestIdx < 0 {
			continue
		}
		attacker := remainingUnits[bestIdx]
		assignUnitToBucket(attacker, buckets[code], assignedTeamByUser)
		remainingUnits = append(remainingUnits[:bestIdx], remainingUnits[bestIdx+1:]...)
	}

	// Stage 4: satisfy remaining mandatory role quotas team-by-team.
	for _, role := range []string{"setter", "libero", "central"} {
		for _, code := range teamCodes {
			target := 0
			if byTeam, ok := roleTargets[role]; ok {
				target = byTeam[code]
			}
			for buckets[code].RoleCounts[role] < target {
				bestIdx := -1
				bestScore := math.MaxFloat64
				for idx, unit := range remainingUnits {
					if unit.Role != role {
						continue
					}
					bucket := buckets[code]
					if bucket.PlayerCount+unit.Size > bucket.Capacity {
						continue
					}
					score := teamSplitPlacementScore(
						unit,
						bucket,
						roleTargets,
						skillTargets[code],
						powerTargets[code],
						relations,
						assignedTeamByUser,
					)
					if bestIdx == -1 || score < bestScore {
						bestIdx = idx
						bestScore = score
					}
				}
				if bestIdx < 0 {
					break
				}
				chosen := remainingUnits[bestIdx]
				assignUnitToBucket(chosen, buckets[code], assignedTeamByUser)
				remainingUnits = append(remainingUnits[:bestIdx], remainingUnits[bestIdx+1:]...)
			}
		}
	}

	// Stage 5: weak-first balancing for the rest.
	sort.SliceStable(remainingUnits, func(i, j int) bool {
		if remainingUnits[i].PowerTotal != remainingUnits[j].PowerTotal {
			return remainingUnits[i].PowerTotal < remainingUnits[j].PowerTotal
		}
		return remainingUnits[i].ID < remainingUnits[j].ID
	})

	for _, unit := range remainingUnits {
		chosen := chooseTeamBucketForUnit(
			unit,
			teamCodes,
			buckets,
			roleTargets,
			skillTargets,
			powerTargets,
			relations,
			assignedTeamByUser,
		)
		assignUnitToBucket(unit, chosen, assignedTeamByUser)
	}

	assignments := make([]TeamSplitAssignmentInput, 0, len(state.Players))
	for _, code := range teamCodes {
		bucket := buckets[code]
		players := make([]TeamSplitPlayer, 0, bucket.PlayerCount)
		for _, unit := range bucket.Units {
			players = append(players, unit.Players...)
		}
		ordered := orderTeamPlayersForLineup(players, roleByUser, skillByUser)
		for pos, p := range ordered {
			assignments = append(assignments, TeamSplitAssignmentInput{
				UserID:   p.UserID,
				Team:     code,
				Position: pos,
			})
		}
	}

	assignments = enforceGuestOwnerAssignments(state.Players, assignments)
	assignments = normalizeTeamAssignmentPositions(assignments)

	if err := s.SaveEventTeamSplit(ctx, chatID, eventID, postID, assignments); err != nil {
		return nil, err
	}
	return s.GetEventTeamSplitState(ctx, chatID, eventID, postID)
}

type teamSplitRelEdge struct {
	Other  int64
	Type   string
	Weight int
}

type teamSplitUnit struct {
	ID             int64
	Role           string
	Players        []TeamSplitPlayer
	Size           int
	SkillTotals    map[string]float64
	PowerTotal     float64
	RelationWeight int
}

type teamSplitBucket struct {
	Code        string
	Capacity    int
	Units       []*teamSplitUnit
	PlayerCount int
	RoleCounts  map[string]int
	SkillTotals map[string]float64
	PowerTotal  float64
}

func buildTeamSplitUnits(
	players []TeamSplitPlayer,
	roleByUser map[int64]string,
	skillByUser map[int64]map[string]float64,
	relations map[int64][]teamSplitRelEdge,
) []*teamSplitUnit {
	owners := make([]TeamSplitPlayer, 0, len(players))
	guestsByOwner := make(map[int64][]TeamSplitPlayer)
	standaloneGuests := make([]TeamSplitPlayer, 0)

	for _, p := range players {
		if p.GuestOwnerID > 0 {
			guestsByOwner[p.GuestOwnerID] = append(guestsByOwner[p.GuestOwnerID], p)
			continue
		}
		if p.UserID < 0 {
			standaloneGuests = append(standaloneGuests, p)
			continue
		}
		owners = append(owners, p)
	}

	sort.SliceStable(owners, func(i, j int) bool { return owners[i].UserID < owners[j].UserID })
	sort.SliceStable(standaloneGuests, func(i, j int) bool { return standaloneGuests[i].UserID < standaloneGuests[j].UserID })

	units := make([]*teamSplitUnit, 0, len(owners)+len(standaloneGuests))
	for _, owner := range owners {
		unitPlayers := make([]TeamSplitPlayer, 0, 1+len(guestsByOwner[owner.UserID]))
		unitPlayers = append(unitPlayers, owner)
		if guests := guestsByOwner[owner.UserID]; len(guests) > 0 {
			sort.SliceStable(guests, func(i, j int) bool { return guests[i].UserID < guests[j].UserID })
			unitPlayers = append(unitPlayers, guests...)
		}
		units = append(units, newTeamSplitUnit(owner.UserID, unitPlayers, roleByUser, skillByUser, relations))
		delete(guestsByOwner, owner.UserID)
	}

	// Defensive fallback for inconsistent data where a guest exists without owner in the roster.
	for ownerID, guests := range guestsByOwner {
		for _, guest := range guests {
			unitID := guest.UserID
			if ownerID > 0 {
				unitID = ownerID
			}
			units = append(units, newTeamSplitUnit(unitID, []TeamSplitPlayer{guest}, roleByUser, skillByUser, relations))
		}
	}
	for _, guest := range standaloneGuests {
		units = append(units, newTeamSplitUnit(guest.UserID, []TeamSplitPlayer{guest}, roleByUser, skillByUser, relations))
	}

	return units
}

func newTeamSplitUnit(
	id int64,
	players []TeamSplitPlayer,
	roleByUser map[int64]string,
	skillByUser map[int64]map[string]float64,
	relations map[int64][]teamSplitRelEdge,
) *teamSplitUnit {
	role := teamSplitPlayerRole(players[0], roleByUser)
	skillTotals := map[string]float64{
		"receive": 0,
		"serve":   0,
		"set":     0,
		"defense": 0,
		"attack":  0,
		"block":   0,
	}
	powerTotal := 0.0
	relationWeight := 0
	for _, p := range players {
		pRole := teamSplitPlayerRole(p, roleByUser)
		skills := teamSplitPlayerSkills(p, skillByUser)
		for code, value := range skills {
			skillTotals[code] += value
		}
		powerTotal += teamSplitPlayerPower(pRole, skills)
		for _, edge := range relations[p.UserID] {
			relationWeight += edge.Weight
		}
	}
	return &teamSplitUnit{
		ID:             id,
		Role:           role,
		Players:        players,
		Size:           len(players),
		SkillTotals:    skillTotals,
		PowerTotal:     powerTotal,
		RelationWeight: relationWeight,
	}
}

func buildTeamSplitRoleTargets(units []*teamSplitUnit, teamCodes []string) map[string]map[string]int {
	countByRole := map[string]int{
		"setter":  0,
		"libero":  0,
		"central": 0,
	}
	for _, unit := range units {
		if _, ok := countByRole[unit.Role]; ok {
			countByRole[unit.Role]++
		}
	}

	targets := map[string]map[string]int{
		"setter":  distributeTeamRoleTargets(minInt(countByRole["setter"], len(teamCodes)), 1, teamCodes),
		"libero":  distributeTeamRoleTargets(minInt(countByRole["libero"], len(teamCodes)), 1, teamCodes),
		"central": distributeTeamRoleTargets(minInt(countByRole["central"], 2*len(teamCodes)), 2, teamCodes),
	}
	return targets
}

func distributeTeamRoleTargets(total int, maxPerTeam int, teamCodes []string) map[string]int {
	out := make(map[string]int, len(teamCodes))
	for _, code := range teamCodes {
		out[code] = 0
	}
	if total <= 0 || len(teamCodes) == 0 || maxPerTeam <= 0 {
		return out
	}

	remaining := total
	for remaining > 0 {
		progressed := false
		for _, code := range teamCodes {
			if remaining == 0 {
				break
			}
			if out[code] >= maxPerTeam {
				continue
			}
			out[code]++
			remaining--
			progressed = true
		}
		if !progressed {
			break
		}
	}
	return out
}

func buildTeamSplitStatTargets(
	units []*teamSplitUnit,
	teamCodes []string,
	capacity map[string]int,
) (map[string]map[string]float64, map[string]float64) {
	totalPlayers := 0
	totalPower := 0.0
	totalSkills := map[string]float64{
		"receive": 0,
		"serve":   0,
		"set":     0,
		"defense": 0,
		"attack":  0,
		"block":   0,
	}
	for _, unit := range units {
		totalPlayers += unit.Size
		totalPower += unit.PowerTotal
		for code, value := range unit.SkillTotals {
			totalSkills[code] += value
		}
	}
	if totalPlayers <= 0 {
		return map[string]map[string]float64{}, map[string]float64{}
	}

	skillTargets := make(map[string]map[string]float64, len(teamCodes))
	powerTargets := make(map[string]float64, len(teamCodes))
	for _, code := range teamCodes {
		ratio := float64(capacity[code]) / float64(totalPlayers)
		skillTargets[code] = map[string]float64{
			"receive": totalSkills["receive"] * ratio,
			"serve":   totalSkills["serve"] * ratio,
			"set":     totalSkills["set"] * ratio,
			"defense": totalSkills["defense"] * ratio,
			"attack":  totalSkills["attack"] * ratio,
			"block":   totalSkills["block"] * ratio,
		}
		powerTargets[code] = totalPower * ratio
	}
	return skillTargets, powerTargets
}

func chooseTeamBucketForUnit(
	unit *teamSplitUnit,
	teamCodes []string,
	buckets map[string]*teamSplitBucket,
	roleTargets map[string]map[string]int,
	skillTargets map[string]map[string]float64,
	powerTargets map[string]float64,
	relations map[int64][]teamSplitRelEdge,
	assignedTeamByUser map[int64]string,
) *teamSplitBucket {
	var chosen *teamSplitBucket
	best := math.MaxFloat64

	for _, code := range teamCodes {
		bucket := buckets[code]
		score := teamSplitPlacementScore(
			unit,
			bucket,
			roleTargets,
			skillTargets[code],
			powerTargets[code],
			relations,
			assignedTeamByUser,
		)
		if chosen == nil || score < best || (score == best && bucket.PlayerCount < chosen.PlayerCount) {
			chosen = bucket
			best = score
		}
	}
	if chosen == nil {
		return buckets[teamCodes[0]]
	}
	return chosen
}

func teamSplitPlacementScore(
	unit *teamSplitUnit,
	bucket *teamSplitBucket,
	roleTargets map[string]map[string]int,
	skillTarget map[string]float64,
	powerTarget float64,
	relations map[int64][]teamSplitRelEdge,
	assignedTeamByUser map[int64]string,
) float64 {
	penalty := 0.0
	nextPlayers := bucket.PlayerCount + unit.Size
	if overflow := nextPlayers - bucket.Capacity; overflow > 0 {
		penalty += float64(overflow) * 60.0
	}
	penalty += math.Abs(float64(nextPlayers-bucket.Capacity)) * 0.9

	roleOrder := []string{"setter", "libero", "central"}
	for _, role := range roleOrder {
		target := 0
		if byTeam, ok := roleTargets[role]; ok {
			target = byTeam[bucket.Code]
		}
		current := bucket.RoleCounts[role]
		add := 0
		if unit.Role == role {
			add = 1
		}
		next := current + add
		if next < target {
			penalty += float64(target-next) * 52.0
		}
		if next > target {
			excessWeight := 6.0
			if role == "central" {
				excessWeight = 4.5
			}
			penalty += float64(next-target) * excessWeight
		}
	}

	nextPower := bucket.PowerTotal + unit.PowerTotal
	penalty += math.Abs(nextPower-powerTarget) * 0.42

	// Keep strongest centrals opposite strongest attackers.
	if unit.Role == "central" {
		centralStrength := unit.SkillTotals["block"]*1.1 + unit.SkillTotals["attack"]*0.35
		penalty += centralStrength * teamSplitBucketAttackerLoad(bucket) * 0.08
	} else if unit.Role == "attacker" {
		attackerStrength := unit.SkillTotals["attack"]
		penalty += attackerStrength * teamSplitBucketCentralLoad(bucket) * 0.08
	}

	skillWeight := map[string]float64{
		"attack":  0.45,
		"block":   0.48,
		"set":     0.42,
		"receive": 0.37,
		"defense": 0.33,
		"serve":   0.28,
	}
	for code, target := range skillTarget {
		nextSkill := bucket.SkillTotals[code] + unit.SkillTotals[code]
		penalty += math.Abs(nextSkill-target) * skillWeight[code]
	}

	for _, p := range unit.Players {
		for _, edge := range relations[p.UserID] {
			otherTeam, ok := assignedTeamByUser[edge.Other]
			if !ok {
				continue
			}
			w := float64(edge.Weight)
			switch edge.Type {
			case "prefer_together":
				if otherTeam != bucket.Code {
					penalty += w * 9.0
				} else {
					penalty -= w * 1.4
				}
			case "avoid_together":
				if otherTeam == bucket.Code {
					penalty += w * 11.0
				} else {
					penalty -= w * 0.45
				}
			}
		}
	}

	return penalty
}

func teamSplitUnitSkillAvg(unit *teamSplitUnit, code string) float64 {
	if unit == nil || unit.Size <= 0 {
		return 0
	}
	return unit.SkillTotals[code] / float64(unit.Size)
}

func teamSplitBucketSetterQuality(bucket *teamSplitBucket) float64 {
	if bucket == nil || len(bucket.Units) == 0 {
		return 0
	}
	total := 0.0
	count := 0
	for _, unit := range bucket.Units {
		if unit.Role != "setter" {
			continue
		}
		total += teamSplitUnitSkillAvg(unit, "set")
		count++
	}
	if count == 0 {
		return 0
	}
	return total / float64(count)
}

func teamSplitBucketAttackerLoad(bucket *teamSplitBucket) float64 {
	if bucket == nil || len(bucket.Units) == 0 {
		return 0
	}
	total := 0.0
	for _, unit := range bucket.Units {
		if unit.Role == "attacker" {
			total += teamSplitUnitSkillAvg(unit, "attack")
		}
	}
	return total
}

func teamSplitBucketCentralLoad(bucket *teamSplitBucket) float64 {
	if bucket == nil || len(bucket.Units) == 0 {
		return 0
	}
	total := 0.0
	for _, unit := range bucket.Units {
		if unit.Role != "central" {
			continue
		}
		blockAvg := teamSplitUnitSkillAvg(unit, "block")
		attackAvg := teamSplitUnitSkillAvg(unit, "attack")
		total += blockAvg*1.1 + attackAvg*0.25
	}
	return total
}

func teamSplitBucketSetterUsers(bucket *teamSplitBucket) map[int64]struct{} {
	out := make(map[int64]struct{})
	if bucket == nil {
		return out
	}
	for _, unit := range bucket.Units {
		if unit.Role != "setter" {
			continue
		}
		for _, p := range unit.Players {
			out[p.UserID] = struct{}{}
		}
	}
	return out
}

func teamSplitUnitPreferToUsers(unit *teamSplitUnit, users map[int64]struct{}, relations map[int64][]teamSplitRelEdge) float64 {
	if unit == nil || len(users) == 0 {
		return 0
	}
	score := 0.0
	for _, p := range unit.Players {
		for _, edge := range relations[p.UserID] {
			if edge.Type != "prefer_together" {
				continue
			}
			if _, ok := users[edge.Other]; ok {
				score += float64(edge.Weight)
			}
		}
	}
	return score
}

func assignUnitToBucket(unit *teamSplitUnit, bucket *teamSplitBucket, assignedTeamByUser map[int64]string) {
	bucket.Units = append(bucket.Units, unit)
	bucket.PlayerCount += unit.Size
	bucket.PowerTotal += unit.PowerTotal
	if unit.Role != "" {
		bucket.RoleCounts[unit.Role]++
	}
	for code, value := range unit.SkillTotals {
		bucket.SkillTotals[code] += value
	}
	for _, p := range unit.Players {
		assignedTeamByUser[p.UserID] = bucket.Code
	}
}

const (
	teamSplitFormationSetterTarget  = 7.0
	teamSplitFormationReceiveTarget = 6.0
	teamSplitFormationCentralTarget = 2
)

func orderTeamPlayersForLineup(
	players []TeamSplitPlayer,
	roleByUser map[int64]string,
	skillByUser map[int64]map[string]float64,
) []TeamSplitPlayer {
	if len(players) <= 1 {
		out := make([]TeamSplitPlayer, len(players))
		copy(out, players)
		return out
	}

	available := make([]TeamSplitPlayer, len(players))
	copy(available, players)
	ordered := make([]TeamSplitPlayer, 0, len(players))
	slotOrder := []int{2, 3, 4, 5, 1, 6}
	scheme := teamSplitRecommendedSchemeForLineup(players, roleByUser, skillByUser)
	if scheme == "5/1" {
		// Fill scarce roles first for 5/1:
		// 2-setter, 1-central, 4-central, then attacking trio.
		slotOrder = []int{2, 1, 4, 5, 3, 6}
	}

	for _, slot := range slotOrder {
		if len(available) == 0 || len(ordered) >= 6 {
			break
		}
		bestIdx := -1
		bestScore := -math.MaxFloat64
		for idx, p := range available {
			score := teamSplitLineupSlotScore(slot, scheme, p, roleByUser, skillByUser)
			if bestIdx == -1 || score > bestScore {
				bestIdx = idx
				bestScore = score
			}
		}
		if bestIdx < 0 {
			break
		}
		ordered = append(ordered, available[bestIdx])
		available = append(available[:bestIdx], available[bestIdx+1:]...)
	}

	sort.SliceStable(available, func(i, j int) bool {
		si := teamSplitBenchScore(available[i], roleByUser, skillByUser)
		sj := teamSplitBenchScore(available[j], roleByUser, skillByUser)
		if si != sj {
			return si > sj
		}
		return available[i].UserID < available[j].UserID
	})

	return append(ordered, available...)
}

func teamSplitBenchScore(player TeamSplitPlayer, roleByUser map[int64]string, skillByUser map[int64]map[string]float64) float64 {
	role := teamSplitPlayerRole(player, roleByUser)
	skills := teamSplitPlayerSkills(player, skillByUser)
	return teamSplitPlayerPower(role, skills) + player.Rating*0.85
}

func teamSplitRecommendedSchemeForLineup(
	players []TeamSplitPlayer,
	roleByUser map[int64]string,
	skillByUser map[int64]map[string]float64,
) string {
	if len(players) == 0 {
		return "4/2"
	}

	setters := make([]float64, 0, 2)
	receiveTotal := 0.0
	centralCount := 0
	for _, player := range players {
		role := teamSplitPlayerRole(player, roleByUser)
		skills := teamSplitPlayerSkills(player, skillByUser)
		receiveTotal += skills["receive"]
		if role == "setter" {
			setters = append(setters, skills["set"])
		}
		if role == "central" {
			centralCount++
		}
	}

	sort.SliceStable(setters, func(i, j int) bool { return setters[i] > setters[j] })
	bestSetter := 0.0
	if len(setters) > 0 {
		bestSetter = setters[0]
	}

	avgReceive := receiveTotal / float64(len(players))
	useFiveOne := len(setters) > 0 &&
		bestSetter >= teamSplitFormationSetterTarget &&
		centralCount >= teamSplitFormationCentralTarget &&
		avgReceive >= teamSplitFormationReceiveTarget
	if useFiveOne {
		return "5/1"
	}
	return "4/2"
}

func teamSplitLineupSlotScore(
	slot int,
	scheme string,
	player TeamSplitPlayer,
	roleByUser map[int64]string,
	skillByUser map[int64]map[string]float64,
) float64 {
	role := teamSplitPlayerRole(player, roleByUser)
	skills := teamSplitPlayerSkills(player, skillByUser)

	base := teamSplitPlayerPower(role, skills) + player.Rating*0.55
	attack := skills["attack"]
	block := skills["block"]
	setScore := skills["set"]
	receive := skills["receive"]
	defense := skills["defense"]
	serve := skills["serve"]

	roleBonus := func(target string, bonus float64) float64 {
		if role == target {
			return bonus
		}
		return 0
	}

	if scheme == "5/1" {
		switch slot {
		case 1:
			return base + block*2.4 + attack*1.0 + serve*0.4 + roleBonus("central", 6.2)
		case 2:
			return base + setScore*2.9 + serve*0.6 + defense*0.7 + roleBonus("setter", 7.1)
		case 3:
			return base + attack*1.9 + receive*1.1 + defense*0.6 + serve*0.5 + roleBonus("attacker", 4.2)
		case 4:
			return base + block*2.3 + attack*1.0 + serve*0.3 + roleBonus("central", 5.8)
		case 5:
			return base + attack*2.5 + serve*1.0 + block*0.5 + roleBonus("attacker", 5.0)
		case 6:
			return base + receive*1.9 + defense*1.5 + attack*1.0 + serve*0.5 + roleBonus("attacker", 3.7) + roleBonus("libero", 2.8)
		default:
			return base
		}
	}

	switch slot {
	case 1:
		return base + serve*1.5 + receive*1.2 + defense*0.8 + roleBonus("attacker", 1.1)
	case 2:
		return base + setScore*2.3 + serve*0.6 + defense*0.7 + roleBonus("setter", 4.5)
	case 3:
		return base + block*2.2 + attack*1.1 + serve*0.4 + roleBonus("central", 3.7)
	case 4:
		return base + attack*2.0 + serve*1.0 + receive*0.5 + roleBonus("attacker", 3.1)
	case 5:
		return base + receive*2.1 + defense*2.0 + setScore*0.4 + roleBonus("libero", 4.3)
	case 6:
		return base + block*1.2 + defense*1.0 + receive*0.9 + attack*0.8 + roleBonus("central", 1.7)
	default:
		return base
	}
}

func teamSplitPlayerRole(player TeamSplitPlayer, roleByUser map[int64]string) string {
	if player.UserID > 0 {
		if normalized, ok := normalizePlayerType(roleByUser[player.UserID]); ok {
			return normalized
		}
	}
	if player.GuestOwnerID > 0 {
		if normalized, ok := normalizePlayerType(roleByUser[player.GuestOwnerID]); ok {
			return normalized
		}
	}
	return ""
}

func teamSplitPlayerSkills(player TeamSplitPlayer, skillByUser map[int64]map[string]float64) map[string]float64 {
	out := map[string]float64{
		"receive": 5.0,
		"serve":   5.0,
		"set":     5.0,
		"defense": 5.0,
		"attack":  5.0,
		"block":   5.0,
	}
	if skills, ok := skillByUser[player.UserID]; ok {
		for code := range out {
			if value, ok := skills[code]; ok && value > 0 {
				out[code] = value
			}
		}
		return out
	}
	// For synthetic guests try to inherit owner's profile if available. Fallback is neutral defaults.
	if player.UserID < 0 && player.GuestOwnerID > 0 {
		if skills, ok := skillByUser[player.GuestOwnerID]; ok {
			for code := range out {
				if value, ok := skills[code]; ok && value > 0 {
					out[code] = value
				}
			}
		}
	}
	return out
}

func teamSplitPlayerPower(role string, skills map[string]float64) float64 {
	receive := skills["receive"]
	serve := skills["serve"]
	setScore := skills["set"]
	defense := skills["defense"]
	attack := skills["attack"]
	block := skills["block"]

	power := attack*1.15 + block*1.12 + setScore*1.02 + receive*0.95 + defense*0.88 + serve*0.78
	switch role {
	case "setter":
		power += setScore*0.48 + serve*0.12
	case "libero":
		power += receive*0.42 + defense*0.38
	case "central":
		power += block*0.56 + attack*0.18
	case "attacker":
		power += attack*0.34 + serve*0.11
	}
	return power
}

func teamSplitRolePriority(role string) int {
	switch role {
	case "setter":
		return 0
	case "central":
		return 1
	case "libero":
		return 2
	case "attacker":
		return 3
	default:
		return 4
	}
}

func enforceGuestOwnerAssignments(players []TeamSplitPlayer, assignments []TeamSplitAssignmentInput) []TeamSplitAssignmentInput {
	ownerByGuest := make(map[int64]int64)
	for _, p := range players {
		if p.UserID < 0 && p.GuestOwnerID > 0 {
			ownerByGuest[p.UserID] = p.GuestOwnerID
		}
	}
	if len(ownerByGuest) == 0 {
		return assignments
	}

	byUser := make(map[int64]TeamSplitAssignmentInput, len(assignments))
	for _, item := range assignments {
		byUser[item.UserID] = item
	}
	for guestID, ownerID := range ownerByGuest {
		owner, ok := byUser[ownerID]
		if !ok {
			continue
		}
		guest, ok := byUser[guestID]
		if !ok {
			guest = TeamSplitAssignmentInput{
				UserID: guestID,
				Team:   owner.Team,
			}
		}
		guest.Team = owner.Team
		if guest.Position < owner.Position {
			guest.Position = owner.Position + 1
		}
		byUser[guestID] = guest
	}

	out := make([]TeamSplitAssignmentInput, 0, len(byUser))
	for _, item := range byUser {
		out = append(out, item)
	}
	return out
}

func normalizeTeamAssignmentPositions(assignments []TeamSplitAssignmentInput) []TeamSplitAssignmentInput {
	byTeam := map[string][]TeamSplitAssignmentInput{
		"A":          {},
		"B":          {},
		"C":          {},
		"unassigned": {},
	}
	for _, item := range assignments {
		team := normalizeTeamValue(item.Team)
		byTeam[team] = append(byTeam[team], TeamSplitAssignmentInput{
			UserID:   item.UserID,
			Team:     team,
			Position: item.Position,
		})
	}

	out := make([]TeamSplitAssignmentInput, 0, len(assignments))
	for _, code := range []string{"A", "B", "C", "unassigned"} {
		list := byTeam[code]
		sort.SliceStable(list, func(i, j int) bool {
			if list[i].Position != list[j].Position {
				return list[i].Position < list[j].Position
			}
			return list[i].UserID < list[j].UserID
		})
		for idx, item := range list {
			item.Position = idx
			out = append(out, item)
		}
	}
	return out
}

func validateTeamSplitAssignments(
	players []TeamSplitPlayer,
	assignments []TeamSplitAssignmentInput,
	roleByUser map[int64]string,
	teamCodes []string,
	capacity map[string]int,
	roleTargets map[string]map[string]int,
) bool {
	if len(players) == 0 || len(assignments) == 0 {
		return false
	}

	allowedTeams := make(map[string]struct{}, len(teamCodes))
	for _, code := range teamCodes {
		allowedTeams[normalizeTeamValue(code)] = struct{}{}
	}

	byUser := make(map[int64]TeamSplitAssignmentInput, len(assignments))
	for _, item := range assignments {
		if item.UserID == 0 {
			continue
		}
		normalized := item
		normalized.Team = normalizeTeamValue(item.Team)
		byUser[item.UserID] = normalized
	}

	teamCounts := make(map[string]int, len(teamCodes))
	roleCounts := make(map[string]map[string]int, len(teamCodes))
	for _, code := range teamCodes {
		teamCounts[code] = 0
		roleCounts[code] = map[string]int{"setter": 0, "libero": 0, "central": 0}
	}

	for _, p := range players {
		item, ok := byUser[p.UserID]
		if !ok {
			return false
		}
		if _, ok := allowedTeams[item.Team]; !ok {
			return false
		}
		teamCounts[item.Team]++
		if p.GuestOwnerID > 0 {
			if owner, ok := byUser[p.GuestOwnerID]; ok {
				if normalizeTeamValue(owner.Team) != item.Team {
					return false
				}
			}
		}
		role := teamSplitPlayerRole(p, roleByUser)
		if _, ok := roleCounts[item.Team][role]; ok {
			roleCounts[item.Team][role]++
		}
	}

	for _, code := range teamCodes {
		if teamCounts[code] > capacity[code] {
			return false
		}
		for _, role := range []string{"setter", "libero", "central"} {
			target := 0
			if byTeam, ok := roleTargets[role]; ok {
				target = byTeam[code]
			}
			if roleCounts[code][role] < target {
				return false
			}
		}
	}
	return true
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type teamSplitServiceReq struct {
	Players []struct {
		UserID       int64              `json:"userID"`
		GuestOwnerID int64              `json:"guestOwnerID,omitempty"`
		Rating       float64            `json:"rating"`
		PlayerType   string             `json:"playerType"`
		Skills       map[string]float64 `json:"skills,omitempty"`
	} `json:"players"`
	Relations []struct {
		UserAID      int64  `json:"userAID"`
		UserBID      int64  `json:"userBID"`
		RelationType string `json:"relationType"`
		Weight       int    `json:"weight"`
	} `json:"relations"`
	TeamCodes   []string                  `json:"teamCodes,omitempty"`
	Capacity    map[string]int            `json:"capacity,omitempty"`
	RoleTargets map[string]map[string]int `json:"roleTargets,omitempty"`
	LineupSize  int                       `json:"lineupSize,omitempty"`
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
	skillByUser map[int64]map[string]float64,
	relationRows []teamSplitRelationRow,
	teamCodes []string,
	capacity map[string]int,
	roleTargets map[string]map[string]int,
) ([]TeamSplitAssignmentInput, error) {
	reqBody := teamSplitServiceReq{
		TeamCodes:   teamCodes,
		Capacity:    capacity,
		RoleTargets: roleTargets,
		LineupSize:  6,
	}
	reqBody.Players = make([]struct {
		UserID       int64              `json:"userID"`
		GuestOwnerID int64              `json:"guestOwnerID,omitempty"`
		Rating       float64            `json:"rating"`
		PlayerType   string             `json:"playerType"`
		Skills       map[string]float64 `json:"skills,omitempty"`
	}, 0, len(players))
	for _, p := range players {
		role := teamSplitPlayerRole(p, roleByUser)
		skills := teamSplitPlayerSkills(p, skillByUser)
		reqBody.Players = append(reqBody.Players, struct {
			UserID       int64              `json:"userID"`
			GuestOwnerID int64              `json:"guestOwnerID,omitempty"`
			Rating       float64            `json:"rating"`
			PlayerType   string             `json:"playerType"`
			Skills       map[string]float64 `json:"skills,omitempty"`
		}{
			UserID:       p.UserID,
			GuestOwnerID: p.GuestOwnerID,
			Rating:       p.Rating,
			PlayerType:   role,
			Skills:       skills,
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
