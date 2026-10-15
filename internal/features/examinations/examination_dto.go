package examinations

import (
	"gin-quickstart/internal/platform/models"
	"gin-quickstart/internal/platform/utils"
)

type PatientSummary struct {
	ID                  string        `json:"id" example:"3f2b8c1e-5d4a-4e7b-9a1c-2b6f0d8e4a71"`
	MedicalRecordNumber string        `json:"medical_record_number" example:"RM-000001"`
	FullName            string        `json:"full_name" example:"Siti Rahmawati"`
	BirthDate           string        `json:"birth_date" example:"1990-05-17"`
	Age                 int           `json:"age" example:"36"`
	Gender              models.Gender `json:"gender" enums:"Male,Female" example:"Female"`
}

type VisitSummary struct {
	ID          string             `json:"id" example:"104dd35e-2b3a-4b2a-b38c-8bdd0e1ecc79"`
	VisitDate   string             `json:"visit_date" example:"2026-10-15"`
	VisitType   models.VisitType   `json:"visit_type" enums:"baru,kontrol" example:"baru"`
	VisitStatus models.VisitStatus `json:"visit_status" enums:"Registered,Served,Completed,Paid,Cancelled" example:"Served"`
	Poli        string             `json:"poli" example:"General Clinic"`
	Doctor      string             `json:"doctor" example:"dr. Andi Wijaya"`
	Patient     PatientSummary     `json:"patient"`
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
		Visit: VisitSummary{
			ID:          v.ID.String(),
			VisitDate:   v.VisitDate.Format("2006-01-02"),
			VisitType:   v.VisitType,
			VisitStatus: v.VisitStatus,
			Poli:        v.PoliName(),
			Doctor:      v.DoctorName(),
			Patient: PatientSummary{
				ID:                  v.Patient.ID.String(),
				MedicalRecordNumber: v.Patient.MedicalRecordNumber,
				FullName:            v.Patient.FullName,
				BirthDate:           v.Patient.BirthDate.Format("2006-01-02"),
				Age:                 utils.AgeOn(v.Patient.BirthDate, v.VisitDate),
				Gender:              v.Patient.Gender,
			},
		},
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
