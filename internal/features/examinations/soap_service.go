package examinations

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"gin-quickstart/internal/platform/utils"

	"gin-quickstart/internal/platform/models"
)

func (s *service) GetSoap(ctx context.Context, visitID uuid.UUID, actor *utils.TokenClaims) (*SoapView, error) {
	if _, err := s.authorize(ctx, visitID, actor, false); err != nil {
		return nil, err
	}
	return s.view(ctx, visitID)
}

func (s *service) SaveSoap(ctx context.Context, visitID uuid.UUID, req SoapRequest, actor *utils.TokenClaims) (*SoapView, error) {
	visit, err := s.authorize(ctx, visitID, actor, true)
	if err != nil {
		return nil, err
	}
	n, err := buildNote(req)
	if err != nil {
		return nil, err
	}
	n.VisitID, n.DoctorID, n.Status = visitID, *visit.DoctorID, models.SoapDraft

	existing, err := s.repo.FindSoap(ctx, visitID)
	switch {
	case errors.Is(err, ErrSoapNotFound):
		if req.Version != 0 {
			return nil, ErrVersionConflict
		}
		err = s.repo.SaveSoap(ctx, n, nil)
	case err != nil:
		return nil, err
	default:
		if existing.Status == models.SoapFinal {
			return nil, ErrFinalized
		}
		if req.Version != existing.Version {
			return nil, ErrVersionConflict
		}
		n.ID, n.CreatedAt, n.Version = existing.ID, existing.CreatedAt, existing.Version+1
		err = s.repo.SaveSoap(ctx, n, &existing.Version)
	}
	if err != nil {
		return nil, err
	}
	return s.view(ctx, visitID)
}

func (s *service) FinalizeSoap(ctx context.Context, visitID uuid.UUID, actor *utils.TokenClaims) (*SoapView, error) {
	if _, err := s.authorize(ctx, visitID, actor, true); err != nil {
		return nil, err
	}
	note, err := s.repo.FindSoap(ctx, visitID)
	if err != nil {
		return nil, err
	}
	if note.Status == models.SoapFinal {
		return nil, ErrFinalized
	}

	vital, err := s.repo.FindVital(ctx, visitID)
	if err != nil && !errors.Is(err, ErrVitalNotFound) {
		return nil, err
	}
	if missing := missingForFinal(note, vital != nil); len(missing) > 0 {
		return nil, IncompleteError{Missing: missing}
	}

	snapshot, err := json.Marshal(toVitalResponse(vital))
	if err != nil {
		return nil, err
	}
	if err := s.repo.FinalizeSoap(ctx, note.ID, string(snapshot), actor.UserID); err != nil {
		return nil, err
	}
	return s.view(ctx, visitID)
}

func (s *service) AddAddendum(ctx context.Context, visitID uuid.UUID, req AddendumRequest, actor *utils.TokenClaims) (*SoapView, error) {
	if _, err := s.authorize(ctx, visitID, actor, false); err != nil {
		return nil, err
	}
	note, err := s.repo.FindSoap(ctx, visitID)
	if err != nil {
		return nil, err
	}
	err = s.repo.AddAddendum(ctx, &models.SoapAddendum{
		SoapNoteID:   note.ID,
		AuthorUserID: actor.UserID,
		Content:      strings.TrimSpace(req.Content),
	})
	if err != nil {
		return nil, err
	}
	return s.view(ctx, visitID)
}

// view loads the note and the live vital signs shown in its "O".
func (s *service) view(ctx context.Context, visitID uuid.UUID) (*SoapView, error) {
	note, err := s.repo.FindSoap(ctx, visitID)
	if err != nil {
		return nil, err
	}
	vital, err := s.repo.FindVital(ctx, visitID)
	if err != nil && !errors.Is(err, ErrVitalNotFound) {
		return nil, err
	}
	return &SoapView{Note: note, Vital: vital}, nil
}

// buildNote turns a draft request into a note, checking the rules that span several fields.
func buildNote(req SoapRequest) (*models.SoapNote, error) {
	n := &models.SoapNote{
		Subjective:        strings.TrimSpace(req.Subjective),
		ObjectiveFindings: strings.TrimSpace(req.ObjectiveFindings),
		AssessmentNotes:   strings.TrimSpace(req.AssessmentNotes),
		Plan:              strings.TrimSpace(req.Plan),
		FollowUp:          req.FollowUp,
		ReferralNotes:     strings.TrimSpace(req.ReferralNotes),
	}

	seen := map[string]bool{}
	add := func(in DiagnosisInput, role string) error {
		code := strings.ToUpper(strings.TrimSpace(in.ICD10Code))
		if seen[code] {
			return InputError("diagnosis " + code + " is listed more than once")
		}
		seen[code] = true
		n.Diagnoses = append(n.Diagnoses, models.SoapDiagnosis{ICD10Code: code, Name: strings.TrimSpace(in.Name), Role: role})
		return nil
	}
	if req.PrimaryDiagnosis != nil {
		if err := add(*req.PrimaryDiagnosis, models.DiagnosisPrimary); err != nil {
			return nil, err
		}
	}
	if len(req.SecondaryDiagnoses) > 0 && req.PrimaryDiagnosis == nil {
		return nil, InputError("primary_diagnosis is required when secondary_diagnoses are sent")
	}
	for _, in := range req.SecondaryDiagnoses {
		if err := add(in, models.DiagnosisSecondary); err != nil {
			return nil, err
		}
	}

	if req.ControlDate != "" {
		if req.FollowUp != models.FollowUpOutpatient {
			return nil, InputError("control_date is only allowed with follow_up outpatient_followup")
		}
		d, err := time.Parse("2006-01-02", req.ControlDate)
		if err != nil {
			return nil, InputError("control_date must use the format YYYY-MM-DD")
		}
		if !d.After(utils.Today()) {
			return nil, InputError("control_date must be after today")
		}
		n.ControlDate = &d
	}
	if n.ReferralNotes != "" && req.FollowUp != models.FollowUpInpatient && req.FollowUp != models.FollowUpICU {
		return nil, InputError("referral_notes is only allowed with refer_inpatient or refer_icu")
	}
	return n, nil
}

// missingForFinal lists the fields that must be filled before a note can be finalized.
func missingForFinal(n *models.SoapNote, hasVitals bool) []string {
	var missing []string
	need := func(ok bool, field string) {
		if !ok {
			missing = append(missing, field)
		}
	}

	need(hasVitals, "vital_signs")
	need(n.Subjective != "", "subjective")
	need(n.Plan != "", "plan")
	need(hasPrimary(n), "primary_diagnosis")
	need(n.FollowUp != "", "follow_up")
	if n.FollowUp == models.FollowUpOutpatient {
		need(n.ControlDate != nil, "control_date")
	}
	if n.FollowUp == models.FollowUpInpatient || n.FollowUp == models.FollowUpICU {
		need(n.ReferralNotes != "", "referral_notes")
	}
	return missing
}

func hasPrimary(n *models.SoapNote) bool {
	for _, d := range n.Diagnoses {
		if d.Role == models.DiagnosisPrimary {
			return true
		}
	}
	return false
}
