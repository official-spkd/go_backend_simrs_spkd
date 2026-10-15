package queues

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"gin-quickstart/internal/platform/models"
)

var (
	ErrCounterNotFound = errors.New("counter not found")
	ErrQueueNotFound   = errors.New("queue not found")
	ErrNoWaiting       = errors.New("there are no waiting queues for today")
	ErrCounterBusy     = errors.New("counter already has an active queue; complete or skip it first")
	ErrNoActiveQueue   = errors.New("counter has no active queue")
	ErrInvalidState    = errors.New("action is not allowed for the current queue status")
)

var activeStatuses = []models.TicketStatus{models.TicketCalling, models.TicketServing}

type Repository interface {
	SeedCounters(ctx context.Context, names []string) error
	ListCounters(ctx context.Context) ([]models.Counter, error)
	FindCounter(ctx context.Context, id uuid.UUID) (*models.Counter, error)
	SetCounterActive(ctx context.Context, id uuid.UUID, active bool) error

	// ActiveQueues returns every CALLING or SERVING queue, one per busy counter.
	ActiveQueues(ctx context.Context) ([]models.Queue, error)
	CountWaiting(ctx context.Context, date time.Time) (int64, error)
	ListQueues(ctx context.Context, q ListQuery) ([]models.Queue, int64, error)

	// CallNext atomically hands the oldest waiting queue of the date to the counter.
	CallNext(ctx context.Context, counterID uuid.UUID, date, now time.Time) (*models.Queue, error)
	// Transition changes the counter's active queue if its status is one of from.
	Transition(ctx context.Context, counterID uuid.UUID, from []models.TicketStatus, apply func(q *models.Queue, now time.Time)) (*models.Queue, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) SeedCounters(ctx context.Context, names []string) error {
	counters := make([]models.Counter, len(names))
	for i, n := range names {
		counters[i] = models.Counter{Name: n, IsActive: true}
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "name"}}, DoNothing: true}).
		Create(&counters).Error
}

func (r *repository) ListCounters(ctx context.Context) ([]models.Counter, error) {
	var out []models.Counter
	if err := r.db.WithContext(ctx).Order("name").Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *repository) FindCounter(ctx context.Context, id uuid.UUID) (*models.Counter, error) {
	var c models.Counter
	err := r.db.WithContext(ctx).First(&c, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrCounterNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *repository) SetCounterActive(ctx context.Context, id uuid.UUID, active bool) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var c models.Counter
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&c, "id = ?", id).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCounterNotFound
		}
		if err != nil {
			return err
		}

		if !active {
			var busy int64
			err := tx.Model(&models.Queue{}).Where("counter_id = ? AND status IN ?", id, activeStatuses).Count(&busy).Error
			if err != nil {
				return err
			}
			if busy > 0 {
				return ErrCounterBusy
			}
		}
		return tx.Model(&c).Update("is_active", active).Error
	})
}

func (r *repository) ActiveQueues(ctx context.Context) ([]models.Queue, error) {
	var out []models.Queue
	err := r.withDetails(r.db.WithContext(ctx)).Where("status IN ?", activeStatuses).Find(&out).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *repository) CountWaiting(ctx context.Context, date time.Time) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&models.Queue{}).
		Where("queue_date = ? AND status = ?", date, models.TicketWaiting).Count(&n).Error
	return n, err
}

func (r *repository) ListQueues(ctx context.Context, q ListQuery) ([]models.Queue, int64, error) {
	tx := r.db.WithContext(ctx).Model(&models.Queue{}).Where("queue_date = ?", q.Date)
	if q.Status != "" {
		tx = tx.Where("status = ?", q.Status)
	}
	if q.CounterID != "" {
		tx = tx.Where("counter_id = ?", q.CounterID)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var out []models.Queue
	err := r.withDetails(tx).Order("queue_sequence").
		Limit(q.Limit).Offset((q.Page - 1) * q.Limit).Find(&out).Error
	if err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (r *repository) CallNext(ctx context.Context, counterID uuid.UUID, date, now time.Time) (*models.Queue, error) {
	var calledID uuid.UUID

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var active int64
		err := tx.Model(&models.Queue{}).Where("counter_id = ? AND status IN ?", counterID, activeStatuses).Count(&active).Error
		if err != nil {
			return err
		}
		if active > 0 {
			return ErrCounterBusy
		}

		// SKIP LOCKED makes concurrent counters take different rows instead of waiting for each other.
		var next models.Queue
		err = tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("queue_date = ? AND status = ?", date, models.TicketWaiting).
			Order("created_at, id").Limit(1).Take(&next).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNoWaiting
		}
		if err != nil {
			return err
		}

		next.CounterID = &counterID
		next.Status = models.TicketCalling
		next.CalledAt = &now
		next.LastCalledAt = &now
		next.CallCount = 1
		if err := save(tx, &next); err != nil {
			return err
		}
		calledID = next.ID
		return nil
	})
	if err != nil {
		return nil, err
	}
	return r.findQueue(ctx, calledID)
}

func (r *repository) Transition(ctx context.Context, counterID uuid.UUID, from []models.TicketStatus, apply func(q *models.Queue, now time.Time)) (*models.Queue, error) {
	var changedID uuid.UUID

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var q models.Queue
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("counter_id = ? AND status IN ?", counterID, activeStatuses).Take(&q).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNoActiveQueue
		}
		if err != nil {
			return err
		}

		allowed := false
		for _, s := range from {
			allowed = allowed || q.Status == s
		}
		if !allowed {
			return fmt.Errorf("%w (current status: %s)", ErrInvalidState, q.Status)
		}

		apply(&q, time.Now())
		if err := save(tx, &q); err != nil {
			return err
		}
		changedID = q.ID
		return nil
	})
	if err != nil {
		return nil, err
	}
	return r.findQueue(ctx, changedID)
}

func (r *repository) findQueue(ctx context.Context, id uuid.UUID) (*models.Queue, error) {
	var q models.Queue
	err := r.withDetails(r.db.WithContext(ctx)).First(&q, "queues.id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrQueueNotFound
	}
	if err != nil {
		return nil, err
	}
	return &q, nil
}

// withDetails loads the counter, visit and patient (soft-deleted patients included).
func (r *repository) withDetails(tx *gorm.DB) *gorm.DB {
	return tx.Preload("Counter").Preload("Visit").Preload("Visit.Poli").Preload("Visit.Doctor").
		Preload("Visit.Patient", func(db *gorm.DB) *gorm.DB { return db.Unscoped() })
}

// save stores the queue and mirrors its status onto the visit's clinic queue status.
func save(tx *gorm.DB, q *models.Queue) error {
	if err := tx.Omit(clause.Associations).Save(q).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrCounterBusy
		}
		return err
	}
	return tx.Model(&models.Visit{}).Where("id = ?", q.VisitID).
		Update("queue_status", visitQueueStatus(q.Status)).Error
}

func visitQueueStatus(s models.TicketStatus) models.QueueStatus {
	switch s {
	case models.TicketCalling, models.TicketServing:
		return models.QueueCalled
	case models.TicketCompleted:
		return models.QueueDone
	case models.TicketSkipped:
		return models.QueueSkipped
	default:
		return models.QueueWaiting
	}
}
