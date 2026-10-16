package cashier

import "time"

type ShiftListRequest struct {
	Date string `form:"date" json:"date" binding:"omitempty,datetime=2006-01-02" example:"2026-10-16"`
}

// CloseShiftRequest records the cash counted in the drawer. A note is required when it differs
// from the cash the system recorded.
type CloseShiftRequest struct {
	CountedCash *int64 `json:"counted_cash" binding:"required,min=0" example:"350000"`
	Note        string `json:"note" binding:"max=500" example:"All receipts match."`
}

type ShiftSummaryResponse struct {
	Transactions int   `json:"transactions" example:"4"`
	Cash         int64 `json:"cash" example:"350000"`
	QRIS         int64 `json:"qris" example:"120000"`
	VA           int64 `json:"va" example:"0"`
	Total        int64 `json:"total" example:"470000"`
	CashCount    int   `json:"cash_count" example:"3"`
	QRISCount    int   `json:"qris_count" example:"1"`
	VACount      int   `json:"va_count" example:"0"`
}

type CashierRef struct {
	ID   string `json:"id"`
	Name string `json:"name" example:"Rina"`
}

type ShiftResponse struct {
	ID             string               `json:"id"`
	Cashier        CashierRef           `json:"cashier"`
	OpenedAt       time.Time            `json:"opened_at"`
	ClosedAt       *time.Time           `json:"closed_at"`
	IsOpen         bool                 `json:"is_open" example:"true"`
	CountedCash    *int64               `json:"counted_cash"`
	CashDifference *int64               `json:"cash_difference"`
	Total          *int64               `json:"total"`
	Note           string               `json:"note"`
	Summary        ShiftSummaryResponse `json:"summary"`
}

func toShiftResponse(v *ShiftView) ShiftResponse {
	s := v.Shift
	out := ShiftResponse{
		ID:             s.ID.String(),
		Cashier:        CashierRef{ID: s.CashierUserID.String()},
		OpenedAt:       s.OpenedAt,
		ClosedAt:       s.ClosedAt,
		IsOpen:         s.ClosedAt == nil,
		CountedCash:    s.CountedCash,
		CashDifference: s.CashDifference,
		Total:          s.Total,
		Note:           s.Note,
		Summary:        toShiftSummaryResponse(v.Summary),
	}
	if s.Cashier != nil {
		out.Cashier.Name = s.Cashier.Name
	}
	return out
}

func toShiftSummaryResponse(s ShiftSummary) ShiftSummaryResponse {
	return ShiftSummaryResponse{
		Transactions: s.Transactions, Cash: s.Cash, QRIS: s.QRIS, VA: s.VA, Total: s.Total(),
		CashCount: s.CashCount, QRISCount: s.QRISCount, VACount: s.VACount,
	}
}

func toShiftResponses(views []ShiftView) []ShiftResponse {
	out := make([]ShiftResponse, len(views))
	for i := range views {
		out[i] = toShiftResponse(&views[i])
	}
	return out
}
