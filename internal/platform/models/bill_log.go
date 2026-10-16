package models

import (
	"time"

	"github.com/google/uuid"
)

// BillLog is the audit trail of a bill: every change, finalization, payment and print.
type BillLog struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	BillID    uuid.UUID `gorm:"type:uuid;not null;index"`
	Action    string    `gorm:"size:30;not null"`
	Detail    string    `gorm:"type:text;not null;default:''"`
	Reason    string    `gorm:"type:text;not null;default:''"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"`
	CreatedAt time.Time
}
