package examinations

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"gin-quickstart/internal/platform/models"
)

var (
	ErrVitalNotFound   = errors.New("vital signs have not been recorded for this visit")
	ErrVersionNotFound = errors.New("vital signs version not found")
	ErrSoapNotFound    = errors.New("soap note has not been started for this visit")
	ErrVersionConflict = errors.New("the record was changed by someone else; reload and try again")
	ErrFinalized       = errors.New("the soap note is finalized and can no longer be changed")
	ErrNotFinalized    = errors.New("addenda can only be added to a finalized soap note")
)

type Repository interface {
	FindVital(ctx context.Context, visitID uuid.UUID) (*models.VitalAssessment, error)
	// SaveVital creates the record when expectedVersion is nil, otherwise updates it only if the
	// stored version still equals *expectedVersion. It also appends entry to the version history;
	// baseline (the state being replaced) is added first if that record has no history yet.
	SaveVital(ctx context.Context, v *models.VitalAssessment, expectedVersion *int, entry models.VitalVersion, baseline *models.VitalVersion) error
	ListVitalVersions(ctx context.Context, vitalID uuid.UUID) ([]models.VitalVersion, error)
	FindVitalVersion(ctx context.Context, vitalID uuid.UUID, version int) (*models.VitalVersion, error)

	FindSoap(ctx context.Context, visitID uuid.UUID) (*models.SoapNote, error)
	// SaveSoap saves a draft with its diagnoses and moves a registered visit to Served.
	SaveSoap(ctx context.Context, n *models.SoapNote, expectedVersion *int) error
	// FinalizeSoap locks the note and moves the visit to Completed.
	FinalizeSoap(ctx context.Context, noteID uuid.UUID, snapshot string, userID uuid.UUID) error
	AddAddendum(ctx context.Context, a *models.SoapAddendum) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) FindVital(ctx context.Context, visitID uuid.UUID) (*models.VitalAssessment, error) {
	var v models.VitalAssessment
	err := r.db.WithContext(ctx).First(&v, "visit_id = ?", visitID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrVitalNotFound
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *repository) SaveVital(ctx context.Context, v *models.VitalAssessment, expectedVersion *int, entry models.VitalVersion, baseline *models.VitalVersion) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if expectedVersion == nil {
			err := tx.Omit(clause.Associations).Create(v).Error
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return ErrVersionConflict
			}
			if err != nil {
				return err
			}
		} else {
			res := tx.Model(&models.VitalAssessment{}).
				Where("id = ? AND version = ?", v.ID, *expectedVersion).
				Select("*").Omit("id", "visit_id", "created_at", clause.Associations).
				Updates(v)
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return ErrVersionConflict
			}
		}

		if baseline != nil {
			var existing int64
			if err := tx.Model(&models.VitalVersion{}).Where("vital_assessment_id = ?", v.ID).Count(&existing).Error; err != nil {
				return err
			}
			if existing == 0 {
				baseline.VitalAssessmentID = v.ID
				if err := tx.Create(baseline).Error; err != nil {
					return err
				}
			}
		}

		entry.VitalAssessmentID = v.ID
		return tx.Create(&entry).Error
	})
}

func (r *repository) ListVitalVersions(ctx context.Context, vitalID uuid.UUID) ([]models.VitalVersion, error) {
	var out []models.VitalVersion
	err := r.db.WithContext(ctx).Where("vital_assessment_id = ?", vitalID).Order("version DESC").Find(&out).Error
	return out, err
}

func (r *repository) FindVitalVersion(ctx context.Context, vitalID uuid.UUID, version int) (*models.VitalVersion, error) {
	var vv models.VitalVersion
	err := r.db.WithContext(ctx).First(&vv, "vital_assessment_id = ? AND version = ?", vitalID, version).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrVersionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &vv, nil
}

func (r *repository) FindSoap(ctx context.Context, visitID uuid.UUID) (*models.SoapNote, error) {
	var n models.SoapNote
	err := r.db.WithContext(ctx).
		Preload("Diagnoses", func(db *gorm.DB) *gorm.DB { return db.Order("sort_order") }).
		Preload("Addenda", func(db *gorm.DB) *gorm.DB { return db.Order("created_at") }).
		First(&n, "visit_id = ?", visitID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrSoapNotFound
	}
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *repository) SaveSoap(ctx context.Context, n *models.SoapNote, expectedVersion *int) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if expectedVersion == nil {
			err := tx.Omit(clause.Associations).Create(n).Error
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return ErrVersionConflict
			}
			if err != nil {
				return err
			}
		} else {
			res := tx.Model(&models.SoapNote{}).
				Where("id = ? AND version = ? AND status = ?", n.ID, *expectedVersion, models.SoapDraft).
				Select("*").Omit("id", "visit_id", "created_at", clause.Associations).
				Updates(n)
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return ErrVersionConflict
			}
			if err := tx.Delete(&models.SoapDiagnosis{}, "soap_note_id = ?", n.ID).Error; err != nil {
				return err
			}
		}

		for i := range n.Diagnoses {
			n.Diagnoses[i].SoapNoteID = n.ID
			n.Diagnoses[i].SortOrder = i
		}
		if len(n.Diagnoses) > 0 {
			if err := tx.Create(&n.Diagnoses).Error; err != nil {
				return err
			}
		}

		return tx.Model(&models.Visit{}).
			Where("id = ? AND visit_status = ?", n.VisitID, models.VisitRegistered).
			Update("visit_status", models.VisitServed).Error
	})
}

func (r *repository) FinalizeSoap(ctx context.Context, noteID uuid.UUID, snapshot string, userID uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var n models.SoapNote
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&n, "id = ?", noteID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSoapNotFound
		}
		if err != nil {
			return err
		}
		if n.Status == models.SoapFinal {
			return ErrFinalized
		}

		now := time.Now()
		err = tx.Model(&models.SoapNote{}).Where("id = ?", noteID).Updates(map[string]any{
			"status":                    models.SoapFinal,
			"objective_vitals_snapshot": snapshot,
			"finalized_at":              now,
			"finalized_by_user_id":      userID,
			"version":                   n.Version + 1,
		}).Error
		if err != nil {
			return err
		}

		return tx.Model(&models.Visit{}).
			Where("id = ? AND visit_status IN ?", n.VisitID, []models.VisitStatus{models.VisitRegistered, models.VisitServed}).
			Update("visit_status", models.VisitCompleted).Error
	})
}

func (r *repository) AddAddendum(ctx context.Context, a *models.SoapAddendum) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var n models.SoapNote
		err := tx.Clauses(clause.Locking{Strength: "SHARE"}).Select("id", "status").First(&n, "id = ?", a.SoapNoteID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSoapNotFound
		}
		if err != nil {
			return err
		}
		if n.Status != models.SoapFinal {
			return ErrNotFinalized
		}
		return tx.Create(a).Error
	})
}
