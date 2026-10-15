package examinations

import (
	"time"

	"github.com/google/uuid"

	"gin-quickstart/internal/platform/models"
)

const (
	ConsciousnessCompos    = "compos_mentis"
	ConsciousnessApathy    = "apathy"
	ConsciousnessDelirium  = "delirium"
	ConsciousnessSomnolent = "somnolence"
	ConsciousnessStupor    = "stupor"
	ConsciousnessComa      = "coma"
)

// VitalAssessment is the initial assessment and vital signs of one visit.
type VitalAssessment struct {
	ID               uuid.UUID     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	VisitID          uuid.UUID     `gorm:"type:uuid;not null;uniqueIndex"`
	Visit            *models.Visit `gorm:"foreignKey:VisitID"`
	MeasuredByUserID uuid.UUID     `gorm:"type:uuid;not null"`
	MeasuredAt       time.Time     `gorm:"not null"`

	SystolicBP       int      `gorm:"type:smallint;not null"`             // mmHg
	DiastolicBP      int      `gorm:"type:smallint;not null"`             // mmHg
	Pulse            int      `gorm:"type:smallint;not null"`             // beats/minute
	RespiratoryRate  int      `gorm:"type:smallint;not null"`             // breaths/minute
	Temperature      float64  `gorm:"type:numeric(3,1);not null"`         // °C
	SpO2             int      `gorm:"column:spo2;type:smallint;not null"` // %
	HeightCM         *float64 `gorm:"column:height_cm;type:numeric(4,1)"`
	WeightKG         *float64 `gorm:"column:weight_kg;type:numeric(5,2)"`
	BMI              *float64 `gorm:"column:bmi;type:numeric(4,1)"` // computed by the server
	BMICategory      string   `gorm:"column:bmi_category;size:30;not null;default:''"`
	Consciousness    string   `gorm:"size:20;not null"`
	PainScore        int      `gorm:"type:smallint;not null"` // 0-10
	FallRisk         bool     `gorm:"not null"`
	FallRiskLevel    string   `gorm:"size:20;not null;default:''"`
	Allergies        string   `gorm:"type:text;not null;default:''"`
	NoKnownAllergies bool     `gorm:"not null"`
	ChiefComplaint   string   `gorm:"type:text;not null"`

	Version   int `gorm:"not null;default:0"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
