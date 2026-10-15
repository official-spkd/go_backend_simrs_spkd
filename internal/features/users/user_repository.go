package users

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"gin-quickstart/internal/platform/models"
)

var (
	ErrNotFound      = errors.New("user not found")
	ErrAlreadyExists = errors.New("nip, email, SIP number or STR number is already registered")
)

type Repository interface {
	List(ctx context.Context, q ListQuery) ([]models.User, int64, error)
	FindByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	Update(ctx context.Context, id uuid.UUID, fields map[string]any) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) List(ctx context.Context, q ListQuery) ([]models.User, int64, error) {
	tx := r.db.WithContext(ctx).Model(&models.User{}).Joins("Role").Preload("Poli")

	if q.Search != "" {
		like := "%" + escapeLike(strings.ToLower(q.Search)) + "%"
		tx = tx.Where(`LOWER(users.name) LIKE ? OR LOWER(users.email) LIKE ? OR users.nip LIKE ?`, like, like, like)
	}
	if q.Role != "" {
		tx = tx.Where(`"Role".name = ?`, q.Role)
	}
	if q.IsActive != nil {
		tx = tx.Where("users.is_active = ?", *q.IsActive)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var out []models.User
	err := tx.Order("users.created_at DESC, users.id").
		Limit(q.Limit).Offset((q.Page - 1) * q.Limit).
		Find(&out).Error
	if err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (r *repository) FindByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var u models.User
	err := r.db.WithContext(ctx).Preload("Role").Preload("Poli").First(&u, "users.id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *repository) Update(ctx context.Context, id uuid.UUID, fields map[string]any) error {
	res := r.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", id).Updates(fields)
	if errors.Is(res.Error, gorm.ErrDuplicatedKey) {
		return ErrAlreadyExists
	}
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Delete(&models.User{}, "id = ?", id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// escapeLike neutralizes LIKE wildcards in user input.
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
