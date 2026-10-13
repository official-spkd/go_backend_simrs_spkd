package roles

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var (
	ErrInvalidName = errors.New("name must be lowercase letters, digits or underscores, and start with a letter")
	ErrSystemRole  = errors.New("built-in roles cannot be deleted")

	namePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
)

type Service interface {
	List(ctx context.Context) ([]Role, error)
	Get(ctx context.Context, id uuid.UUID) (*Role, error)
	Create(ctx context.Context, req CreateRequest) (*Role, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*Role, error)
	Delete(ctx context.Context, id uuid.UUID) error
	SeedDefaults(ctx context.Context) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) List(ctx context.Context) ([]Role, error) {
	return s.repo.List(ctx)
}

func (s *service) Get(ctx context.Context, id uuid.UUID) (*Role, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) Create(ctx context.Context, req CreateRequest) (*Role, error) {
	name := strings.TrimSpace(req.Name)
	if !namePattern.MatchString(name) {
		return nil, ErrInvalidName
	}

	role := &Role{
		Name:        name,
		DisplayName: strings.TrimSpace(req.DisplayName),
		Description: strings.TrimSpace(req.Description),
	}
	if err := s.repo.Create(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}

func (s *service) Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*Role, error) {
	role, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	role.DisplayName = strings.TrimSpace(req.DisplayName)
	role.Description = strings.TrimSpace(req.Description)
	if err := s.repo.Update(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	role, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if role.IsSystem {
		return ErrSystemRole
	}
	return s.repo.Delete(ctx, id)
}

func (s *service) SeedDefaults(ctx context.Context) error {
	return s.repo.Seed(ctx, Defaults())
}
