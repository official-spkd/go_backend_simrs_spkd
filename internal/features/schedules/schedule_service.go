package schedules

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"gin-quickstart/internal/platform/models"
	"gin-quickstart/internal/platform/utils"
)

// horizonDays is how far ahead sessions are generated and quota usage is checked.
const horizonDays = 30

var (
	ErrDoctorNotFound = models.ErrDoctorNotFound
	ErrDoctorInactive = errors.New("doctor is not active")
	ErrInvalidDate    = errors.New("date must use the format YYYY-MM-DD")
	ErrPastDate       = errors.New("date cannot be in the past")
	ErrInvalidTime    = errors.New("time must use the format HH:MM")
	ErrInvalidRange   = errors.New("end time must be after start time")
	ErrQuotaRequired  = errors.New("total quota must be at least 1")
	ErrBelowFilled    = errors.New("total quota cannot be lower than the registered visits")
	ErrInUse          = errors.New("schedule still has registered visits")
)

// DoctorFinder is satisfied by the doctors feature's service.
type DoctorFinder interface {
	Get(ctx context.Context, id uuid.UUID) (*models.Doctor, error)
}

type Service interface {
	Create(ctx context.Context, req CreateRequest) (*Response, error)
	Get(ctx context.Context, id uuid.UUID) (*Response, error)
	List(ctx context.Context, q ListQuery) ([]Response, int64, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*Response, error)
	Delete(ctx context.Context, id uuid.UUID) error

	CreateException(ctx context.Context, req ExceptionRequest) (*ExceptionResponse, error)
	ListExceptions(ctx context.Context, q ExceptionQuery) ([]ExceptionResponse, int64, error)
	DeleteException(ctx context.Context, id uuid.UUID) error

	Sessions(ctx context.Context, q SessionQuery) ([]SessionResponse, error)

	Availability(ctx context.Context, q AvailabilityQuery) ([]AvailabilityResponse, error)
}

type service struct {
	repo    Repository
	doctors DoctorFinder
}

func NewService(repo Repository, doctors DoctorFinder) Service {
	return &service{repo: repo, doctors: doctors}
}

func (s *service) Create(ctx context.Context, req CreateRequest) (*Response, error) {
	doctor, err := s.activeDoctor(ctx, req.DoctorID)
	if err != nil {
		return nil, err
	}
	d, err := parseDetails(req.UpdateRequest)
	if err != nil {
		return nil, err
	}
	peak, err := s.checkQuota(ctx, doctor.ID, *req.Weekday, d.total())
	if err != nil {
		return nil, err
	}

	sched := &models.DoctorSchedule{DoctorID: doctor.ID, Doctor: doctor, Weekday: *req.Weekday}
	d.applyTo(sched)
	if err := s.repo.Create(ctx, sched); err != nil {
		return nil, err
	}
	resp := toResponse(sched, peak)
	return &resp, nil
}

func (s *service) Get(ctx context.Context, id uuid.UUID) (*Response, error) {
	sched, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	peak, err := s.peakFilled(ctx, sched.DoctorID, sched.Weekday)
	if err != nil {
		return nil, err
	}
	resp := toResponse(sched, peak)
	return &resp, nil
}

func (s *service) List(ctx context.Context, q ListQuery) ([]Response, int64, error) {
	found, total, err := s.repo.List(ctx, q)
	if err != nil {
		return nil, 0, err
	}
	if len(found) == 0 {
		return []Response{}, total, nil
	}

	doctorIDs := make([]uuid.UUID, 0, len(found))
	for i := range found {
		doctorIDs = append(doctorIDs, found[i].DoctorID)
	}
	today := utils.Today()
	counts, err := s.repo.FilledCounts(ctx, doctorIDs, today, today.AddDate(0, 0, horizonDays))
	if err != nil {
		return nil, 0, err
	}

	data := make([]Response, 0, len(found))
	for i := range found {
		data = append(data, toResponse(&found[i], peakOf(counts, found[i].DoctorID, found[i].Weekday, today)))
	}
	return data, total, nil
}

func (s *service) Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*Response, error) {
	sched, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	d, err := parseDetails(req)
	if err != nil {
		return nil, err
	}
	peak, err := s.checkQuota(ctx, sched.DoctorID, sched.Weekday, d.total())
	if err != nil {
		return nil, err
	}

	d.applyTo(sched)
	if err := s.repo.Update(ctx, sched); err != nil {
		return nil, err
	}
	resp := toResponse(sched, peak)
	return &resp, nil
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	sched, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	peak, err := s.peakFilled(ctx, sched.DoctorID, sched.Weekday)
	if err != nil {
		return err
	}
	if peak > 0 {
		return fmt.Errorf("%w (%d)", ErrInUse, peak)
	}
	return s.repo.Delete(ctx, id)
}

// details is the validated, normalized form of UpdateRequest.
type details struct {
	start, end    string
	jkn, nonJKN   int
	effectiveDate time.Time
}

func (d details) total() int {
	return d.jkn + d.nonJKN
}

func (d details) applyTo(s *models.DoctorSchedule) {
	s.StartTime, s.EndTime = d.start, d.end
	s.JKNQuota, s.NonJKNQuota = d.jkn, d.nonJKN
	s.EffectiveDate = d.effectiveDate
}

func parseDetails(req UpdateRequest) (details, error) {
	start, err := time.Parse("15:04", req.StartTime)
	if err != nil {
		return details{}, ErrInvalidTime
	}
	end, err := time.Parse("15:04", req.EndTime)
	if err != nil {
		return details{}, ErrInvalidTime
	}
	if !end.After(start) {
		return details{}, ErrInvalidRange
	}
	if *req.JKNQuota+*req.NonJKNQuota < 1 {
		return details{}, ErrQuotaRequired
	}
	effective, err := parseDate(req.EffectiveDate)
	if err != nil {
		return details{}, err
	}
	if effective.Before(utils.Today()) {
		return details{}, ErrPastDate
	}
	return details{
		start: start.Format("15:04"), end: end.Format("15:04"),
		jkn: *req.JKNQuota, nonJKN: *req.NonJKNQuota, effectiveDate: effective,
	}, nil
}

func parseDate(raw string) (time.Time, error) {
	d, err := time.Parse(dateLayout, raw)
	if err != nil {
		return time.Time{}, ErrInvalidDate
	}
	return d, nil
}

// activeDoctor resolves a doctor ID from a request body.
func (s *service) activeDoctor(ctx context.Context, rawID string) (*models.Doctor, error) {
	doctor, err := s.findDoctor(ctx, rawID)
	if err != nil {
		return nil, err
	}
	if !doctor.IsActive {
		return nil, ErrDoctorInactive
	}
	return doctor, nil
}

func (s *service) findDoctor(ctx context.Context, rawID string) (*models.Doctor, error) {
	id, err := uuid.Parse(rawID)
	if err != nil {
		return nil, ErrDoctorNotFound
	}
	return s.doctors.Get(ctx, id)
}

// checkQuota rejects a total below the visits already registered on that weekday and returns
// that highest count.
func (s *service) checkQuota(ctx context.Context, doctorID uuid.UUID, weekday, total int) (int, error) {
	peak, err := s.peakFilled(ctx, doctorID, weekday)
	if err != nil {
		return 0, err
	}
	if total < peak {
		return 0, fmt.Errorf("%w (%d)", ErrBelowFilled, peak)
	}
	return peak, nil
}

func (s *service) peakFilled(ctx context.Context, doctorID uuid.UUID, weekday int) (int, error) {
	today := utils.Today()
	counts, err := s.repo.FilledCounts(ctx, []uuid.UUID{doctorID}, today, today.AddDate(0, 0, horizonDays))
	if err != nil {
		return 0, err
	}
	return peakOf(counts, doctorID, weekday, today), nil
}

// peakOf is the highest registered-visit count on the given weekday within the horizon.
func peakOf(counts map[FilledKey]int, doctorID uuid.UUID, weekday int, from time.Time) int {
	peak := 0
	for i := 0; i <= horizonDays; i++ {
		d := from.AddDate(0, 0, i)
		if int(d.Weekday()) != weekday {
			continue
		}
		peak = max(peak, counts[FilledKey{DoctorID: doctorID, Date: d.Format(dateLayout)}])
	}
	return peak
}
