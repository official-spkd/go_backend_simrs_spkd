package examinations

import (
	"context"

	"github.com/google/uuid"

	"gin-quickstart/internal/platform/utils"
)

func (s *service) History(ctx context.Context, visitID uuid.UUID, page, limit int, actor *utils.TokenClaims) ([]HistoryEntry, int64, error) {
	visit, err := s.authorize(ctx, visitID, actor, false)
	if err != nil {
		return nil, 0, err
	}
	visits, total, err := s.repo.ListHistory(ctx, visit.PatientID, visit.ID, page, limit)
	if err != nil {
		return nil, 0, err
	}

	ids := make([]uuid.UUID, len(visits))
	for i := range visits {
		ids[i] = visits[i].ID
	}
	notes, err := s.repo.FindFinalNotes(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	files, err := s.repo.FindAttachments(ctx, ids)
	if err != nil {
		return nil, 0, err
	}

	out := make([]HistoryEntry, len(visits))
	for i := range visits {
		v := &visits[i]
		out[i] = HistoryEntry{Visit: v, Note: notes[v.ID], Attachments: files[v.ID]}
	}
	return out, total, nil
}
