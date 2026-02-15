package postgres

import "time"

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
