package files

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"gin-quickstart/internal/platform/models"
)

var ErrNotFound = errors.New("file not found")

type Repository interface {
	Create(ctx context.Context, f *models.File) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.File, error)
	ListUploaded(ctx context.Context, resourceType string, resourceID uuid.UUID) ([]models.File, error)
	// CountActive counts the uploaded files plus the uploads still allowed to finish.
	CountActive(ctx context.Context, resourceType string, resourceID uuid.UUID, now time.Time) (int64, error)
	// Transition updates the file only if it is still in the given status, so two requests cannot
	// both finish the same upload. It reports whether the file was updated.
	Transition(ctx context.Context, f *models.File, from string) (bool, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, f *models.File) error {
	return r.db.WithContext(ctx).Create(f).Error
}

func (r *repository) FindByID(ctx context.Context, id uuid.UUID) (*models.File, error) {
	var f models.File
	err := r.db.WithContext(ctx).First(&f, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *repository) ListUploaded(ctx context.Context, resourceType string, resourceID uuid.UUID) ([]models.File, error) {
	var out []models.File
	err := r.db.WithContext(ctx).
		Where("resource_type = ? AND resource_id = ? AND status = ?", resourceType, resourceID, models.FileUploaded).
		Order("created_at").Find(&out).Error
	return out, err
}

func (r *repository) CountActive(ctx context.Context, resourceType string, resourceID uuid.UUID, now time.Time) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&models.File{}).
		Where("resource_type = ? AND resource_id = ?", resourceType, resourceID).
		Where("status = ? OR (status = ? AND upload_expires_at > ?)", models.FileUploaded, models.FilePending, now).
		Count(&n).Error
	return n, err
}

func (r *repository) Transition(ctx context.Context, f *models.File, from string) (bool, error) {
	res := r.db.WithContext(ctx).Model(&models.File{}).
		Where("id = ? AND status = ?", f.ID, from).
		Select("status", "size_bytes", "uploaded_at").Updates(f)
	return res.RowsAffected > 0, res.Error
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.File{}, "id = ?", id).Error
}
