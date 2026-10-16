package models

import (
	"time"

	"github.com/google/uuid"
)

// CashierShift groups the payments one cashier received. A cashier has at most one open shift;
// closing it records the cash count and locks its total.
type CashierShift struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CashierUserID uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:idx_cashier_shifts_open,where:closed_at IS NULL"`
	Cashier       *User     `gorm:"foreignKey:CashierUserID"`
	OpenedAt      time.Time `gorm:"not null;index"`
	ClosedAt      *time.Time

	// Set when the shift is closed. CashDifference is the counted cash minus the recorded cash.
	CountedCash    *int64
	CashDifference *int64
	Total          *int64
	Note           string `gorm:"type:text;not null;default:''"`

	CreatedAt time.Time
	UpdatedAt time.Time
}
