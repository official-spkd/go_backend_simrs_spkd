package patients

import (
	"time"

	"gin-quickstart/internal/platform/models"
)

const dateLayout = "2006-01-02"

// PatientRequest is used for both create and update.
type PatientRequest struct {
	NIK       string        `json:"nik" binding:"required,numeric,len=16,nik" example:"3171234567890001"`
	FullName  string        `json:"full_name" binding:"required,notblank,min=3,max=150,person_name" example:"Siti Rahmawati"`
	BirthDate string        `json:"birth_date" binding:"required,datetime=2006-01-02" example:"1990-05-17"`
	Gender    models.Gender `json:"gender" binding:"required,oneof=Male Female" enums:"Male,Female" example:"Female"`

	BirthPlace    string `json:"birth_place" binding:"required,notblank,max=100" example:"Jakarta"`
	Religion      string `json:"religion" binding:"omitempty,notblank,max=50" example:"Islam"`
	Education     string `json:"education" binding:"omitempty,notblank,max=50" example:"Bachelor"`
	Occupation    string `json:"occupation" binding:"omitempty,notblank,max=100" example:"Teacher"`
	MaritalStatus string `json:"marital_status" binding:"omitempty,oneof=Single Married Divorced Widowed" enums:"Single,Married,Divorced,Widowed" example:"Married"`
	BloodType     string `json:"blood_type" binding:"omitempty,oneof=A B AB O" enums:"A,B,AB,O" example:"O"`
	Allergies     string `json:"allergies" binding:"omitempty,notblank,max=1000" example:"Penicillin, peanuts"`

	Phone string `json:"phone" binding:"required,phone" example:"081234567890"`
	Email string `json:"email" binding:"omitempty,email,max=150" example:"siti@example.com"`

	IDCardAddress   string `json:"id_card_address" binding:"required,notblank,min=5,max=500" example:"Jl. Melati No. 5, Jakarta Selatan"`
	DomicileAddress string `json:"domicile_address" binding:"omitempty,notblank,min=5,max=500" example:"Jl. Mawar No. 9, Depok"`

	EmergencyContactName         string `json:"emergency_contact_name" binding:"required,notblank,min=3,max=150,person_name" example:"Budi Santoso"`
	EmergencyContactRelationship string `json:"emergency_contact_relationship" binding:"required,notblank,max=50" example:"Husband"`
	EmergencyContactPhone        string `json:"emergency_contact_phone" binding:"required,phone" example:"081298765432"`

	Payer models.Payer `json:"payer" binding:"required,oneof=General BPJS Insurance" enums:"General,BPJS,Insurance" example:"BPJS"`
}

// ListRequest holds the validated query parameters of the list endpoint.
type ListRequest struct {
	Search string        `form:"search" json:"search" binding:"omitempty,max=100"`
	Gender models.Gender `form:"gender" json:"gender" binding:"omitempty,oneof=Male Female"`
	Payer  models.Payer  `form:"payer" json:"payer" binding:"omitempty,oneof=General BPJS Insurance"`
}

type ListQuery struct {
	Search string
	Gender models.Gender
	Payer  models.Payer
	Page   int
	Limit  int
}

type Response struct {
	ID                  string        `json:"id" example:"3f2b8c1e-5d4a-4e7b-9a1c-2b6f0d8e4a71"`
	MedicalRecordNumber string        `json:"medical_record_number" example:"RM-000001"`
	NIK                 string        `json:"nik" example:"3171234567890001"`
	FullName            string        `json:"full_name" example:"Siti Rahmawati"`
	BirthDate           string        `json:"birth_date" example:"1990-05-17"`
	Gender              models.Gender `json:"gender" enums:"Male,Female" example:"Female"`

	BirthPlace    string `json:"birth_place" example:"Jakarta"`
	Religion      string `json:"religion" example:"Islam"`
	Education     string `json:"education" example:"Bachelor"`
	Occupation    string `json:"occupation" example:"Teacher"`
	MaritalStatus string `json:"marital_status" example:"Married"`
	BloodType     string `json:"blood_type" example:"O"`
	Allergies     string `json:"allergies" example:"Penicillin, peanuts"`

	Phone string `json:"phone" example:"081234567890"`
	Email string `json:"email" example:"siti@example.com"`

	IDCardAddress   string `json:"id_card_address" example:"Jl. Melati No. 5, Jakarta Selatan"`
	DomicileAddress string `json:"domicile_address" example:"Jl. Mawar No. 9, Depok"`

	EmergencyContactName         string `json:"emergency_contact_name" example:"Budi Santoso"`
	EmergencyContactRelationship string `json:"emergency_contact_relationship" example:"Husband"`
	EmergencyContactPhone        string `json:"emergency_contact_phone" example:"081298765432"`

	Payer models.Payer `json:"payer" enums:"General,BPJS,Insurance" example:"BPJS"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toResponse(p *models.Patient) Response {
	return Response{
		ID:                           p.ID.String(),
		MedicalRecordNumber:          p.MedicalRecordNumber,
		NIK:                          p.NIK,
		FullName:                     p.FullName,
		BirthDate:                    p.BirthDate.Format(dateLayout),
		Gender:                       p.Gender,
		BirthPlace:                   p.BirthPlace,
		Religion:                     p.Religion,
		Education:                    p.Education,
		Occupation:                   p.Occupation,
		MaritalStatus:                p.MaritalStatus,
		BloodType:                    p.BloodType,
		Allergies:                    p.Allergies,
		Phone:                        p.Phone,
		Email:                        p.Email,
		IDCardAddress:                p.IDCardAddress,
		DomicileAddress:              p.DomicileAddress,
		EmergencyContactName:         p.EmergencyContactName,
		EmergencyContactRelationship: p.EmergencyContactRelationship,
		EmergencyContactPhone:        p.EmergencyContactPhone,
		Payer:                        p.Payer,
		CreatedAt:                    p.CreatedAt,
		UpdatedAt:                    p.UpdatedAt,
	}
}
