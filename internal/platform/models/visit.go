package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Penjamin is who pays for the visit.
type Penjamin string

const (
	PenjaminUmum     Penjamin = "umum"
	PenjaminBPJS     Penjamin = "bpjs"
	PenjaminAsuransi Penjamin = "asuransi"
)

// VisitType tells a first visit (baru) from a follow-up (kontrol).
type VisitType string

const (
	VisitTypeNew      VisitType = "baru"
	VisitTypeFollowUp VisitType = "kontrol"
)

// ErrVisitNotFound is shared so other features can detect a missing visit without importing the visits feature.
var ErrVisitNotFound = errors.New("visit not found")

type VisitStatus string

const (
	VisitRegistered VisitStatus = "Registered"
	VisitServed     VisitStatus = "Served"
	VisitCompleted  VisitStatus = "Completed"
	VisitPaid       VisitStatus = "Paid"
	VisitCancelled  VisitStatus = "Cancelled"
)

type QueueStatus string

const (
	QueueWaiting QueueStatus = "Waiting"
	QueueCalled  QueueStatus = "Called"
	QueueSkipped QueueStatus = "Skipped"
	QueueDone    QueueStatus = "Done"
)

type SatuSehatStatus string

const (
	SatuSehatNotSent SatuSehatStatus = "NotSent"
	SatuSehatSent    SatuSehatStatus = "Sent"
)

// Visit is an outpatient registration: one patient queued for one clinic (poli) on one date.
type Visit struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	PatientID uuid.UUID `gorm:"type:uuid;not null;index"`
	Patient   Patient   `gorm:"foreignKey:PatientID"`

	VisitDate time.Time `gorm:"type:date;not null;index;uniqueIndex:idx_visits_poli_queue,priority:1"`
	// PoliID and DoctorID are the source of truth; names are always looked up from polis and
	// doctors. Both are NULL only for visits registered before they became managed entities.
	PoliID        *uuid.UUID `gorm:"type:uuid;index;uniqueIndex:idx_visits_poli_queue,priority:2"`
	Poli          *Poli      `gorm:"foreignKey:PoliID"`
	QueueSequence int        `gorm:"not null;uniqueIndex:idx_visits_poli_queue,priority:3"`
	QueueNumber   string     `gorm:"size:10;not null"`
	DoctorID      *uuid.UUID `gorm:"type:uuid;index"`
	Doctor        *Doctor    `gorm:"foreignKey:DoctorID"`

	// LegacyPoli and LegacyDoctor hold the clinic and doctor text of old visits. New visits leave
	// them empty.
	LegacyPoli   *string `gorm:"column:poli;size:100"`
	LegacyDoctor *string `gorm:"column:doctor;size:150"`

	// Penjamin keeps the original "payer" column name so existing databases migrate in place.
	Penjamin        Penjamin        `gorm:"column:payer;size:20;not null"`
	VisitType       VisitType       `gorm:"size:20;not null;default:baru"`
	VisitStatus     VisitStatus     `gorm:"size:20;not null;index"`
	QueueStatus     QueueStatus     `gorm:"size:20;not null"`
	SatuSehatStatus SatuSehatStatus `gorm:"size:20;not null"`

	ResponsiblePerson string `gorm:"size:150"`
	// ChiefComplaint keeps the original "complaint" column name so existing databases migrate in place.
	ChiefComplaint string `gorm:"column:complaint;type:text;not null"`
	Allergies      string `gorm:"type:text"`
	IsApproved     bool   `gorm:"not null;default:false"`

	// Only for Penjamin == bpjs.
	BPJSCardNumber      string `gorm:"column:bpjs_card_number;size:20"`
	FKTPReferralNumber  string `gorm:"column:fktp_referral_number;size:50"`
	ControlLetterNumber string `gorm:"size:50"`
	DocumentType        string `gorm:"size:50"`

	// Only for Penjamin == asuransi. CoverageLimit is in Rupiah.
	InsuranceName string `gorm:"size:150"`
	PolicyNumber  string `gorm:"size:50"`
	CoverageLimit *int64

	CreatedAt time.Time
	UpdatedAt time.Time
}

// PoliName is the clinic name from polis, or the stored text for legacy visits.
func (v *Visit) PoliName() string {
	if v.Poli != nil {
		return v.Poli.Name
	}
	if v.LegacyPoli != nil {
		return *v.LegacyPoli
	}
	return ""
}

// DoctorName is the doctor name from doctors, or the stored text for legacy visits.
func (v *Visit) DoctorName() string {
	if v.Doctor != nil {
		return v.Doctor.Name
	}
	if v.LegacyDoctor != nil {
		return *v.LegacyDoctor
	}
	return ""
}
