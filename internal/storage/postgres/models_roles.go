package postgres

import (
	"time"

	"gorm.io/datatypes"
)

type GroupRole struct {
	ID          uint64         `gorm:"primaryKey"`
	GroupID     uint64         `gorm:"not null;index;uniqueIndex:ux_group_role"`
	Code        string         `gorm:"not null;uniqueIndex:ux_group_role"`
	Title       string         `gorm:"not null"`
	Permissions datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'"`
	IsActive    bool           `gorm:"not null;default:true"`
	CreatedAt   time.Time      `gorm:"not null;default:now()"`
	UpdatedAt   time.Time      `gorm:"not null;default:now()"`
}

func (GroupRole) TableName() string {
	return "group_roles"
}

type GroupRoleAssignment struct {
	ID             uint64    `gorm:"primaryKey"`
	GroupID        uint64    `gorm:"not null;index;uniqueIndex:ux_group_role_assignment"`
	UserTelegramID int64     `gorm:"not null;uniqueIndex:ux_group_role_assignment"`
	RoleCode       string    `gorm:"not null"`
	IsActive       bool      `gorm:"not null;default:true"`
	CreatedAt      time.Time `gorm:"not null;default:now()"`
	UpdatedAt      time.Time `gorm:"not null;default:now()"`
}

func (GroupRoleAssignment) TableName() string {
	return "group_role_assignments"
}
