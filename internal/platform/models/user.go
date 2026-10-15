package models

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"gin-quickstart/internal/features/roles"
)

// ErrUserNotFound is shared so other features can detect a missing user without importing the users feature.
var ErrUserNotFound = errors.New("user not found")

type User struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name         string     `gorm:"size:150;not null"`
	NIP          string     `gorm:"column:nip;size:30;uniqueIndex;not null"`
	Email        string     `gorm:"size:150;uniqueIndex;not null"`
	PasswordHash string     `gorm:"not null"`
	RoleID       uuid.UUID  `gorm:"type:uuid;not null;index"`
	Role         roles.Role `gorm:"foreignKey:RoleID"`
	IsActive     bool       `gorm:"not null;default:true"`

	// DoctorID is set exactly for accounts with the doctor role: the doctor record must exist first.
	DoctorID *uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	Doctor   *Doctor    `gorm:"foreignKey:DoctorID"`

	CreatedAt time.Time
	UpdatedAt time.Time
}
