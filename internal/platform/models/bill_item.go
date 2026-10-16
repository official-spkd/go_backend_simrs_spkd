package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	BillItemAdministration = "administration"
	BillItemConsultation   = "consultation"
	BillItemProcedure      = "procedure"
	BillItemMedication     = "medication"
	BillItemOther          = "other"

	BillItemSourceRegistration = "registration"
	BillItemSourceManual       = "manual"
)

// BillItem is one line of a bill. Manual lines are stored as soon as they are added; derived
// lines are stored when the bill is finalized, so later tariff changes do not affect it.
type BillItem struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	BillID          uuid.UUID  `gorm:"type:uuid;not null;index"`
	Category        string     `gorm:"size:20;not null"`
	Name            string     `gorm:"size:150;not null"`
	Source          string     `gorm:"size:20;not null"`
	TariffCode      string     `gorm:"size:20;not null;default:''"`
	ServedAt        time.Time  `gorm:"not null"`
	Price           int64      `gorm:"not null"`
	Quantity        int        `gorm:"not null"`
	Subtotal        int64      `gorm:"not null"`
	IsManual        bool       `gorm:"not null"`
	Reason          string     `gorm:"type:text;not null;default:''"`
	CreatedByUserID *uuid.UUID `gorm:"type:uuid"`
	CreatedBy       *User      `gorm:"foreignKey:CreatedByUserID"`
	CreatedAt       time.Time
}
