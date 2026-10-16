package examinations

import (
	"time"

	"gin-quickstart/internal/platform/models"
	"gin-quickstart/internal/platform/utils"
)

type PatientSummary struct {
	ID                  string        `json:"id" example:"3f2b8c1e-5d4a-4e7b-9a1c-2b6f0d8e4a71"`
	MedicalRecordNumber string        `json:"medical_record_number" example:"RM-000001"`
	NIK                 string        `json:"nik" example:"3273010101900001"`
	FullName            string        `json:"full_name" example:"Siti Rahmawati"`
	BirthDate           string        `json:"birth_date" example:"1990-05-17"`
	Age                 int           `json:"age" example:"36"`
	Gender              models.Gender `json:"gender" enums:"Male,Female" example:"Female"`
	Allergies           string        `json:"allergies" example:"Penicillin"`
}

// VisitSummary is a visit as the doctor sees it. poli_queue_status is the clinic's own queue,
// moved by the call / skip / start / close actions.
type VisitSummary struct {
	ID              string                 `json:"id" example:"104dd35e-2b3a-4b2a-b38c-8bdd0e1ecc79"`
	VisitDate       string                 `json:"visit_date" example:"2026-10-15"`
	QueueNumber     string                 `json:"queue_number" example:"012"`
	VisitType       models.VisitType       `json:"visit_type" enums:"baru,kontrol" example:"baru"`
	Penjamin        models.Penjamin        `json:"penjamin" enums:"umum,bpjs,asuransi" example:"umum"`
	VisitStatus     models.VisitStatus     `json:"visit_status" enums:"Registered,Served,Completed,Paid,Cancelled" example:"Served"`
	PoliQueueStatus models.PoliQueueStatus `json:"poli_queue_status" enums:"Waiting,Called,Skipped,Examining,Done" example:"Examining"`
	PoliID          *string                `json:"poli_id" example:"6a1f6c2a-7b3e-4a58-8c1d-0e5f4b2a6d93"`
	Poli            string                 `json:"poli" example:"General Clinic"`
	PoliPrefix      string                 `json:"poli_prefix" example:"PD"`
	DoctorID        *string                `json:"doctor_id" example:"9d1f6c2a-7b3e-4a58-8c1d-0e5f4b2a6d93"`
	Doctor          string                 `json:"doctor" example:"dr. Andi Wijaya"`
	ChiefComplaint  string                 `json:"chief_complaint" example:"Fever and cough for three days"`
	Allergies       string                 `json:"allergies" example:"Penicillin"`

	RegisteredAt  time.Time  `json:"registered_at"`
	PoliCalledAt  *time.Time `json:"poli_called_at"`
	PoliCallCount int        `json:"poli_call_count" example:"1"`
	ExamStartedAt *time.Time `json:"exam_started_at"`
	ClosedAt      *time.Time `json:"closed_at"`

	DischargeCondition string `json:"discharge_condition" enums:"recovered,improved,unchanged" example:""`
	DischargeMethod    string `json:"discharge_method" enums:"home,follow_up,referred,inpatient" example:""`

	Patient PatientSummary `json:"patient"`
}

// Prefill holds what was already collected at registration, to start the nurse's form from.
type Prefill struct {
	ChiefComplaint string `json:"chief_complaint" example:"Fever and cough for three days"`
	Allergies      string `json:"allergies" example:"Penicillin"`
}

// ExaminationResponse is everything recorded for a visit. vital_signs and soap_note are null
// until they have been saved.
type ExaminationResponse struct {
	Visit      VisitSummary   `json:"visit"`
	Prefill    Prefill        `json:"prefill"`
	VitalSigns *VitalResponse `json:"vital_signs"`
	SoapNote   *SoapResponse  `json:"soap_note"`
}

func toExaminationResponse(o *Overview) ExaminationResponse {
	v := o.Visit
	r := ExaminationResponse{
		Visit:   toVisitSummary(v),
		Prefill: Prefill{ChiefComplaint: v.ChiefComplaint, Allergies: v.Allergies},
	}
	if o.Vital != nil {
		vr := toVitalResponse(o.Vital)
		r.VitalSigns = &vr
	}
	if o.Soap != nil {
		sr := toSoapResponse(*o.Soap)
		r.SoapNote = &sr
	}
	return r
}

func toVisitSummary(v *models.Visit) VisitSummary {
	r := VisitSummary{
		ID:                 v.ID.String(),
		VisitDate:          v.VisitDate.Format("2006-01-02"),
		QueueNumber:        v.QueueNumber,
		VisitType:          v.VisitType,
		Penjamin:           v.Penjamin,
		VisitStatus:        v.VisitStatus,
		PoliQueueStatus:    v.PoliQueueStatus,
		Poli:               v.PoliName(),
		Doctor:             v.DoctorName(),
		ChiefComplaint:     v.ChiefComplaint,
		Allergies:          v.Allergies,
		RegisteredAt:       v.CreatedAt,
		PoliCalledAt:       v.PoliCalledAt,
		PoliCallCount:      v.PoliCallCount,
		ExamStartedAt:      v.ExamStartedAt,
		ClosedAt:           v.ClosedAt,
		DischargeCondition: v.DischargeCondition,
		DischargeMethod:    v.DischargeMethod,
		Patient: PatientSummary{
			ID:                  v.Patient.ID.String(),
			MedicalRecordNumber: v.Patient.MedicalRecordNumber,
			NIK:                 v.Patient.NIK,
			FullName:            v.Patient.FullName,
			BirthDate:           v.Patient.BirthDate.Format("2006-01-02"),
			Age:                 utils.AgeOn(v.Patient.BirthDate, v.VisitDate),
			Gender:              v.Patient.Gender,
			Allergies:           v.Patient.Allergies,
		},
	}
	if v.PoliID != nil {
		id := v.PoliID.String()
		r.PoliID = &id
	}
	if v.Poli != nil {
		r.PoliPrefix = v.Poli.PrefixCode
	}
	if v.DoctorID != nil {
		id := v.DoctorID.String()
		r.DoctorID = &id
	}
	return r
}
