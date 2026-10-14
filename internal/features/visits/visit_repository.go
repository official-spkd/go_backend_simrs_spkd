package visits

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"gin-quickstart/internal/platform/models"
)

var (
	ErrNotFound          = errors.New("visit not found")
	ErrAlreadyRegistered = errors.New("patient is already registered for this clinic on this date")
)

type Repository interface {
	// Create assigns the next queue number for the visit's clinic and date, then stores the visit.
	Create(ctx context.Context, v *models.Visit) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.Visit, error)
	List(ctx context.Context, q ListQuery) ([]models.Visit, int64, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, v *models.Visit) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Serialize registrations per clinic and date so queue numbers are gap-free and unique.
		lockKey := v.VisitDate.Format(dateLayout) + "|" + strings.ToLower(v.Poli)
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", lockKey).Error; err != nil {
			return err
		}

		var duplicates int64
		err := tx.Model(&models.Visit{}).
			Where("patient_id = ? AND visit_date = ? AND LOWER(poli) = LOWER(?) AND visit_status <> ?",
				v.PatientID, v.VisitDate, v.Poli, models.VisitCancelled).
			Count(&duplicates).Error
		if err != nil {
			return err
		}
		if duplicates > 0 {
			return ErrAlreadyRegistered
		}

		var last int
		err = tx.Model(&models.Visit{}).
			Where("visit_date = ? AND LOWER(poli) = LOWER(?)", v.VisitDate, v.Poli).
			Select("COALESCE(MAX(queue_sequence), 0)").Scan(&last).Error
		if err != nil {
			return err
		}

		v.QueueSequence = last + 1
		v.QueueNumber = fmt.Sprintf("%03d", v.QueueSequence)
		return tx.Create(v).Error
	})
}

func (r *repository) FindByID(ctx context.Context, id uuid.UUID) (*models.Visit, error) {
	var v models.Visit
	err := r.withPatient(r.db.WithContext(ctx)).First(&v, "visits.id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *repository) List(ctx context.Context, q ListQuery) ([]models.Visit, int64, error) {
	tx := r.db.WithContext(ctx).Model(&models.Visit{})

	if q.Search != "" {
		like := "%" + escapeLike(strings.ToLower(q.Search)) + "%"
		// Subquery on the table name skips the soft-delete filter, so visits of deleted patients stay searchable.
		patients := r.db.Table("patients").Select("id").
			Where("LOWER(full_name) LIKE ? OR nik LIKE ? OR LOWER(medical_record_number) LIKE ?", like, like, like)
		tx = tx.Where("patient_id IN (?)", patients)
	}
	if q.Date != nil {
		tx = tx.Where("visit_date = ?", *q.Date)
	}
	if q.Poli != "" {
		tx = tx.Where("LOWER(poli) = LOWER(?)", q.Poli)
	}
	if q.Penjamin != "" {
		tx = tx.Where("payer = ?", q.Penjamin)
	}
	if q.VisitType != "" {
		tx = tx.Where("visit_type = ?", q.VisitType)
	}
	if q.VisitStatus != "" {
		tx = tx.Where("visit_status = ?", q.VisitStatus)
	}
	if q.QueueStatus != "" {
		tx = tx.Where("queue_status = ?", q.QueueStatus)
	}
	if q.SatuSehatStatus != "" {
		tx = tx.Where("satu_sehat_status = ?", q.SatuSehatStatus)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var out []models.Visit
	err := r.withPatient(tx).
		Order("visit_date DESC, poli, queue_sequence").
		Limit(q.Limit).Offset((q.Page - 1) * q.Limit).
		Find(&out).Error
	if err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

// withPatient preloads the patient, including soft-deleted ones, so history stays readable.
func (r *repository) withPatient(tx *gorm.DB) *gorm.DB {
	return tx.Preload("Patient", func(db *gorm.DB) *gorm.DB { return db.Unscoped() })
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
