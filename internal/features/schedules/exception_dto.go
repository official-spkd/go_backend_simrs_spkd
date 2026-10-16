package schedules

import (
	"time"

	"gin-quickstart/internal/platform/models"
)

type ExceptionRequest struct {
	DoctorID           string `json:"doctor_id" binding:"required,uuid" example:"9d1f6c2a-7b3e-4a58-8c1d-0e5f4b2a6d93"`
	Date               string `json:"date" binding:"required,datetime=2006-01-02" example:"2026-10-15"`
	SubstituteDoctorID string `json:"substitute_doctor_id" binding:"omitempty,uuid" example:"1b7e4d90-3a6c-4f21-9d58-6c0a2e8f7b14"`
	Reason             string `json:"reason" binding:"required,notblank,max=200" example:"Annual leave"`
}

type ExceptionListRequest struct {
	DoctorID string `form:"doctor_id" binding:"omitempty,uuid"`
	From     string `form:"from" binding:"omitempty,datetime=2006-01-02"`
	To       string `form:"to" binding:"omitempty,datetime=2006-01-02"`
}

type ExceptionQuery struct {
	DoctorID string
	From     *time.Time
	To       *time.Time
	Page     int
	Limit    int
}

type ExceptionResponse struct {
	ID         string     `json:"id" example:"7e2a9b14-5d3c-4b68-a1f0-3c8d6e2b9a47"`
	Doctor     DoctorRef  `json:"doctor"`
	Date       string     `json:"date" example:"2026-10-15"`
	Substitute *DoctorRef `json:"substitute"`
	Reason     string     `json:"reason" example:"Annual leave"`
	CreatedAt  time.Time  `json:"created_at"`
}

func toExceptionResponse(e *models.ScheduleException) ExceptionResponse {
	r := ExceptionResponse{
		ID:        e.ID.String(),
		Doctor:    toDoctorRef(e.Doctor),
		Date:      e.Date.Format(dateLayout),
		Reason:    e.Reason,
		CreatedAt: e.CreatedAt,
	}
	if e.SubstituteDoctor != nil {
		ref := toDoctorRef(e.SubstituteDoctor)
		r.Substitute = &ref
	}
	return r
}
