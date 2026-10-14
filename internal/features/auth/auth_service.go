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
	ErrPoliNotFound       = models.ErrPoliNotFound
	ErrPoliInactive       = errors.New("poli is not active")
)

var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password"), bcrypt.DefaultCost)

type RoleFinder interface {
	FindByName(ctx context.Context, name string) (*roles.Role, error)
}

// PoliFinder is satisfied by the polis feature's service.
type PoliFinder interface {
	Get(ctx context.Context, id uuid.UUID) (*models.Poli, error)
}

type Service interface {
	Login(ctx context.Context, req LoginRequest) (*LoginResponse, error)
	Register(ctx context.Context, req RegisterRequest, actor *utils.TokenClaims) (*models.User, error)
	Me(ctx context.Context, userID uuid.UUID) (*models.User, error)
}

type service struct {
	repo   Repository
	roles  RoleFinder
	polis  PoliFinder
	tokens *utils.TokenManager
}

func NewService(repo Repository, roles RoleFinder, polis PoliFinder, tokens *utils.TokenManager) Service {
	return &service{repo: repo, roles: roles, polis: polis, tokens: tokens}
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

	var profile *doctorFields
	if role.Name == roles.Doctor {
		if profile, err = s.doctorFields(ctx, req); err != nil {
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
	}
	if profile != nil {
		u.SIPNumber, u.STRNumber = &profile.sip, &profile.str
		u.PoliID, u.Poli = &profile.poli.ID, profile.poli
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

type doctorFields struct {
	sip, str string
	poli     *models.Poli
}

// doctorFields validates the doctor-only fields and resolves the poli they are assigned to.
func (s *service) doctorFields(ctx context.Context, req RegisterRequest) (*doctorFields, error) {
	poliID, err := uuid.Parse(req.PoliID)
	if err != nil {
		return nil, ErrPoliNotFound
	}
	poli, err := s.polis.Get(ctx, poliID)
	if err != nil {
		return nil, err
	}
	if !poli.IsActive {
		return nil, ErrPoliInactive
	}
	return &doctorFields{
		sip:  strings.TrimSpace(req.SIPNumber),
		str:  strings.TrimSpace(req.STRNumber),
		poli: poli,
	}, nil
}
