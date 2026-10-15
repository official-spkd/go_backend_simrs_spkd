package examinations

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"gin-quickstart/internal/platform/models"
	"gin-quickstart/internal/platform/utils"
)

const adultAge = 18

func (s *service) GetVitals(ctx context.Context, visitID uuid.UUID, actor *utils.TokenClaims) (*models.VitalAssessment, error) {
	if _, err := s.authorize(ctx, visitID, actor, false); err != nil {
		return nil, err
	}
	return s.repo.FindVital(ctx, visitID)
}

func (s *service) SaveVitals(ctx context.Context, visitID uuid.UUID, req VitalRequest, actor *utils.TokenClaims) (*models.VitalAssessment, error) {
	visit, err := s.authorize(ctx, visitID, actor, true)
	if err != nil {
		return nil, err
	}
	if err := s.ensureNotFinalized(ctx, visitID); err != nil {
		return nil, err
	}
	if err := validateVitals(req, visit); err != nil {
		return nil, err
	}

	v := &models.VitalAssessment{
		VisitID:          visitID,
		MeasuredByUserID: actor.UserID,
		MeasuredAt:       time.Now(),
		SystolicBP:       req.SystolicBP,
		DiastolicBP:      req.DiastolicBP,
		Pulse:            req.Pulse,
		RespiratoryRate:  req.RespiratoryRate,
		Temperature:      round(req.Temperature, 1),
		SpO2:             req.SpO2,
		HeightCM:         roundPtr(req.HeightCM, 1),
		WeightKG:         roundPtr(req.WeightKG, 2),
		Consciousness:    req.Consciousness,
		PainScore:        *req.PainScore,
		FallRisk:         *req.FallRisk,
		FallRiskLevel:    req.FallRiskLevel,
		Allergies:        strings.TrimSpace(req.Allergies),
		NoKnownAllergies: req.NoKnownAllergies,
		ChiefComplaint:   strings.TrimSpace(req.ChiefComplaint),
	}
	v.BMI, v.BMICategory = bmi(v.HeightCM, v.WeightKG, utils.AgeOn(visit.Patient.BirthDate, visit.VisitDate))

	now := time.Now()
	v.UpdatedAt = now
	reason := strings.TrimSpace(req.ChangeReason)

	existing, err := s.repo.FindVital(ctx, visitID)
	switch {
	case errors.Is(err, ErrVitalNotFound):
		if req.Version != 0 {
			return nil, ErrVersionConflict
		}
		entry, err := newVersion(v, reason, actor.UserID, now)
		if err != nil {
			return nil, err
		}
		err = s.repo.SaveVital(ctx, v, nil, entry, nil)
		if err != nil {
			return nil, err
		}
	case err != nil:
		return nil, err
	default:
		if req.Version != existing.Version {
			return nil, ErrVersionConflict
		}
		if reason == "" {
			return nil, InputError("change_reason is required when changing saved vital signs")
		}
		v.ID, v.CreatedAt, v.Version = existing.ID, existing.CreatedAt, existing.Version+1

		entry, err := newVersion(v, reason, actor.UserID, now)
		if err != nil {
			return nil, err
		}
		baseline, err := newVersion(existing, "Recorded before version history existed", existing.MeasuredByUserID, existing.UpdatedAt)
		if err != nil {
			return nil, err
		}
		if len(changedFields(baseline.Snapshot, entry.Snapshot)) == 0 {
			return nil, InputError("nothing changed; there is no new version to save")
		}
		if err := s.repo.SaveVital(ctx, v, &existing.Version, entry, &baseline); err != nil {
			return nil, err
		}
	}
	return v, nil
}

// validateVitals checks the rules that span several fields.
func validateVitals(req VitalRequest, visit *models.Visit) error {
	if req.SystolicBP <= req.DiastolicBP {
		return InputError("systolic_bp must be greater than diastolic_bp")
	}
	if (req.HeightCM == nil) != (req.WeightKG == nil) {
		return InputError("height_cm and weight_kg must be sent together")
	}
	if req.HeightCM == nil && visit.VisitType == models.VisitTypeNew {
		return InputError("height_cm and weight_kg are required for a first visit")
	}
	if *req.FallRisk && req.FallRiskLevel == "" {
		return InputError("fall_risk_level is required when fall_risk is true")
	}
	if !*req.FallRisk && req.FallRiskLevel != "" {
		return InputError("fall_risk_level is only allowed when fall_risk is true")
	}

	hasAllergies := strings.TrimSpace(req.Allergies) != ""
	if !hasAllergies && !req.NoKnownAllergies {
		return InputError("allergies or no_known_allergies is required")
	}
	if hasAllergies && req.NoKnownAllergies {
		return InputError("allergies and no_known_allergies cannot both be set")
	}
	return nil
}

// bmi computes the body mass index and its Kemenkes RI category. Children under 18 get the value
// but no category, because their BMI is judged against age-based charts.
func bmi(heightCM, weightKG *float64, ageYears int) (*float64, string) {
	if heightCM == nil || weightKG == nil {
		return nil, ""
	}
	m := *heightCM / 100
	value := round(*weightKG/(m*m), 1)

	if ageYears < adultAge {
		return &value, "not_applicable"
	}
	switch {
	case value < 17.0:
		return &value, "severely_underweight"
	case value < 18.5:
		return &value, "underweight"
	case value <= 25.0:
		return &value, "normal"
	case value <= 27.0:
		return &value, "overweight"
	default:
		return &value, "obese"
	}
}

func painCategory(score int) string {
	switch {
	case score == 0:
		return "none"
	case score <= 3:
		return "mild"
	case score <= 6:
		return "moderate"
	default:
		return "severe"
	}
}

// flagsFor returns advisory warnings using adult reference values.
func flagsFor(v *models.VitalAssessment) []Flag {
	flags := []Flag{}
	add := func(field, level, msg string) { flags = append(flags, Flag{Field: field, Level: level, Message: msg}) }

	switch {
	case v.SpO2 < 90:
		add("spo2", "critical", "SpO2 below 90%")
	case v.SpO2 < 95:
		add("spo2", "warning", "SpO2 below 95%")
	}
	switch {
	case v.Temperature >= 40:
		add("temperature", "critical", "Temperature 40 °C or higher")
	case v.Temperature >= 38:
		add("temperature", "warning", "Fever (38 °C or higher)")
	case v.Temperature < 35:
		add("temperature", "warning", "Temperature below 35 °C")
	}
	switch {
	case v.SystolicBP >= 180 || v.DiastolicBP >= 120:
		add("blood_pressure", "critical", "Blood pressure 180/120 or higher")
	case v.SystolicBP >= 140 || v.DiastolicBP >= 90:
		add("blood_pressure", "warning", "Blood pressure 140/90 or higher")
	case v.SystolicBP < 90 || v.DiastolicBP < 60:
		add("blood_pressure", "warning", "Blood pressure below 90/60")
	}
	if v.Pulse < 60 || v.Pulse > 100 {
		add("pulse", "warning", "Pulse outside 60-100 per minute")
	}
	if v.RespiratoryRate < 12 || v.RespiratoryRate > 20 {
		add("respiratory_rate", "warning", "Respiratory rate outside 12-20 per minute")
	}
	return flags
}

func round(v float64, places int) float64 {
	p := math.Pow(10, float64(places))
	return math.Round(v*p) / p
}

func roundPtr(v *float64, places int) *float64 {
	if v == nil {
		return nil
	}
	r := round(*v, places)
	return &r
}

// newVersion captures the vital signs as they are in v, for the history.
func newVersion(v *models.VitalAssessment, reason string, userID uuid.UUID, at time.Time) (models.VitalVersion, error) {
	snapshot, err := json.Marshal(toVitalResponse(v))
	if err != nil {
		return models.VitalVersion{}, err
	}
	return models.VitalVersion{
		Version:         v.Version,
		Snapshot:        string(snapshot),
		ChangeReason:    reason,
		ChangedByUserID: userID,
		ChangedAt:       at,
	}, nil
}

func (s *service) ListVitalVersions(ctx context.Context, visitID uuid.UUID, actor *utils.TokenClaims) ([]VersionSummary, error) {
	vital, err := s.vitalOf(ctx, visitID, actor)
	if err != nil {
		return nil, err
	}
	versions, err := s.repo.ListVitalVersions(ctx, vital.ID)
	if err != nil {
		return nil, err
	}

	out := make([]VersionSummary, len(versions))
	for i := range versions {
		var previous *models.VitalVersion
		if i+1 < len(versions) { // versions are newest first
			previous = &versions[i+1]
		}
		out[i] = toVersionSummary(versions[i], previous)
	}
	return out, nil
}

func (s *service) GetVitalVersion(ctx context.Context, visitID uuid.UUID, version int, actor *utils.TokenClaims) (*VersionDetail, error) {
	vital, err := s.vitalOf(ctx, visitID, actor)
	if err != nil {
		return nil, err
	}
	found, err := s.repo.FindVitalVersion(ctx, vital.ID, version)
	if err != nil {
		return nil, err
	}

	var previous *models.VitalVersion
	if version > 0 {
		if p, err := s.repo.FindVitalVersion(ctx, vital.ID, version-1); err == nil {
			previous = p
		} else if !errors.Is(err, ErrVersionNotFound) {
			return nil, err
		}
	}

	var snapshot VitalResponse
	if err := json.Unmarshal([]byte(found.Snapshot), &snapshot); err != nil {
		return nil, err
	}
	return &VersionDetail{VersionSummary: toVersionSummary(*found, previous), VitalSigns: snapshot}, nil
}

// vitalOf checks access to the visit and returns its vital signs record.
func (s *service) vitalOf(ctx context.Context, visitID uuid.UUID, actor *utils.TokenClaims) (*models.VitalAssessment, error) {
	if _, err := s.authorize(ctx, visitID, actor, false); err != nil {
		return nil, err
	}
	return s.repo.FindVital(ctx, visitID)
}
