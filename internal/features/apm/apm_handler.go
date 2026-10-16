package apm

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"gin-quickstart/internal/features/patients"
	"gin-quickstart/internal/features/visits"
	"gin-quickstart/internal/platform/models"
	"gin-quickstart/internal/platform/utils"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// @Summary		APM: find a returning patient
// @Description	Public (no login), for the self-service registration kiosk (APM). Send exactly one of nik, medical_record_number (RM-000001) or bpjs_number. Returns only what the kiosk needs to confirm the person (no NIK, contact, address or allergies). Strictly rate limited per IP (10 per minute).
// @Tags			apm
// @Produce		json
// @Param			nik						query		string	false	"16-digit NIK"
// @Param			medical_record_number	query		string	false	"Medical record number, e.g. RM-000001"
// @Param			bpjs_number				query		string	false	"13-digit BPJS card number"
// @Success		200						{object}	utils.Response{data=PatientBrief}
// @Failure		400						{object}	utils.ErrorResponse
// @Failure		404						{object}	utils.ErrorResponse
// @Failure		429						{object}	utils.ErrorResponse
// @Router			/api/v1/apm/patients/lookup [get]
func (h *Handler) LookupPatient(c *gin.Context) {
	var req LookupRequest
	if !utils.BindQuery(c, &req) {
		return
	}
	p, err := h.service.LookupPatient(c.Request.Context(), req)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Patient found", p)
}

// @Summary		APM: list polis
// @Description	Public (no login). Active polis only, with only the fields a kiosk shows.
// @Tags			apm
// @Produce		json
// @Success		200	{object}	utils.Response{data=[]PoliBrief}
// @Failure		429	{object}	utils.ErrorResponse
// @Router			/api/v1/apm/polis [get]
func (h *Handler) ListPolis(c *gin.Context) {
	data, err := h.service.ListPolis(c.Request.Context())
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Polis retrieved successfully", data)
}

// @Summary		APM: list doctors
// @Description	Public (no login). Active doctors of active polis, optionally of one poli. No license or account data is exposed.
// @Tags			apm
// @Produce		json
// @Param			poli_id	query		string	false	"Only doctors of this poli"
// @Success		200		{object}	utils.Response{data=[]DoctorBrief}
// @Failure		400		{object}	utils.ErrorResponse
// @Failure		429		{object}	utils.ErrorResponse
// @Router			/api/v1/apm/doctors [get]
func (h *Handler) ListDoctors(c *gin.Context) {
	var req DoctorListRequest
	if !utils.BindQuery(c, &req) {
		return
	}
	data, err := h.service.ListDoctors(c.Request.Context(), req.PoliID)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Doctors retrieved successfully", data)
}

// @Summary		APM: register a new patient
// @Description	Public (no login). Creates a patient record with the same fields and validation as the front desk (NIK must be a valid, unused 16-digit number; bpjs_number optional). Returns the brief patient (id and No. RM) to pass to POST /apm/visits. An existing NIK or BPJS number is refused with 409: that person is a returning patient and should be looked up instead. Strictly rate limited per IP (5 per minute).
// @Tags			apm
// @Accept			json
// @Produce		json
// @Param			body	body		patients.PatientRequest	true	"Patient data"
// @Success		201		{object}	utils.Response{data=PatientBrief}
// @Failure		400		{object}	utils.ErrorResponse
// @Failure		409		{object}	utils.ErrorResponse
// @Failure		429		{object}	utils.ErrorResponse
// @Router			/api/v1/apm/patients [post]
func (h *Handler) RegisterPatient(c *gin.Context) {
	var req patients.PatientRequest
	if !utils.BindJSON(c, &req) {
		return
	}
	p, err := h.service.RegisterPatient(c.Request.Context(), req)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusCreated, "Patient registered successfully", p)
}

// @Summary		APM: register a visit
// @Description	Public (no login). Registers a patient (patient_id from the lookup, or from POST /apm/patients for a new patient) for a visit today with an active doctor of the chosen poli, following the same rules as the front desk (BPJS and insurance fields, responsible person under 18, no double registration). The kiosk cannot pick another date or approve the visit. Returns the ticket to print. Rate limited per IP (60 per minute).
// @Tags			apm
// @Accept			json
// @Produce		json
// @Param			body	body		RegisterRequest	true	"Registration"
// @Success		201		{object}	utils.Response{data=Ticket}
// @Failure		400		{object}	utils.ErrorResponse
// @Failure		404		{object}	utils.ErrorResponse
// @Failure		409		{object}	utils.ErrorResponse
// @Failure		429		{object}	utils.ErrorResponse
// @Router			/api/v1/apm/visits [post]
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if !utils.BindJSON(c, &req) {
		return
	}
	t, err := h.service.Register(c.Request.Context(), req)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusCreated, "Registered successfully", t)
}

func (h *Handler) fail(c *gin.Context, err error) {
	var input InputError
	switch {
	case errors.As(err, &input):
		utils.AbortWithError(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, models.ErrPatientNotFound), errors.Is(err, visits.ErrDoctorNotFound):
		utils.AbortWithError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, visits.ErrAlreadyRegistered), errors.Is(err, patients.ErrAlreadyExists):
		utils.AbortWithError(c, http.StatusConflict, err.Error())
	case errors.Is(err, visits.ErrDoctorInactive), errors.Is(err, visits.ErrDoctorPoliMismatch),
		errors.Is(err, visits.ErrPoliInactive), errors.Is(err, visits.ErrResponsiblePersonRequired),
		errors.Is(err, visits.ErrVisitDateInPast), errors.Is(err, patients.ErrInvalidBirthDate):
		utils.AbortWithError(c, http.StatusBadRequest, err.Error())
	default:
		utils.AbortWithError(c, http.StatusInternalServerError, "internal server error")
	}
}
