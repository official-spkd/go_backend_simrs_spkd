package schedules

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"gin-quickstart/internal/platform/utils"
)

// @Summary		List daily sessions
// @Description	Superadmin only. The doctor's sessions per date, derived from the weekly schedule and leaves (nothing is stored). Defaults to the next 30 days from today; a period can span at most 31 days. A leave gives a "leave" row with zero quota, plus a "substitute" row when another doctor takes over; the substitute row counts the visits of both doctors. filled is the number of non-cancelled visits, so every registration channel shares one quota.
// @Tags			schedules
// @Produce		json
// @Security		BearerAuth
// @Param			doctor_id	query		string	true	"Doctor ID"
// @Param			from		query		string	false	"First date (YYYY-MM-DD), default today"
// @Param			to			query		string	false	"Last date (YYYY-MM-DD), default 29 days after from"
// @Success		200			{object}	utils.Response{data=[]SessionResponse}
// @Failure		400			{object}	utils.ErrorResponse
// @Failure		401			{object}	utils.ErrorResponse
// @Failure		403			{object}	utils.ErrorResponse
// @Router			/api/v1/schedules/sessions [get]
func (h *Handler) ListSessions(c *gin.Context) {
	var req SessionListRequest
	if !utils.BindQuery(c, &req) {
		return
	}
	data, err := h.service.Sessions(c.Request.Context(), SessionQuery{DoctorID: req.DoctorID, From: req.From, To: req.To})
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Sessions retrieved successfully", data)
}
