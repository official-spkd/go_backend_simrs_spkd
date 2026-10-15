package doctors

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"gin-quickstart/internal/platform/models"
)

var (
	ErrPoliNotFound = models.ErrPoliNotFound
	ErrPoliInactive = errors.New("poli is not active")
)

// PoliFinder is satisfied by the polis feature's service.
type PoliFinder interface {
	Get(ctx context.Context, id uuid.UUID) (*models.Poli, error)
}

type Service interface {
	Create(ctx context.Context, req DoctorRequest) (*models.Doctor, error)
	Get(ctx context.Context, id uuid.UUID) (*models.Doctor, error)
	List(ctx context.Context, q ListQuery) ([]Response, int64, error)
	Update(ctx context.Context, id uuid.UUID, req DoctorRequest) (*models.Doctor, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type service struct {
	repo  Repository
	polis PoliFinder
}

func NewService(repo Repository, polis PoliFinder) Service {
	return &service{repo: repo, polis: polis}
}

func (s *service) Create(ctx context.Context, req DoctorRequest) (*models.Doctor, error) {
	poli, err := s.resolvePoli(ctx, req.PoliID, nil)
	if err != nil {
		return nil, err
	}

	d := &models.Doctor{IsActive: true}
	apply(d, req, poli)
	if err := s.repo.Create(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *service) Get(ctx context.Context, id uuid.UUID) (*models.Doctor, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) List(ctx context.Context, q ListQuery) ([]Response, int64, error) {
	q.Search = strings.TrimSpace(q.Search)
	q.Specialization = strings.TrimSpace(q.Specialization)

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

func (s *service) Update(ctx context.Context, id uuid.UUID, req DoctorRequest) (*models.Doctor, error) {
	d, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	poli, err := s.resolvePoli(ctx, req.PoliID, &d.PoliID)
	if err != nil {
		return nil, err
	}

	apply(d, req, poli)
	if err := s.repo.Update(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// resolvePoli loads the poli and requires it to be active, unless the doctor already works
// in it (current), so editing a doctor of a deactivated poli stays possible.
func (s *service) resolvePoli(ctx context.Context, rawID string, current *uuid.UUID) (*models.Poli, error) {
	id, err := uuid.Parse(rawID)
	if err != nil {
		return nil, ErrPoliNotFound
	}
	poli, err := s.polis.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if !poli.IsActive && (current == nil || *current != poli.ID) {
		return nil, ErrPoliInactive
	}
	return poli, nil
}

func apply(d *models.Doctor, req DoctorRequest, poli *models.Poli) {
	d.Name = strings.TrimSpace(req.Name)
	d.Specialization = strings.TrimSpace(req.Specialization)
	d.PoliID, d.Poli = poli.ID, poli
	d.SIPNumber = strings.TrimSpace(req.SIPNumber)
	d.STRNumber = strings.TrimSpace(req.STRNumber)
	d.IHSNumber = nil
	if ihs := strings.TrimSpace(req.IHSNumber); ihs != "" {
		d.IHSNumber = &ihs
	}
	if req.IsActive != nil {
		d.IsActive = *req.IsActive
	}
}
