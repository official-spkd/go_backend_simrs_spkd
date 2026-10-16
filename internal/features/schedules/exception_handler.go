package schedules

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"gin-quickstart/internal/platform/utils"
)

// @Summary		Record a leave
// @Description	Superadmin only. Closes the doctor's session on a date (today or later) and optionally hands it to a substitute doctor. The doctor must have a schedule on that date. A substitute must be a different, active doctor who practices on that weekday and is not on leave that day. When visits are already registered, a substitute is required (409). Visits are not moved or cancelled; the substitute's session counts them against its quota.
// @Tags			schedules
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			body	body		ExceptionRequest	true	"Leave data"
// @Success		201		{object}	utils.Response{data=ExceptionResponse}
// @Failure		400		{object}	utils.ErrorResponse
// @Failure		401		{object}	utils.ErrorResponse
// @Failure		403		{object}	utils.ErrorResponse
// @Failure		409		{object}	utils.ErrorResponse
// @Router			/api/v1/schedules/exceptions [post]
func (h *Handler) CreateException(c *gin.Context) {
	var req ExceptionRequest
	if !utils.BindJSON(c, &req) {
		return
	}
	resp, err := h.service.CreateException(c.Request.Context(), req)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusCreated, "Schedule exception created successfully", resp)
}

// @Summary		List leaves
// @Description	Superadmin only. Leave exceptions ordered by date, optionally for one doctor and a date range.
// @Tags			schedules
// @Produce		json
// @Security		BearerAuth
// @Param			doctor_id	query		string	false	"Only this doctor"
// @Param			from		query		string	false	"First date (YYYY-MM-DD)"
// @Param			to			query		string	false	"Last date (YYYY-MM-DD)"
// @Param			page		query		int		false	"Page number"				default(1)
// @Param			limit		query		int		false	"Items per page (max 100)"	default(20)
// @Success		200			{object}	utils.PaginatedResponse{data=[]ExceptionResponse}
// @Failure		400			{object}	utils.ErrorResponse
// @Failure		401			{object}	utils.ErrorResponse
// @Failure		403			{object}	utils.ErrorResponse
// @Router			/api/v1/schedules/exceptions [get]
func (h *Handler) ListExceptions(c *gin.Context) {
	var req ExceptionListRequest
	if !utils.BindQuery(c, &req) {
		return
	}
	q := ExceptionQuery{DoctorID: req.DoctorID, From: optionalDate(req.From), To: optionalDate(req.To)}
	q.Page, q.Limit = utils.ParsePagination(c)

	data, total, err := h.service.ListExceptions(c.Request.Context(), q)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.SuccessPaginated(c, "Schedule exceptions retrieved successfully", data, utils.NewPagination(q.Page, q.Limit, total))
}

// @Summary		Cancel a leave
// @Description	Superadmin only. Reopens the doctor's session. Only today's and future exceptions can be cancelled (400 for past ones).
// @Tags			schedules
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string	true	"Exception ID"
// @Success		200	{object}	utils.Response
// @Failure		400	{object}	utils.ErrorResponse
// @Failure		401	{object}	utils.ErrorResponse
// @Failure		403	{object}	utils.ErrorResponse
// @Failure		404	{object}	utils.ErrorResponse
// @Router			/api/v1/schedules/exceptions/{id} [delete]
func (h *Handler) DeleteException(c *gin.Context) {
	id, ok := utils.ParseUUIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.service.DeleteException(c.Request.Context(), id); err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Schedule exception deleted successfully", nil)
}
