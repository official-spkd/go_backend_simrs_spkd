package users

import (
	"time"

	"gin-quickstart/internal/platform/models"
)

type UpdateRequest struct {
	Name  string `json:"name" binding:"required,max=150" example:"Dr. Jane Doe"`
	NIP   string `json:"nip" binding:"required,max=30" example:"198001012010011001"`
	Email string `json:"email" binding:"required,email,max=150" example:"jane.doe@hospital.id"`
}

// UpdateRoleRequest changes a user's role. Becoming a doctor needs doctor_id; leaving the doctor
// role clears it.
type UpdateRoleRequest struct {
	Role     string `json:"role" binding:"required" example:"doctor"`
	DoctorID string `json:"doctor_id" binding:"required_if=Role doctor,excluded_unless=Role doctor,omitempty,uuid" example:"9d1f6c2a-7b3e-4a58-8c1d-0e5f4b2a6d93"`
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

	// Present only for doctor accounts.
	DoctorID string `json:"doctor_id,omitempty" example:"9d1f6c2a-7b3e-4a58-8c1d-0e5f4b2a6d93"`
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
	if u.DoctorID != nil {
		r.DoctorID = u.DoctorID.String()
	}
	return r
}
