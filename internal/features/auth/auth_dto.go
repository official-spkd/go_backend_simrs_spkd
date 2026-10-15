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
// which always becomes a superadmin. doctor_id is required when the role is doctor (the doctor
// record must exist first) and rejected for any other role.
type RegisterRequest struct {
	Name     string `json:"name" binding:"required,max=150" example:"Dr. Jane Doe"`
	NIP      string `json:"nip" binding:"required,max=30" example:"198001012010011001"`
	Email    string `json:"email" binding:"required,email,max=150" example:"jane.doe@hospital.id"`
	Password string `json:"password" binding:"required,min=8,max=72" example:"s3cretPass!"`
	Role     string `json:"role" example:"doctor"`

	DoctorID string `json:"doctor_id" binding:"required_if=Role doctor,excluded_unless=Role doctor,omitempty,uuid" example:"9d1f6c2a-7b3e-4a58-8c1d-0e5f4b2a6d93"`
}

type UserResponse struct {
	ID        string    `json:"id" example:"3f2b8c1e-5d4a-4e7b-9a1c-2b6f0d8e4a71"`
	Name      string    `json:"name" example:"Dr. Jane Doe"`
	NIP       string    `json:"nip" example:"198001012010011001"`
	Email     string    `json:"email" example:"jane.doe@hospital.id"`
	Role      string    `json:"role" example:"doctor"`
	IsActive  bool      `json:"is_active" example:"true"`
	CreatedAt time.Time `json:"created_at"`

	// Present only for doctor accounts.
	DoctorID string `json:"doctor_id,omitempty" example:"9d1f6c2a-7b3e-4a58-8c1d-0e5f4b2a6d93"`
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
	if u.DoctorID != nil {
		r.DoctorID = u.DoctorID.String()
	}
	return r
}
