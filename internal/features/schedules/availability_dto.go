package schedules

type AvailabilityRequest struct {
	PoliID   string `form:"poli_id" binding:"omitempty,uuid"`
	DoctorID string `form:"doctor_id" binding:"omitempty,uuid"`
	From     string `form:"from" binding:"omitempty,datetime=2006-01-02"`
	To       string `form:"to" binding:"omitempty,datetime=2006-01-02"`
}

type AvailabilityQuery struct {
	PoliID   string
	DoctorID string
	From     string
	To       string
}

// AvailabilityResponse is what registration needs to know about one doctor on one date: when
// they practice and how much quota is left. Unlike a session list, each doctor and date has one row.
type AvailabilityResponse struct {
	Date       string    `json:"date" example:"2026-10-16"`
	Doctor     DoctorRef `json:"doctor"`
	StartTime  string    `json:"start_time" example:"08:00"`
	EndTime    string    `json:"end_time" example:"12:00"`
	Status     string    `json:"status" enums:"active,substitute,leave" example:"active"`
	TotalQuota int       `json:"total_quota" example:"30"`
	Filled     int       `json:"filled" example:"23"`
	Remaining  int       `json:"remaining" example:"7"`
	// ReplacedDoctor is the doctor on leave or without a session whom this doctor replaces; only on "substitute" rows.
	ReplacedDoctor *DoctorRef `json:"replaced_doctor"`
}
