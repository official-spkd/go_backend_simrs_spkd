package patients

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

// @Summary		Register a patient
// @Description	Front Office only. NIK must be unique and exactly 16 digits.
// @Tags			patients
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			body	body		PatientRequest	true	"Patient data"
// @Success		201		{object}	utils.Response{data=Response}
// @Failure		400		{object}	utils.ErrorResponse
// @Failure		401		{object}	utils.ErrorResponse
// @Failure		403		{object}	utils.ErrorResponse
// @Failure		409		{object}	utils.ErrorResponse
// @Router			/api/v1/patients [post]
func (h *Handler) Create(c *gin.Context) {
	var req PatientRequest
	if !utils.BindJSON(c, &req) {
		return
	}
	p, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusCreated, "Patient registered successfully", toResponse(p))
}

// @Summary		List patients
// @Description	Front Office only. Search by name, NIK or phone; filter by gender and payer.
// @Tags			patients
// @Produce		json
// @Security		BearerAuth
// @Param			search	query		string	false	"Search by name, NIK or phone"
// @Param			gender	query		string	false	"Filter by gender"	Enums(Male, Female)
// @Param			payer	query		string	false	"Filter by payer"	Enums(General, BPJS, Insurance)
// @Param			page	query		int		false	"Page number"		default(1)
// @Param			limit	query		int		false	"Items per page (max 100)"	default(20)
// @Success		200		{object}	utils.PaginatedResponse{data=[]Response}
// @Failure		400		{object}	utils.ErrorResponse
// @Failure		401		{object}	utils.ErrorResponse
// @Failure		403		{object}	utils.ErrorResponse
// @Router			/api/v1/patients [get]
func (h *Handler) List(c *gin.Context) {
	var req ListRequest
	if !utils.BindQuery(c, &req) {
		return
	}
	q := ListQuery{Search: req.Search, Gender: req.Gender, Payer: req.Payer}
	q.Page, q.Limit = utils.ParsePagination(c)

	data, total, err := h.service.List(c.Request.Context(), q)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.SuccessPaginated(c, "Patients retrieved successfully", data, utils.NewPagination(q.Page, q.Limit, total))
}

// @Summary		Get a patient
// @Description	Front Office only.
// @Tags			patients
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string	true	"Patient ID"
// @Success		200	{object}	utils.Response{data=Response}
// @Failure		400	{object}	utils.ErrorResponse
// @Failure		401	{object}	utils.ErrorResponse
// @Failure		403	{object}	utils.ErrorResponse
// @Failure		404	{object}	utils.ErrorResponse
// @Router			/api/v1/patients/{id} [get]
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
	utils.Success(c, http.StatusOK, "Patient retrieved successfully", toResponse(p))
}

// @Summary		Update a patient
// @Description	Front Office only. Replaces all editable fields.
// @Tags			patients
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			id		path		string			true	"Patient ID"
// @Param			body	body		PatientRequest	true	"Patient data"
// @Success		200		{object}	utils.Response{data=Response}
// @Failure		400		{object}	utils.ErrorResponse
// @Failure		401		{object}	utils.ErrorResponse
// @Failure		403		{object}	utils.ErrorResponse
// @Failure		404		{object}	utils.ErrorResponse
// @Failure		409		{object}	utils.ErrorResponse
// @Router			/api/v1/patients/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	id, ok := utils.ParseUUIDParam(c, "id")
	if !ok {
		return
	}
	var req PatientRequest
	if !utils.BindJSON(c, &req) {
		return
	}
	p, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Patient updated successfully", toResponse(p))
}

// @Summary		Delete a patient
// @Description	Front Office only. The record is soft-deleted and its NIK can be registered again.
// @Tags			patients
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string	true	"Patient ID"
// @Success		200	{object}	utils.Response
// @Failure		400	{object}	utils.ErrorResponse
// @Failure		401	{object}	utils.ErrorResponse
// @Failure		403	{object}	utils.ErrorResponse
// @Failure		404	{object}	utils.ErrorResponse
// @Router			/api/v1/patients/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	id, ok := utils.ParseUUIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Patient deleted successfully", nil)
}

func (h *Handler) fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		utils.AbortWithError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrAlreadyExists):
		utils.AbortWithError(c, http.StatusConflict, err.Error())
	case errors.Is(err, ErrInvalidBirthDate):
		utils.AbortWithError(c, http.StatusBadRequest, err.Error())
	default:
		utils.AbortWithError(c, http.StatusInternalServerError, "internal server error")
	}
}
