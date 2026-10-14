package models

import (
	"time"

	"github.com/google/uuid"

	"gin-quickstart/internal/features/roles"
)

type User struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name         string     `gorm:"size:150;not null"`
	NIP          string     `gorm:"column:nip;size:30;uniqueIndex;not null"`
	Email        string     `gorm:"size:150;uniqueIndex;not null"`
	PasswordHash string     `gorm:"not null"`
	RoleID       uuid.UUID  `gorm:"type:uuid;not null;index"`
	Role         roles.Role `gorm:"foreignKey:RoleID"`
	IsActive     bool       `gorm:"not null;default:true"`

	// Doctor-only fields, NULL for every other role. PoliID is the clinic the doctor works in.
	SIPNumber *string    `gorm:"column:sip_number;size:50;uniqueIndex"` // practice license (Surat Izin Praktik)
	STRNumber *string    `gorm:"column:str_number;size:50;uniqueIndex"` // registration certificate (Surat Tanda Registrasi)
	PoliID    *uuid.UUID `gorm:"type:uuid;index"`
	Poli      *Poli      `gorm:"foreignKey:PoliID"`

	CreatedAt time.Time
	UpdatedAt time.Time
}
