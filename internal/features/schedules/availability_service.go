package schedules

import (
	"context"

	"github.com/google/uuid"

	"gin-quickstart/internal/platform/models"
)

// defaultAvailabilityDays is one week, the window the front desk looks at.
const defaultAvailabilityDays = 7

// Availability lists the sessions of active doctors that have a weekly template, one row per
// doctor and date. A doctor on leave shows as "leave" with no quota; a substitute shows as
// "substitute" with the visits of the replaced doctor counted against the quota.
func (s *service) Availability(ctx context.Context, q AvailabilityQuery) ([]AvailabilityResponse, error) {
	from, to, err := resolvePeriod(q.From, q.To, defaultAvailabilityDays)
	if err != nil {
		return nil, err
	}
	found, err := s.repo.ListTemplates(ctx, q.DoctorID, q.PoliID)
	if err != nil {
		return nil, err
	}

	templates := make(map[templateKey]*models.DoctorSchedule, len(found))
	doctors := make([]*models.Doctor, 0)
	seen := make(map[uuid.UUID]bool)
	for i := range found {
		doc := found[i].Doctor
		if doc == nil || !doc.IsActive {
			continue
		}
		templates[templateKey{found[i].DoctorID, found[i].Weekday}] = &found[i]
		if !seen[doc.ID] {
			seen[doc.ID] = true
			doctors = append(doctors, doc)
		}
	}
	if len(doctors) == 0 {
		return []AvailabilityResponse{}, nil
	}
	ids := make([]uuid.UUID, 0, len(doctors))
	for _, d := range doctors {
		ids = append(ids, d.ID)
	}

	leaves, err := s.repo.ExceptionsInRange(ctx, ids, from, to)
	if err != nil {
		return nil, err
	}
	onLeave := make(map[FilledKey]bool, len(leaves))
	for _, e := range leaves {
		onLeave[FilledKey{e.DoctorID, e.Date.Format(dateLayout)}] = true
	}
	substitutions, err := s.repo.SubstitutionsInRange(ctx, ids, from, to)
	if err != nil {
		return nil, err
	}
	replaced := make(map[FilledKey]models.ScheduleException, len(substitutions))
	countIDs := append([]uuid.UUID{}, ids...)
	for _, e := range substitutions {
		replaced[FilledKey{*e.SubstituteDoctorID, e.Date.Format(dateLayout)}] = e
		countIDs = append(countIDs, e.DoctorID)
	}
	counts, err := s.repo.FilledCounts(ctx, countIDs, from, to)
	if err != nil {
		return nil, err
	}

	out := []AvailabilityResponse{}
	for _, doc := range doctors {
		for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
			t := templates[templateKey{doc.ID, int(d.Weekday())}]
			if t == nil || d.Before(t.EffectiveDate) {
				continue
			}
			key := FilledKey{doc.ID, d.Format(dateLayout)}
			row := AvailabilityResponse{
				Date: key.Date, Doctor: toDoctorRef(doc), StartTime: t.StartTime, EndTime: t.EndTime,
				Status: StatusActive, TotalQuota: t.TotalQuota(), Filled: counts[key],
			}
			switch other, isSubstitute := replaced[key]; {
			case onLeave[key]:
				row.Status, row.TotalQuota, row.Filled = StatusLeave, 0, 0
			case isSubstitute:
				row.Status = StatusSubstitute
				row.Filled += counts[FilledKey{other.DoctorID, key.Date}]
				ref := toDoctorRef(other.Doctor)
				row.ReplacedDoctor = &ref
			}
			row.Remaining = max(0, row.TotalQuota-row.Filled)
			out = append(out, row)
		}
	}
	return out, nil
}
