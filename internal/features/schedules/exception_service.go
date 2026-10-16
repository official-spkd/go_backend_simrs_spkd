package schedules

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"gin-quickstart/internal/platform/models"
	"gin-quickstart/internal/platform/utils"
)

var (
	ErrNoSession             = errors.New("doctor has no session on this date")
	ErrSubstituteRequired    = errors.New("a substitute doctor is required because visits are already registered")
	ErrSubstituteSame        = errors.New("substitute must be a different doctor")
	ErrSubstituteUnavailable = errors.New("substitute doctor has no available session on this date")
)

func (s *service) CreateException(ctx context.Context, req ExceptionRequest) (*ExceptionResponse, error) {
	doctor, err := s.findDoctor(ctx, req.DoctorID)
	if err != nil {
		return nil, err
	}
	date, err := parseDate(req.Date)
	if err != nil {
		return nil, err
	}
	if date.Before(utils.Today()) {
		return nil, ErrPastDate
	}
	own, err := s.templateOn(ctx, doctor.ID, date)
	if err != nil {
		return nil, err
	}
	// Without a substitute there is nothing to close; with one, a doctor who has no session that
	// day can still be covered so the poli keeps a doctor.
	if own == nil && req.SubstituteDoctorID == "" {
		return nil, ErrNoSession
	}

	var substitute *models.Doctor
	if req.SubstituteDoctorID != "" {
		substitute, err = s.resolveSubstitute(ctx, doctor.ID, req.SubstituteDoctorID, date)
		if err != nil {
			return nil, err
		}
	}

	counts, err := s.repo.FilledCounts(ctx, []uuid.UUID{doctor.ID}, date, date)
	if err != nil {
		return nil, err
	}
	if substitute == nil && counts[FilledKey{DoctorID: doctor.ID, Date: date.Format(dateLayout)}] > 0 {
		return nil, ErrSubstituteRequired
	}

	e := &models.ScheduleException{
		DoctorID: doctor.ID, Doctor: doctor, Date: date,
		SubstituteDoctor: substitute, Reason: strings.TrimSpace(req.Reason),
	}
	if substitute != nil {
		e.SubstituteDoctorID = &substitute.ID
	}
	if err := s.repo.CreateException(ctx, e); err != nil {
		return nil, err
	}
	resp := toExceptionResponse(e)
	return &resp, nil
}

func (s *service) ListExceptions(ctx context.Context, q ExceptionQuery) ([]ExceptionResponse, int64, error) {
	found, total, err := s.repo.ListExceptions(ctx, q)
	if err != nil {
		return nil, 0, err
	}
	data := make([]ExceptionResponse, 0, len(found))
	for i := range found {
		data = append(data, toExceptionResponse(&found[i]))
	}
	return data, total, nil
}

func (s *service) DeleteException(ctx context.Context, id uuid.UUID) error {
	e, err := s.repo.FindException(ctx, id)
	if err != nil {
		return err
	}
	if e.Date.Before(utils.Today()) {
		return ErrPastDate
	}
	return s.repo.DeleteException(ctx, id)
}

// resolveSubstitute loads the substitute and checks it can take the session: a different, active
// doctor who practices on that weekday and is not on leave that day.
func (s *service) resolveSubstitute(ctx context.Context, doctorID uuid.UUID, rawID string, date time.Time) (*models.Doctor, error) {
	substitute, err := s.activeDoctor(ctx, rawID)
	if err != nil {
		return nil, err
	}
	if substitute.ID == doctorID {
		return nil, ErrSubstituteSame
	}
	tmpl, err := s.templateOn(ctx, substitute.ID, date)
	if err != nil {
		return nil, err
	}
	if tmpl == nil {
		return nil, ErrSubstituteUnavailable
	}
	leaves, err := s.repo.ExceptionsInRange(ctx, []uuid.UUID{substitute.ID}, date, date)
	if err != nil {
		return nil, err
	}
	if len(leaves) > 0 {
		return nil, ErrSubstituteUnavailable
	}
	return substitute, nil
}

// templateOn returns the doctor's template that applies on date, or nil when there is none.
func (s *service) templateOn(ctx context.Context, doctorID uuid.UUID, date time.Time) (*models.DoctorSchedule, error) {
	found, err := s.repo.ListByDoctors(ctx, []uuid.UUID{doctorID})
	if err != nil {
		return nil, err
	}
	for i := range found {
		if found[i].Weekday == int(date.Weekday()) && !date.Before(found[i].EffectiveDate) {
			return &found[i], nil
		}
	}
	return nil, nil
}
