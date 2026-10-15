package examinations

import (
	"time"

	"github.com/google/uuid"

	"gin-quickstart/internal/platform/models"
)

const (
	SoapDraft = "draft"
	SoapFinal = "final"

	FollowUpDischarge  = "discharge"
	FollowUpOutpatient = "outpatient_followup"
	FollowUpInpatient  = "refer_inpatient"
	FollowUpICU        = "refer_icu"

	DiagnosisPrimary   = "primary"
	DiagnosisSecondary = "secondary"
)

// SoapNote is the doctor's SOAP note of one visit. A final note is immutable; corrections are
// added as addenda.
type SoapNote struct {
	ID       uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	VisitID  uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex"`
	Visit    *models.Visit  `gorm:"foreignKey:VisitID"`
	DoctorID uuid.UUID      `gorm:"type:uuid;not null;index"`
	Doctor   *models.Doctor `gorm:"foreignKey:DoctorID"`
	Status   string         `gorm:"size:10;not null"`

	Subjective        string `gorm:"type:text;not null;default:''"`
	ObjectiveFindings string `gorm:"type:text;not null;default:''"`
	// ObjectiveVitalsSnapshot freezes the vital signs shown in "O" when the note is finalized.
	ObjectiveVitalsSnapshot *string    `gorm:"type:jsonb"`
	AssessmentNotes         string     `gorm:"type:text;not null;default:''"`
	Plan                    string     `gorm:"type:text;not null;default:''"`
	FollowUp                string     `gorm:"size:25;not null;default:''"`
	ControlDate             *time.Time `gorm:"type:date"`
	ReferralNotes           string     `gorm:"type:text;not null;default:''"`

	FinalizedAt       *time.Time
	FinalizedByUserID *uuid.UUID `gorm:"type:uuid"`

	Version   int `gorm:"not null;default:0"`
	CreatedAt time.Time
	UpdatedAt time.Time

	Diagnoses []SoapDiagnosis `gorm:"foreignKey:SoapNoteID;constraint:OnDelete:CASCADE"`
	Addenda   []SoapAddendum  `gorm:"foreignKey:SoapNoteID;constraint:OnDelete:CASCADE"`
}

// SoapDiagnosis is an ICD-10 diagnosis of a note. There is no ICD-10 master table yet, so the
// code is validated by format only and the name is typed by the doctor.
type SoapDiagnosis struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	SoapNoteID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_soap_diagnosis_code,priority:1;uniqueIndex:idx_soap_diagnosis_primary,where:role = 'primary'"`
	ICD10Code  string    `gorm:"column:icd10_code;size:10;not null;uniqueIndex:idx_soap_diagnosis_code,priority:2"`
	Name       string    `gorm:"size:255;not null;default:''"`
	Role       string    `gorm:"size:10;not null"`
	SortOrder  int       `gorm:"not null"`
}

// SoapAddendum is a correction or note appended to a finalized SOAP note.
type SoapAddendum struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	SoapNoteID   uuid.UUID `gorm:"type:uuid;not null;index"`
	AuthorUserID uuid.UUID `gorm:"type:uuid;not null"`
	Content      string    `gorm:"type:text;not null"`
	CreatedAt    time.Time
}
