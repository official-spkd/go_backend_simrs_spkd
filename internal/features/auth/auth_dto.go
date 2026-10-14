package auth

import (
	"time"

	"gin-quickstart/internal/platform/models"
)

type LoginRequest struct {
	Identifier string `json:"identifier" binding:"required" example:"198001012010011001"` // NIP or email
	Password   string `json:"password" binding:"required" example:"s3cretPass!"`
}

// RegisterRequest creates a staff account. Role is ignored for the very first account,
// which always becomes a superadmin. sip_number, str_number and poli_id are required when the
// role is doctor and rejected for any other role.
type RegisterRequest struct {
	Name     string `json:"name" binding:"required,max=150" example:"Dr. Jane Doe"`
	NIP      string `json:"nip" binding:"required,max=30" example:"198001012010011001"`
	Email    string `json:"email" binding:"required,email,max=150" example:"jane.doe@hospital.id"`
	Password string `json:"password" binding:"required,min=8,max=72" example:"s3cretPass!"`
	Role     string `json:"role" example:"doctor"`

	SIPNumber string `json:"sip_number" binding:"required_if=Role doctor,excluded_unless=Role doctor,omitempty,license_number,min=5,max=50" example:"446/001/SIP-D/2026"`
	STRNumber string `json:"str_number" binding:"required_if=Role doctor,excluded_unless=Role doctor,omitempty,license_number,min=5,max=50" example:"3111100220123456"`
	PoliID    string `json:"poli_id" binding:"required_if=Role doctor,excluded_unless=Role doctor,omitempty,uuid" example:"3f2b8c1e-5d4a-4e7b-9a1c-2b6f0d8e4a71"`
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

type UserResponse struct {
	ID        string    `json:"id" example:"3f2b8c1e-5d4a-4e7b-9a1c-2b6f0d8e4a71"`
	Name      string    `json:"name" example:"Dr. Jane Doe"`
	NIP       string    `json:"nip" example:"198001012010011001"`
	Email     string    `json:"email" example:"jane.doe@hospital.id"`
	Role      string    `json:"role" example:"doctor"`
	IsActive  bool      `json:"is_active" example:"true"`
	CreatedAt time.Time `json:"created_at"`

	// Present only for doctors.
	DoctorProfile *DoctorProfileResponse `json:"doctor_profile,omitempty"`
}

type LoginResponse struct {
	AccessToken string       `json:"access_token"`
	TokenType   string       `json:"token_type" example:"Bearer"`
	ExpiresIn   int64        `json:"expires_in" example:"28800"` // seconds
	User        UserResponse `json:"user"`
}

func toUserResponse(u *models.User) UserResponse {
	r := UserResponse{
		ID:        u.ID.String(),
		Name:      u.Name,
		NIP:       u.NIP,
		Email:     u.Email,
		Role:      u.Role.Name,
		IsActive:  u.IsActive,
		CreatedAt: u.CreatedAt,
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
