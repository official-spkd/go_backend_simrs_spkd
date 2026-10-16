package schedules

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"gin-quickstart/internal/platform/utils"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// @Summary		Create a weekly schedule
// @Description	Superadmin only. Creates the weekly template of one doctor for one weekday (0 = Sunday ... 6 = Saturday). The total quota is jkn_quota + non_jkn_quota and must be at least 1. Times use HH:MM and the end must be after the start. effective_date cannot be in the past. A doctor has at most one template per weekday.
// @Tags			schedules
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			body	body		CreateRequest	true	"Schedule data"
// @Success		201		{object}	utils.Response{data=Response}
// @Failure		400		{object}	utils.ErrorResponse
// @Failure		401		{object}	utils.ErrorResponse
// @Failure		403		{object}	utils.ErrorResponse
// @Failure		409		{object}	utils.ErrorResponse
// @Router			/api/v1/schedules [post]
func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
	if !utils.BindJSON(c, &req) {
		return
	}
	resp, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusCreated, "Schedule created successfully", resp)
}

// @Summary		List weekly schedules
// @Description	Superadmin only. Weekly templates ordered Monday to Sunday. min_quota is the highest number of registered visits on that weekday over the next 30 days; the total quota cannot go below it.
// @Tags			schedules
// @Produce		json
// @Security		BearerAuth
// @Param			doctor_id	query		string	false	"Only this doctor"
// @Param			poli_id		query		string	false	"Only doctors of this poli"
// @Param			weekday		query		int		false	"0 = Sunday ... 6 = Saturday"
// @Param			page		query		int		false	"Page number"				default(1)
// @Param			limit		query		int		false	"Items per page (max 100)"	default(20)
// @Success		200			{object}	utils.PaginatedResponse{data=[]Response}
// @Failure		400			{object}	utils.ErrorResponse
// @Failure		401			{object}	utils.ErrorResponse
// @Failure		403			{object}	utils.ErrorResponse
// @Router			/api/v1/schedules [get]
func (h *Handler) List(c *gin.Context) {
	var req ListRequest
	if !utils.BindQuery(c, &req) {
		return
	}
	q := ListQuery{DoctorID: req.DoctorID, PoliID: req.PoliID, Weekday: req.Weekday}
	q.Page, q.Limit = utils.ParsePagination(c)

	data, total, err := h.service.List(c.Request.Context(), q)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.SuccessPaginated(c, "Schedules retrieved successfully", data, utils.NewPagination(q.Page, q.Limit, total))
}

// @Summary		Get a weekly schedule
// @Description	Superadmin only.
// @Tags			schedules
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string	true	"Schedule ID"
// @Success		200	{object}	utils.Response{data=Response}
// @Failure		400	{object}	utils.ErrorResponse
// @Failure		401	{object}	utils.ErrorResponse
// @Failure		403	{object}	utils.ErrorResponse
// @Failure		404	{object}	utils.ErrorResponse
// @Router			/api/v1/schedules/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	id, ok := utils.ParseUUIDParam(c, "id")
	if !ok {
		return
	}
	resp, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Schedule retrieved successfully", resp)
}

// @Summary		Update a weekly schedule
// @Description	Superadmin only. Replaces times, quotas and effective date; the doctor and weekday cannot change. The total quota cannot be lower than the visits already registered on that weekday in the next 30 days (409); existing visits are never cancelled.
// @Tags			schedules
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			id		path		string			true	"Schedule ID"
// @Param			body	body		UpdateRequest	true	"Schedule data"
// @Success		200		{object}	utils.Response{data=Response}
// @Failure		400		{object}	utils.ErrorResponse
// @Failure		401		{object}	utils.ErrorResponse
// @Failure		403		{object}	utils.ErrorResponse
// @Failure		404		{object}	utils.ErrorResponse
// @Failure		409		{object}	utils.ErrorResponse
// @Router			/api/v1/schedules/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	id, ok := utils.ParseUUIDParam(c, "id")
	if !ok {
		return
	}
	var req UpdateRequest
	if !utils.BindJSON(c, &req) {
		return
	}
	resp, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Schedule updated successfully", resp)
}

// @Summary		Delete a weekly schedule
// @Description	Superadmin only. A schedule with visits registered on that weekday in the next 30 days cannot be deleted (409).
// @Tags			schedules
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string	true	"Schedule ID"
// @Success		200	{object}	utils.Response
// @Failure		400	{object}	utils.ErrorResponse
// @Failure		401	{object}	utils.ErrorResponse
// @Failure		403	{object}	utils.ErrorResponse
// @Failure		404	{object}	utils.ErrorResponse
// @Failure		409	{object}	utils.ErrorResponse
// @Router			/api/v1/schedules/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	id, ok := utils.ParseUUIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Schedule deleted successfully", nil)
}

func (h *Handler) fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrExceptionNotFound):
		utils.AbortWithError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrAlreadyExists), errors.Is(err, ErrExceptionExists), errors.Is(err, ErrBelowFilled),
		errors.Is(err, ErrInUse), errors.Is(err, ErrSubstituteRequired):
		utils.AbortWithError(c, http.StatusConflict, err.Error())
	case errors.Is(err, ErrDoctorNotFound), errors.Is(err, ErrDoctorInactive), errors.Is(err, ErrInvalidDate),
		errors.Is(err, ErrPastDate), errors.Is(err, ErrInvalidTime), errors.Is(err, ErrInvalidRange),
		errors.Is(err, ErrQuotaRequired), errors.Is(err, ErrNoSession), errors.Is(err, ErrSubstituteSame),
		errors.Is(err, ErrSubstituteUnavailable), errors.Is(err, ErrInvalidPeriod), errors.Is(err, ErrPeriodTooBig):
		utils.AbortWithError(c, http.StatusBadRequest, err.Error())
	default:
		utils.AbortWithError(c, http.StatusInternalServerError, "internal server error")
	}
}

// optionalDate converts a query date that binding already validated; empty means not given.
func optionalDate(raw string) *time.Time {
	d, err := time.Parse(dateLayout, raw)
	if err != nil {
		return nil
	}
	return &d
}
