package queues

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"gin-quickstart/internal/platform/models"
	"gin-quickstart/internal/platform/utils"
)

var defaultCounters = []string{"Loket 1", "Loket 2", "Loket 3"}

var ErrCounterInactive = errors.New("counter is inactive")

type counterState struct {
	Counter models.Counter
	Current *models.Queue
}

type display struct {
	Date         time.Time
	WaitingCount int64
	Counters     []counterState
}

type Service interface {
	SeedCounters(ctx context.Context) error
	ListCounters(ctx context.Context) ([]counterState, error)
	SetCounterActive(ctx context.Context, id uuid.UUID, active bool) (*counterState, error)
	Display(ctx context.Context) (*display, error)
	ListQueues(ctx context.Context, q ListQuery) ([]QueueResponse, int64, error)

	Next(ctx context.Context, counterID uuid.UUID) (*models.Queue, error)
	Recall(ctx context.Context, counterID uuid.UUID) (*models.Queue, error)
	Serve(ctx context.Context, counterID uuid.UUID) (*models.Queue, error)
	Complete(ctx context.Context, counterID uuid.UUID) (*models.Queue, error)
	Skip(ctx context.Context, counterID uuid.UUID) (*models.Queue, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) SeedCounters(ctx context.Context) error {
	return s.repo.SeedCounters(ctx, defaultCounters)
}

func (s *service) ListCounters(ctx context.Context) ([]counterState, error) {
	counters, err := s.repo.ListCounters(ctx)
	if err != nil {
		return nil, err
	}
	active, err := s.repo.ActiveQueues(ctx)
	if err != nil {
		return nil, err
	}

	byCounter := make(map[uuid.UUID]*models.Queue, len(active))
	for i := range active {
		if active[i].CounterID != nil {
			byCounter[*active[i].CounterID] = &active[i]
		}
	}

	states := make([]counterState, len(counters))
	for i, c := range counters {
		states[i] = counterState{Counter: c, Current: byCounter[c.ID]}
	}
	return states, nil
}

func (s *service) SetCounterActive(ctx context.Context, id uuid.UUID, active bool) (*counterState, error) {
	if err := s.repo.SetCounterActive(ctx, id, active); err != nil {
		return nil, err
	}
	states, err := s.ListCounters(ctx)
	if err != nil {
		return nil, err
	}
	for i := range states {
		if states[i].Counter.ID == id {
			return &states[i], nil
		}
	}
	return nil, ErrCounterNotFound
}

func (s *service) Display(ctx context.Context) (*display, error) {
	states, err := s.ListCounters(ctx)
	if err != nil {
		return nil, err
	}
	today := utils.Today()
	waiting, err := s.repo.CountWaiting(ctx, today)
	if err != nil {
		return nil, err
	}
	return &display{Date: today, WaitingCount: waiting, Counters: states}, nil
}

func (s *service) ListQueues(ctx context.Context, q ListQuery) ([]QueueResponse, int64, error) {
	found, total, err := s.repo.ListQueues(ctx, q)
	if err != nil {
		return nil, 0, err
	}
	data := make([]QueueResponse, 0, len(found))
	for i := range found {
		data = append(data, toQueueResponse(&found[i]))
	}
	return data, total, nil
}

func (s *service) Next(ctx context.Context, counterID uuid.UUID) (*models.Queue, error) {
	counter, err := s.repo.FindCounter(ctx, counterID)
	if err != nil {
		return nil, err
	}
	if !counter.IsActive {
		return nil, ErrCounterInactive
	}
	return s.repo.CallNext(ctx, counterID, utils.Today(), time.Now())
}

// Recall refreshes the call time so displays ring the bell again.
func (s *service) Recall(ctx context.Context, counterID uuid.UUID) (*models.Queue, error) {
	return s.transition(ctx, counterID, []models.TicketStatus{models.TicketCalling}, func(q *models.Queue, now time.Time) {
		q.LastCalledAt = &now
		q.CallCount++
	})
}

// Serve marks the called patient as present and being served.
func (s *service) Serve(ctx context.Context, counterID uuid.UUID) (*models.Queue, error) {
	return s.transition(ctx, counterID, []models.TicketStatus{models.TicketCalling}, func(q *models.Queue, now time.Time) {
		q.Status = models.TicketServing
		q.ServingStartedAt = &now
	})
}

func (s *service) Complete(ctx context.Context, counterID uuid.UUID) (*models.Queue, error) {
	return s.transition(ctx, counterID, activeStatuses, func(q *models.Queue, now time.Time) {
		q.Status = models.TicketCompleted
		q.CompletedAt = &now
	})
}

// Skip is for patients who did not show up when called, so it only applies to CALLING queues.
func (s *service) Skip(ctx context.Context, counterID uuid.UUID) (*models.Queue, error) {
	return s.transition(ctx, counterID, []models.TicketStatus{models.TicketCalling}, func(q *models.Queue, now time.Time) {
		q.Status = models.TicketSkipped
		q.SkippedAt = &now
	})
}

func (s *service) transition(ctx context.Context, counterID uuid.UUID, from []models.TicketStatus, apply func(*models.Queue, time.Time)) (*models.Queue, error) {
	if _, err := s.repo.FindCounter(ctx, counterID); err != nil {
		return nil, err
	}
	return s.repo.Transition(ctx, counterID, from, apply)
}
