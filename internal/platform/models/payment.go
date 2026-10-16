package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	PaymentCash = "cash"
	PaymentQRIS = "qris"
	PaymentVA   = "va"
)

// Payment is money received for a bill, in Rupiah. Received and Change are kept for cash only.
type Payment struct {
	ID            uuid.UUID     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	BillID        uuid.UUID     `gorm:"type:uuid;not null;index"`
	ShiftID       uuid.UUID     `gorm:"type:uuid;not null;index"`
	Shift         *CashierShift `gorm:"foreignKey:ShiftID"`
	Method        string        `gorm:"size:10;not null"`
	Amount        int64         `gorm:"not null"`
	Received      *int64
	Change        *int64
	Reference     string    `gorm:"size:50;not null;default:''"`
	CashierUserID uuid.UUID `gorm:"type:uuid;not null;index"`
	Cashier       *User     `gorm:"foreignKey:CashierUserID"`
	PaidAt        time.Time `gorm:"not null"`
	CreatedAt     time.Time
}
