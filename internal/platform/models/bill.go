package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	BillDraft = "draft"
	BillFinal = "final"
	BillPaid  = "paid"
)

// Bill is the cashier's bill of one visit. While draft, the registration and consultation lines
// are derived from the visit; finalizing freezes them as BillItem rows and issues the invoice
// number. Paying issues the receipt number and moves the visit to Paid.
type Bill struct {
	ID      uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	VisitID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	Visit   *Visit    `gorm:"foreignKey:VisitID"`
	Status  string    `gorm:"size:10;not null;index"`

	// Scheme is who pays the bill; it starts as the visit's penjamin. CoverageLimit applies to
	// asuransi and Excess (class upgrade) to bpjs. All amounts are in Rupiah.
	Scheme         Penjamin `gorm:"size:20;not null"`
	CoverageLimit  int64    `gorm:"not null;default:0"`
	Excess         int64    `gorm:"not null;default:0"`
	DiscountAmount int64    `gorm:"not null;default:0"`
	DiscountReason string   `gorm:"type:text;not null;default:''"`

	InvoiceNumber     *string `gorm:"size:30;uniqueIndex"`
	FinalizedAt       *time.Time
	FinalizedByUserID *uuid.UUID `gorm:"type:uuid"`
	ReceiptNumber     *string    `gorm:"size:30;uniqueIndex"`
	PaidAt            *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time

	Items    []BillItem `gorm:"foreignKey:BillID;constraint:OnDelete:CASCADE"`
	Payments []Payment  `gorm:"foreignKey:BillID;constraint:OnDelete:CASCADE"`
}
