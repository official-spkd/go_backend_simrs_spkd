package apm

import (
	"context"
	"regexp"
	"strings"
	"time"

	"gin-quickstart/internal/features/patients"
	"gin-quickstart/internal/features/schedules"
	"gin-quickstart/internal/features/visits"
	"gin-quickstart/internal/platform/models"
	"gin-quickstart/internal/platform/utils"
)

var mrnPattern = regexp.MustCompile(`^(?i)RM-[0-9]{1,12}$`)

// InputError is a request that is well-formed but not acceptable (maps to 400).
type InputError string

func (e InputError) Error() string { return string(e) }

// VisitRegistrar is satisfied by the visits feature's service, which owns every registration rule.
type VisitRegistrar interface {
	Create(ctx context.Context, req visits.CreateRequest) (*models.Visit, error)
}

// PatientRegistrar is satisfied by the patients feature's service, which owns the patient rules.
type PatientRegistrar interface {
	Create(ctx context.Context, req patients.PatientRequest) (*models.Patient, error)
}

// SessionFinder is satisfied by the schedules feature's service.
type SessionFinder interface {
	Availability(ctx context.Context, q schedules.AvailabilityQuery) ([]schedules.AvailabilityResponse, error)
}

type Service interface {
	LookupPatient(ctx context.Context, req LookupRequest) (*PatientBrief, error)
	ListPolis(ctx context.Context) ([]PoliBrief, error)
	ListDoctors(ctx context.Context, poliID string) ([]DoctorBrief, error)
	// ListSessions returns today's sessions only; the kiosk cannot look at other dates.
	ListSessions(ctx context.Context, poliID string) ([]SessionBrief, error)
	// RegisterPatient creates a new patient record; the kiosk then registers a visit for them.
	RegisterPatient(ctx context.Context, req patients.PatientRequest) (*PatientBrief, error)
	Register(ctx context.Context, req RegisterRequest) (*Ticket, error)
}

type service struct {
	repo     Repository
	visits   VisitRegistrar
	patients PatientRegistrar
	sessions SessionFinder
}

func NewService(repo Repository, visits VisitRegistrar, patients PatientRegistrar, sessions SessionFinder) Service {
	return &service{repo: repo, visits: visits, patients: patients, sessions: sessions}
}

func (s *service) RegisterPatient(ctx context.Context, req patients.PatientRequest) (*PatientBrief, error) {
	p, err := s.patients.Create(ctx, req)
	if err != nil {
		return nil, err
	}
	brief := toPatientBrief(p)
	return &brief, nil
}

func (s *service) LookupPatient(ctx context.Context, req LookupRequest) (*PatientBrief, error) {
	nik := strings.TrimSpace(req.NIK)
	mrn := strings.ToUpper(strings.TrimSpace(req.MedicalRecordNumber))
	bpjs := strings.TrimSpace(req.BPJSNumber)

	var column, value string
	given := 0
	for _, c := range []struct{ col, val string }{{"nik", nik}, {"medical_record_number", mrn}, {"bpjs_number", bpjs}} {
		if c.val != "" {
			column, value = c.col, c.val
			given++
		}
	}
	if given != 1 {
		return nil, InputError("send exactly one of nik, medical_record_number or bpjs_number")
	}
	if column == "medical_record_number" && !mrnPattern.MatchString(value) {
		return nil, InputError("medical_record_number must look like RM-000001")
	}

	p, err := s.repo.FindPatient(ctx, column, value)
	if err != nil {
		return nil, err
	}
	brief := toPatientBrief(p)
	return &brief, nil
}

func (s *service) ListPolis(ctx context.Context) ([]PoliBrief, error) {
	found, err := s.repo.ListPolis(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]PoliBrief, len(found))
	for i := range found {
		out[i] = toPoliBrief(&found[i])
	}
	return out, nil
}

func (s *service) ListDoctors(ctx context.Context, poliID string) ([]DoctorBrief, error) {
	found, err := s.repo.ListDoctors(ctx, poliID)
	if err != nil {
		return nil, err
	}
	out := make([]DoctorBrief, len(found))
	for i := range found {
		out[i] = toDoctorBrief(&found[i])
	}
	return out, nil
}

func (s *service) ListSessions(ctx context.Context, poliID string) ([]SessionBrief, error) {
	today := utils.Today().Format("2006-01-02")
	found, err := s.sessions.Availability(ctx, schedules.AvailabilityQuery{PoliID: poliID, From: today, To: today})
	if err != nil {
		return nil, err
	}
	out := make([]SessionBrief, len(found))
	for i, a := range found {
		out[i] = SessionBrief{DoctorID: a.Doctor.ID, StartTime: a.StartTime, EndTime: a.EndTime, Status: a.Status, TotalQuota: a.TotalQuota, Remaining: a.Remaining}
		if a.ReplacedDoctor != nil {
			out[i].ReplacedDoctorName = &a.ReplacedDoctor.Name
		}
	}
	return out, nil
}

// Register creates today's visit through the visits feature, so the kiosk follows exactly the same
// rules as the front desk, and returns what the kiosk prints.
func (s *service) Register(ctx context.Context, req RegisterRequest) (*Ticket, error) {
	visit, err := s.visits.Create(ctx, visits.CreateRequest{
		PatientID:           req.PatientID,
		PoliID:              req.PoliID,
		DoctorID:            req.DoctorID,
		VisitDate:           utils.Today().Format("2006-01-02"),
		Penjamin:            req.Penjamin,
		VisitType:           req.VisitType,
		ChiefComplaint:      req.ChiefComplaint,
		ResponsiblePerson:   req.ResponsiblePerson,
		BPJSCardNumber:      req.BPJSCardNumber,
		FKTPReferralNumber:  req.FKTPReferralNumber,
		ControlLetterNumber: req.ControlLetterNumber,
		DocumentType:        req.DocumentType,
		InsuranceName:       req.InsuranceName,
		PolicyNumber:        req.PolicyNumber,
		CoverageLimit:       req.CoverageLimit,
	})
	if err != nil {
		return nil, err
	}

	queueNumber, err := s.repo.FindQueueNumber(ctx, visit.ID)
	if err != nil {
		return nil, err
	}

	t := &Ticket{
		VisitID:             visit.ID.String(),
		VisitDate:           visit.VisitDate.Format("2006-01-02"),
		QueueNumber:         queueNumber,
		PoliQueueNumber:     visit.QueueNumber,
		Penjamin:            visit.Penjamin,
		PatientName:         visit.Patient.FullName,
		MedicalRecordNumber: visit.Patient.MedicalRecordNumber,
		PrintedAt:           time.Now(),
	}
	if visit.Poli != nil {
		t.Poli = toPoliBrief(visit.Poli)
	}
	if visit.Doctor != nil {
		t.Doctor = toDoctorBrief(visit.Doctor)
	}
	return t, nil
}
