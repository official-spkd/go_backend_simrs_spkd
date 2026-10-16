package examinations

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"gin-quickstart/internal/platform/models"
	"gin-quickstart/internal/platform/utils"
)

// Patients waiting at the poli can be called again after being called or skipped.
var callable = []models.PoliQueueStatus{models.PoliQueueWaiting, models.PoliQueueCalled, models.PoliQueueSkipped}

func (s *service) Worklist(ctx context.Context, q WorklistQuery, actor *utils.TokenClaims) ([]WorklistEntry, int64, error) {
	doctorID, err := s.linkedDoctor(ctx, actor)
	if err != nil {
		return nil, 0, err
	}
	q.DoctorID = doctorID
	q.Search = strings.TrimSpace(q.Search)

	visits, total, err := s.repo.ListWorklist(ctx, q)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]uuid.UUID, len(visits))
	for i := range visits {
		ids[i] = visits[i].ID
	}
	records, err := s.repo.FindRecords(ctx, ids)
	if err != nil {
		return nil, 0, err
	}

	out := make([]WorklistEntry, len(visits))
	for i := range visits {
		v := &visits[i]
		out[i] = WorklistEntry{Visit: v, HasVitals: records.HasVitals[v.ID], SoapStatus: records.SoapStatus[v.ID]}
	}
	return out, total, nil
}

func (s *service) Call(ctx context.Context, visitID uuid.UUID, actor *utils.TokenClaims) (*models.Visit, error) {
	visit, err := s.queueVisit(ctx, visitID, actor)
	if err != nil {
		return nil, err
	}
	if visit.VisitStatus != models.VisitRegistered {
		return nil, ErrCannotCall
	}
	return s.move(ctx, visitID, FlowGuard{VisitStatus: models.VisitRegistered, PoliQueue: callable}, FlowChange{
		Set:       map[string]any{"poli_queue_status": models.PoliQueueCalled, "poli_called_at": time.Now()},
		CountCall: true,
	})
}

func (s *service) Recall(ctx context.Context, visitID uuid.UUID, actor *utils.TokenClaims) (*models.Visit, error) {
	visit, err := s.queueVisit(ctx, visitID, actor)
	if err != nil {
		return nil, err
	}
	if visit.VisitStatus != models.VisitRegistered || visit.PoliQueueStatus != models.PoliQueueCalled {
		return nil, ErrNotCalled
	}
	guard := FlowGuard{VisitStatus: models.VisitRegistered, PoliQueue: []models.PoliQueueStatus{models.PoliQueueCalled}}
	return s.move(ctx, visitID, guard, FlowChange{Set: map[string]any{"poli_called_at": time.Now()}, CountCall: true})
}

func (s *service) Skip(ctx context.Context, visitID uuid.UUID, actor *utils.TokenClaims) (*models.Visit, error) {
	visit, err := s.queueVisit(ctx, visitID, actor)
	if err != nil {
		return nil, err
	}
	if visit.VisitStatus != models.VisitRegistered {
		return nil, ErrCannotCall
	}
	guard := FlowGuard{VisitStatus: models.VisitRegistered, PoliQueue: []models.PoliQueueStatus{models.PoliQueueWaiting, models.PoliQueueCalled}}
	return s.move(ctx, visitID, guard, FlowChange{Set: map[string]any{"poli_queue_status": models.PoliQueueSkipped}})
}

func (s *service) StartExam(ctx context.Context, visitID uuid.UUID, actor *utils.TokenClaims) (*models.Visit, error) {
	visit, err := s.queueVisit(ctx, visitID, actor)
	if err != nil {
		return nil, err
	}
	if visit.VisitStatus != models.VisitRegistered {
		return nil, ErrExamStarted
	}
	return s.move(ctx, visitID, FlowGuard{VisitStatus: models.VisitRegistered, PoliQueue: callable}, FlowChange{Set: map[string]any{
		"visit_status":      models.VisitServed,
		"poli_queue_status": models.PoliQueueExamining,
		"exam_started_at":   time.Now(),
	}})
}

func (s *service) Close(ctx context.Context, visitID uuid.UUID, req CloseRequest, actor *utils.TokenClaims) (*models.Visit, error) {
	visit, err := s.authorize(ctx, visitID, actor, true)
	if err != nil {
		return nil, err
	}
	if err := ensureExamining(visit); err != nil {
		return nil, err
	}

	note, err := s.repo.FindSoap(ctx, visitID)
	if errors.Is(err, ErrSoapNotFound) {
		return nil, ErrSoapNotFinal
	}
	if err != nil {
		return nil, err
	}
	if note.Status != models.SoapFinal {
		return nil, ErrSoapNotFinal
	}
	if req.DischargeMethod == models.DischargeFollowUp && note.ControlDate == nil {
		return nil, InputError("discharge_method follow_up needs a control_date in the soap note")
	}

	return s.move(ctx, visitID, FlowGuard{VisitStatus: models.VisitServed}, FlowChange{Set: map[string]any{
		"visit_status":        models.VisitCompleted,
		"poli_queue_status":   models.PoliQueueDone,
		"closed_at":           time.Now(),
		"discharge_condition": req.DischargeCondition,
		"discharge_method":    req.DischargeMethod,
	}})
}

// queueVisit authorizes a poli queue action, which only makes sense on the visit's own date.
func (s *service) queueVisit(ctx context.Context, visitID uuid.UUID, actor *utils.TokenClaims) (*models.Visit, error) {
	visit, err := s.authorize(ctx, visitID, actor, true)
	if err != nil {
		return nil, err
	}
	if visit.VisitDate.Format(time.DateOnly) != utils.Today().Format(time.DateOnly) {
		return nil, ErrQueueNotToday
	}
	return visit, nil
}

func (s *service) move(ctx context.Context, visitID uuid.UUID, guard FlowGuard, change FlowChange) (*models.Visit, error) {
	if err := s.repo.UpdateVisitFlow(ctx, visitID, guard, change); err != nil {
		return nil, err
	}
	return s.visits.Get(ctx, visitID)
}

// ensureExamining allows clinical writes only between "start examination" and "close visit".
func ensureExamining(v *models.Visit) error {
	switch v.VisitStatus {
	case models.VisitServed:
		return nil
	case models.VisitRegistered:
		return ErrExamNotStarted
	default:
		return ErrVisitClosed
	}
}
