package schedules

import (
	"time"

	"gin-quickstart/internal/platform/models"
)

const dateLayout = "2006-01-02"

// UpdateRequest holds the editable fields of a weekly template. The doctor and weekday identify
// a template and cannot change.
type UpdateRequest struct {
	StartTime     string `json:"start_time" binding:"required,len=5" example:"08:00"`
	EndTime       string `json:"end_time" binding:"required,len=5" example:"12:00"`
	JKNQuota      *int   `json:"jkn_quota" binding:"required,gte=0,lte=1000" example:"20"`
	NonJKNQuota   *int   `json:"non_jkn_quota" binding:"required,gte=0,lte=1000" example:"10"`
	EffectiveDate string `json:"effective_date" binding:"required,datetime=2006-01-02" example:"2026-10-01"`
}

type CreateRequest struct {
	DoctorID string `json:"doctor_id" binding:"required,uuid" example:"9d1f6c2a-7b3e-4a58-8c1d-0e5f4b2a6d93"`
	Weekday  *int   `json:"weekday" binding:"required,gte=0,lte=6" example:"2"`
	UpdateRequest
}

type ListRequest struct {
	DoctorID string `form:"doctor_id" binding:"omitempty,uuid"`
	PoliID   string `form:"poli_id" binding:"omitempty,uuid"`
	Weekday  *int   `form:"weekday" binding:"omitempty,gte=0,lte=6"`
}

type ListQuery struct {
	DoctorID string
	PoliID   string
	Weekday  *int
	Page     int
	Limit    int
}

type DoctorRef struct {
	ID             string `json:"id" example:"9d1f6c2a-7b3e-4a58-8c1d-0e5f4b2a6d93"`
	Name           string `json:"name" example:"dr. Maya Putri, Sp.PD"`
	Specialization string `json:"specialization" example:"Internal Medicine"`
	PoliID         string `json:"poli_id" example:"3f2b8c1e-5d4a-4e7b-9a1c-2b6f0d8e4a71"`
	PoliCode       string `json:"poli_code" example:"PD"`
	PoliName       string `json:"poli_name" example:"Internal Medicine"`
}

type Response struct {
	ID            string    `json:"id" example:"5c0e1a7d-2f4b-4c39-8a6e-7d1b3f9e0a52"`
	Doctor        DoctorRef `json:"doctor"`
	Weekday       int       `json:"weekday" example:"2"`
	StartTime     string    `json:"start_time" example:"08:00"`
	EndTime       string    `json:"end_time" example:"12:00"`
	JKNQuota      int       `json:"jkn_quota" example:"20"`
	NonJKNQuota   int       `json:"non_jkn_quota" example:"10"`
	TotalQuota    int       `json:"total_quota" example:"30"`
	EffectiveDate string    `json:"effective_date" example:"2026-10-01"`
	// MinQuota is the highest number of registered visits on this weekday over the next 30 days;
	// the total quota cannot go below it.
	MinQuota  int       `json:"min_quota" example:"23"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toDoctorRef(d *models.Doctor) DoctorRef {
	if d == nil {
		return DoctorRef{}
	}
	ref := DoctorRef{ID: d.ID.String(), Name: d.Name, Specialization: d.Specialization, PoliID: d.PoliID.String()}
	if d.Poli != nil {
		ref.PoliCode, ref.PoliName = d.Poli.PrefixCode, d.Poli.Name
	}
	return ref
}

func toResponse(s *models.DoctorSchedule, minQuota int) Response {
	return Response{
		ID:            s.ID.String(),
		Doctor:        toDoctorRef(s.Doctor),
		Weekday:       s.Weekday,
		StartTime:     s.StartTime,
		EndTime:       s.EndTime,
		JKNQuota:      s.JKNQuota,
		NonJKNQuota:   s.NonJKNQuota,
		TotalQuota:    s.TotalQuota(),
		EffectiveDate: s.EffectiveDate.Format(dateLayout),
		MinQuota:      minQuota,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}
}
