package visits

import (
	"time"

	"gin-quickstart/internal/platform/models"
)

const dateLayout = "2006-01-02"

type CreateRequest struct {
	PatientID string `json:"patient_id" binding:"required,uuid" example:"3f2b8c1e-5d4a-4e7b-9a1c-2b6f0d8e4a71"`

	VisitDate string `json:"visit_date" binding:"required,datetime=2006-01-02" example:"2026-10-20"`
	Poli      string `json:"poli" binding:"required,notblank,max=100" example:"General Clinic"`
	Doctor    string `json:"doctor" binding:"required,notblank,max=150" example:"dr. Andi Wijaya"`

	Penjamin          models.Penjamin  `json:"penjamin" binding:"required,oneof=umum bpjs asuransi" enums:"umum,bpjs,asuransi" example:"bpjs"`
	VisitType         models.VisitType `json:"visit_type" binding:"required,oneof=baru kontrol" enums:"baru,kontrol" example:"baru"`
	ResponsiblePerson string           `json:"responsible_person" binding:"omitempty,notblank,min=3,max=150,person_name" example:"Budi Santoso"`
	ChiefComplaint    string           `json:"chief_complaint" binding:"required,notblank,max=1000" example:"Fever and cough for three days"`
	IsApproved        *bool            `json:"is_approved" example:"false"`
	Allergies         string           `json:"allergies" binding:"omitempty,notblank,max=1000" example:"Penicillin"`

	// BPJS only: required when penjamin is bpjs, rejected otherwise.
	BPJSCardNumber      string `json:"bpjs_card_number" binding:"required_if=Penjamin bpjs,excluded_unless=Penjamin bpjs,omitempty,bpjs_id" example:"0001234567890"`
	FKTPReferralNumber  string `json:"fktp_referral_number" binding:"excluded_unless=Penjamin bpjs,omitempty,notblank,max=50" example:"0123R0010125Y000001"`
	ControlLetterNumber string `json:"control_letter_number" binding:"excluded_unless=Penjamin bpjs,omitempty,notblank,max=50" example:"0123R0010125K000001"`
	DocumentType        string `json:"document_type" binding:"excluded_unless=Penjamin bpjs,omitempty,notblank,max=50" example:"Referral letter"`

	// Asuransi only: insurance_name and policy_number are required when penjamin is asuransi, rejected otherwise.
	InsuranceName string `json:"insurance_name" binding:"required_if=Penjamin asuransi,excluded_unless=Penjamin asuransi,omitempty,notblank,max=150" example:"Prudential"`
	PolicyNumber  string `json:"policy_number" binding:"required_if=Penjamin asuransi,excluded_unless=Penjamin asuransi,omitempty,notblank,max=50" example:"POL-2026-001234"`
	CoverageLimit *int64 `json:"coverage_limit" binding:"excluded_unless=Penjamin asuransi,omitempty,gte=0" example:"15000000"` // in Rupiah
}

type ListRequest struct {
	Search          string                 `form:"search" json:"search" binding:"omitempty,max=100"`
	Date            string                 `form:"date" json:"date" binding:"omitempty,datetime=2006-01-02"`
	Poli            string                 `form:"poli" json:"poli" binding:"omitempty,max=100"`
	Penjamin        models.Penjamin        `form:"penjamin" json:"penjamin" binding:"omitempty,oneof=umum bpjs asuransi"`
	VisitType       models.VisitType       `form:"visit_type" json:"visit_type" binding:"omitempty,oneof=baru kontrol"`
	VisitStatus     models.VisitStatus     `form:"visit_status" json:"visit_status" binding:"omitempty,oneof=Registered Served Completed Paid Cancelled"`
	QueueStatus     models.QueueStatus     `form:"queue_status" json:"queue_status" binding:"omitempty,oneof=Waiting Called Skipped Done"`
	SatuSehatStatus models.SatuSehatStatus `form:"satu_sehat_status" json:"satu_sehat_status" binding:"omitempty,oneof=NotSent Sent"`
}

type ListQuery struct {
	Search          string
	Date            *time.Time
	Poli            string
	Penjamin        models.Penjamin
	VisitType       models.VisitType
	VisitStatus     models.VisitStatus
	QueueStatus     models.QueueStatus
	SatuSehatStatus models.SatuSehatStatus
	Page            int
	Limit           int
}

type PatientSummary struct {
	ID                  string        `json:"id" example:"3f2b8c1e-5d4a-4e7b-9a1c-2b6f0d8e4a71"`
	MedicalRecordNumber string        `json:"medical_record_number" example:"RM-000001"`
	NIK                 string        `json:"nik" example:"3171015705900001"`
	FullName            string        `json:"full_name" example:"Siti Rahmawati"`
	BirthDate           string        `json:"birth_date" example:"1990-05-17"`
	Gender              models.Gender `json:"gender" enums:"Male,Female" example:"Female"`
	BloodType           string        `json:"blood_type" example:"O"`
	Phone               string        `json:"phone" example:"081234567890"`
}

type Response struct {
	ID          string `json:"id" example:"9d1f6c2a-7b3e-4a58-8c1d-0e5f4b2a6d93"`
	QueueNumber string `json:"queue_number" example:"001"`
	VisitDate   string `json:"visit_date" example:"2026-10-20"`
	Poli        string `json:"poli" example:"General Clinic"`
	Doctor      string `json:"doctor" example:"dr. Andi Wijaya"`

	Penjamin        models.Penjamin        `json:"penjamin" enums:"umum,bpjs,asuransi" example:"bpjs"`
	VisitType       models.VisitType       `json:"visit_type" enums:"baru,kontrol" example:"baru"`
	VisitStatus     models.VisitStatus     `json:"visit_status" enums:"Registered,Served,Completed,Paid,Cancelled" example:"Registered"`
	QueueStatus     models.QueueStatus     `json:"queue_status" enums:"Waiting,Called,Skipped,Done" example:"Waiting"`
	SatuSehatStatus models.SatuSehatStatus `json:"satu_sehat_status" enums:"NotSent,Sent" example:"NotSent"`

	ResponsiblePerson string `json:"responsible_person" example:"Budi Santoso"`
	ChiefComplaint    string `json:"chief_complaint" example:"Fever and cough for three days"`
	Allergies         string `json:"allergies" example:"Penicillin"`
	IsApproved        bool   `json:"is_approved" example:"false"`

	// Present only when penjamin is bpjs.
	BPJSCardNumber      string `json:"bpjs_card_number,omitempty" example:"0001234567890"`
	FKTPReferralNumber  string `json:"fktp_referral_number,omitempty" example:"0123R0010125Y000001"`
	ControlLetterNumber string `json:"control_letter_number,omitempty" example:"0123R0010125K000001"`
	DocumentType        string `json:"document_type,omitempty" example:"Referral letter"`

	// Present only when penjamin is asuransi. CoverageLimit is in Rupiah.
	InsuranceName string `json:"insurance_name,omitempty" example:"Prudential"`
	PolicyNumber  string `json:"policy_number,omitempty" example:"POL-2026-001234"`
	CoverageLimit *int64 `json:"coverage_limit,omitempty" example:"15000000"`

	Patient PatientSummary `json:"patient"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toResponse(v *models.Visit) Response {
	p := v.Patient
	return Response{
		ID:                  v.ID.String(),
		QueueNumber:         v.QueueNumber,
		VisitDate:           v.VisitDate.Format(dateLayout),
		Poli:                v.Poli,
		Doctor:              v.Doctor,
		Penjamin:            v.Penjamin,
		VisitType:           v.VisitType,
		VisitStatus:         v.VisitStatus,
		QueueStatus:         v.QueueStatus,
		SatuSehatStatus:     v.SatuSehatStatus,
		ResponsiblePerson:   v.ResponsiblePerson,
		ChiefComplaint:      v.ChiefComplaint,
		Allergies:           v.Allergies,
		IsApproved:          v.IsApproved,
		BPJSCardNumber:      v.BPJSCardNumber,
		FKTPReferralNumber:  v.FKTPReferralNumber,
		ControlLetterNumber: v.ControlLetterNumber,
		DocumentType:        v.DocumentType,
		InsuranceName:       v.InsuranceName,
		PolicyNumber:        v.PolicyNumber,
		CoverageLimit:       v.CoverageLimit,
		Patient: PatientSummary{
			ID:                  p.ID.String(),
			MedicalRecordNumber: p.MedicalRecordNumber,
			NIK:                 p.NIK,
			FullName:            p.FullName,
			BirthDate:           p.BirthDate.Format(dateLayout),
			Gender:              p.Gender,
			BloodType:           p.BloodType,
			Phone:               p.Phone,
		},
		CreatedAt: v.CreatedAt,
		UpdatedAt: v.UpdatedAt,
	}
}
