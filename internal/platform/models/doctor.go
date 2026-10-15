package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrDoctorNotFound is shared so other features can detect a missing doctor without importing the doctors feature.
var ErrDoctorNotFound = errors.New("doctor not found")

// Doctor is a practitioner who sees patients in one poli. It is independent of user accounts.
type Doctor struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name           string    `gorm:"size:150;not null;index"`
	Specialization string    `gorm:"size:100;not null"`
	PoliID         uuid.UUID `gorm:"type:uuid;not null;index"`
	Poli           *Poli     `gorm:"foreignKey:PoliID"`
	SIPNumber      string    `gorm:"column:sip_number;size:50;not null;uniqueIndex"` // practice license
	STRNumber      string    `gorm:"column:str_number;size:50;not null;uniqueIndex"` // registration certificate
	IHSNumber      *string   `gorm:"column:ihs_number;size:20;uniqueIndex"`          // Satu Sehat practitioner ID
	IsActive       bool      `gorm:"not null"`                                       // no DB default: GORM would turn false into true
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
