package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"gin-quickstart/internal/features/roles"
	"gin-quickstart/internal/platform/models"
	"gin-quickstart/internal/platform/utils"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInactiveAccount    = errors.New("account is deactivated")
	ErrUnauthenticated    = errors.New("authentication required")
	ErrForbidden          = errors.New("only superadmin can register new users")
	ErrRoleRequired       = errors.New("role is required")
	ErrInvalidRole        = errors.New("role does not exist")
	ErrDoctorNotFound     = models.ErrDoctorNotFound
	ErrDoctorInactive     = errors.New("doctor is not active")
)

var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password"), bcrypt.DefaultCost)

type RoleFinder interface {
	FindByName(ctx context.Context, name string) (*roles.Role, error)
}

// DoctorFinder is satisfied by the doctors feature's service.
type DoctorFinder interface {
	Get(ctx context.Context, id uuid.UUID) (*models.Doctor, error)
}

type Service interface {
	Login(ctx context.Context, req LoginRequest) (*LoginResponse, error)
	Register(ctx context.Context, req RegisterRequest, actor *utils.TokenClaims) (*models.User, error)
	Me(ctx context.Context, userID uuid.UUID) (*models.User, error)
}

type service struct {
	repo    Repository
	roles   RoleFinder
	doctors DoctorFinder
	tokens  *utils.TokenManager
}

func NewService(repo Repository, roles RoleFinder, doctors DoctorFinder, tokens *utils.TokenManager) Service {
	return &service{repo: repo, roles: roles, doctors: doctors, tokens: tokens}
}

func (s *service) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	u, err := s.repo.FindByIdentifier(ctx, strings.TrimSpace(req.Identifier))
	if errors.Is(err, ErrUserNotFound) {
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(req.Password))
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	if !u.IsActive {
		return nil, ErrInactiveAccount
	}

	accessToken, expiresAt, err := s.tokens.Generate(u.ID, u.Role.Name)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(time.Until(expiresAt).Seconds()),
		User:        toUserResponse(u),
	}, nil
}

// Register creates a user. The first account in the system is open to anyone and becomes
// superadmin; after that only an authenticated superadmin can register new accounts.
func (s *service) Register(ctx context.Context, req RegisterRequest, actor *utils.TokenClaims) (*models.User, error) {
	count, err := s.repo.Count(ctx)
	if err != nil {
		return nil, err
	}

	roleName := strings.TrimSpace(req.Role)
	if count == 0 {
		roleName = roles.Superadmin
	} else {
		if actor == nil {
			return nil, ErrUnauthenticated
		}
		if actor.Role != roles.Superadmin {
			return nil, ErrForbidden
		}
		if roleName == "" {
			return nil, ErrRoleRequired
		}
	}

	role, err := s.roles.FindByName(ctx, roleName)
	if errors.Is(err, roles.ErrNotFound) {
		return nil, ErrInvalidRole
	}
	if err != nil {
		return nil, err
	}

	var doctorID *uuid.UUID
	if role.Name == roles.Doctor {
		if doctorID, err = s.activeDoctorID(ctx, req.DoctorID); err != nil {
			return nil, err
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	u := &models.User{
		Name:         strings.TrimSpace(req.Name),
		NIP:          strings.TrimSpace(req.NIP),
		Email:        strings.ToLower(strings.TrimSpace(req.Email)),
		PasswordHash: string(hash),
		RoleID:       role.ID,
		Role:         *role,
		IsActive:     true,
		DoctorID:     doctorID,
	}
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *service) Me(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	u, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !u.IsActive {
		return nil, ErrInactiveAccount
	}
	return u, nil
}

// activeDoctorID checks that the doctor record exists and is active before an account is made for it.
func (s *service) activeDoctorID(ctx context.Context, rawID string) (*uuid.UUID, error) {
	id, err := uuid.Parse(rawID)
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
	return &doctor.ID, nil
}
