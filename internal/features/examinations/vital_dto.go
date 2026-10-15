package examinations

import "time"

// VitalRequest saves the vital signs and initial assessment of a visit. Send version 0 when
// creating and the current version when updating.
type VitalRequest struct {
	SystolicBP      int     `json:"systolic_bp" binding:"required,gte=40,lte=300" example:"120"`
	DiastolicBP     int     `json:"diastolic_bp" binding:"required,gte=20,lte=200" example:"80"`
	Pulse           int     `json:"pulse" binding:"required,gte=20,lte=250" example:"82"`
	RespiratoryRate int     `json:"respiratory_rate" binding:"required,gte=4,lte=80" example:"18"`
	Temperature     float64 `json:"temperature" binding:"required,gte=30,lte=45" example:"36.8"`
	SpO2            int     `json:"spo2" binding:"required,gte=50,lte=100" example:"98"`

	// Height and weight go together; they are required for a first visit (visit_type baru).
	HeightCM *float64 `json:"height_cm" binding:"omitempty,gte=30,lte=250" example:"165"`
	WeightKG *float64 `json:"weight_kg" binding:"omitempty,gte=0.5,lte=400" example:"60"`

	Consciousness string `json:"consciousness" binding:"required,oneof=compos_mentis apathy delirium somnolence stupor coma" enums:"compos_mentis,apathy,delirium,somnolence,stupor,coma" example:"compos_mentis"`
	PainScore     *int   `json:"pain_score" binding:"required,gte=0,lte=10" example:"2"`

	FallRisk      *bool  `json:"fall_risk" binding:"required" example:"true"`
	FallRiskLevel string `json:"fall_risk_level" binding:"omitempty,oneof=low moderate high" enums:"low,moderate,high" example:"low"` // required when fall_risk is true

	// Either list the allergies or confirm there are none; an empty text alone is not an answer.
	Allergies        string `json:"allergies" binding:"omitempty,max=1000" example:"Penicillin, shrimp"`
	NoKnownAllergies bool   `json:"no_known_allergies" example:"false"`

	ChiefComplaint string `json:"chief_complaint" binding:"required,notblank,max=1000" example:"Fever and cough for three days"`

	Version int `json:"version" binding:"gte=0" example:"0"`
}

// Flag is an advisory warning about a measurement, computed with adult reference values.
type Flag struct {
	Field   string `json:"field" example:"spo2"`
	Level   string `json:"level" enums:"warning,critical" example:"warning"`
	Message string `json:"message" example:"SpO2 below 95%"`
}

type VitalResponse struct {
	VisitID          string    `json:"visit_id" example:"104dd35e-2b3a-4b2a-b38c-8bdd0e1ecc79"`
	MeasuredByUserID string    `json:"measured_by_user_id" example:"5b815a95-e4d9-40c1-bfc0-dccbe1f38722"`
	MeasuredAt       time.Time `json:"measured_at"`

	SystolicBP      int      `json:"systolic_bp" example:"120"`
	DiastolicBP     int      `json:"diastolic_bp" example:"80"`
	Pulse           int      `json:"pulse" example:"82"`
	RespiratoryRate int      `json:"respiratory_rate" example:"18"`
	Temperature     float64  `json:"temperature" example:"36.8"`
	SpO2            int      `json:"spo2" example:"98"`
	HeightCM        *float64 `json:"height_cm" example:"165"`
	WeightKG        *float64 `json:"weight_kg" example:"60"`
	BMI             *float64 `json:"bmi" example:"22"`
	BMICategory     string   `json:"bmi_category" enums:"severely_underweight,underweight,normal,overweight,obese,not_applicable" example:"normal"`

	Consciousness string `json:"consciousness" example:"compos_mentis"`
	PainScore     int    `json:"pain_score" example:"2"`
	PainCategory  string `json:"pain_category" enums:"none,mild,moderate,severe" example:"mild"`

	FallRisk      bool   `json:"fall_risk" example:"true"`
	FallRiskLevel string `json:"fall_risk_level" example:"low"`

	Allergies        string `json:"allergies" example:"Penicillin, shrimp"`
	NoKnownAllergies bool   `json:"no_known_allergies" example:"false"`
	ChiefComplaint   string `json:"chief_complaint" example:"Fever and cough for three days"`

	Flags     []Flag    `json:"flags"`
	Version   int       `json:"version" example:"0"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toVitalResponse(v *VitalAssessment) VitalResponse {
	return VitalResponse{
		VisitID:          v.VisitID.String(),
		MeasuredByUserID: v.MeasuredByUserID.String(),
		MeasuredAt:       v.MeasuredAt,
		SystolicBP:       v.SystolicBP,
		DiastolicBP:      v.DiastolicBP,
		Pulse:            v.Pulse,
		RespiratoryRate:  v.RespiratoryRate,
		Temperature:      v.Temperature,
		SpO2:             v.SpO2,
		HeightCM:         v.HeightCM,
		WeightKG:         v.WeightKG,
		BMI:              v.BMI,
		BMICategory:      v.BMICategory,
		Consciousness:    v.Consciousness,
		PainScore:        v.PainScore,
		PainCategory:     painCategory(v.PainScore),
		FallRisk:         v.FallRisk,
		FallRiskLevel:    v.FallRiskLevel,
		Allergies:        v.Allergies,
		NoKnownAllergies: v.NoKnownAllergies,
		ChiefComplaint:   v.ChiefComplaint,
		Flags:            flagsFor(v),
		Version:          v.Version,
		UpdatedAt:        v.UpdatedAt,
	}
}
