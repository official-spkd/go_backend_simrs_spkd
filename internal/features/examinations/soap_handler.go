package examinations

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"gin-quickstart/internal/platform/utils"
)

func (h *Handler) soapResponse(c *gin.Context, message string, view *SoapView) {
	utils.Success(c, http.StatusOK, message, toSoapResponse(*view))
}

// @Summary		Get the SOAP note
// @Description	Doctor only (own visits). Objective.vital_signs is pulled in automatically: live values while the note is a draft, the frozen snapshot once final. 404 until a draft has been saved.
// @Tags			examinations
// @Produce		json
// @Security		BearerAuth
// @Param			visit_id	path		string	true	"Visit ID"
// @Success		200			{object}	utils.Response{data=SoapResponse}
// @Failure		400			{object}	utils.ErrorResponse
// @Failure		401			{object}	utils.ErrorResponse
// @Failure		403			{object}	utils.ErrorResponse
// @Failure		404			{object}	utils.ErrorResponse
// @Router			/api/v1/examinations/{visit_id}/soap-note [get]
func (h *Handler) GetSoap(c *gin.Context) {
	visitID, actor, ok := h.request(c)
	if !ok {
		return
	}
	view, err := h.service.GetSoap(c.Request.Context(), visitID, actor)
	if err != nil {
		h.fail(c, err)
		return
	}
	h.soapResponse(c, "SOAP note retrieved successfully", view)
}

// @Summary		Save the SOAP note draft
// @Description	Doctor only (own visits). Creates the draft on the first call (version 0) and replaces it afterwards (send the current version, otherwise 409). The first save moves the visit from Registered to Served. Diagnoses are ICD-10 codes (format only; no code list is checked yet): one primary, up to 10 secondary. control_date only goes with outpatient_followup, referral_notes only with refer_inpatient or refer_icu. Not allowed once final.
// @Tags			examinations
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			visit_id	path		string		true	"Visit ID"
// @Param			body		body		SoapRequest	true	"SOAP draft"
// @Success		200			{object}	utils.Response{data=SoapResponse}
// @Failure		400			{object}	utils.ErrorResponse
// @Failure		401			{object}	utils.ErrorResponse
// @Failure		403			{object}	utils.ErrorResponse
// @Failure		404			{object}	utils.ErrorResponse
// @Failure		409			{object}	utils.ErrorResponse
// @Router			/api/v1/examinations/{visit_id}/soap-note [put]
func (h *Handler) SaveSoap(c *gin.Context) {
	visitID, actor, ok := h.request(c)
	if !ok {
		return
	}
	var req SoapRequest
	if !utils.BindJSON(c, &req) {
		return
	}
	view, err := h.service.SaveSoap(c.Request.Context(), visitID, req, actor)
	if err != nil {
		h.fail(c, err)
		return
	}
	h.soapResponse(c, "SOAP note saved successfully", view)
}

// @Summary		Finalize the SOAP note
// @Description	Doctor only (own visits). Needs the vital signs, subjective, plan, a primary diagnosis and a follow-up (plus control_date or referral_notes where they apply), otherwise 422 with the missing fields. Freezes the vital signs shown in the objective part, locks the note and the vital signs, and moves the visit to Completed. Corrections afterwards go through addenda.
// @Tags			examinations
// @Produce		json
// @Security		BearerAuth
// @Param			visit_id	path		string	true	"Visit ID"
// @Success		200			{object}	utils.Response{data=SoapResponse}
// @Failure		400			{object}	utils.ErrorResponse
// @Failure		401			{object}	utils.ErrorResponse
// @Failure		403			{object}	utils.ErrorResponse
// @Failure		404			{object}	utils.ErrorResponse
// @Failure		409			{object}	utils.ErrorResponse
// @Failure		422			{object}	utils.ErrorResponse
// @Router			/api/v1/examinations/{visit_id}/soap-note/finalize [post]
func (h *Handler) FinalizeSoap(c *gin.Context) {
	visitID, actor, ok := h.request(c)
	if !ok {
		return
	}
	view, err := h.service.FinalizeSoap(c.Request.Context(), visitID, actor)
	if err != nil {
		h.fail(c, err)
		return
	}
	h.soapResponse(c, "SOAP note finalized successfully", view)
}

// @Summary		Add an addendum
// @Description	Doctor only (own visits). Appends a correction or note to a finalized SOAP note. Addenda cannot be edited or deleted.
// @Tags			examinations
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			visit_id	path		string			true	"Visit ID"
// @Param			body		body		AddendumRequest	true	"Addendum"
// @Success		201			{object}	utils.Response{data=SoapResponse}
// @Failure		400			{object}	utils.ErrorResponse
// @Failure		401			{object}	utils.ErrorResponse
// @Failure		403			{object}	utils.ErrorResponse
// @Failure		404			{object}	utils.ErrorResponse
// @Failure		409			{object}	utils.ErrorResponse
// @Router			/api/v1/examinations/{visit_id}/soap-note/addenda [post]
func (h *Handler) AddAddendum(c *gin.Context) {
	visitID, actor, ok := h.request(c)
	if !ok {
		return
	}
	var req AddendumRequest
	if !utils.BindJSON(c, &req) {
		return
	}
	view, err := h.service.AddAddendum(c.Request.Context(), visitID, req, actor)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusCreated, "Addendum added successfully", toSoapResponse(*view))
}
