package roles

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

// @Summary	List roles
// @Description	Superadmin only.
// @Tags		roles
// @Produce	json
// @Security	BearerAuth
// @Success	200	{object}	utils.Response{data=[]models.Role}
// @Failure	401	{object}	utils.ErrorResponse
// @Failure	403	{object}	utils.ErrorResponse
// @Router		/api/v1/roles [get]
func (h *Handler) List(c *gin.Context) {
	out, err := h.service.List(c.Request.Context())
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Roles retrieved successfully", out)
}

// @Summary	Get a role
// @Description	Superadmin only.
// @Tags		roles
// @Produce	json
// @Param		id	path		string	true	"Role ID"
// @Security	BearerAuth
// @Success	200	{object}	utils.Response{data=models.Role}
// @Failure	400	{object}	utils.ErrorResponse
// @Failure	401	{object}	utils.ErrorResponse
// @Failure	403	{object}	utils.ErrorResponse
// @Failure	404	{object}	utils.ErrorResponse
// @Router		/api/v1/roles/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	id, ok := utils.ParseUUIDParam(c, "id")
	if !ok {
		return
	}
	role, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Role retrieved successfully", role)
}

// @Summary	Create a role
// @Description	Superadmin only.
// @Tags		roles
// @Accept		json
// @Produce	json
// @Param		body	body		CreateRequest	true	"Role data"
// @Security	BearerAuth
// @Success	201	{object}	utils.Response{data=models.Role}
// @Failure	400		{object}	utils.ErrorResponse
// @Failure	401	{object}	utils.ErrorResponse
// @Failure	403	{object}	utils.ErrorResponse
// @Failure	409		{object}	utils.ErrorResponse
// @Router		/api/v1/roles [post]
func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.AbortWithError(c, http.StatusBadRequest, err.Error())
		return
	}
	role, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusCreated, "Role created successfully", role)
}

// @Summary	Update a role's display name and description
// @Description	Superadmin only.
// @Tags		roles
// @Accept		json
// @Produce	json
// @Param		id		path		string			true	"Role ID"
// @Param		body	body		UpdateRequest	true	"Role data"
// @Security	BearerAuth
// @Success	200	{object}	utils.Response{data=models.Role}
// @Failure	400		{object}	utils.ErrorResponse
// @Failure	401	{object}	utils.ErrorResponse
// @Failure	403	{object}	utils.ErrorResponse
// @Failure	404		{object}	utils.ErrorResponse
// @Router		/api/v1/roles/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	id, ok := utils.ParseUUIDParam(c, "id")
	if !ok {
		return
	}
	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.AbortWithError(c, http.StatusBadRequest, err.Error())
		return
	}
	role, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Role updated successfully", role)
}

// @Summary	Delete a role
// @Description	Superadmin only.
// @Tags		roles
// @Param		id	path	string	true	"Role ID"
// @Security	BearerAuth
// @Success	200	{object}	utils.Response
// @Failure	400	{object}	utils.ErrorResponse
// @Failure	401	{object}	utils.ErrorResponse
// @Failure	403	{object}	utils.ErrorResponse
// @Failure	404	{object}	utils.ErrorResponse
// @Failure	409	{object}	utils.ErrorResponse
// @Router		/api/v1/roles/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	id, ok := utils.ParseUUIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Role deleted successfully", nil)
}

func (h *Handler) fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		utils.AbortWithError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrAlreadyExists), errors.Is(err, ErrInUse):
		utils.AbortWithError(c, http.StatusConflict, err.Error())
	case errors.Is(err, ErrInvalidName):
		utils.AbortWithError(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrSystemRole):
		utils.AbortWithError(c, http.StatusForbidden, err.Error())
	default:
		utils.AbortWithError(c, http.StatusInternalServerError, "internal server error")
	}
}
