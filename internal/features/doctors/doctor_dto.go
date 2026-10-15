package doctors

import (
	"time"

	"gin-quickstart/internal/platform/models"
)

// DoctorRequest is used for both create and update. On create, a missing is_active means true;
// on update it leaves the current value unchanged.
type DoctorRequest struct {
	Name           string `json:"name" binding:"required,notblank,min=3,max=150,person_name" example:"dr. Andi Wijaya, Sp.PD"`
	Specialization string `json:"specialization" binding:"required,notblank,max=100" example:"Internal Medicine"`
	PoliID         string `json:"poli_id" binding:"required,uuid" example:"3f2b8c1e-5d4a-4e7b-9a1c-2b6f0d8e4a71"`
	SIPNumber      string `json:"sip_number" binding:"required,license_number,min=5,max=50" example:"446/001/SIP-D/2026"`
	STRNumber      string `json:"str_number" binding:"required,license_number,min=5,max=50" example:"3111100220123456"`
	IHSNumber      string `json:"ihs_number" binding:"omitempty,numeric,min=5,max=20" example:"10009880728"`
	IsActive       *bool  `json:"is_active" example:"true"`
}

type ListRequest struct {
	Search         string `form:"search" json:"search" binding:"omitempty,max=100"`
	PoliID         string `form:"poli_id" json:"poli_id" binding:"omitempty,uuid"`
	Specialization string `form:"specialization" json:"specialization" binding:"omitempty,max=100"`
	IsActive       *bool  `form:"is_active" json:"is_active"`
	HasAccount     *bool  `form:"has_account" json:"has_account"`
}

type ListQuery struct {
	Search         string
	PoliID         string
	Specialization string
	IsActive       *bool
	HasAccount     *bool
	Page           int
	Limit          int
}

type PoliRef struct {
	ID         string `json:"id" example:"3f2b8c1e-5d4a-4e7b-9a1c-2b6f0d8e4a71"`
	PrefixCode string `json:"prefix_code" example:"UMUM"`
	Name       string `json:"name" example:"General Clinic"`
	Room       string `json:"room" example:"Room 101"`
}

type Response struct {
	ID             string    `json:"id" example:"9d1f6c2a-7b3e-4a58-8c1d-0e5f4b2a6d93"`
	Name           string    `json:"name" example:"dr. Andi Wijaya, Sp.PD"`
	Specialization string    `json:"specialization" example:"Internal Medicine"`
	Poli           PoliRef   `json:"poli"`
	SIPNumber      string    `json:"sip_number" example:"446/001/SIP-D/2026"`
	STRNumber      string    `json:"str_number" example:"3111100220123456"`
	IHSNumber      *string   `json:"ihs_number" example:"10009880728"`
	IsActive       bool      `json:"is_active" example:"true"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func toResponse(d *models.Doctor) Response {
	r := Response{
		ID:             d.ID.String(),
		Name:           d.Name,
		Specialization: d.Specialization,
		SIPNumber:      d.SIPNumber,
		STRNumber:      d.STRNumber,
		IHSNumber:      d.IHSNumber,
		IsActive:       d.IsActive,
		CreatedAt:      d.CreatedAt,
		UpdatedAt:      d.UpdatedAt,
	}
	if d.Poli != nil {
		r.Poli = PoliRef{
			ID:         d.Poli.ID.String(),
			PrefixCode: d.Poli.PrefixCode,
			Name:       d.Poli.Name,
			Room:       d.Poli.Room,
		}
	}
	return r
}
