package polis

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

// @Summary		Create a poli
// @Description	Superadmin only. The prefix code (letters only, up to 5) is stored in uppercase and must be unique, as must the name.
// @Tags			polis
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			body	body		PoliRequest	true	"Poli data"
// @Success		201		{object}	utils.Response{data=Response}
// @Failure		400		{object}	utils.ErrorResponse
// @Failure		401		{object}	utils.ErrorResponse
// @Failure		403		{object}	utils.ErrorResponse
// @Failure		409		{object}	utils.ErrorResponse
// @Router			/api/v1/polis [post]
func (h *Handler) Create(c *gin.Context) {
	var req PoliRequest
	if !utils.BindJSON(c, &req) {
		return
	}
	p, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusCreated, "Poli created successfully", toResponse(p))
}

// @Summary		List polis
// @Description	Any signed-in user, so it can feed clinic dropdowns (use is_active=true for selectable clinics). Search by name, prefix code or room; filter by status.
// @Tags			polis
// @Produce		json
// @Security		BearerAuth
// @Param			search		query		string	false	"Search by name, prefix code or room"
// @Param			is_active	query		bool	false	"Filter by status"
// @Param			page		query		int		false	"Page number"				default(1)
// @Param			limit		query		int		false	"Items per page (max 100)"	default(20)
// @Success		200			{object}	utils.PaginatedResponse{data=[]Response}
// @Failure		400			{object}	utils.ErrorResponse
// @Failure		401			{object}	utils.ErrorResponse
// @Router			/api/v1/polis [get]
func (h *Handler) List(c *gin.Context) {
	var req ListRequest
	if !utils.BindQuery(c, &req) {
		return
	}
	q := ListQuery{Search: req.Search, IsActive: req.IsActive}
	q.Page, q.Limit = utils.ParsePagination(c)

	data, total, err := h.service.List(c.Request.Context(), q)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.SuccessPaginated(c, "Polis retrieved successfully", data, utils.NewPagination(q.Page, q.Limit, total))
}

// @Summary		Get a poli
// @Description	Any signed-in user.
// @Tags			polis
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string	true	"Poli ID"
// @Success		200	{object}	utils.Response{data=Response}
// @Failure		400	{object}	utils.ErrorResponse
// @Failure		401	{object}	utils.ErrorResponse
// @Failure		404	{object}	utils.ErrorResponse
// @Router			/api/v1/polis/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	id, ok := utils.ParseUUIDParam(c, "id")
	if !ok {
		return
	}
	p, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Poli retrieved successfully", toResponse(p))
}

// @Summary		Update a poli
// @Description	Superadmin only. Replaces all editable fields; a missing is_active keeps the current status.
// @Tags			polis
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			id		path		string		true	"Poli ID"
// @Param			body	body		PoliRequest	true	"Poli data"
// @Success		200		{object}	utils.Response{data=Response}
// @Failure		400		{object}	utils.ErrorResponse
// @Failure		401		{object}	utils.ErrorResponse
// @Failure		403		{object}	utils.ErrorResponse
// @Failure		404		{object}	utils.ErrorResponse
// @Failure		409		{object}	utils.ErrorResponse
// @Router			/api/v1/polis/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	id, ok := utils.ParseUUIDParam(c, "id")
	if !ok {
		return
	}
	var req PoliRequest
	if !utils.BindJSON(c, &req) {
		return
	}
	p, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Poli updated successfully", toResponse(p))
}

// @Summary		Delete a poli
// @Description	Superadmin only. A poli that doctors or visits refer to cannot be deleted (409). Existing visits keep their clinic name, so history is not affected.
// @Tags			polis
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string	true	"Poli ID"
// @Success		200	{object}	utils.Response
// @Failure		400	{object}	utils.ErrorResponse
// @Failure		401	{object}	utils.ErrorResponse
// @Failure		403	{object}	utils.ErrorResponse
// @Failure		404	{object}	utils.ErrorResponse
// @Failure		409	{object}	utils.ErrorResponse
// @Router			/api/v1/polis/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	id, ok := utils.ParseUUIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Poli deleted successfully", nil)
}

func (h *Handler) fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		utils.AbortWithError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrAlreadyExists), errors.Is(err, ErrInUse):
		utils.AbortWithError(c, http.StatusConflict, err.Error())
	default:
		utils.AbortWithError(c, http.StatusInternalServerError, "internal server error")
	}
}
