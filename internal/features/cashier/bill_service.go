package cashier

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"gin-quickstart/internal/platform/models"
	"gin-quickstart/internal/platform/utils"
)

const minReasonLength = 15

// BillView is a visit's bill as the cashier sees it. Bill is nil while nothing has been stored
// for the visit yet; Items are then derived from the visit alone.
type BillView struct {
	Visit       *models.Visit
	CompletedAt *time.Time
	Bill        *models.Bill
	Items       []models.BillItem
	Totals      Totals
	Tariff      TariffInfo
}

// Status is the bill status; a visit without a stored bill is a draft.
func (v *BillView) Status() string {
	if v.Bill == nil {
		return models.BillDraft
	}
	return v.Bill.Status
}

// TariffInfo is the tariff version the derived lines are priced with.
type TariffInfo struct {
	Version       int
	EffectiveDate time.Time
}

// Totals follow the payment scheme: bpjs covers everything except the class-upgrade excess,
// asuransi covers up to the coverage limit, and umum leaves everything to the patient.
type Totals struct {
	Subtotal       int64
	Discount       int64
	Net            int64
	PayerCovered   int64
	PatientPortion int64
	Excess         int64
	Paid           int64
	Remaining      int64
}

func computeTotals(b *models.Bill, items []models.BillItem) Totals {
	var t Totals
	for _, i := range items {
		t.Subtotal += i.Subtotal
	}
	t.Discount = b.DiscountAmount
	t.Net = max(0, t.Subtotal-t.Discount)

	switch b.Scheme {
	case models.PenjaminBPJS:
		t.Excess = min(b.Excess, t.Net)
		t.PatientPortion = t.Excess
		t.PayerCovered = t.Net - t.Excess
	case models.PenjaminAsuransi:
		t.PayerCovered = min(t.Net, b.CoverageLimit)
		t.PatientPortion = t.Net - t.PayerCovered
		t.Excess = t.PatientPortion
	default:
		t.PatientPortion = t.Net
	}

	for _, p := range b.Payments {
		t.Paid += p.Amount
	}
	t.Remaining = max(0, t.PatientPortion-t.Paid)
	return t
}

func (s *service) Worklist(ctx context.Context, date time.Time, page, limit int) ([]BillView, int64, error) {
	visits, total, err := s.repo.ListVisits(ctx, date, page, limit)
	if err != nil {
		return nil, 0, err
	}
	ptrs := make([]*models.Visit, len(visits))
	for i := range visits {
		ptrs[i] = &visits[i]
	}
	views, err := s.views(ctx, ptrs, nil)
	return views, total, err
}

func (s *service) ListDocuments(ctx context.Context, q DocumentQuery) ([]BillView, int64, error) {
	bills, total, err := s.repo.ListDocuments(ctx, q)
	if err != nil {
		return nil, 0, err
	}
	visits := make([]*models.Visit, len(bills))
	byVisit := make(map[uuid.UUID]*models.Bill, len(bills))
	for i := range bills {
		visits[i] = bills[i].Visit
		byVisit[bills[i].VisitID] = &bills[i]
	}
	views, err := s.views(ctx, visits, byVisit)
	return views, total, err
}

func (s *service) Get(ctx context.Context, visitID uuid.UUID) (*BillView, error) {
	visit, err := s.repo.FindVisit(ctx, visitID)
	if err != nil {
		return nil, err
	}
	views, err := s.views(ctx, []*models.Visit{visit}, nil)
	if err != nil {
		return nil, err
	}
	return &views[0], nil
}

// views builds the bill of each visit. bills may be passed when already loaded.
func (s *service) views(ctx context.Context, visits []*models.Visit, bills map[uuid.UUID]*models.Bill) ([]BillView, error) {
	ids := make([]uuid.UUID, len(visits))
	for i, v := range visits {
		ids[i] = v.ID
	}
	var err error
	if bills == nil {
		if bills, err = s.repo.FindBills(ctx, ids); err != nil {
			return nil, err
		}
	}
	completed, err := s.repo.CompletedAt(ctx, ids)
	if err != nil {
		return nil, err
	}
	tariffs, err := s.repo.FindTariffs(ctx, []string{TariffRegistration, TariffGeneralConsultation, TariffSpecialConsultation})
	if err != nil {
		return nil, err
	}

	out := make([]BillView, len(visits))
	for i, v := range visits {
		view := BillView{Visit: v, Bill: bills[v.ID]}
		if at, ok := completed[v.ID]; ok {
			view.CompletedAt = &at
		}
		b := view.Bill
		if b == nil {
			b = newBill(v)
		}
		view.Items = b.Items
		if b.Status == models.BillDraft {
			view.Items = append(derivedItems(v, tariffs), b.Items...)
		}
		sortItems(view.Items)
		view.Totals = computeTotals(b, view.Items)
		view.Tariff = tariffInfo(tariffs)
		out[i] = view
	}
	return out, nil
}

// newBill is the draft a visit starts with: its penjamin, and the insurer's coverage limit
// recorded at registration.
func newBill(v *models.Visit) *models.Bill {
	b := &models.Bill{VisitID: v.ID, Status: models.BillDraft, Scheme: v.Penjamin}
	if v.Penjamin == models.PenjaminAsuransi && v.CoverageLimit != nil {
		b.CoverageLimit = *v.CoverageLimit
	}
	return b
}

// derivedItems are the registration fee and the consultation of the visit's doctor, priced for
// the visit's penjamin. General practitioners and dentists charge the general consultation.
func derivedItems(v *models.Visit, tariffs map[string]models.Tariff) []models.BillItem {
	consultation := TariffSpecialConsultation
	if v.Doctor == nil || isGeneralPractice(v.Doctor.Specialization) {
		consultation = TariffGeneralConsultation
	}

	var out []models.BillItem
	add := func(code, category string) {
		t, ok := tariffs[code]
		if !ok {
			return
		}
		price := t.PriceFor(v.Penjamin)
		out = append(out, models.BillItem{
			Category: category, Name: t.Name, Source: models.BillItemSourceRegistration, TariffCode: code,
			ServedAt: v.CreatedAt, Price: price, Quantity: 1, Subtotal: price,
		})
	}
	add(TariffRegistration, models.BillItemAdministration)
	add(consultation, models.BillItemConsultation)
	return out
}

func isGeneralPractice(specialization string) bool {
	s := strings.ToLower(specialization)
	return s == "" || strings.Contains(s, "umum") || strings.Contains(s, "gigi") || strings.Contains(s, "general") || strings.Contains(s, "dent")
}

var categoryOrder = []string{models.BillItemAdministration, models.BillItemConsultation, models.BillItemProcedure, models.BillItemMedication, models.BillItemOther}

func sortItems(items []models.BillItem) {
	rank := func(c string) int {
		if i := slices.Index(categoryOrder, c); i >= 0 {
			return i
		}
		return len(categoryOrder)
	}
	slices.SortStableFunc(items, func(a, b models.BillItem) int { return cmp.Compare(rank(a.Category), rank(b.Category)) })
}

func tariffInfo(tariffs map[string]models.Tariff) TariffInfo {
	var info TariffInfo
	for _, t := range tariffs {
		info.Version = max(info.Version, t.Version)
		if t.EffectiveDate.After(info.EffectiveDate) {
			info.EffectiveDate = t.EffectiveDate
		}
	}
	return info
}

// draft loads the visit's bill for a change, creating it on the first one.
func (s *service) draft(ctx context.Context, visitID uuid.UUID) (*BillView, error) {
	view, err := s.Get(ctx, visitID)
	if err != nil {
		return nil, err
	}
	if view.Visit.VisitStatus == models.VisitCancelled {
		return nil, ErrVisitCancelled
	}
	if view.Bill == nil {
		if view.Bill, err = s.repo.EnsureBill(ctx, newBill(view.Visit)); err != nil {
			return nil, err
		}
	}
	if view.Bill.Status != models.BillDraft {
		return nil, ErrBillNotDraft
	}
	return view, nil
}

func (s *service) SetScheme(ctx context.Context, visitID uuid.UUID, req SchemeRequest, actor *utils.TokenClaims) (*BillView, error) {
	view, err := s.draft(ctx, visitID)
	if err != nil {
		return nil, err
	}
	scheme := models.Penjamin(req.Scheme)
	var limit, excess int64
	switch scheme {
	case models.PenjaminAsuransi:
		limit = deref(req.CoverageLimit)
	case models.PenjaminBPJS:
		excess = deref(req.Excess)
	}

	err = s.repo.UpdateDraft(ctx, view.Bill.ID, map[string]any{"scheme": scheme, "coverage_limit": limit, "excess": excess}, models.BillLog{
		BillID: view.Bill.ID, Action: "scheme_changed", UserID: actor.UserID,
		Detail: fmt.Sprintf("scheme %s, coverage limit %d, excess %d", scheme, limit, excess),
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, visitID)
}

func (s *service) SetDiscount(ctx context.Context, visitID uuid.UUID, req DiscountRequest, actor *utils.TokenClaims) (*BillView, error) {
	view, err := s.draft(ctx, visitID)
	if err != nil {
		return nil, err
	}
	reason := strings.TrimSpace(req.Reason)
	if req.Amount > 0 && utf8.RuneCountInString(reason) < minReasonLength {
		return nil, InputError(fmt.Sprintf("a discount needs a reason of at least %d characters", minReasonLength))
	}
	if req.Amount > view.Totals.Subtotal {
		return nil, InputError("the discount cannot exceed the bill subtotal")
	}
	if req.Amount == 0 {
		reason = ""
	}

	err = s.repo.UpdateDraft(ctx, view.Bill.ID, map[string]any{"discount_amount": req.Amount, "discount_reason": reason}, models.BillLog{
		BillID: view.Bill.ID, Action: "discount_changed", UserID: actor.UserID, Reason: reason,
		Detail: fmt.Sprintf("discount %d", req.Amount),
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, visitID)
}

func (s *service) AddItem(ctx context.Context, visitID uuid.UUID, req ItemRequest, actor *utils.TokenClaims) (*BillView, error) {
	view, err := s.draft(ctx, visitID)
	if err != nil {
		return nil, err
	}
	reason := strings.TrimSpace(req.Reason)
	if utf8.RuneCountInString(reason) < minReasonLength {
		return nil, InputError(fmt.Sprintf("a manual item needs a reason of at least %d characters", minReasonLength))
	}

	name := strings.TrimSpace(req.Name)
	item := &models.BillItem{
		BillID: view.Bill.ID, Category: models.BillItemOther, Name: name, Source: models.BillItemSourceManual,
		ServedAt: time.Now(), Price: req.Price, Quantity: req.Quantity, Subtotal: req.Price * int64(req.Quantity),
		IsManual: true, Reason: reason, CreatedByUserID: &actor.UserID,
	}
	err = s.repo.AddItem(ctx, item, models.BillLog{
		BillID: view.Bill.ID, Action: "item_added", UserID: actor.UserID, Reason: reason,
		Detail: fmt.Sprintf("%s, %d x %d", name, req.Quantity, req.Price),
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, visitID)
}

func (s *service) Finalize(ctx context.Context, visitID uuid.UUID, actor *utils.TokenClaims) (*BillView, error) {
	view, err := s.draft(ctx, visitID)
	if err != nil {
		return nil, err
	}
	if view.Visit.VisitStatus != models.VisitCompleted {
		return nil, ErrVisitNotCompleted
	}

	var derived []models.BillItem
	for _, i := range view.Items {
		if !i.IsManual {
			i.BillID = view.Bill.ID
			derived = append(derived, i)
		}
	}
	err = s.repo.Finalize(ctx, view.Bill.ID, derived, actor.UserID, models.BillLog{BillID: view.Bill.ID, Action: "finalized", UserID: actor.UserID})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, visitID)
}

// Pay settles the whole remaining patient portion at once. The remaining amount is recomputed
// here, never taken from the client; each part is capped to what is still left.
func (s *service) Pay(ctx context.Context, visitID uuid.UUID, req PaymentRequest, actor *utils.TokenClaims) (*BillView, error) {
	view, err := s.Get(ctx, visitID)
	if err != nil {
		return nil, err
	}
	if view.Status() == models.BillPaid {
		return nil, ErrBillPaid
	}
	if view.Status() != models.BillFinal {
		return nil, ErrBillNotFinal
	}

	remaining := view.Totals.Remaining
	var sum int64
	for _, p := range req.Payments {
		sum += p.Amount
	}
	if remaining == 0 && sum > 0 {
		return nil, InputError("the patient has nothing left to pay")
	}
	if remaining > 0 && sum < remaining {
		return nil, InputError(fmt.Sprintf("the payment is %d short of the remaining %d", remaining-sum, remaining))
	}

	var payments []models.Payment
	for _, p := range req.Payments {
		amount := min(p.Amount, remaining)
		if amount == 0 {
			continue
		}
		remaining -= amount
		pay := models.Payment{Method: p.Method, Amount: amount, Reference: strings.TrimSpace(p.Reference)}
		if p.Method == models.PaymentCash && p.Received != nil {
			if *p.Received < amount {
				return nil, InputError("the cash received is less than the cash amount")
			}
			change := *p.Received - amount
			pay.Received, pay.Change = p.Received, &change
		}
		payments = append(payments, pay)
	}
	if len(payments) == 0 {
		// A zero patient portion still closes the bill and issues a receipt; the zero payment
		// keeps the shift's record of settled bills complete.
		payments = append(payments, models.Payment{Method: models.PaymentCash, Reference: string(view.Bill.Scheme)})
	}

	err = s.repo.Pay(ctx, view.Bill.ID, visitID, payments, actor.UserID, models.BillLog{BillID: view.Bill.ID, Action: "paid", UserID: actor.UserID})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, visitID)
}

func (s *service) RecordPrint(ctx context.Context, visitID uuid.UUID, req PrintRequest, actor *utils.TokenClaims) error {
	view, err := s.Get(ctx, visitID)
	if err != nil {
		return err
	}
	if view.Status() == models.BillDraft {
		return ErrBillNotFound
	}
	detail := deref(view.Bill.InvoiceNumber)
	if view.Bill.ReceiptNumber != nil {
		detail += " / " + *view.Bill.ReceiptNumber
	}
	return s.repo.AddLog(ctx, models.BillLog{BillID: view.Bill.ID, Action: "printed_" + req.Kind, Detail: detail, UserID: actor.UserID})
}

func deref[T any](p *T) T {
	var zero T
	if p == nil {
		return zero
	}
	return *p
}
