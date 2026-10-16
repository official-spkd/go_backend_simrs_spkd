package models

import (
	"time"

	"github.com/google/uuid"
)

// Tariff is a billable service with one price per penjamin, in Rupiah.
type Tariff struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Code           string    `gorm:"size:20;not null;uniqueIndex"`
	Name           string    `gorm:"size:150;not null"`
	PriceGeneral   int64     `gorm:"not null"`
	PriceBPJS      int64     `gorm:"column:price_bpjs;not null"`
	PriceInsurance int64     `gorm:"not null"`
	EffectiveDate  time.Time `gorm:"type:date;not null"`
	Version        int       `gorm:"not null"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// PriceFor returns the price charged to a visit with the given penjamin.
func (t *Tariff) PriceFor(p Penjamin) int64 {
	switch p {
	case PenjaminBPJS:
		return t.PriceBPJS
	case PenjaminAsuransi:
		return t.PriceInsurance
	default:
		return t.PriceGeneral
	}
}
