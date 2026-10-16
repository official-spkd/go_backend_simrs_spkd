package cashier

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"gin-quickstart/internal/platform/models"
	"gin-quickstart/internal/platform/utils"
)

const minShiftNoteLength = 10

// ShiftView is a shift with the money it received.
type ShiftView struct {
	Shift   *models.CashierShift
	Summary ShiftSummary
}

func (s *service) ListShifts(ctx context.Context, date time.Time, page, limit int) ([]ShiftView, int64, error) {
	from := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.Local)
	// Still-open shifts belong to today's recap even when they were opened earlier.
	shifts, total, err := s.repo.ListShifts(ctx, from, from.AddDate(0, 0, 1), date.Equal(utils.Today()), page, limit)
	if err != nil {
		return nil, 0, err
	}
	ptrs := make([]*models.CashierShift, len(shifts))
	for i := range shifts {
		ptrs[i] = &shifts[i]
	}
	views, err := s.shiftViews(ctx, ptrs)
	return views, total, err
}

func (s *service) CurrentShift(ctx context.Context, actor *utils.TokenClaims) (*ShiftView, error) {
	shift, err := s.repo.FindOpenShift(ctx, actor.UserID)
	if errors.Is(err, ErrShiftNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	views, err := s.shiftViews(ctx, []*models.CashierShift{shift})
	if err != nil {
		return nil, err
	}
	return &views[0], nil
}

func (s *service) CloseShift(ctx context.Context, id uuid.UUID, req CloseShiftRequest, actor *utils.TokenClaims) (*ShiftView, error) {
	shift, err := s.repo.FindShift(ctx, id)
	if err != nil {
		return nil, err
	}
	if shift.ClosedAt != nil {
		return nil, ErrShiftClosed
	}
	if shift.CashierUserID != actor.UserID {
		return nil, ErrNotYourShift
	}

	sums, err := s.repo.Summaries(ctx, []uuid.UUID{id})
	if err != nil {
		return nil, err
	}
	cash := sums[id].Cash
	counted := *req.CountedCash
	note := strings.TrimSpace(req.Note)
	if counted != cash && utf8.RuneCountInString(note) < minShiftNoteLength {
		return nil, InputError(fmt.Sprintf("a cash difference needs a note of at least %d characters", minShiftNoteLength))
	}

	if err := s.repo.CloseShift(ctx, id, counted, cash, note); err != nil {
		return nil, err
	}
	if shift, err = s.repo.FindShift(ctx, id); err != nil {
		return nil, err
	}
	views, err := s.shiftViews(ctx, []*models.CashierShift{shift})
	if err != nil {
		return nil, err
	}
	return &views[0], nil
}

func (s *service) shiftViews(ctx context.Context, shifts []*models.CashierShift) ([]ShiftView, error) {
	ids := make([]uuid.UUID, len(shifts))
	for i, sh := range shifts {
		ids[i] = sh.ID
	}
	sums, err := s.repo.Summaries(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]ShiftView, len(shifts))
	for i, sh := range shifts {
		out[i] = ShiftView{Shift: sh, Summary: sums[sh.ID]}
	}
	return out, nil
}
