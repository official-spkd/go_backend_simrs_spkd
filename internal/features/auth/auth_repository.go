package auth

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"gin-quickstart/internal/platform/models"
)

var (
	ErrUserNotFound  = errors.New("user not found")
	ErrAlreadyExists = errors.New("nip, email or doctor already has an account")
)

type Repository interface {
	Create(ctx context.Context, u *models.User) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	FindByIdentifier(ctx context.Context, identifier string) (*models.User, error)
	Count(ctx context.Context) (int64, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, u *models.User) error {
	err := r.db.WithContext(ctx).Create(u).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrAlreadyExists
	}
	return err
}

func (r *repository) FindByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	return r.findOne(ctx, "users.id = ?", id)
}

// FindByIdentifier matches by email when the identifier contains "@", otherwise by NIP.
func (r *repository) FindByIdentifier(ctx context.Context, identifier string) (*models.User, error) {
	if strings.Contains(identifier, "@") {
		return r.findOne(ctx, "users.email = ?", strings.ToLower(identifier))
	}
	return r.findOne(ctx, "users.nip = ?", identifier)
}

func (r *repository) Count(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&models.User{}).Count(&n).Error
	return n, err
}

func (r *repository) findOne(ctx context.Context, query string, arg any) (*models.User, error) {
	var u models.User
	err := r.db.WithContext(ctx).Preload("Role").Where(query, arg).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}
