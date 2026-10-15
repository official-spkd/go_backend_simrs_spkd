package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"gin-quickstart/internal/platform/middleware"
	"gin-quickstart/internal/platform/utils"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// @Summary	Sign in with NIP or email
// @Tags		auth
// @Accept		json
// @Produce	json
// @Param		body	body		LoginRequest	true	"Credentials"
// @Success	200	{object}	utils.Response{data=LoginResponse}
// @Failure	400		{object}	utils.ErrorResponse
// @Failure	401		{object}	utils.ErrorResponse
// @Failure	403		{object}	utils.ErrorResponse
// @Router		/api/v1/auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if !utils.BindJSON(c, &req) {
		return
	}

	res, err := h.service.Login(c.Request.Context(), req)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Login successful", res)
}

// @Summary		Register a user
// @Description	Open for the very first account (becomes superadmin). Afterwards only a superadmin can register accounts. For role doctor, doctor_id (an active doctor from GET /doctors?has_account=false, one account per doctor) is required; for other roles it is rejected.
// @Tags			auth
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			body	body		RegisterRequest	true	"User data"
// @Success	201	{object}	utils.Response{data=UserResponse}
// @Failure		400		{object}	utils.ErrorResponse
// @Failure		401		{object}	utils.ErrorResponse
// @Failure		403		{object}	utils.ErrorResponse
// @Failure		409		{object}	utils.ErrorResponse
// @Router			/api/v1/auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if !utils.BindJSON(c, &req) {
		return
	}

	actor, _ := middleware.ClaimsFrom(c)
	u, err := h.service.Register(c.Request.Context(), req, actor)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusCreated, "User registered successfully", toUserResponse(u))
}

// @Summary	Get the signed-in user
// @Tags		auth
// @Produce	json
// @Security	BearerAuth
// @Success	200	{object}	utils.Response{data=UserResponse}
// @Failure	401	{object}	utils.ErrorResponse
// @Router		/api/v1/auth/me [get]
func (h *Handler) Me(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		utils.AbortWithError(c, http.StatusUnauthorized, "unauthenticated")
		return
	}

	u, err := h.service.Me(c.Request.Context(), claims.UserID)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Profile retrieved successfully", toUserResponse(u))
}

func (h *Handler) fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidCredentials), errors.Is(err, ErrUnauthenticated), errors.Is(err, ErrUserNotFound):
		utils.AbortWithError(c, http.StatusUnauthorized, err.Error())
	case errors.Is(err, ErrInactiveAccount), errors.Is(err, ErrForbidden):
		utils.AbortWithError(c, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrRoleRequired), errors.Is(err, ErrInvalidRole),
		errors.Is(err, ErrDoctorNotFound), errors.Is(err, ErrDoctorInactive):
		utils.AbortWithError(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrAlreadyExists):
		utils.AbortWithError(c, http.StatusConflict, err.Error())
	default:
		utils.AbortWithError(c, http.StatusInternalServerError, "internal server error")
	}
}
