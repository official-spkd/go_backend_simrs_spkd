package patients

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"gin-quickstart/internal/platform/models"
)

var ErrInvalidBirthDate = errors.New("birth_date must be between 1900-01-01 and today")

var earliestBirthDate = time.Date(1900, time.January, 1, 0, 0, 0, 0, time.UTC)

type Service interface {
	Create(ctx context.Context, req PatientRequest) (*models.Patient, error)
	Get(ctx context.Context, id uuid.UUID) (*models.Patient, error)
	List(ctx context.Context, q ListQuery) ([]Response, int64, error)
	Update(ctx context.Context, id uuid.UUID, req PatientRequest) (*models.Patient, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, req PatientRequest) (*models.Patient, error) {
	p := &models.Patient{}
	if err := apply(p, req); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *service) Get(ctx context.Context, id uuid.UUID) (*models.Patient, error) {
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

func (s *service) Update(ctx context.Context, id uuid.UUID, req PatientRequest) (*models.Patient, error) {
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := apply(p, req); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// apply copies the normalized request onto the patient.
func apply(p *models.Patient, req PatientRequest) error {
	birthDate, err := time.Parse(dateLayout, req.BirthDate)
	if err != nil {
		return err
	}
	if birthDate.After(time.Now()) || birthDate.Before(earliestBirthDate) {
		return ErrInvalidBirthDate
	}

	p.NIK = strings.TrimSpace(req.NIK)
	p.FullName = strings.TrimSpace(req.FullName)
	p.BirthDate = birthDate
	p.Gender = req.Gender
	p.BPJSNumber = nil
	if bpjs := strings.TrimSpace(req.BPJSNumber); bpjs != "" {
		p.BPJSNumber = &bpjs
	}
	p.BirthPlace = strings.TrimSpace(req.BirthPlace)
	p.Religion = strings.TrimSpace(req.Religion)
	p.Education = strings.TrimSpace(req.Education)
	p.Occupation = strings.TrimSpace(req.Occupation)
	p.MaritalStatus = req.MaritalStatus
	p.BloodType = req.BloodType
	p.Allergies = strings.TrimSpace(req.Allergies)
	p.Phone = req.Phone
	p.Email = strings.ToLower(strings.TrimSpace(req.Email))
	p.IDCardAddress = strings.TrimSpace(req.IDCardAddress)
	p.DomicileAddress = strings.TrimSpace(req.DomicileAddress)
	p.EmergencyContactName = strings.TrimSpace(req.EmergencyContactName)
	p.EmergencyContactRelationship = strings.TrimSpace(req.EmergencyContactRelationship)
	p.EmergencyContactPhone = req.EmergencyContactPhone
	p.Payer = req.Payer
	return nil
}
