package schedules

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"gin-quickstart/internal/platform/models"
)

var (
	ErrNotFound          = errors.New("schedule not found")
	ErrExceptionNotFound = errors.New("schedule exception not found")
	ErrAlreadyExists     = errors.New("doctor already has a schedule for this weekday")
	ErrExceptionExists   = errors.New("doctor already has an exception on this date")
)

// FilledKey identifies the registered visits of one doctor on one date (YYYY-MM-DD).
type FilledKey struct {
	DoctorID uuid.UUID
	Date     string
}

type Repository interface {
	Create(ctx context.Context, s *models.DoctorSchedule) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.DoctorSchedule, error)
	List(ctx context.Context, q ListQuery) ([]models.DoctorSchedule, int64, error)
	ListByDoctors(ctx context.Context, doctorIDs []uuid.UUID) ([]models.DoctorSchedule, error)
	Update(ctx context.Context, s *models.DoctorSchedule) error
	Delete(ctx context.Context, id uuid.UUID) error
	// FilledCounts counts the non-cancelled visits per doctor and date; it is the quota usage.
	FilledCounts(ctx context.Context, doctorIDs []uuid.UUID, from, to time.Time) (map[FilledKey]int, error)

	CreateException(ctx context.Context, e *models.ScheduleException) error
	FindException(ctx context.Context, id uuid.UUID) (*models.ScheduleException, error)
	ListExceptions(ctx context.Context, q ExceptionQuery) ([]models.ScheduleException, int64, error)
	ExceptionsInRange(ctx context.Context, doctorIDs []uuid.UUID, from, to time.Time) ([]models.ScheduleException, error)
	DeleteException(ctx context.Context, id uuid.UUID) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// Monday first, as the weekly template is shown.
const weekdayOrder = "CASE WHEN weekday = 0 THEN 7 ELSE weekday END"

func (r *repository) Create(ctx context.Context, s *models.DoctorSchedule) error {
	err := r.db.WithContext(ctx).Omit("Doctor").Create(s).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrAlreadyExists
	}
	return err
}

func (r *repository) FindByID(ctx context.Context, id uuid.UUID) (*models.DoctorSchedule, error) {
	var s models.DoctorSchedule
	err := r.db.WithContext(ctx).Preload("Doctor.Poli").First(&s, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *repository) List(ctx context.Context, q ListQuery) ([]models.DoctorSchedule, int64, error) {
	tx := r.db.WithContext(ctx).Model(&models.DoctorSchedule{})

	if q.DoctorID != "" {
		tx = tx.Where("doctor_id = ?", q.DoctorID)
	}
	if q.PoliID != "" {
		tx = tx.Where("doctor_id IN (SELECT id FROM doctors WHERE poli_id = ?)", q.PoliID)
	}
	if q.Weekday != nil {
		tx = tx.Where("weekday = ?", *q.Weekday)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var out []models.DoctorSchedule
	err := tx.Preload("Doctor.Poli").Order(weekdayOrder + ", doctor_id").
		Limit(q.Limit).Offset((q.Page - 1) * q.Limit).Find(&out).Error
	if err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (r *repository) ListByDoctors(ctx context.Context, doctorIDs []uuid.UUID) ([]models.DoctorSchedule, error) {
	var out []models.DoctorSchedule
	err := r.db.WithContext(ctx).Preload("Doctor.Poli").Where("doctor_id IN ?", doctorIDs).Find(&out).Error
	return out, err
}

func (r *repository) Update(ctx context.Context, s *models.DoctorSchedule) error {
	return r.db.WithContext(ctx).Omit("Doctor").Save(s).Error
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Delete(&models.DoctorSchedule{}, "id = ?", id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *repository) FilledCounts(ctx context.Context, doctorIDs []uuid.UUID, from, to time.Time) (map[FilledKey]int, error) {
	var rows []struct {
		DoctorID  uuid.UUID
		VisitDate time.Time
		Filled    int
	}
	err := r.db.WithContext(ctx).Model(&models.Visit{}).
		Select("doctor_id, visit_date, COUNT(*) AS filled").
		Where("doctor_id IN ?", doctorIDs).
		Where("visit_date BETWEEN ? AND ?", from, to).
		Where("visit_status <> ?", models.VisitCancelled).
		Group("doctor_id, visit_date").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	out := make(map[FilledKey]int, len(rows))
	for _, row := range rows {
		out[FilledKey{DoctorID: row.DoctorID, Date: row.VisitDate.Format(dateLayout)}] = row.Filled
	}
	return out, nil
}

func (r *repository) CreateException(ctx context.Context, e *models.ScheduleException) error {
	err := r.db.WithContext(ctx).Omit("Doctor", "SubstituteDoctor").Create(e).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrExceptionExists
	}
	return err
}

func (r *repository) FindException(ctx context.Context, id uuid.UUID) (*models.ScheduleException, error) {
	var e models.ScheduleException
	err := r.db.WithContext(ctx).Preload("Doctor.Poli").Preload("SubstituteDoctor.Poli").First(&e, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrExceptionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *repository) ListExceptions(ctx context.Context, q ExceptionQuery) ([]models.ScheduleException, int64, error) {
	tx := r.db.WithContext(ctx).Model(&models.ScheduleException{})

	if q.DoctorID != "" {
		tx = tx.Where("doctor_id = ?", q.DoctorID)
	}
	if q.From != nil {
		tx = tx.Where("date >= ?", *q.From)
	}
	if q.To != nil {
		tx = tx.Where("date <= ?", *q.To)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var out []models.ScheduleException
	err := tx.Preload("Doctor.Poli").Preload("SubstituteDoctor.Poli").Order("date, id").
		Limit(q.Limit).Offset((q.Page - 1) * q.Limit).Find(&out).Error
	if err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (r *repository) ExceptionsInRange(ctx context.Context, doctorIDs []uuid.UUID, from, to time.Time) ([]models.ScheduleException, error) {
	var out []models.ScheduleException
	err := r.db.WithContext(ctx).Preload("Doctor.Poli").Preload("SubstituteDoctor.Poli").
		Where("doctor_id IN ?", doctorIDs).Where("date BETWEEN ? AND ?", from, to).
		Order("date").Find(&out).Error
	return out, err
}

func (r *repository) DeleteException(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Delete(&models.ScheduleException{}, "id = ?", id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrExceptionNotFound
	}
	return nil
}
