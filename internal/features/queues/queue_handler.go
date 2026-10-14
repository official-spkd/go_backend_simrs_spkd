package queues

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"gin-quickstart/internal/platform/models"
	"gin-quickstart/internal/platform/utils"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// @Summary		List counters
// @Description	Front Office only. Each counter comes with the queue it is handling right now (or null).
// @Tags			queues
// @Produce		json
// @Security		BearerAuth
// @Success		200	{object}	utils.Response{data=[]CounterState}
// @Failure		401	{object}	utils.ErrorResponse
// @Failure		403	{object}	utils.ErrorResponse
// @Router			/api/v1/counters [get]
func (h *Handler) ListCounters(c *gin.Context) {
	states, err := h.service.ListCounters(c.Request.Context())
	if err != nil {
		h.fail(c, err)
		return
	}
	data := make([]CounterState, len(states))
	for i := range states {
		data[i] = toCounterState(states[i])
	}
	utils.Success(c, http.StatusOK, "Counters retrieved successfully", data)
}

// @Summary		Activate or deactivate a counter
// @Description	Front Office only. A counter that is handling a queue cannot be deactivated.
// @Tags			queues
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			id		path		string						true	"Counter ID"
// @Param			body	body		UpdateCounterStatusRequest	true	"New status"
// @Success		200		{object}	utils.Response{data=CounterState}
// @Failure		400		{object}	utils.ErrorResponse
// @Failure		401		{object}	utils.ErrorResponse
// @Failure		403		{object}	utils.ErrorResponse
// @Failure		404		{object}	utils.ErrorResponse
// @Failure		409		{object}	utils.ErrorResponse
// @Router			/api/v1/counters/{id}/status [patch]
func (h *Handler) SetCounterStatus(c *gin.Context) {
	id, ok := utils.ParseUUIDParam(c, "id")
	if !ok {
		return
	}
	var req UpdateCounterStatusRequest
	if !utils.BindJSON(c, &req) {
		return
	}
	state, err := h.service.SetCounterActive(c.Request.Context(), id, *req.IsActive)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Counter status updated successfully", toCounterState(*state))
}

// @Summary		List queues
// @Description	Front Office only. Queues of one date (default today) in call order.
// @Tags			queues
// @Produce		json
// @Security		BearerAuth
// @Param			date		query		string	false	"Queue date (YYYY-MM-DD), default today"
// @Param			status		query		string	false	"Queue status"	Enums(WAITING, CALLING, SERVING, COMPLETED, SKIPPED)
// @Param			counter_id	query		string	false	"Counter ID"
// @Param			page		query		int		false	"Page number"				default(1)
// @Param			limit		query		int		false	"Items per page (max 100)"	default(20)
// @Success		200			{object}	utils.PaginatedResponse{data=[]QueueResponse}
// @Failure		400			{object}	utils.ErrorResponse
// @Failure		401			{object}	utils.ErrorResponse
// @Failure		403			{object}	utils.ErrorResponse
// @Router			/api/v1/queues [get]
func (h *Handler) ListQueues(c *gin.Context) {
	var req ListRequest
	if !utils.BindQuery(c, &req) {
		return
	}
	q := ListQuery{Date: utils.Today(), Status: req.Status, CounterID: req.CounterID}
	if req.Date != "" {
		d, err := time.Parse(dateLayout, req.Date)
		if err != nil {
			utils.AbortWithError(c, http.StatusBadRequest, "date must use the format YYYY-MM-DD")
			return
		}
		q.Date = d
	}
	q.Page, q.Limit = utils.ParsePagination(c)

	data, total, err := h.service.ListQueues(c.Request.Context(), q)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.SuccessPaginated(c, "Queues retrieved successfully", data, utils.NewPagination(q.Page, q.Limit, total))
}

// @Summary		Call the next queue
// @Description	Front Office only. Gives the oldest waiting queue of today to this counter (status CALLING). Concurrent counters never receive the same queue.
// @Tags			queues
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string	true	"Counter ID"
// @Success		200	{object}	utils.Response{data=QueueResponse}
// @Failure		400	{object}	utils.ErrorResponse
// @Failure		401	{object}	utils.ErrorResponse
// @Failure		403	{object}	utils.ErrorResponse
// @Failure		404	{object}	utils.ErrorResponse
// @Failure		409	{object}	utils.ErrorResponse
// @Router			/api/v1/counters/{id}/next [post]
func (h *Handler) Next(c *gin.Context) {
	h.act(c, "Next queue called successfully", h.service.Next)
}

// @Summary		Recall the current queue
// @Description	Front Office only. Refreshes last_called_at and call_count of the counter's CALLING queue so displays ring the bell again.
// @Tags			queues
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string	true	"Counter ID"
// @Success		200	{object}	utils.Response{data=QueueResponse}
// @Failure		400	{object}	utils.ErrorResponse
// @Failure		401	{object}	utils.ErrorResponse
// @Failure		403	{object}	utils.ErrorResponse
// @Failure		404	{object}	utils.ErrorResponse
// @Failure		409	{object}	utils.ErrorResponse
// @Router			/api/v1/counters/{id}/recall [post]
func (h *Handler) Recall(c *gin.Context) {
	h.act(c, "Queue recalled successfully", h.service.Recall)
}

// @Summary		Start serving the current queue
// @Description	Front Office only. The called patient is present: CALLING becomes SERVING.
// @Tags			queues
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string	true	"Counter ID"
// @Success		200	{object}	utils.Response{data=QueueResponse}
// @Failure		400	{object}	utils.ErrorResponse
// @Failure		401	{object}	utils.ErrorResponse
// @Failure		403	{object}	utils.ErrorResponse
// @Failure		404	{object}	utils.ErrorResponse
// @Failure		409	{object}	utils.ErrorResponse
// @Router			/api/v1/counters/{id}/serve [post]
func (h *Handler) Serve(c *gin.Context) {
	h.act(c, "Queue is now being served", h.service.Serve)
}

// @Summary		Complete the current queue
// @Description	Front Office only. A CALLING or SERVING queue becomes COMPLETED and the counter is free again.
// @Tags			queues
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string	true	"Counter ID"
// @Success		200	{object}	utils.Response{data=QueueResponse}
// @Failure		400	{object}	utils.ErrorResponse
// @Failure		401	{object}	utils.ErrorResponse
// @Failure		403	{object}	utils.ErrorResponse
// @Failure		404	{object}	utils.ErrorResponse
// @Failure		409	{object}	utils.ErrorResponse
// @Router			/api/v1/counters/{id}/complete [post]
func (h *Handler) Complete(c *gin.Context) {
	h.act(c, "Queue completed successfully", h.service.Complete)
}

// @Summary		Skip the current queue
// @Description	Front Office only. The called patient did not show up: CALLING becomes SKIPPED and the counter is free again.
// @Tags			queues
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string	true	"Counter ID"
// @Success		200	{object}	utils.Response{data=QueueResponse}
// @Failure		400	{object}	utils.ErrorResponse
// @Failure		401	{object}	utils.ErrorResponse
// @Failure		403	{object}	utils.ErrorResponse
// @Failure		404	{object}	utils.ErrorResponse
// @Failure		409	{object}	utils.ErrorResponse
// @Router			/api/v1/counters/{id}/skip [post]
func (h *Handler) Skip(c *gin.Context) {
	h.act(c, "Queue skipped successfully", h.service.Skip)
}

// @Summary		Waiting-room display data
// @Description	Public (no login) so TV screens can poll it. Shows only queue numbers and counters, never patient data. A changed last_called_at means the bell should ring again.
// @Tags			display
// @Produce		json
// @Success		200	{object}	utils.Response{data=DisplayResponse}
// @Router			/api/v1/display/queues [get]
func (h *Handler) Display(c *gin.Context) {
	d, err := h.service.Display(c.Request.Context())
	if err != nil {
		h.fail(c, err)
		return
	}
	counters := make([]DisplayCounter, len(d.Counters))
	for i := range d.Counters {
		counters[i] = toDisplayCounter(d.Counters[i])
	}
	utils.Success(c, http.StatusOK, "Display data retrieved successfully", DisplayResponse{
		Date:         d.Date.Format(dateLayout),
		WaitingCount: d.WaitingCount,
		Counters:     counters,
		GeneratedAt:  time.Now(),
	})
}

// act runs a queue action for the counter in the path and writes the resulting queue.
func (h *Handler) act(c *gin.Context, message string, action func(context.Context, uuid.UUID) (*models.Queue, error)) {
	id, ok := utils.ParseUUIDParam(c, "id")
	if !ok {
		return
	}
	q, err := action(c.Request.Context(), id)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, message, toQueueResponse(q))
}

func (h *Handler) fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrCounterNotFound), errors.Is(err, ErrQueueNotFound), errors.Is(err, ErrNoWaiting):
		utils.AbortWithError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrCounterBusy), errors.Is(err, ErrCounterInactive),
		errors.Is(err, ErrNoActiveQueue), errors.Is(err, ErrInvalidState):
		utils.AbortWithError(c, http.StatusConflict, err.Error())
	default:
		utils.AbortWithError(c, http.StatusInternalServerError, "internal server error")
	}
}
