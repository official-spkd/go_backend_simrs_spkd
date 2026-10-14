package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Gender string

const (
	GenderMale   Gender = "Male"
	GenderFemale Gender = "Female"
)

type Payer string

const (
	PayerGeneral   Payer = "General"
	PayerBPJS      Payer = "BPJS"
	PayerInsurance Payer = "Insurance"
)

type Patient struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	NIK       string    `gorm:"column:nik;size:16;not null;uniqueIndex:idx_patients_nik,where:deleted_at IS NULL"`
	FullName  string    `gorm:"size:150;not null;index"`
	BirthDate time.Time `gorm:"type:date;not null"`
	Gender    Gender    `gorm:"size:10;not null"`

	BirthPlace    string `gorm:"size:100;not null"`
	Religion      string `gorm:"size:50"`
	Education     string `gorm:"size:50"`
	Occupation    string `gorm:"size:100"`
	MaritalStatus string `gorm:"size:20"`
	BloodType     string `gorm:"size:2"`
	Allergies     string `gorm:"type:text"`

	Phone string `gorm:"size:20;not null"`
	Email string `gorm:"size:150"`

	IDCardAddress   string `gorm:"column:id_card_address;type:text;not null"`
	DomicileAddress string `gorm:"type:text"`

	EmergencyContactName         string `gorm:"size:150;not null"`
	EmergencyContactRelationship string `gorm:"size:50;not null"`
	EmergencyContactPhone        string `gorm:"size:20;not null"`

	Payer Payer `gorm:"size:20;not null"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}
