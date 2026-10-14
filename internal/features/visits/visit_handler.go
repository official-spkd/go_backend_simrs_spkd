package visits

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

// @Summary		Register a patient to the queue
// @Description	Front Office only. The patient must already be registered; pass their patient_id. Queue numbers restart at 001 for each clinic and date. BPJS fields are required when penjamin is bpjs and insurance fields when it is asuransi; they are rejected for any other penjamin.
// @Tags			visits
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			body	body		CreateRequest	true	"Registration data"
// @Success		201		{object}	utils.Response{data=Response}
// @Failure		400		{object}	utils.ErrorResponse
// @Failure		401		{object}	utils.ErrorResponse
// @Failure		403		{object}	utils.ErrorResponse
// @Failure		404		{object}	utils.ErrorResponse
// @Failure		409		{object}	utils.ErrorResponse
// @Router			/api/v1/visits [post]
func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
	if !utils.BindJSON(c, &req) {
		return
	}
	v, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusCreated, "Patient registered to the queue successfully", toResponse(v))
}

// @Summary		List visits
// @Description	Front Office only. Newest dates first, then by clinic and queue number.
// @Tags			visits
// @Produce		json
// @Security		BearerAuth
// @Param			search				query		string	false	"Search by patient name, NIK or medical record number"
// @Param			date				query		string	false	"Visit date (YYYY-MM-DD)"
// @Param			poli				query		string	false	"Clinic name"
// @Param			penjamin			query		string	false	"Penjamin"				Enums(umum, bpjs, asuransi)
// @Param			visit_type			query		string	false	"Visit type"			Enums(baru, kontrol)
// @Param			visit_status		query		string	false	"Visit status"			Enums(Registered, Served, Completed, Paid, Cancelled)
// @Param			queue_status		query		string	false	"Clinic queue status"	Enums(Waiting, Called, Skipped, Done)
// @Param			satu_sehat_status	query		string	false	"Satu Sehat status"		Enums(NotSent, Sent)
// @Param			page				query		int		false	"Page number"			default(1)
// @Param			limit				query		int		false	"Items per page (max 100)"	default(20)
// @Success		200					{object}	utils.PaginatedResponse{data=[]Response}
// @Failure		400					{object}	utils.ErrorResponse
// @Failure		401					{object}	utils.ErrorResponse
// @Failure		403					{object}	utils.ErrorResponse
// @Router			/api/v1/visits [get]
func (h *Handler) List(c *gin.Context) {
	var req ListRequest
	if !utils.BindQuery(c, &req) {
		return
	}
	q := ListQuery{
		Search:          req.Search,
		Poli:            req.Poli,
		Penjamin:        req.Penjamin,
		VisitType:       req.VisitType,
		VisitStatus:     req.VisitStatus,
		QueueStatus:     req.QueueStatus,
		SatuSehatStatus: req.SatuSehatStatus,
	}
	if req.Date != "" {
		d, err := time.Parse(dateLayout, req.Date)
		if err != nil {
			utils.AbortWithError(c, http.StatusBadRequest, "date must use the format YYYY-MM-DD")
			return
		}
		q.Date = &d
	}
	q.Page, q.Limit = utils.ParsePagination(c)

	data, total, err := h.service.List(c.Request.Context(), q)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.SuccessPaginated(c, "Visits retrieved successfully", data, utils.NewPagination(q.Page, q.Limit, total))
}

// @Summary		Get a visit
// @Description	Front Office only.
// @Tags			visits
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string	true	"Visit ID"
// @Success		200	{object}	utils.Response{data=Response}
// @Failure		400	{object}	utils.ErrorResponse
// @Failure		401	{object}	utils.ErrorResponse
// @Failure		403	{object}	utils.ErrorResponse
// @Failure		404	{object}	utils.ErrorResponse
// @Router			/api/v1/visits/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	id, ok := utils.ParseUUIDParam(c, "id")
	if !ok {
		return
	}
	v, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Visit retrieved successfully", toResponse(v))
}

func (h *Handler) fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrPatientNotFound):
		utils.AbortWithError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrAlreadyRegistered):
		utils.AbortWithError(c, http.StatusConflict, err.Error())
	case errors.Is(err, ErrVisitDateInPast), errors.Is(err, ErrResponsiblePersonRequired):
		utils.AbortWithError(c, http.StatusBadRequest, err.Error())
	default:
		utils.AbortWithError(c, http.StatusInternalServerError, "internal server error")
	}
}
