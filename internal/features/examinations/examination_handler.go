package examinations

import (
	"errors"
	"net/http"

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

// @Summary		Get a visit's examination
// @Description	Doctor only, and only for visits assigned to the signed-in doctor. Returns the visit summary, values to prefill the form from, the vital signs, and the SOAP note (each null until saved).
// @Tags			examinations
// @Produce		json
// @Security		BearerAuth
// @Param			visit_id	path		string	true	"Visit ID"
// @Success		200			{object}	utils.Response{data=ExaminationResponse}
// @Failure		400			{object}	utils.ErrorResponse
// @Failure		401			{object}	utils.ErrorResponse
// @Failure		403			{object}	utils.ErrorResponse
// @Failure		404			{object}	utils.ErrorResponse
// @Router			/api/v1/examinations/{visit_id} [get]
func (h *Handler) Overview(c *gin.Context) {
	visitID, actor, ok := h.request(c)
	if !ok {
		return
	}
	o, err := h.service.Overview(c.Request.Context(), visitID, actor)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Examination retrieved successfully", toExaminationResponse(o))
}

// request reads the visit ID from the path and the signed-in user from the session.
func (h *Handler) request(c *gin.Context) (uuid.UUID, *utils.TokenClaims, bool) {
	visitID, ok := utils.ParseUUIDParam(c, "visit_id")
	if !ok {
		return uuid.Nil, nil, false
	}
	actor, ok := middleware.ClaimsFrom(c)
	if !ok {
		utils.AbortWithError(c, http.StatusUnauthorized, "unauthenticated")
		return uuid.Nil, nil, false
	}
	return visitID, actor, true
}

func (h *Handler) fail(c *gin.Context, err error) {
	var input InputError
	var incomplete IncompleteError

	switch {
	case errors.As(err, &input):
		utils.AbortWithError(c, http.StatusBadRequest, err.Error())
	case errors.As(err, &incomplete):
		utils.AbortWithError(c, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, ErrVisitNotFound), errors.Is(err, ErrVitalNotFound), errors.Is(err, ErrSoapNotFound):
		utils.AbortWithError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrDoctorNotLinked), errors.Is(err, ErrNotYourVisit):
		utils.AbortWithError(c, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrNoDoctor), errors.Is(err, ErrVisitCancelled), errors.Is(err, ErrVisitNotToday),
		errors.Is(err, ErrVersionConflict), errors.Is(err, ErrFinalized), errors.Is(err, ErrNotFinalized):
		utils.AbortWithError(c, http.StatusConflict, err.Error())
	default:
		utils.AbortWithError(c, http.StatusInternalServerError, "internal server error")
	}
}
