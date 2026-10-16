package schedules

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"gin-quickstart/internal/platform/models"
	"gin-quickstart/internal/platform/utils"
)

// maxSessionDays caps one sessions request; it covers the 30-day horizon plus today.
const maxSessionDays = horizonDays + 1

var (
	ErrInvalidPeriod = errors.New("to must not be before from")
	ErrPeriodTooBig  = errors.New("period cannot be longer than 31 days")
)

type templateKey struct {
	doctorID uuid.UUID
	weekday  int
}

// Sessions derives the daily sessions of one doctor from the weekly template and the leave
// exceptions. Nothing is stored: quota usage always comes from the registered visits.
func (s *service) Sessions(ctx context.Context, q SessionQuery) ([]SessionResponse, error) {
	doctor, err := s.findDoctor(ctx, q.DoctorID)
	if err != nil {
		return nil, err
	}
	today := utils.Today()
	from := today
	if q.From != "" {
		if from, err = parseDate(q.From); err != nil {
			return nil, err
		}
	}
	to := from.AddDate(0, 0, horizonDays-1)
	if q.To != "" {
		if to, err = parseDate(q.To); err != nil {
			return nil, err
		}
	}
	if to.Before(from) {
		return nil, ErrInvalidPeriod
	}
	if to.After(from.AddDate(0, 0, maxSessionDays-1)) {
		return nil, ErrPeriodTooBig
	}

	exceptions, err := s.repo.ExceptionsInRange(ctx, []uuid.UUID{doctor.ID}, from, to)
	if err != nil {
		return nil, err
	}
	leaveByDate := make(map[string]*models.ScheduleException, len(exceptions))
	doctorIDs := []uuid.UUID{doctor.ID}
	for i := range exceptions {
		leaveByDate[exceptions[i].Date.Format(dateLayout)] = &exceptions[i]
		if exceptions[i].SubstituteDoctorID != nil {
			doctorIDs = append(doctorIDs, *exceptions[i].SubstituteDoctorID)
		}
	}

	schedules, err := s.repo.ListByDoctors(ctx, doctorIDs)
	if err != nil {
		return nil, err
	}
	templates := make(map[templateKey]*models.DoctorSchedule, len(schedules))
	for i := range schedules {
		templates[templateKey{schedules[i].DoctorID, schedules[i].Weekday}] = &schedules[i]
	}
	counts, err := s.repo.FilledCounts(ctx, doctorIDs, from, to)
	if err != nil {
		return nil, err
	}

	out := []SessionResponse{}
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		own := templates[templateKey{doctor.ID, int(d.Weekday())}]
		if own == nil || d.Before(own.EffectiveDate) {
			continue
		}
		date := d.Format(dateLayout)
		ownFilled := counts[FilledKey{DoctorID: doctor.ID, Date: date}]

		leave := leaveByDate[date]
		if leave == nil {
			status := StatusActive
			if d.Before(today) {
				status = StatusCompleted
			}
			out = append(out, newSession(date, doctor, own, ownFilled, status))
			continue
		}

		reason, exceptionID := leave.Reason, leave.ID.String()
		row := newSession(date, doctor, own, 0, StatusLeave)
		row.TotalQuota, row.JKNQuota, row.NonJKNQuota, row.Remaining = 0, 0, 0, 0
		row.Reason, row.ExceptionID = &reason, &exceptionID
		out = append(out, row)

		if leave.SubstituteDoctor == nil {
			continue
		}
		sub := templates[templateKey{leave.SubstituteDoctor.ID, int(d.Weekday())}]
		if sub == nil {
			continue // the substitute's template was removed after the leave was recorded
		}
		// The substitute absorbs the visits already registered for the doctor on leave.
		filled := counts[FilledKey{DoctorID: sub.DoctorID, Date: date}] + ownFilled
		row = newSession(date, leave.SubstituteDoctor, sub, filled, StatusSubstitute)
		replaced := toDoctorRef(doctor)
		row.ReplacedDoctor, row.ExceptionID = &replaced, &exceptionID
		out = append(out, row)
	}
	return out, nil
}

func newSession(date string, doctor *models.Doctor, t *models.DoctorSchedule, filled int, status string) SessionResponse {
	return SessionResponse{
		Date: date, Doctor: toDoctorRef(doctor),
		StartTime: t.StartTime, EndTime: t.EndTime, Status: status,
		TotalQuota: t.TotalQuota(), JKNQuota: t.JKNQuota, NonJKNQuota: t.NonJKNQuota,
		Filled: filled, Remaining: max(0, t.TotalQuota()-filled),
	}
}
