package apm

import (
	"time"

	"gin-quickstart/internal/platform/models"
)

// LookupRequest finds a returning patient by exactly one identifier.
type LookupRequest struct {
	NIK                 string `form:"nik" json:"nik" binding:"omitempty,numeric,len=16"`
	MedicalRecordNumber string `form:"medical_record_number" json:"medical_record_number" binding:"omitempty,max=20"`
	BPJSNumber          string `form:"bpjs_number" json:"bpjs_number" binding:"omitempty,numeric,len=13"`
}

type DoctorListRequest struct {
	PoliID string `form:"poli_id" json:"poli_id" binding:"omitempty,uuid"`
}

// RegisterRequest registers a patient (returning, or just created through POST /apm/patients) for a visit today. The visit date, approval and
// allergies are not asked: the kiosk can only register for today and cannot approve a visit.
type RegisterRequest struct {
	PatientID string `json:"patient_id" binding:"required,uuid" example:"3f2b8c1e-5d4a-4e7b-9a1c-2b6f0d8e4a71"`
	PoliID    string `json:"poli_id" binding:"required,uuid" example:"7a1c2b6f-0d8e-4a71-9a1c-3f2b8c1e5d4a"`
	DoctorID  string `json:"doctor_id" binding:"required,uuid" example:"9d1f6c2a-7b3e-4a58-8c1d-0e5f4b2a6d93"`

	Penjamin          models.Penjamin  `json:"penjamin" binding:"required,oneof=umum bpjs asuransi" enums:"umum,bpjs,asuransi" example:"bpjs"`
	VisitType         models.VisitType `json:"visit_type" binding:"required,oneof=baru kontrol" enums:"baru,kontrol" example:"kontrol"`
	ChiefComplaint    string           `json:"chief_complaint" binding:"required,notblank,max=1000" example:"Fever and cough"`
	ResponsiblePerson string           `json:"responsible_person" binding:"omitempty,notblank,min=3,max=150,person_name" example:"Budi Santoso"` // required for patients under 18

	// BPJS only: required when penjamin is bpjs, rejected otherwise.
	BPJSCardNumber      string `json:"bpjs_card_number" binding:"required_if=Penjamin bpjs,excluded_unless=Penjamin bpjs,omitempty,bpjs_id" example:"0001234567890"`
	FKTPReferralNumber  string `json:"fktp_referral_number" binding:"excluded_unless=Penjamin bpjs,omitempty,notblank,max=50" example:"0123R0010125Y000001"`
	ControlLetterNumber string `json:"control_letter_number" binding:"excluded_unless=Penjamin bpjs,omitempty,notblank,max=50" example:"0123R0010125K000001"`
	DocumentType        string `json:"document_type" binding:"excluded_unless=Penjamin bpjs,omitempty,notblank,max=50" example:"Referral letter"`

	// Asuransi only: insurance_name and policy_number are required when penjamin is asuransi.
	InsuranceName string `json:"insurance_name" binding:"required_if=Penjamin asuransi,excluded_unless=Penjamin asuransi,omitempty,notblank,max=150" example:"Prudential"`
	PolicyNumber  string `json:"policy_number" binding:"required_if=Penjamin asuransi,excluded_unless=Penjamin asuransi,omitempty,notblank,max=50" example:"POL-2026-001234"`
	CoverageLimit *int64 `json:"coverage_limit" binding:"excluded_unless=Penjamin asuransi,omitempty,gte=0" example:"15000000"`
}

// PatientBrief is the little a kiosk needs to confirm who is standing in front of it. Contact,
// address, allergies and the NIK are deliberately left out because this endpoint has no login.
type PatientBrief struct {
	ID                  string        `json:"id" example:"3f2b8c1e-5d4a-4e7b-9a1c-2b6f0d8e4a71"`
	MedicalRecordNumber string        `json:"medical_record_number" example:"RM-000001"`
	FullName            string        `json:"full_name" example:"Siti Rahmawati"`
	BirthDate           string        `json:"birth_date" example:"1990-05-17"`
	Gender              models.Gender `json:"gender" enums:"Male,Female" example:"Female"`
	BPJSNumber          *string       `json:"bpjs_number" example:"0001234567890"`
	Payer               models.Payer  `json:"payer" enums:"General,BPJS,Insurance" example:"BPJS"`
}

type PoliBrief struct {
	ID         string `json:"id" example:"7a1c2b6f-0d8e-4a71-9a1c-3f2b8c1e5d4a"`
	PrefixCode string `json:"prefix_code" example:"UMUM"`
	Name       string `json:"name" example:"General Clinic"`
	Room       string `json:"room" example:"Room 101"`
}

type DoctorBrief struct {
	ID             string    `json:"id" example:"9d1f6c2a-7b3e-4a58-8c1d-0e5f4b2a6d93"`
	Name           string    `json:"name" example:"dr. Andi Wijaya, Sp.PD"`
	Specialization string    `json:"specialization" example:"Internal Medicine"`
	Poli           PoliBrief `json:"poli"`
}

type SessionListRequest struct {
	PoliID string `form:"poli_id" json:"poli_id" binding:"omitempty,uuid"`
}

// SessionBrief is today's practice session of one doctor: only hours and quota, which is what a
// kiosk needs to hide doctors that cannot take a patient.
type SessionBrief struct {
	DoctorID   string `json:"doctor_id" example:"9d1f6c2a-7b3e-4a58-8c1d-0e5f4b2a6d93"`
	StartTime  string `json:"start_time" example:"08:00"`
	EndTime    string `json:"end_time" example:"12:00"`
	Status     string `json:"status" enums:"active,substitute,leave" example:"active"`
	TotalQuota int    `json:"total_quota" example:"30"`
	Remaining  int    `json:"remaining" example:"7"`
	// ReplacedDoctorName is the doctor this one replaces today (leave or no session); null otherwise.
	ReplacedDoctorName *string `json:"replaced_doctor_name" example:"dr. Maya Putri, Sp.PD"`
}

// Ticket is what the kiosk prints after a successful registration.
type Ticket struct {
	VisitID             string          `json:"visit_id" example:"104dd35e-2b3a-4b2a-b38c-8bdd0e1ecc79"`
	VisitDate           string          `json:"visit_date" example:"2026-10-16"`
	QueueNumber         string          `json:"queue_number" example:"A007"`     // registration desk queue, called at the counters
	PoliQueueNumber     string          `json:"poli_queue_number" example:"003"` // queue inside the clinic
	Penjamin            models.Penjamin `json:"penjamin" enums:"umum,bpjs,asuransi" example:"bpjs"`
	PatientName         string          `json:"patient_name" example:"Siti Rahmawati"`
	MedicalRecordNumber string          `json:"medical_record_number" example:"RM-000001"`
	Poli                PoliBrief       `json:"poli"`
	Doctor              DoctorBrief     `json:"doctor"`
	PrintedAt           time.Time       `json:"printed_at"`
}

func toPatientBrief(p *models.Patient) PatientBrief {
	return PatientBrief{
		ID:                  p.ID.String(),
		MedicalRecordNumber: p.MedicalRecordNumber,
		FullName:            p.FullName,
		BirthDate:           p.BirthDate.Format("2006-01-02"),
		Gender:              p.Gender,
		BPJSNumber:          p.BPJSNumber,
		Payer:               p.Payer,
	}
}

func toPoliBrief(p *models.Poli) PoliBrief {
	return PoliBrief{ID: p.ID.String(), PrefixCode: p.PrefixCode, Name: p.Name, Room: p.Room}
}

func toDoctorBrief(d *models.Doctor) DoctorBrief {
	b := DoctorBrief{ID: d.ID.String(), Name: d.Name, Specialization: d.Specialization}
	if d.Poli != nil {
		b.Poli = toPoliBrief(d.Poli)
	}
	return b
}
