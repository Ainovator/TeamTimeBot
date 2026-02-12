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

type PollTemplate struct {
	ID        uint64         `gorm:"primaryKey"`
	GroupID   uint64         `gorm:"not null;index;uniqueIndex:ux_group_template_name"`
	Name      string         `gorm:"not null;uniqueIndex:ux_group_template_name"`
	Question  string         `gorm:"not null"`
	Options   datatypes.JSON `gorm:"type:jsonb;not null"`
	IsActive  bool           `gorm:"not null;default:true"`
	CreatedAt time.Time      `gorm:"not null;default:now()"`
	UpdatedAt time.Time      `gorm:"not null;default:now()"`
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
