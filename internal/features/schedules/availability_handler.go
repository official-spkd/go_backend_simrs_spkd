package schedules

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"gin-quickstart/internal/platform/utils"
)

// @Summary		List doctor availability
// @Description	Front office and superadmin. One row per active doctor and date from the weekly schedules: practice hours, total quota, registered visits and what is left. Defaults to 7 days from today; a period spans at most 31 days. A "leave" row has no quota; a "substitute" row counts the visits of the doctor on leave. Registration (POST /visits) refuses a doctor without a session, on leave, or with no quota left.
// @Tags			schedules
// @Produce		json
// @Security		BearerAuth
// @Param			poli_id		query		string	false	"Only doctors of this poli"
// @Param			doctor_id	query		string	false	"Only this doctor"
// @Param			from		query		string	false	"First date (YYYY-MM-DD), default today"
// @Param			to			query		string	false	"Last date (YYYY-MM-DD), default 6 days after from"
// @Success		200			{object}	utils.Response{data=[]AvailabilityResponse}
// @Failure		400			{object}	utils.ErrorResponse
// @Failure		401			{object}	utils.ErrorResponse
// @Failure		403			{object}	utils.ErrorResponse
// @Router			/api/v1/availability [get]
func (h *Handler) ListAvailability(c *gin.Context) {
	var req AvailabilityRequest
	if !utils.BindQuery(c, &req) {
		return
	}
	data, err := h.service.Availability(c.Request.Context(), AvailabilityQuery{PoliID: req.PoliID, DoctorID: req.DoctorID, From: req.From, To: req.To})
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Availability retrieved successfully", data)
}
