package polis

import (
	"time"

	"gin-quickstart/internal/platform/models"
)

// PoliRequest is used for both create and update. On create, a missing is_active means true;
// on update it leaves the current value unchanged. doctor_count is not an input: it is
// maintained automatically from the active doctors assigned to the poli.
type PoliRequest struct {
	PrefixCode string `json:"prefix_code" binding:"required,alpha,min=1,max=5" example:"UMUM"`
	Name       string `json:"name" binding:"required,notblank,min=3,max=100" example:"General Clinic"`
	Room       string `json:"room" binding:"required,notblank,max=100" example:"Room 101"`
	IsActive   *bool  `json:"is_active" example:"true"`
}

type ListRequest struct {
	Search   string `form:"search" json:"search" binding:"omitempty,max=100"`
	IsActive *bool  `form:"is_active" json:"is_active"`
}

type ListQuery struct {
	Search   string
	IsActive *bool
	Page     int
	Limit    int
}

type Response struct {
	ID          string    `json:"id" example:"3f2b8c1e-5d4a-4e7b-9a1c-2b6f0d8e4a71"`
	PrefixCode  string    `json:"prefix_code" example:"UMUM"`
	Name        string    `json:"name" example:"General Clinic"`
	Room        string    `json:"room" example:"Room 101"`
	IsActive    bool      `json:"is_active" example:"true"`
	DoctorCount int       `json:"doctor_count" example:"3"` // read-only
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toResponse(p *models.Poli) Response {
	return Response{
		ID:          p.ID.String(),
		PrefixCode:  p.PrefixCode,
		Name:        p.Name,
		Room:        p.Room,
		IsActive:    p.IsActive,
		DoctorCount: p.DoctorCount,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}
