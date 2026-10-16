package schedules

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"gin-quickstart/internal/platform/models"
	"gin-quickstart/internal/platform/utils"
)

type fakeRepo struct {
	schedules  []models.DoctorSchedule
	exceptions []models.ScheduleException
	filled     map[FilledKey]int
}

func (f *fakeRepo) Create(_ context.Context, s *models.DoctorSchedule) error {
	f.schedules = append(f.schedules, *s)
	return nil
}
func (f *fakeRepo) FindByID(context.Context, uuid.UUID) (*models.DoctorSchedule, error) {
	return nil, ErrNotFound
}
func (f *fakeRepo) List(context.Context, ListQuery) ([]models.DoctorSchedule, int64, error) {
	return nil, 0, nil
}
func (f *fakeRepo) ListByDoctors(_ context.Context, ids []uuid.UUID) ([]models.DoctorSchedule, error) {
	var out []models.DoctorSchedule
	for _, s := range f.schedules {
		for _, id := range ids {
			if s.DoctorID == id {
				out = append(out, s)
			}
		}
	}
	return out, nil
}
func (f *fakeRepo) ListTemplates(_ context.Context, doctorID, _ string) ([]models.DoctorSchedule, error) {
	var out []models.DoctorSchedule
	for _, sc := range f.schedules {
		if doctorID == "" || sc.DoctorID.String() == doctorID {
			out = append(out, sc)
		}
	}
	return out, nil
}
func (f *fakeRepo) SubstitutionsInRange(_ context.Context, ids []uuid.UUID, from, to time.Time) ([]models.ScheduleException, error) {
	var out []models.ScheduleException
	for _, e := range f.exceptions {
		for _, id := range ids {
			if e.SubstituteDoctorID != nil && *e.SubstituteDoctorID == id && !e.Date.Before(from) && !e.Date.After(to) {
				out = append(out, e)
			}
		}
	}
	return out, nil
}
func (f *fakeRepo) Update(context.Context, *models.DoctorSchedule) error { return nil }
func (f *fakeRepo) Delete(context.Context, uuid.UUID) error              { return nil }
func (f *fakeRepo) FilledCounts(context.Context, []uuid.UUID, time.Time, time.Time) (map[FilledKey]int, error) {
	return f.filled, nil
}
func (f *fakeRepo) CreateException(_ context.Context, e *models.ScheduleException) error {
	f.exceptions = append(f.exceptions, *e)
	return nil
}
func (f *fakeRepo) FindException(context.Context, uuid.UUID) (*models.ScheduleException, error) {
	return nil, ErrExceptionNotFound
}
func (f *fakeRepo) ListExceptions(context.Context, ExceptionQuery) ([]models.ScheduleException, int64, error) {
	return nil, 0, nil
}
func (f *fakeRepo) ExceptionsInRange(_ context.Context, ids []uuid.UUID, from, to time.Time) ([]models.ScheduleException, error) {
	var out []models.ScheduleException
	for _, e := range f.exceptions {
		for _, id := range ids {
			if e.DoctorID == id && !e.Date.Before(from) && !e.Date.After(to) {
				out = append(out, e)
			}
		}
	}
	return out, nil
}
func (f *fakeRepo) DeleteException(context.Context, uuid.UUID) error { return nil }

type fakeDoctors map[uuid.UUID]*models.Doctor

func (f fakeDoctors) Get(_ context.Context, id uuid.UUID) (*models.Doctor, error) {
	if d, ok := f[id]; ok {
		return d, nil
	}
	return nil, ErrDoctorNotFound
}

func intp(v int) *int { return &v }

// nextWeekday returns the first date after today that falls on the given weekday.
func nextWeekday(w time.Weekday) time.Time {
	d := utils.Today().AddDate(0, 0, 1)
	for d.Weekday() != w {
		d = d.AddDate(0, 0, 1)
	}
	return d
}

func newFixture() (*service, *fakeRepo, *models.Doctor, *models.Doctor) {
	maya := &models.Doctor{ID: uuid.New(), Name: "dr. Maya", IsActive: true}
	yudha := &models.Doctor{ID: uuid.New(), Name: "dr. Yudha", IsActive: true}
	repo := &fakeRepo{filled: map[FilledKey]int{}}
	for _, d := range []*models.Doctor{maya, yudha} {
		repo.schedules = append(repo.schedules, models.DoctorSchedule{
			ID: uuid.New(), DoctorID: d.ID, Doctor: d, Weekday: int(time.Tuesday), StartTime: "08:00", EndTime: "12:00",
			JKNQuota: 20, NonJKNQuota: 10, EffectiveDate: utils.Today(),
		})
	}
	svc := NewService(repo, fakeDoctors{maya.ID: maya, yudha.ID: yudha}).(*service)
	return svc, repo, maya, yudha
}

func TestCreateValidation(t *testing.T) {
	svc, repo, maya, _ := newFixture()
	tuesday := nextWeekday(time.Tuesday)
	repo.filled[FilledKey{maya.ID, tuesday.Format(dateLayout)}] = 23

	base := func() CreateRequest {
		return CreateRequest{DoctorID: maya.ID.String(), Weekday: intp(int(time.Wednesday)), UpdateRequest: UpdateRequest{
			StartTime: "08:00", EndTime: "12:00", JKNQuota: intp(20), NonJKNQuota: intp(10), EffectiveDate: utils.Today().Format(dateLayout),
		}}
	}
	tests := []struct {
		name   string
		mutate func(*CreateRequest)
		want   error
	}{
		{"valid", func(*CreateRequest) {}, nil},
		{"end not after start", func(r *CreateRequest) { r.EndTime = "08:00" }, ErrInvalidRange},
		{"bad time", func(r *CreateRequest) { r.StartTime = "25:00" }, ErrInvalidTime},
		{"zero total quota", func(r *CreateRequest) { r.JKNQuota, r.NonJKNQuota = intp(0), intp(0) }, ErrQuotaRequired},
		{"effective date in the past", func(r *CreateRequest) { r.EffectiveDate = utils.Today().AddDate(0, 0, -1).Format(dateLayout) }, ErrPastDate},
		{"unknown doctor", func(r *CreateRequest) { r.DoctorID = uuid.NewString() }, ErrDoctorNotFound},
		{"below registered visits", func(r *CreateRequest) {
			r.Weekday, r.JKNQuota, r.NonJKNQuota = intp(int(time.Tuesday)), intp(10), intp(10)
		}, ErrBelowFilled},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := base()
			tt.mutate(&req)
			_, err := svc.Create(context.Background(), req)
			if !errors.Is(err, tt.want) {
				t.Fatalf("got %v, want %v", err, tt.want)
			}
		})
	}
}

func TestCreateReportsPeakAsMinQuota(t *testing.T) {
	svc, repo, maya, _ := newFixture()
	tuesday := nextWeekday(time.Tuesday)
	repo.filled[FilledKey{maya.ID, tuesday.Format(dateLayout)}] = 23

	resp, err := svc.Create(context.Background(), CreateRequest{DoctorID: maya.ID.String(), Weekday: intp(int(time.Tuesday)), UpdateRequest: UpdateRequest{
		StartTime: "08:00", EndTime: "12:00", JKNQuota: intp(20), NonJKNQuota: intp(10), EffectiveDate: utils.Today().Format(dateLayout),
	}})
	if err != nil {
		t.Fatal(err)
	}
	if resp.MinQuota != 23 || resp.TotalQuota != 30 {
		t.Fatalf("min_quota=%d total_quota=%d, want 23 and 30", resp.MinQuota, resp.TotalQuota)
	}
}

func TestLeaveNeedsSubstituteWhenVisitsExist(t *testing.T) {
	svc, repo, maya, yudha := newFixture()
	tuesday := nextWeekday(time.Tuesday)
	key := tuesday.Format(dateLayout)
	repo.filled[FilledKey{maya.ID, key}] = 8

	req := ExceptionRequest{DoctorID: maya.ID.String(), Date: key, Reason: "Annual leave"}
	if _, err := svc.CreateException(context.Background(), req); !errors.Is(err, ErrSubstituteRequired) {
		t.Fatalf("got %v, want %v", err, ErrSubstituteRequired)
	}

	req.SubstituteDoctorID = maya.ID.String()
	if _, err := svc.CreateException(context.Background(), req); !errors.Is(err, ErrSubstituteSame) {
		t.Fatalf("got %v, want %v", err, ErrSubstituteSame)
	}

	req.SubstituteDoctorID = yudha.ID.String()
	if _, err := svc.CreateException(context.Background(), req); err != nil {
		t.Fatalf("leave with substitute: %v", err)
	}
}

func TestLeaveOnDayWithoutSession(t *testing.T) {
	svc, _, maya, _ := newFixture()
	monday := nextWeekday(time.Monday)
	_, err := svc.CreateException(context.Background(), ExceptionRequest{DoctorID: maya.ID.String(), Date: monday.Format(dateLayout), Reason: "x"})
	if !errors.Is(err, ErrNoSession) {
		t.Fatalf("got %v, want %v", err, ErrNoSession)
	}
}

func TestSubstituteOnLeaveIsRejected(t *testing.T) {
	svc, repo, maya, yudha := newFixture()
	tuesday := nextWeekday(time.Tuesday)
	repo.exceptions = append(repo.exceptions, models.ScheduleException{DoctorID: yudha.ID, Date: tuesday, Reason: "Sick"})

	_, err := svc.CreateException(context.Background(), ExceptionRequest{
		DoctorID: maya.ID.String(), Date: tuesday.Format(dateLayout), SubstituteDoctorID: yudha.ID.String(), Reason: "Annual leave",
	})
	if !errors.Is(err, ErrSubstituteUnavailable) {
		t.Fatalf("got %v, want %v", err, ErrSubstituteUnavailable)
	}
}

func TestSessionsWithLeaveAndSubstitute(t *testing.T) {
	svc, repo, maya, yudha := newFixture()
	tuesday := nextWeekday(time.Tuesday)
	key := tuesday.Format(dateLayout)
	repo.filled[FilledKey{maya.ID, key}] = 5
	repo.filled[FilledKey{yudha.ID, key}] = 8
	repo.exceptions = append(repo.exceptions, models.ScheduleException{
		DoctorID: maya.ID, Date: tuesday, SubstituteDoctorID: &yudha.ID, SubstituteDoctor: yudha, Reason: "Annual leave",
	})

	got, err := svc.Sessions(context.Background(), SessionQuery{DoctorID: maya.ID.String(), From: key, To: key})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d rows, want 2", len(got))
	}
	leave, sub := got[0], got[1]
	if leave.Status != StatusLeave || leave.TotalQuota != 0 || leave.Remaining != 0 {
		t.Fatalf("leave row = %+v", leave)
	}
	// The substitute absorbs the 5 visits of the doctor on leave: 8 + 5 of 30.
	if sub.Status != StatusSubstitute || sub.Filled != 13 || sub.Remaining != 17 || sub.ReplacedDoctor == nil {
		t.Fatalf("substitute row = %+v", sub)
	}
}

func TestSessionsPeriodLimits(t *testing.T) {
	svc, _, maya, _ := newFixture()
	today := utils.Today()

	_, err := svc.Sessions(context.Background(), SessionQuery{DoctorID: maya.ID.String(), From: today.Format(dateLayout), To: today.AddDate(0, 0, -1).Format(dateLayout)})
	if !errors.Is(err, ErrInvalidPeriod) {
		t.Fatalf("got %v, want %v", err, ErrInvalidPeriod)
	}
	_, err = svc.Sessions(context.Background(), SessionQuery{DoctorID: maya.ID.String(), From: today.Format(dateLayout), To: today.AddDate(0, 0, 31).Format(dateLayout)})
	if !errors.Is(err, ErrPeriodTooBig) {
		t.Fatalf("got %v, want %v", err, ErrPeriodTooBig)
	}

	got, err := svc.Sessions(context.Background(), SessionQuery{DoctorID: maya.ID.String()})
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range got {
		if row.Status != StatusActive {
			t.Fatalf("unexpected status %q on %s", row.Status, row.Date)
		}
	}
	if len(got) < 4 || len(got) > 5 {
		t.Fatalf("a Tuesday-only schedule over 30 days gives 4 or 5 sessions, got %d", len(got))
	}
}

func TestAvailabilityOneRowPerDoctorAndDate(t *testing.T) {
	svc, repo, maya, yudha := newFixture()
	tuesday := nextWeekday(time.Tuesday)
	key := tuesday.Format(dateLayout)
	repo.filled[FilledKey{maya.ID, key}] = 5
	repo.filled[FilledKey{yudha.ID, key}] = 8
	repo.exceptions = append(repo.exceptions, models.ScheduleException{DoctorID: maya.ID, Date: tuesday, SubstituteDoctorID: &yudha.ID, SubstituteDoctor: yudha})

	got, err := svc.Availability(context.Background(), AvailabilityQuery{From: key, To: key})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d rows, want one per doctor", len(got))
	}
	byDoctor := map[string]AvailabilityResponse{}
	for _, row := range got {
		byDoctor[row.Doctor.ID] = row
	}
	if r := byDoctor[maya.ID.String()]; r.Status != StatusLeave || r.TotalQuota != 0 || r.Remaining != 0 {
		t.Fatalf("doctor on leave = %+v", r)
	}
	// The substitute's own 8 visits plus the 5 of the doctor on leave.
	if r := byDoctor[yudha.ID.String()]; r.Status != StatusSubstitute || r.Filled != 13 || r.Remaining != 17 {
		t.Fatalf("substitute = %+v", r)
	}
}

func TestAvailabilitySkipsInactiveDoctors(t *testing.T) {
	svc, repo, maya, _ := newFixture()
	maya.IsActive = false
	for i := range repo.schedules {
		if repo.schedules[i].DoctorID == maya.ID {
			repo.schedules[i].Doctor = maya
		}
	}
	got, err := svc.Availability(context.Background(), AvailabilityQuery{})
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range got {
		if row.Doctor.ID == maya.ID.String() {
			t.Fatalf("inactive doctor listed: %+v", row)
		}
	}
}

func TestSubstituteCoversDoctorWithoutSession(t *testing.T) {
	svc, repo, maya, yudha := newFixture()
	monday := nextWeekday(time.Monday) // Maya has no Monday session; Yudha gets one
	repo.schedules = append(repo.schedules, models.DoctorSchedule{
		ID: uuid.New(), DoctorID: yudha.ID, Doctor: yudha, Weekday: int(time.Monday), StartTime: "08:00", EndTime: "12:00",
		JKNQuota: 10, NonJKNQuota: 10, EffectiveDate: utils.Today(),
	})
	key := monday.Format(dateLayout)

	req := ExceptionRequest{DoctorID: maya.ID.String(), Date: key, SubstituteDoctorID: yudha.ID.String(), Reason: "No practice"}
	if _, err := svc.CreateException(context.Background(), req); err != nil {
		t.Fatalf("substitute for a doctor without a session: %v", err)
	}
	if len(repo.exceptions) != 1 {
		t.Fatalf("exception not stored")
	}
	repo.exceptions[0].Doctor = maya

	sessions, err := svc.Sessions(context.Background(), SessionQuery{DoctorID: maya.ID.String(), From: key, To: key})
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 || sessions[0].Status != StatusSubstitute || sessions[0].Doctor.ID != yudha.ID.String() || sessions[0].ReplacedDoctor == nil {
		t.Fatalf("sessions = %+v", sessions)
	}

	avail, err := svc.Availability(context.Background(), AvailabilityQuery{From: key, To: key})
	if err != nil {
		t.Fatal(err)
	}
	if len(avail) != 1 || avail[0].Status != StatusSubstitute || avail[0].ReplacedDoctor == nil || avail[0].ReplacedDoctor.Name != maya.Name {
		t.Fatalf("availability = %+v", avail)
	}
}
