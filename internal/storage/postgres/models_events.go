package postgres

import (
	"time"

	"gorm.io/datatypes"
)

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
	CancelLeadMinutes       int32 `gorm:"not null;default:180"`
	CancelNotifyEnabled     bool  `gorm:"not null;default:false"`
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

type EventInstance struct {
	ID             uint64    `gorm:"primaryKey"`
	GroupID        uint64    `gorm:"not null;index"`
	EventID        uint64    `gorm:"not null;index"`
	PollPostID     *uint64   `gorm:"index"`
	LocalDate      time.Time `gorm:"not null"`
	PlannedStartAt time.Time `gorm:"not null"`
	PlannedEndAt   time.Time `gorm:"not null"`
	Status         string    `gorm:"not null;default:in_voting"`
	EventName      string    `gorm:"not null;default:''"`
	EventType      string    `gorm:"not null;default:training"`
	StartWeekday   int16     `gorm:"not null;default:1"`
	StartTime      string    `gorm:"not null;default:''"`
	EndTime        string    `gorm:"not null;default:''"`
	PublishEnabled bool      `gorm:"not null;default:true"`
	CostAmount     *float64
	MinVotesToHold int32  `gorm:"not null;default:0"`
	CancelLead     int32  `gorm:"not null;default:180"`
	CancelNotify   bool   `gorm:"not null;default:false"`
	SettlementOn   bool   `gorm:"not null;default:true"`
	SettleBefore   bool   `gorm:"not null;default:false"`
	SettleAfter    bool   `gorm:"not null;default:true"`
	AnnounceText   string `gorm:"not null;default:''"`
	AnnounceOn     bool   `gorm:"not null;default:false"`
	AnnounceLead   int16  `gorm:"not null;default:60"`
	TeamsAutoSplit bool   `gorm:"not null;default:false"`
	TeamsPublish   bool   `gorm:"not null;default:false"`
	TeamSize       int16  `gorm:"not null;default:6"`
	PollTemplateID *uint64
	PollTemplate   string         `gorm:"not null;default:''"`
	PollQuestion   string         `gorm:"not null;default:''"`
	PollOptions    datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'"`
	PollCounted    datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'"`
	IsActive       bool           `gorm:"not null;default:true"`
	CreatedAt      time.Time      `gorm:"not null;default:now()"`
	UpdatedAt      time.Time      `gorm:"not null;default:now()"`
}

func (EventInstance) TableName() string {
	return "event_instances"
}
