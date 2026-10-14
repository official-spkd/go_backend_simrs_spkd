package users

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"gin-quickstart/internal/platform/database"
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

// Update applies the column changes and keeps doctor_count of the affected polis in sync,
// all in one transaction.
func (r *repository) Update(ctx context.Context, id uuid.UUID, fields map[string]any) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		before, err := lockPoliID(tx, id)
		if err != nil {
			return err
		}

		err = tx.Model(&models.User{}).Where("id = ?", id).Updates(fields).Error
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrAlreadyExists
		}
		if err != nil {
			return err
		}

		after, err := lockPoliID(tx, id)
		if err != nil {
			return err
		}
		return database.RecountPoliDoctors(tx, poliSet(before, after)...)
	})
}

// Delete removes the user and refreshes doctor_count of the poli they worked in.
func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		poliID, err := lockPoliID(tx, id)
		if err != nil {
			return err
		}
		if err := tx.Delete(&models.User{}, "id = ?", id).Error; err != nil {
			return err
		}
		return database.RecountPoliDoctors(tx, poliSet(poliID)...)
	})
}

// lockPoliID locks the user row and returns the poli they are assigned to (nil if none).
func lockPoliID(tx *gorm.DB, id uuid.UUID) (*uuid.UUID, error) {
	var u models.User
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id", "poli_id").First(&u, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return u.PoliID, nil
}

// poliSet returns the distinct non-nil poli IDs.
func poliSet(ids ...*uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]bool, len(ids))
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if id != nil && !seen[*id] {
			seen[*id] = true
			out = append(out, *id)
		}
	}
	return out
}

// escapeLike neutralizes LIKE wildcards in user input.
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
