package queues

import (
	"time"

	"gin-quickstart/internal/platform/models"
)

const dateLayout = "2006-01-02"

type UpdateCounterStatusRequest struct {
	IsActive *bool `json:"is_active" binding:"required" example:"false"`
}

type ListRequest struct {
	Date      string              `form:"date" json:"date" binding:"omitempty,datetime=2006-01-02"`
	Status    models.TicketStatus `form:"status" json:"status" binding:"omitempty,oneof=WAITING CALLING SERVING COMPLETED SKIPPED"`
	CounterID string              `form:"counter_id" json:"counter_id" binding:"omitempty,uuid"`
	DoctorID  string              `form:"doctor_id" json:"doctor_id" binding:"omitempty,uuid"`
}

type ListQuery struct {
	Date      time.Time
	Status    models.TicketStatus
	CounterID string
	DoctorID  string
	Page      int
	Limit     int
}

type CounterRef struct {
	ID   string `json:"id" example:"3f2b8c1e-5d4a-4e7b-9a1c-2b6f0d8e4a71"`
	Name string `json:"name" example:"Loket 1"`
}

// PoliRef is looked up from polis. Visits registered before polis were managed only have a name;
// their id, prefix_code and room are empty.
type PoliRef struct {
	ID         string `json:"id" example:"3f2b8c1e-5d4a-4e7b-9a1c-2b6f0d8e4a71"`
	PrefixCode string `json:"prefix_code" example:"UMUM"`
	Name       string `json:"name" example:"General Clinic"`
	Room       string `json:"room" example:"Room 101"`
}

// DoctorRef is looked up from doctors. Visits registered before doctors were managed only have a
// name; their id and specialization are empty.
type DoctorRef struct {
	ID             string `json:"id" example:"9d1f6c2a-7b3e-4a58-8c1d-0e5f4b2a6d93"`
	Name           string `json:"name" example:"dr. Andi Wijaya, Sp.PD"`
	Specialization string `json:"specialization" example:"Internal Medicine"`
}

type PatientRef struct {
	MedicalRecordNumber string `json:"medical_record_number" example:"RM-000001"`
	FullName            string `json:"full_name" example:"Siti Rahmawati"`
}

type QueueResponse struct {
	ID          string              `json:"id" example:"9d1f6c2a-7b3e-4a58-8c1d-0e5f4b2a6d93"`
	QueueNumber string              `json:"queue_number" example:"A001"`
	QueueDate   string              `json:"queue_date" example:"2026-10-14"`
	Status      models.TicketStatus `json:"status" enums:"WAITING,CALLING,SERVING,COMPLETED,SKIPPED" example:"CALLING"`
	Counter     *CounterRef         `json:"counter"`

	VisitID         string     `json:"visit_id" example:"104dd35e-2b3a-4b2a-b38c-8bdd0e1ecc79"`
	Poli            PoliRef    `json:"poli"`
	Doctor          DoctorRef  `json:"doctor"`
	PoliQueueNumber string     `json:"poli_queue_number" example:"001"`
	Patient         PatientRef `json:"patient"`

	CalledAt         *time.Time `json:"called_at"`
	LastCalledAt     *time.Time `json:"last_called_at"`
	CallCount        int        `json:"call_count" example:"1"`
	ServingStartedAt *time.Time `json:"serving_started_at"`
	CompletedAt      *time.Time `json:"completed_at"`
	SkippedAt        *time.Time `json:"skipped_at"`
	CreatedAt        time.Time  `json:"created_at"`
}

// CounterState is a counter together with the queue it is handling right now.
type CounterState struct {
	ID       string         `json:"id" example:"3f2b8c1e-5d4a-4e7b-9a1c-2b6f0d8e4a71"`
	Name     string         `json:"name" example:"Loket 1"`
	IsActive bool           `json:"is_active" example:"true"`
	Current  *QueueResponse `json:"current"`
}

// DisplayQueue is what a waiting-room screen may show. It deliberately has no patient data.
type DisplayQueue struct {
	QueueNumber  string              `json:"queue_number" example:"A001"`
	Status       models.TicketStatus `json:"status" enums:"CALLING,SERVING" example:"CALLING"`
	CalledAt     *time.Time          `json:"called_at"`
	LastCalledAt *time.Time          `json:"last_called_at"`
	CallCount    int                 `json:"call_count" example:"2"`
}

type DisplayCounter struct {
	CounterID   string        `json:"counter_id" example:"3f2b8c1e-5d4a-4e7b-9a1c-2b6f0d8e4a71"`
	CounterName string        `json:"counter_name" example:"Loket 1"`
	IsActive    bool          `json:"is_active" example:"true"`
	Current     *DisplayQueue `json:"current"`
}

type DisplayResponse struct {
	Date         string           `json:"date" example:"2026-10-14"`
	WaitingCount int64            `json:"waiting_count" example:"7"`
	Counters     []DisplayCounter `json:"counters"`
	GeneratedAt  time.Time        `json:"generated_at"`
}

func toQueueResponse(q *models.Queue) QueueResponse {
	r := QueueResponse{
		ID:          q.ID.String(),
		QueueNumber: q.QueueNumber,
		QueueDate:   q.QueueDate.Format(dateLayout),
		Status:      q.Status,

		VisitID:         q.VisitID.String(),
		Poli:            poliRef(&q.Visit),
		Doctor:          doctorRef(&q.Visit),
		PoliQueueNumber: q.Visit.QueueNumber,
		Patient: PatientRef{
			MedicalRecordNumber: q.Visit.Patient.MedicalRecordNumber,
			FullName:            q.Visit.Patient.FullName,
		},

		CalledAt:         q.CalledAt,
		LastCalledAt:     q.LastCalledAt,
		CallCount:        q.CallCount,
		ServingStartedAt: q.ServingStartedAt,
		CompletedAt:      q.CompletedAt,
		SkippedAt:        q.SkippedAt,
		CreatedAt:        q.CreatedAt,
	}
	if q.Counter != nil {
		r.Counter = &CounterRef{ID: q.Counter.ID.String(), Name: q.Counter.Name}
	}
	return r
}

func toCounterState(s counterState) CounterState {
	out := CounterState{ID: s.Counter.ID.String(), Name: s.Counter.Name, IsActive: s.Counter.IsActive}
	if s.Current != nil {
		r := toQueueResponse(s.Current)
		out.Current = &r
	}
	return out
}

func toDisplayCounter(s counterState) DisplayCounter {
	out := DisplayCounter{CounterID: s.Counter.ID.String(), CounterName: s.Counter.Name, IsActive: s.Counter.IsActive}
	if q := s.Current; q != nil {
		out.Current = &DisplayQueue{
			QueueNumber:  q.QueueNumber,
			Status:       q.Status,
			CalledAt:     q.CalledAt,
			LastCalledAt: q.LastCalledAt,
			CallCount:    q.CallCount,
		}
	}
	return out
}

func poliRef(v *models.Visit) PoliRef {
	if v.Poli == nil {
		return PoliRef{Name: v.PoliName()}
	}
	return PoliRef{ID: v.Poli.ID.String(), PrefixCode: v.Poli.PrefixCode, Name: v.Poli.Name, Room: v.Poli.Room}
}

func doctorRef(v *models.Visit) DoctorRef {
	if v.Doctor == nil {
		return DoctorRef{Name: v.DoctorName()}
	}
	return DoctorRef{ID: v.Doctor.ID.String(), Name: v.Doctor.Name, Specialization: v.Doctor.Specialization}
}
