package polis

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"gin-quickstart/internal/platform/models"
)

var (
	ErrNotFound      = models.ErrPoliNotFound
	ErrInUse         = errors.New("poli is still referenced by doctors or visits")
	ErrAlreadyExists = errors.New("a poli with this prefix code or name already exists")
)

type Repository interface {
	Create(ctx context.Context, p *models.Poli) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.Poli, error)
	List(ctx context.Context, q ListQuery) ([]models.Poli, int64, error)
	Update(ctx context.Context, p *models.Poli) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, p *models.Poli) error {
	return translate(r.db.WithContext(ctx).Create(p).Error)
}

func (r *repository) FindByID(ctx context.Context, id uuid.UUID) (*models.Poli, error) {
	var p models.Poli
	err := r.db.WithContext(ctx).First(&p, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *repository) List(ctx context.Context, q ListQuery) ([]models.Poli, int64, error) {
	tx := r.db.WithContext(ctx).Model(&models.Poli{})
	if q.Search != "" {
		like := "%" + escapeLike(strings.ToLower(q.Search)) + "%"
		tx = tx.Where("LOWER(name) LIKE ? OR LOWER(prefix_code) LIKE ? OR LOWER(room) LIKE ?", like, like, like)
	}
	if q.IsActive != nil {
		tx = tx.Where("is_active = ?", *q.IsActive)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var out []models.Poli
	err := tx.Order("name").Limit(q.Limit).Offset((q.Page - 1) * q.Limit).Find(&out).Error
	if err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (r *repository) Update(ctx context.Context, p *models.Poli) error {
	return translate(r.db.WithContext(ctx).Omit("DoctorCount").Save(p).Error)
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Delete(&models.Poli{}, "id = ?", id)
	if errors.Is(res.Error, gorm.ErrForeignKeyViolated) {
		return ErrInUse
	}
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func translate(err error) error {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrAlreadyExists
	}
	return err
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
