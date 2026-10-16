package cashier

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"gin-quickstart/internal/platform/utils"
)

// @Summary		List cashier shifts
// @Description	Cashier and management. Shifts opened on the date, newest first, each with what it received per payment method; for today, shifts still open from earlier are included.
// @Tags			cashier
// @Produce		json
// @Security		BearerAuth
// @Param			date	query		string	false	"Date (YYYY-MM-DD), default today"
// @Param			page	query		int		false	"Page number"				default(1)
// @Param			limit	query		int		false	"Items per page (max 100)"	default(20)
// @Success		200		{object}	utils.PaginatedResponse{data=[]ShiftResponse}
// @Failure		400		{object}	utils.ErrorResponse
// @Failure		401		{object}	utils.ErrorResponse
// @Failure		403		{object}	utils.ErrorResponse
// @Router			/api/v1/cashier/shifts [get]
func (h *Handler) ListShifts(c *gin.Context) {
	var req ShiftListRequest
	if !utils.BindQuery(c, &req) {
		return
	}
	date := utils.Today()
	if req.Date != "" {
		date, _ = time.Parse(time.DateOnly, req.Date) // already validated by the binding
	}
	page, limit := utils.ParsePagination(c)

	views, total, err := h.service.ListShifts(c.Request.Context(), date, page, limit)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.SuccessPaginated(c, "Shifts retrieved successfully", toShiftResponses(views), utils.NewPagination(page, limit, total))
}

// @Summary		Get my open shift
// @Description	Cashier only. The signed-in cashier's open shift, or data null when none is open. A shift opens with the first payment and again right after the previous one is closed.
// @Tags			cashier
// @Produce		json
// @Security		BearerAuth
// @Success		200	{object}	utils.Response{data=ShiftResponse}
// @Failure		401	{object}	utils.ErrorResponse
// @Failure		403	{object}	utils.ErrorResponse
// @Router			/api/v1/cashier/shifts/current [get]
func (h *Handler) CurrentShift(c *gin.Context) {
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	view, err := h.service.CurrentShift(c.Request.Context(), actor)
	if err != nil {
		h.fail(c, err)
		return
	}
	if view == nil {
		utils.Success(c, http.StatusOK, "No open shift", nil)
		return
	}
	utils.Success(c, http.StatusOK, "Shift retrieved successfully", toShiftResponse(view))
}

// @Summary		Close a shift
// @Description	Cashier only, for the cashier's own open shift. Records the counted cash and locks the shift's total; a note of at least 10 characters is required when the count differs from the recorded cash. The cashier's next shift opens right away.
// @Tags			cashier
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			id		path		string				true	"Shift ID"
// @Param			body	body		CloseShiftRequest	true	"Cash count"
// @Success		200		{object}	utils.Response{data=ShiftResponse}
// @Failure		400		{object}	utils.ErrorResponse
// @Failure		401		{object}	utils.ErrorResponse
// @Failure		403		{object}	utils.ErrorResponse
// @Failure		404		{object}	utils.ErrorResponse
// @Failure		409		{object}	utils.ErrorResponse
// @Router			/api/v1/cashier/shifts/{id}/close [post]
func (h *Handler) CloseShift(c *gin.Context) {
	id, actor, ok := h.request(c, "id")
	if !ok {
		return
	}
	var req CloseShiftRequest
	if !utils.BindJSON(c, &req) {
		return
	}
	view, err := h.service.CloseShift(c.Request.Context(), id, req, actor)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Shift closed successfully", toShiftResponse(view))
}
