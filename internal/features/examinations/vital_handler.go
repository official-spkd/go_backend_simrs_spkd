package examinations

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"gin-quickstart/internal/platform/utils"
)

// @Summary		Get the vital signs
// @Description	Doctor only (own visits). 404 until the vital signs have been saved.
// @Tags			examinations
// @Produce		json
// @Security		BearerAuth
// @Param			visit_id	path		string	true	"Visit ID"
// @Success		200			{object}	utils.Response{data=VitalResponse}
// @Failure		400			{object}	utils.ErrorResponse
// @Failure		401			{object}	utils.ErrorResponse
// @Failure		403			{object}	utils.ErrorResponse
// @Failure		404			{object}	utils.ErrorResponse
// @Router			/api/v1/examinations/{visit_id}/vital-signs [get]
func (h *Handler) GetVitals(c *gin.Context) {
	visitID, actor, ok := h.request(c)
	if !ok {
		return
	}
	v, err := h.service.GetVitals(c.Request.Context(), visitID, actor)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Vital signs retrieved successfully", toVitalResponse(v))
}

// @Summary		Save the vital signs and initial assessment
// @Description	Doctor only (own visits). Creates the record on the first call (version 0). Every later change creates a new version in the history and needs a change_reason (and the current version, otherwise 409); a request that changes nothing is rejected. BMI and its Kemenkes RI category are computed by the server (category is not_applicable under 18 years); flags are advisory. Not allowed once the SOAP note is final, for cancelled visits, or for visits scheduled for a later date. Either list allergies or set no_known_allergies.
// @Tags			examinations
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			visit_id	path		string			true	"Visit ID"
// @Param			body		body		VitalRequest	true	"Vital signs"
// @Success		200			{object}	utils.Response{data=VitalResponse}
// @Failure		400			{object}	utils.ErrorResponse
// @Failure		401			{object}	utils.ErrorResponse
// @Failure		403			{object}	utils.ErrorResponse
// @Failure		404			{object}	utils.ErrorResponse
// @Failure		409			{object}	utils.ErrorResponse
// @Router			/api/v1/examinations/{visit_id}/vital-signs [put]
func (h *Handler) SaveVitals(c *gin.Context) {
	visitID, actor, ok := h.request(c)
	if !ok {
		return
	}
	var req VitalRequest
	if !utils.BindJSON(c, &req) {
		return
	}
	v, err := h.service.SaveVitals(c.Request.Context(), visitID, req, actor)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Vital signs saved successfully", toVitalResponse(v))
}

// @Summary		List the vital signs history
// @Description	Doctor only (own visits). Every saved version, newest first, with the reason, who made it, when, and which fields differ from the version before. 404 until the vital signs have been saved.
// @Tags			examinations
// @Produce		json
// @Security		BearerAuth
// @Param			visit_id	path		string	true	"Visit ID"
// @Success		200			{object}	utils.Response{data=[]VersionSummary}
// @Failure		400			{object}	utils.ErrorResponse
// @Failure		401			{object}	utils.ErrorResponse
// @Failure		403			{object}	utils.ErrorResponse
// @Failure		404			{object}	utils.ErrorResponse
// @Router			/api/v1/examinations/{visit_id}/vital-signs/versions [get]
func (h *Handler) ListVitalVersions(c *gin.Context) {
	visitID, actor, ok := h.request(c)
	if !ok {
		return
	}
	versions, err := h.service.ListVitalVersions(c.Request.Context(), visitID, actor)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Vital signs history retrieved successfully", versions)
}

// @Summary		Get one version of the vital signs
// @Description	Doctor only (own visits). The vital signs exactly as they were in that version, plus the reason and the fields that changed.
// @Tags			examinations
// @Produce		json
// @Security		BearerAuth
// @Param			visit_id	path		string	true	"Visit ID"
// @Param			version		path		int		true	"Version number"
// @Success		200			{object}	utils.Response{data=VersionDetail}
// @Failure		400			{object}	utils.ErrorResponse
// @Failure		401			{object}	utils.ErrorResponse
// @Failure		403			{object}	utils.ErrorResponse
// @Failure		404			{object}	utils.ErrorResponse
// @Router			/api/v1/examinations/{visit_id}/vital-signs/versions/{version} [get]
func (h *Handler) GetVitalVersion(c *gin.Context) {
	visitID, actor, ok := h.request(c)
	if !ok {
		return
	}
	version, err := strconv.Atoi(c.Param("version"))
	if err != nil || version < 0 {
		utils.AbortWithError(c, http.StatusBadRequest, "invalid version")
		return
	}
	detail, err := h.service.GetVitalVersion(c.Request.Context(), visitID, version, actor)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Vital signs version retrieved successfully", detail)
}
