package models

import (
	"time"

	"github.com/google/uuid"
)

// Counter is a registration desk (loket) that calls patients from the queue.
type Counter struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name      string    `gorm:"size:50;not null;uniqueIndex"`
	IsActive  bool      `gorm:"not null;default:true"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type TicketStatus string

const (
	TicketWaiting   TicketStatus = "WAITING"
	TicketCalling   TicketStatus = "CALLING"
	TicketServing   TicketStatus = "SERVING"
	TicketCompleted TicketStatus = "COMPLETED"
	TicketSkipped   TicketStatus = "SKIPPED"
)

// Queue is a patient's place in the registration desk line. Every visit gets one entry.
// Numbers run per day across all clinics ("A001", "A002", ...), unlike Visit.QueueNumber,
// which restarts for each clinic.
type Queue struct {
	ID      uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	VisitID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	Visit   Visit     `gorm:"foreignKey:VisitID"`

	// A counter has at most one active (CALLING or SERVING) queue at any time.
	CounterID *uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_queues_active_counter,where:status = 'CALLING' OR status = 'SERVING'"`
	Counter   *Counter   `gorm:"foreignKey:CounterID"`

	QueueDate     time.Time    `gorm:"type:date;not null;uniqueIndex:idx_queues_date_sequence,priority:1;index:idx_queues_date_status,priority:1"`
	QueueSequence int          `gorm:"not null;uniqueIndex:idx_queues_date_sequence,priority:2"`
	QueueNumber   string       `gorm:"size:10;not null"`
	Status        TicketStatus `gorm:"size:20;not null;default:WAITING;index:idx_queues_date_status,priority:2"`

	CalledAt         *time.Time
	LastCalledAt     *time.Time // refreshed by every recall so displays can re-trigger the bell
	CallCount        int        `gorm:"not null;default:0"`
	ServingStartedAt *time.Time
	CompletedAt      *time.Time
	SkippedAt        *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}
