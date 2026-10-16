package examinations

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"gin-quickstart/internal/platform/middleware"
	"gin-quickstart/internal/platform/models"
	"gin-quickstart/internal/platform/utils"
)

// @Summary		Doctor worklist
// @Description	Doctor only. The signed-in doctor's visits of one date (default today) in poli queue order, cancelled visits excluded. Each row tells whether vital signs exist and the SOAP note status.
// @Tags			examinations
// @Produce		json
// @Security		BearerAuth
// @Param			date				query		string	false	"Visit date (YYYY-MM-DD), default today"
// @Param			search				query		string	false	"Search by patient name or medical record number"
// @Param			visit_status		query		string	false	"Visit status"			Enums(Registered, Served, Completed, Paid)
// @Param			poli_queue_status	query		string	false	"Poli queue status"		Enums(Waiting, Called, Skipped, Examining, Done)
// @Param			page				query		int		false	"Page number"			default(1)
// @Param			limit				query		int		false	"Items per page (max 100)"	default(20)
// @Success		200					{object}	utils.PaginatedResponse{data=[]WorklistItem}
// @Failure		400					{object}	utils.ErrorResponse
// @Failure		401					{object}	utils.ErrorResponse
// @Failure		403					{object}	utils.ErrorResponse
// @Router			/api/v1/examinations [get]
func (h *Handler) Worklist(c *gin.Context) {
	actor, ok := middleware.ClaimsFrom(c)
	if !ok {
		utils.AbortWithError(c, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var req WorklistRequest
	if !utils.BindQuery(c, &req) {
		return
	}
	q := WorklistQuery{Date: utils.Today(), Search: req.Search, VisitStatus: req.VisitStatus, PoliQueueStatus: req.PoliQueueStatus}
	if req.Date != "" {
		// Already validated by the binding tag.
		q.Date, _ = time.Parse(time.DateOnly, req.Date)
	}
	q.Page, q.Limit = utils.ParsePagination(c)

	entries, total, err := h.service.Worklist(c.Request.Context(), q, actor)
	if err != nil {
		h.fail(c, err)
		return
	}
	items := make([]WorklistItem, len(entries))
	for i, e := range entries {
		items[i] = toWorklistItem(e)
	}
	utils.SuccessPaginated(c, "Worklist retrieved successfully", items, utils.NewPagination(q.Page, q.Limit, total))
}

// @Summary		Call the patient
// @Description	Doctor only (own visits, today only). Calls a patient who is waiting, already called or skipped; the examination must not have started. Sets poli_queue_status to Called, records the call time and counts the call.
// @Tags			examinations
// @Produce		json
// @Security		BearerAuth
// @Param			visit_id	path		string	true	"Visit ID"
// @Success		200			{object}	utils.Response{data=VisitSummary}
// @Failure		400			{object}	utils.ErrorResponse
// @Failure		401			{object}	utils.ErrorResponse
// @Failure		403			{object}	utils.ErrorResponse
// @Failure		404			{object}	utils.ErrorResponse
// @Failure		409			{object}	utils.ErrorResponse
// @Router			/api/v1/examinations/{visit_id}/call [post]
func (h *Handler) Call(c *gin.Context) {
	h.flow(c, "Patient called successfully", h.service.Call)
}

// @Summary		Call the patient again
// @Description	Doctor only (own visits, today only). Repeats the call of a patient whose poli_queue_status is Called.
// @Tags			examinations
// @Produce		json
// @Security		BearerAuth
// @Param			visit_id	path		string	true	"Visit ID"
// @Success		200			{object}	utils.Response{data=VisitSummary}
// @Failure		400			{object}	utils.ErrorResponse
// @Failure		401			{object}	utils.ErrorResponse
// @Failure		403			{object}	utils.ErrorResponse
// @Failure		404			{object}	utils.ErrorResponse
// @Failure		409			{object}	utils.ErrorResponse
// @Router			/api/v1/examinations/{visit_id}/recall [post]
func (h *Handler) Recall(c *gin.Context) {
	h.flow(c, "Patient called again successfully", h.service.Recall)
}

// @Summary		Skip the patient
// @Description	Doctor only (own visits, today only). Marks a waiting or called patient as Skipped; they can be called again later.
// @Tags			examinations
// @Produce		json
// @Security		BearerAuth
// @Param			visit_id	path		string	true	"Visit ID"
// @Success		200			{object}	utils.Response{data=VisitSummary}
// @Failure		400			{object}	utils.ErrorResponse
// @Failure		401			{object}	utils.ErrorResponse
// @Failure		403			{object}	utils.ErrorResponse
// @Failure		404			{object}	utils.ErrorResponse
// @Failure		409			{object}	utils.ErrorResponse
// @Router			/api/v1/examinations/{visit_id}/skip [post]
func (h *Handler) Skip(c *gin.Context) {
	h.flow(c, "Patient skipped successfully", h.service.Skip)
}

// @Summary		Start the examination
// @Description	Doctor only (own visits, today only). Moves the visit from Registered to Served and poli_queue_status to Examining. The SOAP note can only be written after this.
// @Tags			examinations
// @Produce		json
// @Security		BearerAuth
// @Param			visit_id	path		string	true	"Visit ID"
// @Success		200			{object}	utils.Response{data=VisitSummary}
// @Failure		400			{object}	utils.ErrorResponse
// @Failure		401			{object}	utils.ErrorResponse
// @Failure		403			{object}	utils.ErrorResponse
// @Failure		404			{object}	utils.ErrorResponse
// @Failure		409			{object}	utils.ErrorResponse
// @Router			/api/v1/examinations/{visit_id}/start [post]
func (h *Handler) StartExam(c *gin.Context) {
	h.flow(c, "Examination started successfully", h.service.StartExam)
}

// @Summary		Close the visit
// @Description	Doctor only (own visits). Needs a started examination and a finalized SOAP note (409 otherwise); discharge_method follow_up also needs a control_date in the note. Moves the visit to Completed and poli_queue_status to Done, and records the discharge condition and method.
// @Tags			examinations
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			visit_id	path		string			true	"Visit ID"
// @Param			body		body		CloseRequest	true	"Discharge"
// @Success		200			{object}	utils.Response{data=VisitSummary}
// @Failure		400			{object}	utils.ErrorResponse
// @Failure		401			{object}	utils.ErrorResponse
// @Failure		403			{object}	utils.ErrorResponse
// @Failure		404			{object}	utils.ErrorResponse
// @Failure		409			{object}	utils.ErrorResponse
// @Router			/api/v1/examinations/{visit_id}/close [post]
func (h *Handler) Close(c *gin.Context) {
	visitID, actor, ok := h.request(c)
	if !ok {
		return
	}
	var req CloseRequest
	if !utils.BindJSON(c, &req) {
		return
	}
	v, err := h.service.Close(c.Request.Context(), visitID, req, actor)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Visit closed successfully", toVisitSummary(v))
}

type flowAction func(ctx context.Context, visitID uuid.UUID, actor *utils.TokenClaims) (*models.Visit, error)

// flow runs a body-less status change and responds with the updated visit.
func (h *Handler) flow(c *gin.Context, message string, action flowAction) {
	visitID, actor, ok := h.request(c)
	if !ok {
		return
	}
	v, err := action(c.Request.Context(), visitID, actor)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, message, toVisitSummary(v))
}
