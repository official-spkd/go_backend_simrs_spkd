package models

import "time"

// DocumentCounter issues sequential document numbers per key, e.g. "INV/20260101".
type DocumentCounter struct {
	Key       string `gorm:"size:30;primaryKey"`
	Value     int    `gorm:"not null"`
	UpdatedAt time.Time
}
