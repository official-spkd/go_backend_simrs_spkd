package roles

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrNotFound      = errors.New("role not found")
	ErrAlreadyExists = errors.New("role name already exists")
	ErrInUse         = errors.New("role is still assigned to users")
)

type Repository interface {
	List(ctx context.Context) ([]Role, error)
	FindByID(ctx context.Context, id uuid.UUID) (*Role, error)
	Create(ctx context.Context, r *Role) error
	Update(ctx context.Context, r *Role) error
	Delete(ctx context.Context, id uuid.UUID) error
	// Seed inserts the given roles. Existing roles keep their edited fields, but are
	// always marked as built-in.
	Seed(ctx context.Context, roles []Role) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) List(ctx context.Context) ([]Role, error) {
	var out []Role
	if err := r.db.WithContext(ctx).Order("created_at, name").Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *repository) FindByID(ctx context.Context, id uuid.UUID) (*Role, error) {
	var role Role
	err := r.db.WithContext(ctx).First(&role, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *repository) Create(ctx context.Context, role *Role) error {
	err := r.db.WithContext(ctx).Create(role).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrAlreadyExists
	}
	return err
}

func (r *repository) Update(ctx context.Context, role *Role) error {
	return r.db.WithContext(ctx).Save(role).Error
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Delete(&Role{}, "id = ?", id)
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

func (r *repository) Seed(ctx context.Context, roles []Role) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "name"}},
			DoUpdates: clause.AssignmentColumns([]string{"is_system"}),
		}).
		Create(&roles).Error
}
