package cashier

import (
	"time"

	"github.com/google/uuid"

	"gin-quickstart/internal/platform/models"
)

type WorklistRequest struct {
	Date string `form:"date" json:"date" binding:"omitempty,datetime=2006-01-02" example:"2026-10-16"`
}

type DocumentListRequest struct {
	Search string `form:"search" json:"search" binding:"omitempty,max=100"`
	Status string `form:"status" json:"status" binding:"omitempty,oneof=final paid" example:"final"`
}

// SchemeRequest sets who pays the bill. coverage_limit applies to asuransi and excess (BPJS
// class upgrade) to bpjs; each is ignored, and stored as 0, for the other schemes.
type SchemeRequest struct {
	Scheme        string `json:"scheme" binding:"required,oneof=umum bpjs asuransi" example:"asuransi"`
	CoverageLimit *int64 `json:"coverage_limit" binding:"omitempty,min=0" example:"500000"`
	Excess        *int64 `json:"excess" binding:"omitempty,min=0" example:"0"`
}

// DiscountRequest replaces the bill's discount; amount 0 removes it.
type DiscountRequest struct {
	Amount int64  `json:"amount" binding:"min=0" example:"25000"`
	Reason string `json:"reason" binding:"max=500" example:"Patient is a hospital employee family member"`
}

type ItemRequest struct {
	Name     string `json:"name" binding:"required,notblank,max=150" example:"Wound dressing kit"`
	Price    int64  `json:"price" binding:"required,min=1" example:"45000"`
	Quantity int    `json:"quantity" binding:"required,min=1,max=1000" example:"1"`
	Reason   string `json:"reason" binding:"required,max=500" example:"Used during the doctor's procedure"`
}

// PaymentRequest settles the remaining patient portion. Leave payments empty when the patient
// portion is 0.
type PaymentRequest struct {
	Payments []PaymentPart `json:"payments" binding:"max=3,dive"`
}

type PaymentPart struct {
	Method string `json:"method" binding:"required,oneof=cash qris va" example:"cash"`
	Amount int64  `json:"amount" binding:"min=0" example:"105000"`
	// Received is the cash handed over, for cash only; the change is computed from it.
	Received  *int64 `json:"received" binding:"omitempty,min=0" example:"110000"`
	Reference string `json:"reference" binding:"max=50" example:"QR-INV/20261016/0001"`
}

type PrintRequest struct {
	Kind string `json:"kind" binding:"required,oneof=print pdf" example:"print"`
}

type PoliRef struct {
	ID         string `json:"id"`
	Name       string `json:"name" example:"General Clinic"`
	PrefixCode string `json:"prefix_code" example:"UMUM"`
}

type DoctorRef struct {
	ID             string `json:"id"`
	Name           string `json:"name" example:"dr. Andi"`
	Specialization string `json:"specialization" example:"Umum"`
}

type PatientRef struct {
	ID                  string `json:"id"`
	MedicalRecordNumber string `json:"medical_record_number" example:"RM-000123"`
	NIK                 string `json:"nik" example:"3273010101900001"`
	FullName            string `json:"full_name" example:"Budi Santoso"`
	BirthDate           string `json:"birth_date" example:"1990-01-01"`
	Gender              string `json:"gender" example:"Male"`
	Allergies           string `json:"allergies"`
}

// VisitResponse is the part of the visit the cashier needs. completed_at is when the doctor
// finalized the SOAP note.
type VisitResponse struct {
	ID             string     `json:"id"`
	VisitDate      string     `json:"visit_date" example:"2026-10-16"`
	QueueNumber    string     `json:"queue_number" example:"001"`
	VisitStatus    string     `json:"visit_status" example:"Completed"`
	Penjamin       string     `json:"penjamin" example:"umum"`
	Poli           PoliRef    `json:"poli"`
	Doctor         *DoctorRef `json:"doctor"`
	Patient        PatientRef `json:"patient"`
	InsuranceName  string     `json:"insurance_name"`
	PolicyNumber   string     `json:"policy_number"`
	CoverageLimit  *int64     `json:"coverage_limit"`
	BPJSCardNumber string     `json:"bpjs_card_number"`
	RegisteredAt   time.Time  `json:"registered_at"`
	CompletedAt    *time.Time `json:"completed_at"`
}

// TotalsResponse amounts are in Rupiah. remaining is what the patient still has to pay.
type TotalsResponse struct {
	Subtotal       int64 `json:"subtotal" example:"105000"`
	Discount       int64 `json:"discount" example:"0"`
	Net            int64 `json:"net" example:"105000"`
	PayerCovered   int64 `json:"payer_covered" example:"0"`
	PatientPortion int64 `json:"patient_portion" example:"105000"`
	Excess         int64 `json:"excess" example:"0"`
	Paid           int64 `json:"paid" example:"0"`
	Remaining      int64 `json:"remaining" example:"105000"`
}

// SummaryResponse is one row of the worklist or the document list.
type SummaryResponse struct {
	Visit         VisitResponse  `json:"visit"`
	Status        string         `json:"status" example:"draft"`
	Scheme        string         `json:"scheme" example:"umum"`
	InvoiceNumber *string        `json:"invoice_number" example:"INV/20261016/0001"`
	ReceiptNumber *string        `json:"receipt_number" example:"KW/20261016/0001"`
	FinalizedAt   *time.Time     `json:"finalized_at"`
	PaidAt        *time.Time     `json:"paid_at"`
	Totals        TotalsResponse `json:"totals"`
}

// ItemResponse source is registration (derived from the visit) or manual.
type ItemResponse struct {
	ID        *string   `json:"id"`
	Category  string    `json:"category" example:"administration"`
	Name      string    `json:"name" example:"Karcis rawat jalan"`
	Source    string    `json:"source" example:"registration"`
	CreatedBy string    `json:"created_by"`
	ServedAt  time.Time `json:"served_at"`
	Price     int64     `json:"price" example:"30000"`
	Quantity  int       `json:"quantity" example:"1"`
	Subtotal  int64     `json:"subtotal" example:"30000"`
	IsManual  bool      `json:"is_manual"`
	Reason    string    `json:"reason"`
}

type PaymentResponse struct {
	ID        string    `json:"id"`
	Method    string    `json:"method" example:"cash"`
	Amount    int64     `json:"amount" example:"105000"`
	Received  *int64    `json:"received" example:"110000"`
	Change    *int64    `json:"change" example:"5000"`
	Reference string    `json:"reference"`
	Cashier   string    `json:"cashier" example:"Rina"`
	ShiftID   string    `json:"shift_id"`
	PaidAt    time.Time `json:"paid_at"`
}

type BillResponse struct {
	SummaryResponse
	CoverageLimit       int64             `json:"coverage_limit" example:"0"`
	Excess              int64             `json:"excess" example:"0"`
	DiscountReason      string            `json:"discount_reason"`
	TariffVersion       int               `json:"tariff_version" example:"1"`
	TariffEffectiveDate string            `json:"tariff_effective_date" example:"2026-01-01"`
	Items               []ItemResponse    `json:"items"`
	Payments            []PaymentResponse `json:"payments"`
}

func toVisitResponse(v *models.Visit, completedAt *time.Time) VisitResponse {
	out := VisitResponse{
		ID:             v.ID.String(),
		VisitDate:      v.VisitDate.Format(time.DateOnly),
		QueueNumber:    v.QueueNumber,
		VisitStatus:    string(v.VisitStatus),
		Penjamin:       string(v.Penjamin),
		Poli:           PoliRef{Name: v.PoliName()},
		InsuranceName:  v.InsuranceName,
		PolicyNumber:   v.PolicyNumber,
		CoverageLimit:  v.CoverageLimit,
		BPJSCardNumber: v.BPJSCardNumber,
		RegisteredAt:   v.CreatedAt,
		CompletedAt:    completedAt,
		Patient: PatientRef{
			ID:                  v.Patient.ID.String(),
			MedicalRecordNumber: v.Patient.MedicalRecordNumber,
			NIK:                 v.Patient.NIK,
			FullName:            v.Patient.FullName,
			BirthDate:           v.Patient.BirthDate.Format(time.DateOnly),
			Gender:              string(v.Patient.Gender),
			Allergies:           v.Allergies,
		},
	}
	if out.Patient.Allergies == "" {
		out.Patient.Allergies = v.Patient.Allergies
	}
	if v.Poli != nil {
		out.Poli.ID, out.Poli.PrefixCode = v.Poli.ID.String(), v.Poli.PrefixCode
	}
	if v.Doctor != nil {
		out.Doctor = &DoctorRef{ID: v.Doctor.ID.String(), Name: v.Doctor.Name, Specialization: v.Doctor.Specialization}
	} else if name := v.DoctorName(); name != "" {
		out.Doctor = &DoctorRef{Name: name}
	}
	return out
}

func toSummaryResponse(v *BillView) SummaryResponse {
	out := SummaryResponse{
		Visit:  toVisitResponse(v.Visit, v.CompletedAt),
		Status: v.Status(),
		Scheme: string(v.Visit.Penjamin),
		Totals: TotalsResponse(v.Totals),
	}
	if b := v.Bill; b != nil {
		out.Scheme = string(b.Scheme)
		out.InvoiceNumber, out.ReceiptNumber = b.InvoiceNumber, b.ReceiptNumber
		out.FinalizedAt, out.PaidAt = b.FinalizedAt, b.PaidAt
	}
	return out
}

func toSummaryResponses(views []BillView) []SummaryResponse {
	out := make([]SummaryResponse, len(views))
	for i := range views {
		out[i] = toSummaryResponse(&views[i])
	}
	return out
}

func toBillResponse(v *BillView) BillResponse {
	b := v.Bill
	if b == nil {
		b = newBill(v.Visit)
	}
	out := BillResponse{
		SummaryResponse: toSummaryResponse(v),
		CoverageLimit:   b.CoverageLimit,
		Excess:          b.Excess,
		DiscountReason:  b.DiscountReason,
		TariffVersion:   v.Tariff.Version,
		Items:           make([]ItemResponse, len(v.Items)),
		Payments:        make([]PaymentResponse, len(b.Payments)),
	}
	if !v.Tariff.EffectiveDate.IsZero() {
		out.TariffEffectiveDate = v.Tariff.EffectiveDate.Format(time.DateOnly)
	}
	for i, it := range v.Items {
		item := ItemResponse{
			Category: it.Category, Name: it.Name, Source: it.Source, ServedAt: it.ServedAt,
			Price: it.Price, Quantity: it.Quantity, Subtotal: it.Subtotal, IsManual: it.IsManual, Reason: it.Reason,
		}
		if it.ID != uuid.Nil {
			id := it.ID.String()
			item.ID = &id
		}
		if it.CreatedBy != nil {
			item.CreatedBy = it.CreatedBy.Name
		}
		out.Items[i] = item
	}
	for i, p := range b.Payments {
		pay := PaymentResponse{
			ID: p.ID.String(), Method: p.Method, Amount: p.Amount, Received: p.Received, Change: p.Change,
			Reference: p.Reference, ShiftID: p.ShiftID.String(), PaidAt: p.PaidAt,
		}
		if p.Cashier != nil {
			pay.Cashier = p.Cashier.Name
		}
		out.Payments[i] = pay
	}
	return out
}
