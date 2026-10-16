package cashier

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

// request reads a UUID path parameter and the signed-in user from the session.
func (h *Handler) request(c *gin.Context, param string) (uuid.UUID, *utils.TokenClaims, bool) {
	id, ok := utils.ParseUUIDParam(c, param)
	if !ok {
		return uuid.Nil, nil, false
	}
	actor, ok := h.actor(c)
	return id, actor, ok
}

func (h *Handler) actor(c *gin.Context) (*utils.TokenClaims, bool) {
	actor, ok := middleware.ClaimsFrom(c)
	if !ok {
		utils.AbortWithError(c, http.StatusUnauthorized, "unauthenticated")
	}
	return actor, ok
}

func (h *Handler) fail(c *gin.Context, err error) {
	var input InputError

	switch {
	case errors.As(err, &input):
		utils.AbortWithError(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrVisitNotFound), errors.Is(err, ErrBillNotFound), errors.Is(err, ErrShiftNotFound):
		utils.AbortWithError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrNotYourShift):
		utils.AbortWithError(c, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrVisitCancelled), errors.Is(err, ErrVisitNotCompleted), errors.Is(err, ErrBillNotDraft),
		errors.Is(err, ErrBillNotFinal), errors.Is(err, ErrBillPaid), errors.Is(err, ErrShiftClosed), errors.Is(err, ErrShiftChanged):
		utils.AbortWithError(c, http.StatusConflict, err.Error())
	default:
		utils.AbortWithError(c, http.StatusInternalServerError, "internal server error")
	}
}
