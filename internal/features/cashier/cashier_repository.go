package cashier

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"gin-quickstart/internal/platform/models"
)

var (
	ErrVisitNotFound = models.ErrVisitNotFound
	ErrBillNotFound  = errors.New("the bill has not been finalized yet")
	ErrBillNotDraft  = errors.New("the bill is finalized and can no longer be changed")
	ErrBillNotFinal  = errors.New("finalize the bill before taking payment")
	ErrBillPaid      = errors.New("the bill is already paid")
	ErrShiftNotFound = errors.New("shift not found")
	ErrShiftClosed   = errors.New("the shift is already closed")
	ErrShiftChanged  = errors.New("the shift received a payment meanwhile; reload and try again")
)

// ShiftSummary is the money a shift received, per payment method.
type ShiftSummary struct {
	Transactions int
	Cash         int64
	QRIS         int64
	VA           int64
	CashCount    int
	QRISCount    int
	VACount      int
}

func (s ShiftSummary) Total() int64 { return s.Cash + s.QRIS + s.VA }

type DocumentQuery struct {
	Search string
	Status string
	Page   int
	Limit  int
}

type Repository interface {
	SeedTariffs(ctx context.Context, tariffs []models.Tariff) error
	FindTariffs(ctx context.Context, codes []string) (map[string]models.Tariff, error)

	// FindVisit returns the visit with its patient, poli and doctor.
	FindVisit(ctx context.Context, id uuid.UUID) (*models.Visit, error)
	// ListVisits returns the visits of a date that are not cancelled, oldest first.
	ListVisits(ctx context.Context, date time.Time, page, limit int) ([]models.Visit, int64, error)
	// CompletedAt returns when the SOAP note of each visit was finalized.
	CompletedAt(ctx context.Context, visitIDs []uuid.UUID) (map[uuid.UUID]time.Time, error)

	// FindBills returns the bills of the given visits, with items and payments, keyed by visit ID.
	FindBills(ctx context.Context, visitIDs []uuid.UUID) (map[uuid.UUID]*models.Bill, error)
	// ListDocuments returns finalized and paid bills, most recent first.
	ListDocuments(ctx context.Context, q DocumentQuery) ([]models.Bill, int64, error)
	// EnsureBill creates the draft bill of b.VisitID unless one exists, and returns the stored bill.
	EnsureBill(ctx context.Context, b *models.Bill) (*models.Bill, error)

	// UpdateDraft changes fields of a bill that is still draft.
	UpdateDraft(ctx context.Context, billID uuid.UUID, fields map[string]any, log models.BillLog) error
	AddItem(ctx context.Context, item *models.BillItem, log models.BillLog) error
	// Finalize stores the derived lines, issues the invoice number and freezes the bill.
	Finalize(ctx context.Context, billID uuid.UUID, derived []models.BillItem, userID uuid.UUID, log models.BillLog) error
	// Pay records the payments in the cashier's open shift (opening one if needed), issues the
	// receipt number, and moves the bill and its visit to paid.
	Pay(ctx context.Context, billID, visitID uuid.UUID, payments []models.Payment, cashierID uuid.UUID, log models.BillLog) error
	AddLog(ctx context.Context, log models.BillLog) error

	// ListShifts returns the shifts opened on the given date, plus shifts still open when
	// includeOpen is set, newest first.
	ListShifts(ctx context.Context, from, to time.Time, includeOpen bool, page, limit int) ([]models.CashierShift, int64, error)
	FindShift(ctx context.Context, id uuid.UUID) (*models.CashierShift, error)
	FindOpenShift(ctx context.Context, cashierID uuid.UUID) (*models.CashierShift, error)
	Summaries(ctx context.Context, shiftIDs []uuid.UUID) (map[uuid.UUID]ShiftSummary, error)
	// CloseShift closes the shift if its recorded cash still equals expectedCash, then opens the
	// cashier's next shift.
	CloseShift(ctx context.Context, shiftID uuid.UUID, countedCash, expectedCash int64, note string) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) SeedTariffs(ctx context.Context, tariffs []models.Tariff) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "code"}}, DoNothing: true}).
		Create(&tariffs).Error
}

func (r *repository) FindTariffs(ctx context.Context, codes []string) (map[string]models.Tariff, error) {
	var list []models.Tariff
	if err := r.db.WithContext(ctx).Where("code IN ?", codes).Find(&list).Error; err != nil {
		return nil, err
	}
	out := make(map[string]models.Tariff, len(list))
	for _, t := range list {
		out[t.Code] = t
	}
	return out, nil
}

func (r *repository) visits(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Preload("Patient", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).Preload("Poli").Preload("Doctor")
}

func (r *repository) FindVisit(ctx context.Context, id uuid.UUID) (*models.Visit, error) {
	var v models.Visit
	err := r.visits(ctx).First(&v, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrVisitNotFound
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *repository) ListVisits(ctx context.Context, date time.Time, page, limit int) ([]models.Visit, int64, error) {
	tx := r.db.WithContext(ctx).Model(&models.Visit{}).
		Where("visit_date = ? AND visit_status <> ?", date, models.VisitCancelled)

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var out []models.Visit
	err := r.visits(ctx).
		Where("visit_date = ? AND visit_status <> ?", date, models.VisitCancelled).
		Order("created_at, id").Limit(limit).Offset((page - 1) * limit).Find(&out).Error
	if err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (r *repository) CompletedAt(ctx context.Context, visitIDs []uuid.UUID) (map[uuid.UUID]time.Time, error) {
	out := map[uuid.UUID]time.Time{}
	if len(visitIDs) == 0 {
		return out, nil
	}
	var notes []models.SoapNote
	err := r.db.WithContext(ctx).Select("visit_id", "finalized_at").
		Where("visit_id IN ? AND finalized_at IS NOT NULL", visitIDs).Find(&notes).Error
	if err != nil {
		return nil, err
	}
	for _, n := range notes {
		out[n.VisitID] = *n.FinalizedAt
	}
	return out, nil
}

func (r *repository) bills(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).
		Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("created_at, id") }).
		Preload("Items.CreatedBy").
		Preload("Payments", func(db *gorm.DB) *gorm.DB { return db.Order("paid_at, id") }).
		Preload("Payments.Cashier")
}

func (r *repository) FindBills(ctx context.Context, visitIDs []uuid.UUID) (map[uuid.UUID]*models.Bill, error) {
	out := map[uuid.UUID]*models.Bill{}
	if len(visitIDs) == 0 {
		return out, nil
	}
	var list []models.Bill
	if err := r.bills(ctx).Where("visit_id IN ?", visitIDs).Find(&list).Error; err != nil {
		return nil, err
	}
	for i := range list {
		out[list[i].VisitID] = &list[i]
	}
	return out, nil
}

func (r *repository) ListDocuments(ctx context.Context, q DocumentQuery) ([]models.Bill, int64, error) {
	filter := func(tx *gorm.DB) *gorm.DB {
		tx = tx.Joins("JOIN visits ON visits.id = bills.visit_id").
			Joins("JOIN patients ON patients.id = visits.patient_id")
		if q.Status != "" {
			tx = tx.Where("bills.status = ?", q.Status)
		} else {
			tx = tx.Where("bills.status IN ?", []string{models.BillFinal, models.BillPaid})
		}
		if q.Search != "" {
			like := "%" + escapeLike(strings.ToLower(q.Search)) + "%"
			tx = tx.Where("LOWER(patients.full_name) LIKE ? OR LOWER(patients.medical_record_number) LIKE ? OR LOWER(bills.invoice_number) LIKE ? OR LOWER(bills.receipt_number) LIKE ?",
				like, like, like, like)
		}
		return tx
	}

	var total int64
	if err := filter(r.db.WithContext(ctx).Model(&models.Bill{})).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var out []models.Bill
	err := filter(r.bills(ctx)).
		Preload("Visit.Patient", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).Preload("Visit.Poli").Preload("Visit.Doctor").
		Order("COALESCE(bills.paid_at, bills.finalized_at) DESC, bills.id").
		Limit(q.Limit).Offset((q.Page - 1) * q.Limit).Find(&out).Error
	if err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (r *repository) EnsureBill(ctx context.Context, b *models.Bill) (*models.Bill, error) {
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "visit_id"}}, DoNothing: true}).
		Omit("Items", "Payments", "Visit").Create(b).Error
	if err != nil {
		return nil, err
	}
	var out models.Bill
	if err := r.bills(ctx).First(&out, "visit_id = ?", b.VisitID).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *repository) UpdateDraft(ctx context.Context, billID uuid.UUID, fields map[string]any, log models.BillLog) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&models.Bill{}).Where("id = ? AND status = ?", billID, models.BillDraft).Updates(fields)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrBillNotDraft
		}
		return tx.Create(&log).Error
	})
}

func (r *repository) AddItem(ctx context.Context, item *models.BillItem, log models.BillLog) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockBill(tx, item.BillID, models.BillDraft, ErrBillNotDraft); err != nil {
			return err
		}
		if err := tx.Omit("CreatedBy").Create(item).Error; err != nil {
			return err
		}
		return tx.Create(&log).Error
	})
}

func (r *repository) Finalize(ctx context.Context, billID uuid.UUID, derived []models.BillItem, userID uuid.UUID, log models.BillLog) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockBill(tx, billID, models.BillDraft, ErrBillNotDraft); err != nil {
			return err
		}
		now := time.Now()
		number, err := nextNumber(tx, "INV", now)
		if err != nil {
			return err
		}
		if len(derived) > 0 {
			if err := tx.Omit("CreatedBy").Create(&derived).Error; err != nil {
				return err
			}
		}
		err = tx.Model(&models.Bill{}).Where("id = ?", billID).Updates(map[string]any{
			"status":               models.BillFinal,
			"invoice_number":       number,
			"finalized_at":         now,
			"finalized_by_user_id": userID,
		}).Error
		if err != nil {
			return err
		}
		log.Detail = number
		return tx.Create(&log).Error
	})
}

func (r *repository) Pay(ctx context.Context, billID, visitID uuid.UUID, payments []models.Payment, cashierID uuid.UUID, log models.BillLog) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockBill(tx, billID, models.BillFinal, ErrBillPaid); err != nil {
			return err
		}
		shift, err := openShift(tx, cashierID)
		if err != nil {
			return err
		}

		now := time.Now()
		number, err := nextNumber(tx, "KW", now)
		if err != nil {
			return err
		}
		for i := range payments {
			payments[i].BillID = billID
			payments[i].ShiftID = shift.ID
			payments[i].CashierUserID = cashierID
			payments[i].PaidAt = now
		}
		if err := tx.Omit("Shift", "Cashier").Create(&payments).Error; err != nil {
			return err
		}

		err = tx.Model(&models.Bill{}).Where("id = ?", billID).Updates(map[string]any{
			"status":         models.BillPaid,
			"receipt_number": number,
			"paid_at":        now,
		}).Error
		if err != nil {
			return err
		}
		err = tx.Model(&models.Visit{}).
			Where("id = ? AND visit_status = ?", visitID, models.VisitCompleted).
			Update("visit_status", models.VisitPaid).Error
		if err != nil {
			return err
		}
		log.Detail = number
		return tx.Create(&log).Error
	})
}

func (r *repository) AddLog(ctx context.Context, log models.BillLog) error {
	return r.db.WithContext(ctx).Create(&log).Error
}

func (r *repository) ListShifts(ctx context.Context, from, to time.Time, includeOpen bool, page, limit int) ([]models.CashierShift, int64, error) {
	filter := func(tx *gorm.DB) *gorm.DB {
		if includeOpen {
			return tx.Where("(opened_at >= ? AND opened_at < ?) OR closed_at IS NULL", from, to)
		}
		return tx.Where("opened_at >= ? AND opened_at < ?", from, to)
	}

	var total int64
	if err := filter(r.db.WithContext(ctx).Model(&models.CashierShift{})).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var out []models.CashierShift
	err := filter(r.db.WithContext(ctx).Preload("Cashier")).
		Order("opened_at DESC, id").Limit(limit).Offset((page - 1) * limit).Find(&out).Error
	if err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (r *repository) FindShift(ctx context.Context, id uuid.UUID) (*models.CashierShift, error) {
	var s models.CashierShift
	err := r.db.WithContext(ctx).Preload("Cashier").First(&s, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrShiftNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *repository) FindOpenShift(ctx context.Context, cashierID uuid.UUID) (*models.CashierShift, error) {
	var s models.CashierShift
	err := r.db.WithContext(ctx).Preload("Cashier").
		First(&s, "cashier_user_id = ? AND closed_at IS NULL", cashierID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrShiftNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *repository) Summaries(ctx context.Context, shiftIDs []uuid.UUID) (map[uuid.UUID]ShiftSummary, error) {
	if len(shiftIDs) == 0 {
		return map[uuid.UUID]ShiftSummary{}, nil
	}
	return summarize(r.db.WithContext(ctx), shiftIDs)
}

func (r *repository) CloseShift(ctx context.Context, shiftID uuid.UUID, countedCash, expectedCash int64, note string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var s models.CashierShift
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&s, "id = ?", shiftID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrShiftNotFound
		}
		if err != nil {
			return err
		}
		if s.ClosedAt != nil {
			return ErrShiftClosed
		}

		sums, err := summarize(tx, []uuid.UUID{shiftID})
		if err != nil {
			return err
		}
		sum := sums[shiftID]
		if sum.Cash != expectedCash {
			return ErrShiftChanged
		}

		err = tx.Model(&models.CashierShift{}).Where("id = ?", shiftID).Updates(map[string]any{
			"closed_at":       time.Now(),
			"counted_cash":    countedCash,
			"cash_difference": countedCash - sum.Cash,
			"total":           sum.Total(),
			"note":            note,
		}).Error
		if err != nil {
			return err
		}
		return createOpenShift(tx, s.CashierUserID)
	})
}

func summarize(tx *gorm.DB, shiftIDs []uuid.UUID) (map[uuid.UUID]ShiftSummary, error) {
	var rows []summaryRow
	err := tx.Model(&models.Payment{}).
		Select("shift_id, method, COUNT(*) AS count, COALESCE(SUM(amount), 0) AS amount").
		Where("shift_id IN ?", shiftIDs).Group("shift_id, method").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := map[uuid.UUID]ShiftSummary{}
	for _, row := range rows {
		s := out[row.ShiftID]
		s.add(row)
		out[row.ShiftID] = s
	}
	return out, nil
}

type summaryRow struct {
	ShiftID uuid.UUID
	Method  string
	Count   int
	Amount  int64
}

func (s *ShiftSummary) add(row summaryRow) {
	s.Transactions += row.Count
	switch row.Method {
	case models.PaymentCash:
		s.Cash += row.Amount
		s.CashCount += row.Count
	case models.PaymentQRIS:
		s.QRIS += row.Amount
		s.QRISCount += row.Count
	case models.PaymentVA:
		s.VA += row.Amount
		s.VACount += row.Count
	}
}

// lockBill locks the bill row and checks that it has the wanted status, so concurrent edits,
// finalizations and payments of one bill run one after another.
func lockBill(tx *gorm.DB, billID uuid.UUID, status string, wrongStatus error) error {
	var b models.Bill
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id", "status").First(&b, "id = ?", billID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrBillNotFound
	}
	if err != nil {
		return err
	}
	if b.Status != status {
		return wrongStatus
	}
	return nil
}

// createOpenShift relies on the partial unique index that allows one open shift per cashier.
func createOpenShift(tx *gorm.DB, cashierID uuid.UUID) error {
	return tx.Clauses(clause.OnConflict{
		Columns:     []clause.Column{{Name: "cashier_user_id"}},
		TargetWhere: clause.Where{Exprs: []clause.Expression{clause.Expr{SQL: "closed_at IS NULL"}}},
		DoNothing:   true,
	}).Omit("Cashier").Create(&models.CashierShift{CashierUserID: cashierID, OpenedAt: time.Now()}).Error
}

func openShift(tx *gorm.DB, cashierID uuid.UUID) (*models.CashierShift, error) {
	if err := createOpenShift(tx, cashierID); err != nil {
		return nil, err
	}
	var s models.CashierShift
	err := tx.Clauses(clause.Locking{Strength: "SHARE"}).
		First(&s, "cashier_user_id = ? AND closed_at IS NULL", cashierID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// The shift was closed while this payment waited for it.
		return nil, ErrShiftChanged
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// nextNumber issues the next document number of the day, e.g. INV/20260101/0001.
func nextNumber(tx *gorm.DB, prefix string, at time.Time) (string, error) {
	key := prefix + "/" + at.Format("20060102")
	var value int
	err := tx.Raw(`INSERT INTO document_counters (key, value, updated_at) VALUES (?, 1, NOW())
		ON CONFLICT (key) DO UPDATE SET value = document_counters.value + 1, updated_at = NOW()
		RETURNING value`, key).Scan(&value).Error
	if err != nil {
		return "", fmt.Errorf("issue %s number: %w", prefix, err)
	}
	return fmt.Sprintf("%s/%04d", key, value), nil
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
