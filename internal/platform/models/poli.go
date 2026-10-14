package models

import (
	"time"

	"github.com/google/uuid"
)

// Poli is an outpatient clinic.
type Poli struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	PrefixCode  string    `gorm:"size:5;not null;uniqueIndex"`
	Name        string    `gorm:"size:100;not null"`
	Room        string    `gorm:"size:100;not null"`
	IsActive    bool      `gorm:"not null"` // no DB default: GORM would turn an explicit false into true
	DoctorCount int       `gorm:"not null;default:0"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
