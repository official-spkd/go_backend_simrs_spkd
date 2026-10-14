package visits

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"gin-quickstart/internal/platform/models"
)

var (
	ErrPatientNotFound = models.ErrPatientNotFound
	ErrVisitDateInPast = errors.New("visit_date cannot be in the past")

	ErrResponsiblePersonRequired = errors.New("responsible_person is required for patients under 18")
)

// PatientFinder is satisfied by the patients feature's service.
type PatientFinder interface {
	Get(ctx context.Context, id uuid.UUID) (*models.Patient, error)
}

type Service interface {
	Create(ctx context.Context, req CreateRequest) (*models.Visit, error)
	Get(ctx context.Context, id uuid.UUID) (*models.Visit, error)
	List(ctx context.Context, q ListQuery) ([]Response, int64, error)
}

type service struct {
	repo     Repository
	patients PatientFinder
}

func NewService(repo Repository, patients PatientFinder) Service {
	return &service{repo: repo, patients: patients}
}

func (s *service) Create(ctx context.Context, req CreateRequest) (*models.Visit, error) {
	visitDate, err := time.Parse(dateLayout, req.VisitDate)
	if err != nil {
		return nil, err
	}
	if visitDate.Before(today()) {
		return nil, ErrVisitDateInPast
	}

	patientID, err := uuid.Parse(req.PatientID)
	if err != nil {
		return nil, err
	}
	patient, err := s.patients.Get(ctx, patientID)
	if err != nil {
		return nil, err
	}

	responsible := strings.TrimSpace(req.ResponsiblePerson)
	if responsible == "" && ageOn(patient.BirthDate, visitDate) < adultAge {
		return nil, ErrResponsiblePersonRequired
	}

	allergies := strings.TrimSpace(req.Allergies)
	if allergies == "" {
		allergies = patient.Allergies
	}

	v := &models.Visit{
		PatientID:         patient.ID,
		Patient:           *patient,
		VisitDate:         visitDate,
		Poli:              strings.TrimSpace(req.Poli),
		Doctor:            strings.TrimSpace(req.Doctor),
		Penjamin:          req.Penjamin,
		VisitType:         req.VisitType,
		VisitStatus:       models.VisitRegistered,
		QueueStatus:       models.QueueWaiting,
		SatuSehatStatus:   models.SatuSehatNotSent,
		ResponsiblePerson: responsible,
		ChiefComplaint:    strings.TrimSpace(req.ChiefComplaint),
		Allergies:         allergies,
		IsApproved:        req.IsApproved != nil && *req.IsApproved,
	}
	applyPenjaminDetails(v, req)
	if err := s.repo.Create(ctx, v); err != nil {
		return nil, err
	}
	return v, nil
}

func (s *service) Get(ctx context.Context, id uuid.UUID) (*models.Visit, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) List(ctx context.Context, q ListQuery) ([]Response, int64, error) {
	q.Search = strings.TrimSpace(q.Search)
	q.Poli = strings.TrimSpace(q.Poli)

	found, total, err := s.repo.List(ctx, q)
	if err != nil {
		return nil, 0, err
	}

	data := make([]Response, 0, len(found))
	for i := range found {
		data = append(data, toResponse(&found[i]))
	}
	return data, total, nil
}

// today is the current calendar date, as a UTC midnight to compare with parsed dates.
func today() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

// applyPenjaminDetails copies only the detail fields that belong to the chosen penjamin.
func applyPenjaminDetails(v *models.Visit, req CreateRequest) {
	switch req.Penjamin {
	case models.PenjaminBPJS:
		v.BPJSCardNumber = strings.TrimSpace(req.BPJSCardNumber)
		v.FKTPReferralNumber = strings.TrimSpace(req.FKTPReferralNumber)
		v.ControlLetterNumber = strings.TrimSpace(req.ControlLetterNumber)
		v.DocumentType = strings.TrimSpace(req.DocumentType)
	case models.PenjaminAsuransi:
		v.InsuranceName = strings.TrimSpace(req.InsuranceName)
		v.PolicyNumber = strings.TrimSpace(req.PolicyNumber)
		v.CoverageLimit = req.CoverageLimit
	}
}

const adultAge = 18

// ageOn returns the patient's age in whole years on the given date.
func ageOn(birthDate, on time.Time) int {
	age := on.Year() - birthDate.Year()
	if on.Month() < birthDate.Month() || (on.Month() == birthDate.Month() && on.Day() < birthDate.Day()) {
		age--
	}
	return age
}
