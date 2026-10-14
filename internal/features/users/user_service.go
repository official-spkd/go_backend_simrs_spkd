package users

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"gin-quickstart/internal/features/roles"
	"gin-quickstart/internal/platform/models"
)

var (
	ErrInvalidRole  = errors.New("role does not exist")
	ErrSelfAction   = errors.New("you cannot change the role or status of, or delete, your own account")
	ErrNotDoctor    = errors.New("sip_number, str_number and poli_id can only be set for doctors")
	ErrPoliInactive = errors.New("poli is not active")
	ErrPoliNotFound = models.ErrPoliNotFound
)

type RoleFinder interface {
	FindByName(ctx context.Context, name string) (*roles.Role, error)
}

// PoliFinder is satisfied by the polis feature's service.
type PoliFinder interface {
	Get(ctx context.Context, id uuid.UUID) (*models.Poli, error)
}

type Service interface {
	List(ctx context.Context, q ListQuery) ([]Response, int64, error)
	Get(ctx context.Context, id uuid.UUID) (*models.User, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*models.User, error)
	UpdateRole(ctx context.Context, actorID, id uuid.UUID, req UpdateRoleRequest) (*models.User, error)
	UpdateStatus(ctx context.Context, actorID, id uuid.UUID, active bool) (*models.User, error)
	ResetPassword(ctx context.Context, id uuid.UUID, newPassword string) error
	Delete(ctx context.Context, actorID, id uuid.UUID) error
}

type service struct {
	repo  Repository
	roles RoleFinder
	polis PoliFinder
}

func NewService(repo Repository, roles RoleFinder, polis PoliFinder) Service {
	return &service{repo: repo, roles: roles, polis: polis}
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

func (s *service) Get(ctx context.Context, id uuid.UUID) (*models.User, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*models.User, error) {
	fields := map[string]any{
		"name":  strings.TrimSpace(req.Name),
		"nip":   strings.TrimSpace(req.NIP),
		"email": strings.ToLower(strings.TrimSpace(req.Email)),
	}

	if req.SIPNumber != "" || req.STRNumber != "" || req.PoliID != "" {
		current, err := s.repo.FindByID(ctx, id)
		if err != nil {
			return nil, err
		}
		if current.Role.Name != roles.Doctor {
			return nil, ErrNotDoctor
		}
		if req.SIPNumber != "" {
			fields["sip_number"] = strings.TrimSpace(req.SIPNumber)
		}
		if req.STRNumber != "" {
			fields["str_number"] = strings.TrimSpace(req.STRNumber)
		}
		if req.PoliID != "" {
			poli, err := s.activePoli(ctx, req.PoliID)
			if err != nil {
				return nil, err
			}
			fields["poli_id"] = poli.ID
		}
	}

	if err := s.repo.Update(ctx, id, fields); err != nil {
		return nil, err
	}
	return s.repo.FindByID(ctx, id)
}

func (s *service) UpdateRole(ctx context.Context, actorID, id uuid.UUID, req UpdateRoleRequest) (*models.User, error) {
	if actorID == id {
		return nil, ErrSelfAction
	}

	role, err := s.roles.FindByName(ctx, strings.TrimSpace(req.Role))
	if errors.Is(err, roles.ErrNotFound) {
		return nil, ErrInvalidRole
	}
	if err != nil {
		return nil, err
	}

	fields := map[string]any{"role_id": role.ID}
	if role.Name == roles.Doctor {
		poli, err := s.activePoli(ctx, req.PoliID)
		if err != nil {
			return nil, err
		}
		fields["sip_number"] = strings.TrimSpace(req.SIPNumber)
		fields["str_number"] = strings.TrimSpace(req.STRNumber)
		fields["poli_id"] = poli.ID
	} else {
		// Leaving the doctor role clears the doctor-only fields.
		fields["sip_number"], fields["str_number"], fields["poli_id"] = nil, nil, nil
	}

	if err := s.repo.Update(ctx, id, fields); err != nil {
		return nil, err
	}
	return s.repo.FindByID(ctx, id)
}

func (s *service) UpdateStatus(ctx context.Context, actorID, id uuid.UUID, active bool) (*models.User, error) {
	if actorID == id {
		return nil, ErrSelfAction
	}
	if err := s.repo.Update(ctx, id, map[string]any{"is_active": active}); err != nil {
		return nil, err
	}
	return s.repo.FindByID(ctx, id)
}

func (s *service) ResetPassword(ctx context.Context, id uuid.UUID, newPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	return s.repo.Update(ctx, id, map[string]any{"password_hash": string(hash)})
}

func (s *service) Delete(ctx context.Context, actorID, id uuid.UUID) error {
	if actorID == id {
		return ErrSelfAction
	}
	return s.repo.Delete(ctx, id)
}

func (s *service) activePoli(ctx context.Context, rawID string) (*models.Poli, error) {
	id, err := uuid.Parse(rawID)
	if err != nil {
		return nil, ErrPoliNotFound
	}
	poli, err := s.polis.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if !poli.IsActive {
		return nil, ErrPoliInactive
	}
	return poli, nil
}
