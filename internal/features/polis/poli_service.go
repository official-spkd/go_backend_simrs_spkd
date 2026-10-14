package polis

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"gin-quickstart/internal/platform/models"
)

type Service interface {
	Create(ctx context.Context, req PoliRequest) (*models.Poli, error)
	Get(ctx context.Context, id uuid.UUID) (*models.Poli, error)
	List(ctx context.Context, q ListQuery) ([]Response, int64, error)
	Update(ctx context.Context, id uuid.UUID, req PoliRequest) (*models.Poli, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, req PoliRequest) (*models.Poli, error) {
	p := &models.Poli{IsActive: true}
	apply(p, req)
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *service) Get(ctx context.Context, id uuid.UUID) (*models.Poli, error) {
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

func (s *service) Update(ctx context.Context, id uuid.UUID, req PoliRequest) (*models.Poli, error) {
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	apply(p, req)
	if err := s.repo.Update(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func apply(p *models.Poli, req PoliRequest) {
	p.PrefixCode = strings.ToUpper(strings.TrimSpace(req.PrefixCode))
	p.Name = strings.TrimSpace(req.Name)
	p.Room = strings.TrimSpace(req.Room)
	p.DoctorCount = *req.DoctorCount
	if req.IsActive != nil {
		p.IsActive = *req.IsActive
	}
}
