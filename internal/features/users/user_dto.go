package users

import (
	"time"

	"gin-quickstart/internal/platform/models"
)

type UpdateRequest struct {
	Name  string `json:"name" binding:"required,max=150" example:"Dr. Jane Doe"`
	NIP   string `json:"nip" binding:"required,max=30" example:"198001012010011001"`
	Email string `json:"email" binding:"required,email,max=150" example:"jane.doe@hospital.id"`

	// Doctors only; send just the fields to change. Rejected for any other role.
	SIPNumber string `json:"sip_number" binding:"omitempty,license_number,min=5,max=50" example:"446/001/SIP-D/2026"`
	STRNumber string `json:"str_number" binding:"omitempty,license_number,min=5,max=50" example:"3111100220123456"`
	PoliID    string `json:"poli_id" binding:"omitempty,uuid" example:"3f2b8c1e-5d4a-4e7b-9a1c-2b6f0d8e4a71"`
}

// UpdateRoleRequest changes a user's role. Becoming a doctor needs sip_number, str_number and
// poli_id; leaving the doctor role clears them.
type UpdateRoleRequest struct {
	Role string `json:"role" binding:"required" example:"doctor"`

	SIPNumber string `json:"sip_number" binding:"required_if=Role doctor,excluded_unless=Role doctor,omitempty,license_number,min=5,max=50" example:"446/001/SIP-D/2026"`
	STRNumber string `json:"str_number" binding:"required_if=Role doctor,excluded_unless=Role doctor,omitempty,license_number,min=5,max=50" example:"3111100220123456"`
	PoliID    string `json:"poli_id" binding:"required_if=Role doctor,excluded_unless=Role doctor,omitempty,uuid" example:"3f2b8c1e-5d4a-4e7b-9a1c-2b6f0d8e4a71"`
}

type UpdateStatusRequest struct {
	IsActive *bool `json:"is_active" binding:"required" example:"false"`
}

type ResetPasswordRequest struct {
	NewPassword string `json:"new_password" binding:"required,min=8,max=72" example:"n3wS3cretPass!"`
}

type ListQuery struct {
	Search   string
	Role     string
	IsActive *bool
	Page     int
	Limit    int
}

type Response struct {
	ID        string    `json:"id" example:"3f2b8c1e-5d4a-4e7b-9a1c-2b6f0d8e4a71"`
	Name      string    `json:"name" example:"Dr. Jane Doe"`
	NIP       string    `json:"nip" example:"198001012010011001"`
	Email     string    `json:"email" example:"jane.doe@hospital.id"`
	Role      string    `json:"role" example:"doctor"`
	IsActive  bool      `json:"is_active" example:"true"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Present only for doctors.
	DoctorProfile *DoctorProfileResponse `json:"doctor_profile,omitempty"`
}

type PoliRef struct {
	ID         string `json:"id" example:"3f2b8c1e-5d4a-4e7b-9a1c-2b6f0d8e4a71"`
	PrefixCode string `json:"prefix_code" example:"UMUM"`
	Name       string `json:"name" example:"General Clinic"`
	Room       string `json:"room" example:"Room 101"`
}

type DoctorProfileResponse struct {
	SIPNumber string  `json:"sip_number" example:"446/001/SIP-D/2026"`
	STRNumber string  `json:"str_number" example:"3111100220123456"`
	Poli      PoliRef `json:"poli"`
}

func toResponse(u *models.User) Response {
	r := Response{
		ID:        u.ID.String(),
		Name:      u.Name,
		NIP:       u.NIP,
		Email:     u.Email,
		Role:      u.Role.Name,
		IsActive:  u.IsActive,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
	if u.PoliID != nil && u.Poli != nil && u.SIPNumber != nil && u.STRNumber != nil {
		r.DoctorProfile = &DoctorProfileResponse{
			SIPNumber: *u.SIPNumber,
			STRNumber: *u.STRNumber,
			Poli: PoliRef{
				ID:         u.Poli.ID.String(),
				PrefixCode: u.Poli.PrefixCode,
				Name:       u.Poli.Name,
				Room:       u.Poli.Room,
			},
		}
	}
	return r
}
