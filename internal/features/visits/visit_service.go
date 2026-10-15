package visits

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"gin-quickstart/internal/platform/models"
	"gin-quickstart/internal/platform/utils"
)

var (
	ErrPatientNotFound = models.ErrPatientNotFound
	ErrVisitDateInPast = errors.New("visit_date cannot be in the past")

	ErrDoctorNotFound     = models.ErrDoctorNotFound
	ErrDoctorInactive     = errors.New("doctor is not active")
	ErrDoctorPoliMismatch = errors.New("the doctor does not work in this poli")
	ErrPoliInactive       = errors.New("poli is not active")

	ErrResponsiblePersonRequired = errors.New("responsible_person is required for patients under 18")
)

// PatientFinder is satisfied by the patients feature's service.
type PatientFinder interface {
	Get(ctx context.Context, id uuid.UUID) (*models.Patient, error)
}

// DoctorFinder is satisfied by the doctors feature's service. The returned doctor includes its poli.
type DoctorFinder interface {
	Get(ctx context.Context, id uuid.UUID) (*models.Doctor, error)
}

type Service interface {
	Create(ctx context.Context, req CreateRequest) (*models.Visit, error)
	Get(ctx context.Context, id uuid.UUID) (*models.Visit, error)
	List(ctx context.Context, q ListQuery) ([]Response, int64, error)
}

type service struct {
	repo     Repository
	patients PatientFinder
	doctors  DoctorFinder
}

func NewService(repo Repository, patients PatientFinder, doctors DoctorFinder) Service {
	return &service{repo: repo, patients: patients, doctors: doctors}
}

func (s *service) Create(ctx context.Context, req CreateRequest) (*models.Visit, error) {
	visitDate, err := time.Parse(dateLayout, req.VisitDate)
	if err != nil {
		return nil, err
	}
	if visitDate.Before(utils.Today()) {
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

	doctor, err := s.activeDoctor(ctx, req)
	if err != nil {
		return nil, err
	}

	responsible := strings.TrimSpace(req.ResponsiblePerson)
	if responsible == "" && utils.AgeOn(patient.BirthDate, visitDate) < adultAge {
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
		PoliID:            &doctor.PoliID,
		Poli:              doctor.Poli,
		DoctorID:          &doctor.ID,
		Doctor:            doctor,
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

// activeDoctor loads the requested doctor and checks that they can take a visit in the requested poli.
func (s *service) activeDoctor(ctx context.Context, req CreateRequest) (*models.Doctor, error) {
	id, err := uuid.Parse(req.DoctorID)
	if err != nil {
		return nil, ErrDoctorNotFound
	}
	doctor, err := s.doctors.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if !doctor.IsActive {
		return nil, ErrDoctorInactive
	}
	if doctor.PoliID.String() != strings.ToLower(strings.TrimSpace(req.PoliID)) {
		return nil, ErrDoctorPoliMismatch
	}
	if !doctor.Poli.IsActive {
		return nil, ErrPoliInactive
	}
	return doctor, nil
}
