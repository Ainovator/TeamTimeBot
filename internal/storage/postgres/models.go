package postgres

import (
	"time"

	"gorm.io/datatypes"
)

type TelegramGroup struct {
	ID        uint64    `gorm:"primaryKey"`
	ChatID    int64     `gorm:"uniqueIndex;not null"`
	Title     string    `gorm:"not null"`
	Timezone  string    `gorm:"not null;default:UTC"`
	IsActive  bool      `gorm:"not null;default:true"`
	CreatedAt time.Time `gorm:"not null;default:now()"`
	UpdatedAt time.Time `gorm:"not null;default:now()"`
}

func (TelegramGroup) TableName() string {
	return "telegram_groups"
}

type TelegramUser struct {
	ID          uint64 `gorm:"primaryKey"`
	TelegramID  int64  `gorm:"uniqueIndex;not null"`
	Username    string
	FirstName   string
	LastName    string
	Language    string
	IsBot       bool      `gorm:"not null;default:false"`
	FirstSeenAt time.Time `gorm:"not null;default:now()"`
	LastSeenAt  time.Time `gorm:"not null;default:now()"`
	IsActive    bool      `gorm:"not null;default:true"`
	CreatedAt   time.Time `gorm:"not null;default:now()"`
	UpdatedAt   time.Time `gorm:"not null;default:now()"`
}

func (TelegramUser) TableName() string {
	return "telegram_users"
}

type GroupMember struct {
	ID             uint64    `gorm:"primaryKey"`
	GroupID        uint64    `gorm:"not null;index;uniqueIndex:ux_group_member"`
	UserTelegramID int64     `gorm:"not null;uniqueIndex:ux_group_member"`
	PlayerType     string    `gorm:"not null;default:''"`
	Role           string    `gorm:"not null;default:member"`
	Status         string    `gorm:"not null;default:active"`
	JoinedAt       time.Time `gorm:"not null;default:now()"`
	LastSeenAt     time.Time `gorm:"not null;default:now()"`
	IsActive       bool      `gorm:"not null;default:true"`
	CreatedAt      time.Time `gorm:"not null;default:now()"`
	UpdatedAt      time.Time `gorm:"not null;default:now()"`
}

func (GroupMember) TableName() string {
	return "group_members"
}

type PollTemplate struct {
	ID             uint64         `gorm:"primaryKey"`
	GroupID        uint64         `gorm:"not null;index;uniqueIndex:ux_group_template_name"`
	Name           string         `gorm:"not null;uniqueIndex:ux_group_template_name"`
	Question       string         `gorm:"not null"`
	Options        datatypes.JSON `gorm:"type:jsonb;not null"`
	CountedOptions datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'"`
	IsActive       bool           `gorm:"not null;default:true"`
	CreatedAt      time.Time      `gorm:"not null;default:now()"`
	UpdatedAt      time.Time      `gorm:"not null;default:now()"`
}

func (PollTemplate) TableName() string {
	return "poll_templates"
}

type PollSchedule struct {
	ID         uint64 `gorm:"primaryKey"`
	GroupID    uint64 `gorm:"not null;index"`
	TemplateID uint64 `gorm:"not null;index"`
	CronExpr   string `gorm:"not null"`
	NextRunAt  *time.Time
	IsActive   bool      `gorm:"not null;default:true"`
	CreatedAt  time.Time `gorm:"not null;default:now()"`
	UpdatedAt  time.Time `gorm:"not null;default:now()"`
}

func (PollSchedule) TableName() string {
	return "poll_schedules"
}

type GroupEvent struct {
	ID                      uint64  `gorm:"primaryKey"`
	GroupID                 uint64  `gorm:"not null;index"`
	PollTemplateID          *uint64 `gorm:"index"`
	Name                    string  `gorm:"not null"`
	EventType               string  `gorm:"not null;default:training"`
	StartWeekday            int16   `gorm:"not null"`
	PollPublishWeekday      int16   `gorm:"not null"`
	PollPublishTime         string
	StartTime               string `gorm:"not null"`
	EndTime                 string `gorm:"not null"`
	AnnouncementText        string
	AnnouncementEnabled     bool  `gorm:"not null;default:false"`
	AnnouncementLeadMinutes int16 `gorm:"not null;default:60"`
	PublishEnabled          bool  `gorm:"not null;default:true"`
	TeamsAutoSplit          bool  `gorm:"not null;default:false"`
	TeamsPublishList        bool  `gorm:"not null;default:false"`
	TeamSize                int16 `gorm:"not null;default:6"`
	MinVotesToHold          int32 `gorm:"not null;default:0"`
	SettlementEnabled       bool  `gorm:"not null;default:true"`
	SettlementPublishBefore bool  `gorm:"not null;default:false"`
	SettlementPublishAfter  bool  `gorm:"not null;default:true"`
	CostAmount              *float64
	IsActive                bool      `gorm:"not null;default:true"`
	CreatedAt               time.Time `gorm:"not null;default:now()"`
	UpdatedAt               time.Time `gorm:"not null;default:now()"`
}

func (GroupEvent) TableName() string {
	return "group_events"
}

type EventPollPost struct {
	ID                uint64    `gorm:"primaryKey"`
	GroupID           uint64    `gorm:"not null;index"`
	EventID           *uint64   `gorm:"index"`
	TemplateID        uint64    `gorm:"not null;index"`
	TelegramMessageID int64     `gorm:"not null"`
	TelegramPollID    string    `gorm:"index"`
	Status            string    `gorm:"not null;default:open"`
	PublishedAt       time.Time `gorm:"not null;default:now()"`
	CreatedAt         time.Time `gorm:"not null;default:now()"`
	UpdatedAt         time.Time `gorm:"not null;default:now()"`
}

func (EventPollPost) TableName() string {
	return "event_poll_posts"
}

type EventPollVote struct {
	ID        uint64 `gorm:"primaryKey"`
	PostID    uint64 `gorm:"not null;index;uniqueIndex:ux_post_user_vote"`
	UserID    int64  `gorm:"not null;uniqueIndex:ux_post_user_vote"`
	Username  string
	FirstName string
	LastName  string
	Choice    string    `gorm:"not null"`
	Source    string    `gorm:"not null;default:inline"`
	VotedAt   time.Time `gorm:"not null;default:now()"`
	CreatedAt time.Time `gorm:"not null;default:now()"`
	UpdatedAt time.Time `gorm:"not null;default:now()"`
}

func (EventPollVote) TableName() string {
	return "event_poll_votes"
}

type EventTeamSession struct {
	ID        uint64    `gorm:"primaryKey"`
	GroupID   uint64    `gorm:"not null;index"`
	EventID   uint64    `gorm:"not null;index"`
	PostID    uint64    `gorm:"not null;uniqueIndex"`
	CreatedAt time.Time `gorm:"not null;default:now()"`
	UpdatedAt time.Time `gorm:"not null;default:now()"`
}

func (EventTeamSession) TableName() string {
	return "event_team_sessions"
}

type EventTeamAssignment struct {
	ID        uint64    `gorm:"primaryKey"`
	SessionID uint64    `gorm:"not null;index;uniqueIndex:ux_session_user"`
	UserID    int64     `gorm:"not null;uniqueIndex:ux_session_user"`
	Team      string    `gorm:"not null"`
	Position  int32     `gorm:"not null;default:0"`
	CreatedAt time.Time `gorm:"not null;default:now()"`
	UpdatedAt time.Time `gorm:"not null;default:now()"`
}

func (EventTeamAssignment) TableName() string {
	return "event_team_assignments"
}
