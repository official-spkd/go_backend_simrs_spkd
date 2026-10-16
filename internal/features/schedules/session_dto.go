package schedules

// Session statuses.
const (
	StatusActive     = "active"
	StatusCompleted  = "completed"
	StatusLeave      = "leave"
	StatusSubstitute = "substitute"
)

type SessionListRequest struct {
	DoctorID string `form:"doctor_id" binding:"required,uuid"`
	From     string `form:"from" binding:"omitempty,datetime=2006-01-02"`
	To       string `form:"to" binding:"omitempty,datetime=2006-01-02"`
}

type SessionQuery struct {
	DoctorID string
	From     string
	To       string
}

// SessionResponse is one doctor's practice session on one date. A day with leave yields a
// "leave" row, plus a "substitute" row when another doctor takes over.
type SessionResponse struct {
	Date        string    `json:"date" example:"2026-10-13"`
	Doctor      DoctorRef `json:"doctor"`
	StartTime   string    `json:"start_time" example:"08:00"`
	EndTime     string    `json:"end_time" example:"12:00"`
	Status      string    `json:"status" enums:"active,completed,leave,substitute" example:"active"`
	TotalQuota  int       `json:"total_quota" example:"30"`
	JKNQuota    int       `json:"jkn_quota" example:"20"`
	NonJKNQuota int       `json:"non_jkn_quota" example:"10"`
	Filled      int       `json:"filled" example:"23"`
	Remaining   int       `json:"remaining" example:"7"`
	Reason      *string   `json:"reason" example:"Annual leave"`
	// ExceptionID is set on "leave" and "substitute" rows, to cancel the leave.
	ExceptionID *string `json:"exception_id" example:"7e2a9b14-5d3c-4b68-a1f0-3c8d6e2b9a47"`
	// ReplacedDoctor is the doctor on leave; only set on "substitute" rows.
	ReplacedDoctor *DoctorRef `json:"replaced_doctor"`
}
