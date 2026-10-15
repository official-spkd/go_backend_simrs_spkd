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
		lockKey := v.VisitDate.Format(dateLayout) + "|" + v.PoliID.String()
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", lockKey).Error; err != nil {
			return err
		}

		var duplicates int64
		err := tx.Model(&models.Visit{}).
			Where("patient_id = ? AND visit_date = ? AND poli_id = ? AND visit_status <> ?",
				v.PatientID, v.VisitDate, v.PoliID, models.VisitCancelled).
			Count(&duplicates).Error
		if err != nil {
			return err
		}
		if duplicates > 0 {
			return ErrAlreadyRegistered
		}

		var last int
		err = tx.Model(&models.Visit{}).
			Where("visit_date = ? AND poli_id = ?", v.VisitDate, v.PoliID).
			Select("COALESCE(MAX(queue_sequence), 0)").Scan(&last).Error
		if err != nil {
			return err
		}

		v.QueueSequence = last + 1
		v.QueueNumber = fmt.Sprintf("%03d", v.QueueSequence)
		if err := tx.Create(v).Error; err != nil {
			return err
		}
		return enqueue(tx, v)
	})
}

// enqueue puts the visit in the registration desk line with the next number of its date.
// The advisory lock is always taken after the clinic lock above, so lock order never varies.
func enqueue(tx *gorm.DB, v *models.Visit) error {
	if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", "queue|"+v.VisitDate.Format(dateLayout)).Error; err != nil {
		return err
	}

	var last int
	err := tx.Model(&models.Queue{}).Where("queue_date = ?", v.VisitDate).
		Select("COALESCE(MAX(queue_sequence), 0)").Scan(&last).Error
	if err != nil {
		return err
	}

	return tx.Create(&models.Queue{
		VisitID:       v.ID,
		QueueDate:     v.VisitDate,
		QueueSequence: last + 1,
		QueueNumber:   fmt.Sprintf("A%03d", last+1),
		Status:        models.TicketWaiting,
	}).Error
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
	if q.DoctorID != "" {
		tx = tx.Where("doctor_id = ?", q.DoctorID)
	}
	if q.PoliID != "" {
		tx = tx.Where("poli_id = ?", q.PoliID)
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
		Order("visit_date DESC, poli_id, queue_sequence").
		Limit(q.Limit).Offset((q.Page - 1) * q.Limit).
		Find(&out).Error
	if err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

// withPatient preloads the patient, including soft-deleted ones, so history stays readable.
func (r *repository) withPatient(tx *gorm.DB) *gorm.DB {
	return tx.Preload("Poli").Preload("Doctor").Preload("Patient", func(db *gorm.DB) *gorm.DB { return db.Unscoped() })
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
