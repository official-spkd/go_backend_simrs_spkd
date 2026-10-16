package apm

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"gin-quickstart/internal/platform/models"
)

// Repository only reads. The kiosk's one write, registering a visit, goes through the visits feature.
type Repository interface {
	FindPatient(ctx context.Context, column, value string) (*models.Patient, error)
	ListPolis(ctx context.Context) ([]models.Poli, error)
	ListDoctors(ctx context.Context, poliID string) ([]models.Doctor, error)
	FindQueueNumber(ctx context.Context, visitID uuid.UUID) (string, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// FindPatient matches one column, which the service restricts to a known set.
func (r *repository) FindPatient(ctx context.Context, column, value string) (*models.Patient, error) {
	var p models.Patient
	err := r.db.WithContext(ctx).Where(column+" = ?", value).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, models.ErrPatientNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *repository) ListPolis(ctx context.Context) ([]models.Poli, error) {
	var out []models.Poli
	err := r.db.WithContext(ctx).Where("is_active = ?", true).Order("name").Find(&out).Error
	return out, err
}

// ListDoctors returns active doctors of active polis.
func (r *repository) ListDoctors(ctx context.Context, poliID string) ([]models.Doctor, error) {
	tx := r.db.WithContext(ctx).
		Joins("JOIN polis ON polis.id = doctors.poli_id AND polis.is_active = ?", true).
		Where("doctors.is_active = ?", true)
	if poliID != "" {
		tx = tx.Where("doctors.poli_id = ?", poliID)
	}

	var out []models.Doctor
	err := tx.Preload("Poli").Order("doctors.name").Find(&out).Error
	return out, err
}

func (r *repository) FindQueueNumber(ctx context.Context, visitID uuid.UUID) (string, error) {
	var q models.Queue
	err := r.db.WithContext(ctx).Select("queue_number").First(&q, "visit_id = ?", visitID).Error
	return q.QueueNumber, err
}
