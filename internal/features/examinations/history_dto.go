package examinations

import (
	"time"

	"gin-quickstart/internal/platform/models"
)

// HistoryEntry is another visit of the same patient with its finalized note and attachments.
type HistoryEntry struct {
	Visit       *models.Visit
	Note        *models.SoapNote
	Attachments []models.File
}

type AttachmentSummary struct {
	ID          string    `json:"id" example:"2c1f6c2a-7b3e-4a58-8c1d-0e5f4b2a6d93"`
	FileName    string    `json:"file_name" example:"control-notes.pdf"`
	ContentType string    `json:"content_type" example:"application/pdf"`
	SizeBytes   int64     `json:"size_bytes" example:"253952"`
	CreatedAt   time.Time `json:"created_at"`
}

// HistoryItem is one earlier or later visit. soap_note is null unless the note was finalized:
// drafts stay private to the doctor writing them.
type HistoryItem struct {
	Visit       VisitSummary        `json:"visit"`
	SoapNote    *SoapResponse       `json:"soap_note"`
	Attachments []AttachmentSummary `json:"attachments"`
}

func toHistoryItem(e HistoryEntry) HistoryItem {
	r := HistoryItem{Visit: toVisitSummary(e.Visit), Attachments: []AttachmentSummary{}}
	if e.Note != nil {
		sr := toSoapResponse(SoapView{Note: e.Note})
		r.SoapNote = &sr
	}
	for _, f := range e.Attachments {
		r.Attachments = append(r.Attachments, AttachmentSummary{
			ID: f.ID.String(), FileName: f.FileName, ContentType: f.ContentType, SizeBytes: f.SizeBytes, CreatedAt: f.CreatedAt,
		})
	}
	return r
}
