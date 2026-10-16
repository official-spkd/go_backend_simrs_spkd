package cashier

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"gin-quickstart/internal/platform/utils"
)

var (
	ErrVisitCancelled    = errors.New("the visit is cancelled")
	ErrVisitNotCompleted = errors.New("only visits whose service is completed can be finalized")
	ErrNotYourShift      = errors.New("only the cashier of this shift can close it")
)

// InputError is a request that is well-formed but breaks a business rule (maps to 400).
type InputError string

func (e InputError) Error() string { return string(e) }

type Service interface {
	SeedTariffs(ctx context.Context) error

	Worklist(ctx context.Context, date time.Time, page, limit int) ([]BillView, int64, error)
	ListDocuments(ctx context.Context, q DocumentQuery) ([]BillView, int64, error)
	Get(ctx context.Context, visitID uuid.UUID) (*BillView, error)
	SetScheme(ctx context.Context, visitID uuid.UUID, req SchemeRequest, actor *utils.TokenClaims) (*BillView, error)
	SetDiscount(ctx context.Context, visitID uuid.UUID, req DiscountRequest, actor *utils.TokenClaims) (*BillView, error)
	AddItem(ctx context.Context, visitID uuid.UUID, req ItemRequest, actor *utils.TokenClaims) (*BillView, error)
	Finalize(ctx context.Context, visitID uuid.UUID, actor *utils.TokenClaims) (*BillView, error)
	Pay(ctx context.Context, visitID uuid.UUID, req PaymentRequest, actor *utils.TokenClaims) (*BillView, error)
	RecordPrint(ctx context.Context, visitID uuid.UUID, req PrintRequest, actor *utils.TokenClaims) error

	ListShifts(ctx context.Context, date time.Time, page, limit int) ([]ShiftView, int64, error)
	// CurrentShift returns the signed-in cashier's open shift, or nil when none is open. A shift
	// opens with the first payment and again right after the previous one is closed.
	CurrentShift(ctx context.Context, actor *utils.TokenClaims) (*ShiftView, error)
	CloseShift(ctx context.Context, id uuid.UUID, req CloseShiftRequest, actor *utils.TokenClaims) (*ShiftView, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) SeedTariffs(ctx context.Context) error {
	return s.repo.SeedTariffs(ctx, defaultTariffs())
}
