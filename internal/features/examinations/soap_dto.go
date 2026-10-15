package examinations

import (
	"encoding/json"
	"time"
)

type DiagnosisInput struct {
	ICD10Code string `json:"icd10_code" binding:"required,icd10" example:"J06.9"`
	Name      string `json:"name" binding:"omitempty,notblank,max=255" example:"Acute upper respiratory infection, unspecified"`
}

// SoapRequest saves a SOAP draft. Every field is optional for a draft; finalizing needs the
// subjective, plan, primary diagnosis and follow-up. Send version 0 when creating and the current
// version when updating.
type SoapRequest struct {
	Subjective        string `json:"subjective" binding:"omitempty,max=5000" example:"Fever for 3 days, productive cough, no shortness of breath."`
	ObjectiveFindings string `json:"objective_findings" binding:"omitempty,max=5000" example:"Pharynx hyperemic, no rhonchi, no wheezing."`

	PrimaryDiagnosis   *DiagnosisInput  `json:"primary_diagnosis"`
	SecondaryDiagnoses []DiagnosisInput `json:"secondary_diagnoses" binding:"omitempty,max=10,dive"`
	AssessmentNotes    string           `json:"assessment_notes" binding:"omitempty,max=5000" example:"URTI, good general condition."`
	Plan               string           `json:"plan" binding:"omitempty,max=5000" example:"Paracetamol 3x500 mg, drink plenty of fluids, rest."`
	FollowUp           string           `json:"follow_up" binding:"omitempty,oneof=discharge outpatient_followup refer_inpatient refer_icu" enums:"discharge,outpatient_followup,refer_inpatient,refer_icu" example:"outpatient_followup"`
	ControlDate        string           `json:"control_date" binding:"omitempty,datetime=2006-01-02" example:"2026-10-22"` // only with outpatient_followup
	ReferralNotes      string           `json:"referral_notes" binding:"omitempty,max=1000" example:""`                    // only with refer_inpatient / refer_icu

	Version int `json:"version" binding:"gte=0" example:"0"`
}

type AddendumRequest struct {
	Content string `json:"content" binding:"required,notblank,max=5000" example:"Correction: allergy to amoxicillin was confirmed."`
}

type DiagnosisResponse struct {
	ICD10Code string `json:"icd10_code" example:"J06.9"`
	Name      string `json:"name" example:"Acute upper respiratory infection, unspecified"`
}

type AddendumResponse struct {
	ID           string    `json:"id" example:"7d1f6c2a-7b3e-4a58-8c1d-0e5f4b2a6d93"`
	AuthorUserID string    `json:"author_user_id" example:"5b815a95-e4d9-40c1-bfc0-dccbe1f38722"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"created_at"`
}

// Objective is the "O" of SOAP: the doctor's findings plus the vital signs, which are pulled in
// automatically. Once the note is final the vital signs are the frozen snapshot.
type Objective struct {
	Findings   string         `json:"findings"`
	VitalSigns *VitalResponse `json:"vital_signs"`
}

type Assessment struct {
	PrimaryDiagnosis   *DiagnosisResponse  `json:"primary_diagnosis"`
	SecondaryDiagnoses []DiagnosisResponse `json:"secondary_diagnoses"`
	Notes              string              `json:"notes"`
}

type SoapResponse struct {
	VisitID  string `json:"visit_id" example:"104dd35e-2b3a-4b2a-b38c-8bdd0e1ecc79"`
	DoctorID string `json:"doctor_id" example:"9d1f6c2a-7b3e-4a58-8c1d-0e5f4b2a6d93"`
	Status   string `json:"status" enums:"draft,final" example:"draft"`

	Subjective    string     `json:"subjective"`
	Objective     Objective  `json:"objective"`
	Assessment    Assessment `json:"assessment"`
	Plan          string     `json:"plan"`
	FollowUp      string     `json:"follow_up" example:"outpatient_followup"`
	ControlDate   *string    `json:"control_date" example:"2026-10-22"`
	ReferralNotes string     `json:"referral_notes"`

	Addenda     []AddendumResponse `json:"addenda"`
	FinalizedAt *time.Time         `json:"finalized_at"`
	Version     int                `json:"version" example:"0"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

func toDiagnosisResponse(d SoapDiagnosis) DiagnosisResponse {
	return DiagnosisResponse{ICD10Code: d.ICD10Code, Name: d.Name}
}

// SoapView is a note together with the live vital signs it would show in "O" while still a draft.
type SoapView struct {
	Note  *SoapNote
	Vital *VitalAssessment
}

func toSoapResponse(v SoapView) SoapResponse {
	n := v.Note
	r := SoapResponse{
		VisitID:       n.VisitID.String(),
		DoctorID:      n.DoctorID.String(),
		Status:        n.Status,
		Subjective:    n.Subjective,
		Plan:          n.Plan,
		FollowUp:      n.FollowUp,
		ReferralNotes: n.ReferralNotes,
		Addenda:       []AddendumResponse{},
		FinalizedAt:   n.FinalizedAt,
		Version:       n.Version,
		UpdatedAt:     n.UpdatedAt,
	}
	r.Objective = Objective{Findings: n.ObjectiveFindings, VitalSigns: objectiveVitals(n, v.Vital)}
	r.Assessment = Assessment{Notes: n.AssessmentNotes, SecondaryDiagnoses: []DiagnosisResponse{}}
	if n.ControlDate != nil {
		d := n.ControlDate.Format("2006-01-02")
		r.ControlDate = &d
	}
	for _, d := range n.Diagnoses {
		dr := toDiagnosisResponse(d)
		if d.Role == DiagnosisPrimary {
			r.Assessment.PrimaryDiagnosis = &dr
		} else {
			r.Assessment.SecondaryDiagnoses = append(r.Assessment.SecondaryDiagnoses, dr)
		}
	}
	for _, a := range n.Addenda {
		r.Addenda = append(r.Addenda, AddendumResponse{ID: a.ID.String(), AuthorUserID: a.AuthorUserID.String(), Content: a.Content, CreatedAt: a.CreatedAt})
	}
	return r
}

// objectiveVitals returns the frozen snapshot of a final note, or the live vital signs of a draft.
func objectiveVitals(n *SoapNote, live *VitalAssessment) *VitalResponse {
	if n.Status == SoapFinal && n.ObjectiveVitalsSnapshot != nil {
		var frozen VitalResponse
		if json.Unmarshal([]byte(*n.ObjectiveVitalsSnapshot), &frozen) == nil {
			return &frozen
		}
	}
	if live == nil {
		return nil
	}
	r := toVitalResponse(live)
	return &r
}
