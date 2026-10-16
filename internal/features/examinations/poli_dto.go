package examinations

import (
	"time"

	"github.com/google/uuid"

	"gin-quickstart/internal/platform/models"
)

type WorklistRequest struct {
	Date            string                 `form:"date" binding:"omitempty,datetime=2006-01-02"`
	Search          string                 `form:"search" binding:"omitempty,max=100"`
	VisitStatus     models.VisitStatus     `form:"visit_status" binding:"omitempty,oneof=Registered Served Completed Paid"`
	PoliQueueStatus models.PoliQueueStatus `form:"poli_queue_status" binding:"omitempty,oneof=Waiting Called Skipped Examining Done"`
}

type WorklistQuery struct {
	DoctorID        uuid.UUID
	Date            time.Time
	Search          string
	VisitStatus     models.VisitStatus
	PoliQueueStatus models.PoliQueueStatus
	Page            int
	Limit           int
}

// WorklistEntry is a visit with what has been recorded for it so far.
type WorklistEntry struct {
	Visit      *models.Visit
	HasVitals  bool
	SoapStatus string
}

// WorklistItem is one row of the doctor's worklist. soap_status is empty until a draft exists.
type WorklistItem struct {
	VisitSummary
	HasVitalSigns bool   `json:"has_vital_signs" example:"true"`
	SoapStatus    string `json:"soap_status" enums:",draft,final" example:"draft"`
}

func toWorklistItem(e WorklistEntry) WorklistItem {
	return WorklistItem{VisitSummary: toVisitSummary(e.Visit), HasVitalSigns: e.HasVitals, SoapStatus: e.SoapStatus}
}

// CloseRequest ends the visit. follow_up needs a control date in the finalized SOAP note.
type CloseRequest struct {
	DischargeCondition string `json:"discharge_condition" binding:"required,oneof=recovered improved unchanged" enums:"recovered,improved,unchanged" example:"improved"`
	DischargeMethod    string `json:"discharge_method" binding:"required,oneof=home follow_up referred inpatient" enums:"home,follow_up,referred,inpatient" example:"follow_up"`
}
