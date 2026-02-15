package postgres

import "time"

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
