package examinations

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"gin-quickstart/internal/platform/models"
	"gin-quickstart/internal/platform/utils"
)

var (
	ErrVisitNotFound   = models.ErrVisitNotFound
	ErrDoctorNotLinked = errors.New("your account is not linked to a doctor record")
	ErrNoDoctor        = errors.New("this visit has no doctor assigned")
	ErrNotYourVisit    = errors.New("this visit belongs to another doctor")
	ErrVisitCancelled  = errors.New("the visit is cancelled")
	ErrVisitNotToday   = errors.New("the visit is scheduled for a later date")
	ErrQueueNotToday   = errors.New("only today's visits can be called or examined")
	ErrCannotCall      = errors.New("the patient can only be called before the examination starts")
	ErrNotCalled       = errors.New("the patient has not been called")
	ErrExamStarted     = errors.New("the examination has already started")
	ErrExamNotStarted  = errors.New("the examination has not been started")
	ErrVisitClosed     = errors.New("the visit is already closed")
	ErrSoapNotFinal    = errors.New("the soap note must be finalized before the visit can be closed")
)

// InputError is a request that is well-formed but breaks a business rule (maps to 400).
type InputError string

func (e InputError) Error() string { return string(e) }

// IncompleteError lists what is still missing to finalize a note (maps to 422).
type IncompleteError struct{ Missing []string }

func (e IncompleteError) Error() string {
	return "cannot finalize, missing: " + strings.Join(e.Missing, ", ")
}

// VisitFinder is satisfied by the visits feature's service. The returned visit includes its
// patient, poli and doctor.
type VisitFinder interface {
	Get(ctx context.Context, id uuid.UUID) (*models.Visit, error)
}

// UserFinder is satisfied by the users feature's service.
type UserFinder interface {
	Get(ctx context.Context, id uuid.UUID) (*models.User, error)
}

// Overview is everything recorded for a visit so far.
type Overview struct {
	Visit *models.Visit
	Vital *models.VitalAssessment
	Soap  *SoapView
}

type Service interface {
	Overview(ctx context.Context, visitID uuid.UUID, actor *utils.TokenClaims) (*Overview, error)

	GetVitals(ctx context.Context, visitID uuid.UUID, actor *utils.TokenClaims) (*models.VitalAssessment, error)
	SaveVitals(ctx context.Context, visitID uuid.UUID, req VitalRequest, actor *utils.TokenClaims) (*models.VitalAssessment, error)
	ListVitalVersions(ctx context.Context, visitID uuid.UUID, actor *utils.TokenClaims) ([]VersionSummary, error)
	GetVitalVersion(ctx context.Context, visitID uuid.UUID, version int, actor *utils.TokenClaims) (*VersionDetail, error)

	GetSoap(ctx context.Context, visitID uuid.UUID, actor *utils.TokenClaims) (*SoapView, error)
	SaveSoap(ctx context.Context, visitID uuid.UUID, req SoapRequest, actor *utils.TokenClaims) (*SoapView, error)
	FinalizeSoap(ctx context.Context, visitID uuid.UUID, actor *utils.TokenClaims) (*SoapView, error)
	AddAddendum(ctx context.Context, visitID uuid.UUID, req AddendumRequest, actor *utils.TokenClaims) (*SoapView, error)

	// Worklist lists the signed-in doctor's visits of one date, in poli queue order.
	Worklist(ctx context.Context, q WorklistQuery, actor *utils.TokenClaims) ([]WorklistEntry, int64, error)
	Call(ctx context.Context, visitID uuid.UUID, actor *utils.TokenClaims) (*models.Visit, error)
	Recall(ctx context.Context, visitID uuid.UUID, actor *utils.TokenClaims) (*models.Visit, error)
	Skip(ctx context.Context, visitID uuid.UUID, actor *utils.TokenClaims) (*models.Visit, error)
	StartExam(ctx context.Context, visitID uuid.UUID, actor *utils.TokenClaims) (*models.Visit, error)
	Close(ctx context.Context, visitID uuid.UUID, req CloseRequest, actor *utils.TokenClaims) (*models.Visit, error)

	// History lists the patient's earlier and later visits (any doctor), with final notes only.
	History(ctx context.Context, visitID uuid.UUID, page, limit int, actor *utils.TokenClaims) ([]HistoryEntry, int64, error)

	// AuthorizeAttachment checks that the doctor may read (write false) or change (write true) the
	// files attached to the visit's SOAP note. It returns *utils.AccessError to refuse.
	AuthorizeAttachment(ctx context.Context, visitID uuid.UUID, actor *utils.TokenClaims, write bool) error
}

type service struct {
	repo   Repository
	visits VisitFinder
	users  UserFinder
}

func NewService(repo Repository, visits VisitFinder, users UserFinder) Service {
	return &service{repo: repo, visits: visits, users: users}
}

func (s *service) Overview(ctx context.Context, visitID uuid.UUID, actor *utils.TokenClaims) (*Overview, error) {
	visit, err := s.authorize(ctx, visitID, actor, false)
	if err != nil {
		return nil, err
	}
	out := &Overview{Visit: visit}

	if out.Vital, err = s.repo.FindVital(ctx, visitID); err != nil && !errors.Is(err, ErrVitalNotFound) {
		return nil, err
	}
	if note, err := s.repo.FindSoap(ctx, visitID); err == nil {
		out.Soap = &SoapView{Note: note, Vital: out.Vital}
	} else if !errors.Is(err, ErrSoapNotFound) {
		return nil, err
	}
	return out, nil
}

// authorize loads the visit and checks that the signed-in doctor owns it. Writes additionally
// require a visit that is not cancelled and not scheduled for a later date.
func (s *service) authorize(ctx context.Context, visitID uuid.UUID, actor *utils.TokenClaims, write bool) (*models.Visit, error) {
	visit, err := s.visits.Get(ctx, visitID)
	if err != nil {
		return nil, err
	}

	doctorID, err := s.linkedDoctor(ctx, actor)
	if err != nil {
		return nil, err
	}
	if visit.DoctorID == nil {
		return nil, ErrNoDoctor
	}
	if *visit.DoctorID != doctorID {
		return nil, ErrNotYourVisit
	}

	if write {
		if visit.VisitStatus == models.VisitCancelled {
			return nil, ErrVisitCancelled
		}
		if visit.VisitDate.After(utils.Today()) {
			return nil, ErrVisitNotToday
		}
	}
	return visit, nil
}

// linkedDoctor returns the doctor record the signed-in account belongs to.
func (s *service) linkedDoctor(ctx context.Context, actor *utils.TokenClaims) (uuid.UUID, error) {
	user, err := s.users.Get(ctx, actor.UserID)
	if errors.Is(err, models.ErrUserNotFound) || (err == nil && user.DoctorID == nil) {
		return uuid.Nil, ErrDoctorNotLinked
	}
	if err != nil {
		return uuid.Nil, err
	}
	return *user.DoctorID, nil
}

// ensureNotFinalized rejects changes to a visit whose SOAP note is already final.
func (s *service) ensureNotFinalized(ctx context.Context, visitID uuid.UUID) error {
	note, err := s.repo.FindSoap(ctx, visitID)
	if errors.Is(err, ErrSoapNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if note.Status == models.SoapFinal {
		return ErrFinalized
	}
	return nil
}
