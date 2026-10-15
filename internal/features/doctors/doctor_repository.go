package doctors

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrNotFound      = errors.New("doctor not found")
	ErrAlreadyExists = errors.New("sip, str or ihs number is already registered")
)

type Repository interface {
	Create(ctx context.Context, d *Doctor) error
	FindByID(ctx context.Context, id uuid.UUID) (*Doctor, error)
	List(ctx context.Context, q ListQuery) ([]Doctor, int64, error)
	Update(ctx context.Context, d *Doctor) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// Create, Update and Delete refresh doctor_count of the affected polis in the same transaction.

func (r *repository) Create(ctx context.Context, d *Doctor) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit("Poli").Create(d).Error; err != nil {
			return err
		}
		return recountPoliDoctors(tx, d.PoliID)
	})
	return translate(err)
}

func (r *repository) Update(ctx context.Context, d *Doctor) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var old Doctor
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id", "poli_id").First(&old, "id = ?", d.ID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}

		if err := tx.Omit("Poli").Save(d).Error; err != nil {
			return err
		}
		return recountPoliDoctors(tx, old.PoliID, d.PoliID)
	})
	return translate(err)
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var d Doctor
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id", "poli_id").First(&d, "id = ?", id).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if err := tx.Delete(&d).Error; err != nil {
			return err
		}
		return recountPoliDoctors(tx, d.PoliID)
	})
}

func (r *repository) FindByID(ctx context.Context, id uuid.UUID) (*Doctor, error) {
	var d Doctor
	err := r.db.WithContext(ctx).Preload("Poli").First(&d, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *repository) List(ctx context.Context, q ListQuery) ([]Doctor, int64, error) {
	tx := r.db.WithContext(ctx).Model(&Doctor{})

	if q.PoliID != "" {
		tx = tx.Where("poli_id = ?", q.PoliID)
	}
	if q.Specialization != "" {
		tx = tx.Where("LOWER(specialization) = LOWER(?)", q.Specialization)
	}
	if q.IsActive != nil {
		tx = tx.Where("is_active = ?", *q.IsActive)
	}
	if q.Search != "" {
		like := "%" + escapeLike(strings.ToLower(q.Search)) + "%"
		tx = tx.Where("LOWER(name) LIKE ? OR LOWER(sip_number) LIKE ? OR LOWER(str_number) LIKE ? OR ihs_number LIKE ?", like, like, like, like)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var out []Doctor
	err := tx.Preload("Poli").Order("name, id").
		Limit(q.Limit).Offset((q.Page - 1) * q.Limit).Find(&out).Error
	if err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

const recountSQL = `UPDATE polis SET doctor_count =
	(SELECT COUNT(*) FROM doctors WHERE doctors.poli_id = polis.id AND doctors.is_active)`

// recountPoliDoctors recalculates doctor_count of the given polis from the active doctors.
// Call it inside the same transaction as the doctor change.
func recountPoliDoctors(tx *gorm.DB, poliIDs ...uuid.UUID) error {
	if len(poliIDs) == 0 {
		return nil
	}
	return tx.Exec(recountSQL+" WHERE polis.id IN ?", poliIDs).Error
}

// RecountAllPoliDoctors recalculates doctor_count of every poli; used at startup to repair drift.
func RecountAllPoliDoctors(db *gorm.DB) error {
	return db.Exec(recountSQL).Error
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
