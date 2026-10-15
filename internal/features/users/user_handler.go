package users

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"gin-quickstart/internal/platform/middleware"
	"gin-quickstart/internal/platform/utils"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// @Summary		List users
// @Description	Superadmin only. Supports search by name, NIP or email, and filters by role and status.
// @Tags			users
// @Produce		json
// @Security		BearerAuth
// @Param			search		query		string	false	"Search by name, NIP or email"
// @Param			role		query		string	false	"Filter by role name"
// @Param			is_active	query		bool	false	"Filter by status"
// @Param			page		query		int		false	"Page number"		default(1)
// @Param			limit		query		int		false	"Items per page (max 100)"	default(20)
// @Success	200	{object}	utils.PaginatedResponse{data=[]Response}
// @Failure		400			{object}	utils.ErrorResponse
// @Failure		401			{object}	utils.ErrorResponse
// @Failure		403			{object}	utils.ErrorResponse
// @Router			/api/v1/users [get]
func (h *Handler) List(c *gin.Context) {
	q := ListQuery{Search: c.Query("search"), Role: c.Query("role")}
	q.Page, q.Limit = utils.ParsePagination(c)

	if v := c.Query("is_active"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			utils.AbortWithError(c, http.StatusBadRequest, "is_active must be true or false")
			return
		}
		q.IsActive = &b
	}

	users, total, err := h.service.List(c.Request.Context(), q)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.SuccessPaginated(c, "Users retrieved successfully", users, utils.NewPagination(q.Page, q.Limit, total))
}

// @Summary		Get a user
// @Description	Superadmin only.
// @Tags			users
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string	true	"User ID"
// @Success	200	{object}	utils.Response{data=Response}
// @Failure		400	{object}	utils.ErrorResponse
// @Failure		401	{object}	utils.ErrorResponse
// @Failure		403	{object}	utils.ErrorResponse
// @Failure		404	{object}	utils.ErrorResponse
// @Router			/api/v1/users/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	id, ok := utils.ParseUUIDParam(c, "id")
	if !ok {
		return
	}
	u, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "User retrieved successfully", toResponse(u))
}

// @Summary		Update a user's profile
// @Description	Superadmin only. Changes name, NIP and email.
// @Tags			users
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			id		path		string			true	"User ID"
// @Param			body	body		UpdateRequest	true	"Profile data"
// @Success	200	{object}	utils.Response{data=Response}
// @Failure		400		{object}	utils.ErrorResponse
// @Failure		401		{object}	utils.ErrorResponse
// @Failure		403		{object}	utils.ErrorResponse
// @Failure		404		{object}	utils.ErrorResponse
// @Failure		409		{object}	utils.ErrorResponse
// @Router			/api/v1/users/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	id, ok := utils.ParseUUIDParam(c, "id")
	if !ok {
		return
	}
	var req UpdateRequest
	if !utils.BindJSON(c, &req) {
		return
	}
	u, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "User updated successfully", toResponse(u))
}

// @Summary		Change a user's role
// @Description	Superadmin only. You cannot change your own role. Becoming a doctor requires doctor_id (an active doctor without an account); leaving the doctor role unlinks it.
// @Tags			users
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			id		path		string				true	"User ID"
// @Param			body	body		UpdateRoleRequest	true	"New role"
// @Success	200	{object}	utils.Response{data=Response}
// @Failure		400		{object}	utils.ErrorResponse
// @Failure		401		{object}	utils.ErrorResponse
// @Failure		403		{object}	utils.ErrorResponse
// @Failure		404		{object}	utils.ErrorResponse
// @Router			/api/v1/users/{id}/role [patch]
func (h *Handler) UpdateRole(c *gin.Context) {
	actorID, id, ok := h.actorAndTarget(c)
	if !ok {
		return
	}
	var req UpdateRoleRequest
	if !utils.BindJSON(c, &req) {
		return
	}
	u, err := h.service.UpdateRole(c.Request.Context(), actorID, id, req)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "User role updated successfully", toResponse(u))
}

// @Summary		Activate or deactivate a user
// @Description	Superadmin only. You cannot change your own status.
// @Tags			users
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			id		path		string				true	"User ID"
// @Param			body	body		UpdateStatusRequest	true	"New status"
// @Success	200	{object}	utils.Response{data=Response}
// @Failure		400		{object}	utils.ErrorResponse
// @Failure		401		{object}	utils.ErrorResponse
// @Failure		403		{object}	utils.ErrorResponse
// @Failure		404		{object}	utils.ErrorResponse
// @Router			/api/v1/users/{id}/status [patch]
func (h *Handler) UpdateStatus(c *gin.Context) {
	actorID, id, ok := h.actorAndTarget(c)
	if !ok {
		return
	}
	var req UpdateStatusRequest
	if !utils.BindJSON(c, &req) {
		return
	}
	u, err := h.service.UpdateStatus(c.Request.Context(), actorID, id, *req.IsActive)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "User status updated successfully", toResponse(u))
}

// @Summary		Reset a user's password
// @Description	Superadmin only. Sets a new password without needing the old one.
// @Tags			users
// @Accept			json
// @Security		BearerAuth
// @Param			id		path	string					true	"User ID"
// @Param			body	body	ResetPasswordRequest	true	"New password"
// @Success	200	{object}	utils.Response
// @Failure		400	{object}	utils.ErrorResponse
// @Failure		401	{object}	utils.ErrorResponse
// @Failure		403	{object}	utils.ErrorResponse
// @Failure		404	{object}	utils.ErrorResponse
// @Router			/api/v1/users/{id}/password [put]
func (h *Handler) ResetPassword(c *gin.Context) {
	id, ok := utils.ParseUUIDParam(c, "id")
	if !ok {
		return
	}
	var req ResetPasswordRequest
	if !utils.BindJSON(c, &req) {
		return
	}
	if err := h.service.ResetPassword(c.Request.Context(), id, req.NewPassword); err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Password reset successfully", nil)
}

// @Summary		Delete a user
// @Description	Superadmin only. You cannot delete your own account.
// @Tags			users
// @Security		BearerAuth
// @Param			id	path	string	true	"User ID"
// @Success	200	{object}	utils.Response
// @Failure		400	{object}	utils.ErrorResponse
// @Failure		401	{object}	utils.ErrorResponse
// @Failure		403	{object}	utils.ErrorResponse
// @Failure		404	{object}	utils.ErrorResponse
// @Router			/api/v1/users/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	actorID, id, ok := h.actorAndTarget(c)
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), actorID, id); err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "User deleted successfully", nil)
}

// actorAndTarget returns the signed-in user's ID and the target user's ID from the path.
func (h *Handler) actorAndTarget(c *gin.Context) (actorID, id uuid.UUID, ok bool) {
	claims, found := middleware.ClaimsFrom(c)
	if !found {
		utils.AbortWithError(c, http.StatusUnauthorized, "unauthenticated")
		return uuid.Nil, uuid.Nil, false
	}
	id, ok = utils.ParseUUIDParam(c, "id")
	return claims.UserID, id, ok
}

func (h *Handler) fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		utils.AbortWithError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrAlreadyExists):
		utils.AbortWithError(c, http.StatusConflict, err.Error())
	case errors.Is(err, ErrInvalidRole), errors.Is(err, ErrDoctorNotFound), errors.Is(err, ErrDoctorInactive):
		utils.AbortWithError(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrSelfAction):
		utils.AbortWithError(c, http.StatusForbidden, err.Error())
	default:
		utils.AbortWithError(c, http.StatusInternalServerError, "internal server error")
	}
}
