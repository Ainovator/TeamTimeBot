package postgres

import "time"

type EventPollPost struct {
	ID                uint64    `gorm:"primaryKey"`
	GroupID           uint64    `gorm:"not null;index"`
	EventID           *uint64   `gorm:"index"`
	InstanceID        *uint64   `gorm:"index"`
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
	PostID    uint64 `gorm:"not null;index;uniqueIndex:ux_post_user_choice"`
	UserID    int64  `gorm:"not null;uniqueIndex:ux_post_user_choice;index"`
	Username  string
	FirstName string
	LastName  string
	Choice    string    `gorm:"not null;uniqueIndex:ux_post_user_choice;index"`
	Source    string    `gorm:"not null;default:inline"`
	VotedAt   time.Time `gorm:"not null;default:now()"`
	CreatedAt time.Time `gorm:"not null;default:now()"`
	UpdatedAt time.Time `gorm:"not null;default:now()"`
}

func (EventPollVote) TableName() string {
	return "event_poll_votes"
}
