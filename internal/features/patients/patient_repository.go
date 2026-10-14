package patients

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"gin-quickstart/internal/platform/models"
)

var (
	ErrNotFound      = errors.New("patient not found")
	ErrAlreadyExists = errors.New("a patient with this NIK is already registered")
)

type Repository interface {
	Create(ctx context.Context, p *models.Patient) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.Patient, error)
	List(ctx context.Context, q ListQuery) ([]models.Patient, int64, error)
	Update(ctx context.Context, p *models.Patient) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, p *models.Patient) error {
	return translate(r.db.WithContext(ctx).Create(p).Error)
}

func (r *repository) FindByID(ctx context.Context, id uuid.UUID) (*models.Patient, error) {
	var p models.Patient
	err := r.db.WithContext(ctx).First(&p, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *repository) List(ctx context.Context, q ListQuery) ([]models.Patient, int64, error) {
	tx := r.db.WithContext(ctx).Model(&models.Patient{})

	if q.Search != "" {
		like := "%" + escapeLike(strings.ToLower(q.Search)) + "%"
		tx = tx.Where("LOWER(full_name) LIKE ? OR nik LIKE ? OR phone LIKE ?", like, like, like)
	}
	if q.Gender != "" {
		tx = tx.Where("gender = ?", q.Gender)
	}
	if q.Payer != "" {
		tx = tx.Where("payer = ?", q.Payer)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var out []models.Patient
	err := tx.Order("created_at DESC, id").
		Limit(q.Limit).Offset((q.Page - 1) * q.Limit).
		Find(&out).Error
	if err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (r *repository) Update(ctx context.Context, p *models.Patient) error {
	return translate(r.db.WithContext(ctx).Save(p).Error)
}

// Delete soft-deletes the patient so medical history is never lost.
func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Delete(&models.Patient{}, "id = ?", id)
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
