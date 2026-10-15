package doctors

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"gin-quickstart/internal/platform/utils"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// @Summary		Create a doctor
// @Description	Superadmin only. The poli must exist and be active. SIP, STR and IHS numbers must be unique; ihs_number is optional. The poli's doctor count is updated automatically.
// @Tags			doctors
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			body	body		DoctorRequest	true	"Doctor data"
// @Success		201		{object}	utils.Response{data=Response}
// @Failure		400		{object}	utils.ErrorResponse
// @Failure		401		{object}	utils.ErrorResponse
// @Failure		403		{object}	utils.ErrorResponse
// @Failure		409		{object}	utils.ErrorResponse
// @Router			/api/v1/doctors [post]
func (h *Handler) Create(c *gin.Context) {
	var req DoctorRequest
	if !utils.BindJSON(c, &req) {
		return
	}
	d, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusCreated, "Doctor created successfully", toResponse(d))
}

// @Summary		List doctors
// @Description	Any signed-in user, so it can feed doctor dropdowns (use poli_id and is_active=true). Search by name, SIP, STR or IHS number.
// @Tags			doctors
// @Produce		json
// @Security		BearerAuth
// @Param			search			query		string	false	"Search by name, SIP, STR or IHS number"
// @Param			poli_id			query		string	false	"Only doctors of this poli"
// @Param			specialization	query		string	false	"Exact specialization (case-insensitive)"
// @Param			is_active		query		bool	false	"Filter by status"
// @Param			has_account		query		bool	false	"false = only doctors without a login account yet (for the account form)"
// @Param			page			query		int		false	"Page number"				default(1)
// @Param			limit			query		int		false	"Items per page (max 100)"	default(20)
// @Success		200				{object}	utils.PaginatedResponse{data=[]Response}
// @Failure		400				{object}	utils.ErrorResponse
// @Failure		401				{object}	utils.ErrorResponse
// @Router			/api/v1/doctors [get]
func (h *Handler) List(c *gin.Context) {
	var req ListRequest
	if !utils.BindQuery(c, &req) {
		return
	}
	q := ListQuery{Search: req.Search, PoliID: req.PoliID, Specialization: req.Specialization, IsActive: req.IsActive, HasAccount: req.HasAccount}
	q.Page, q.Limit = utils.ParsePagination(c)

	data, total, err := h.service.List(c.Request.Context(), q)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.SuccessPaginated(c, "Doctors retrieved successfully", data, utils.NewPagination(q.Page, q.Limit, total))
}

// @Summary		Get a doctor
// @Description	Any signed-in user.
// @Tags			doctors
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string	true	"Doctor ID"
// @Success		200	{object}	utils.Response{data=Response}
// @Failure		400	{object}	utils.ErrorResponse
// @Failure		401	{object}	utils.ErrorResponse
// @Failure		404	{object}	utils.ErrorResponse
// @Router			/api/v1/doctors/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	id, ok := utils.ParseUUIDParam(c, "id")
	if !ok {
		return
	}
	d, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Doctor retrieved successfully", toResponse(d))
}

// @Summary		Update a doctor
// @Description	Superadmin only. Replaces all editable fields; a missing is_active keeps the current status. Moving a doctor to another poli updates both polis' doctor counts. The new poli must be active.
// @Tags			doctors
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			id		path		string			true	"Doctor ID"
// @Param			body	body		DoctorRequest	true	"Doctor data"
// @Success		200		{object}	utils.Response{data=Response}
// @Failure		400		{object}	utils.ErrorResponse
// @Failure		401		{object}	utils.ErrorResponse
// @Failure		403		{object}	utils.ErrorResponse
// @Failure		404		{object}	utils.ErrorResponse
// @Failure		409		{object}	utils.ErrorResponse
// @Router			/api/v1/doctors/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	id, ok := utils.ParseUUIDParam(c, "id")
	if !ok {
		return
	}
	var req DoctorRequest
	if !utils.BindJSON(c, &req) {
		return
	}
	d, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Doctor updated successfully", toResponse(d))
}

// @Summary		Delete a doctor
// @Description	Superadmin only. A doctor that a user account is linked to cannot be deleted (409). The poli's doctor count is updated automatically.
// @Tags			doctors
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string	true	"Doctor ID"
// @Success		200	{object}	utils.Response
// @Failure		400	{object}	utils.ErrorResponse
// @Failure		401	{object}	utils.ErrorResponse
// @Failure		403	{object}	utils.ErrorResponse
// @Failure		404	{object}	utils.ErrorResponse
// @Failure		409	{object}	utils.ErrorResponse
// @Router			/api/v1/doctors/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	id, ok := utils.ParseUUIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Doctor deleted successfully", nil)
}

func (h *Handler) fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		utils.AbortWithError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrAlreadyExists), errors.Is(err, ErrInUse):
		utils.AbortWithError(c, http.StatusConflict, err.Error())
	case errors.Is(err, ErrPoliNotFound), errors.Is(err, ErrPoliInactive):
		utils.AbortWithError(c, http.StatusBadRequest, err.Error())
	default:
		utils.AbortWithError(c, http.StatusInternalServerError, "internal server error")
	}
}
