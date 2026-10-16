package examinations

import (
	"github.com/gin-gonic/gin"

	"gin-quickstart/internal/platform/utils"
)

// @Summary		Patient visit history
// @Description	Doctor only, opened from one of the doctor's own visits. Lists the patient's other visits (any poli and doctor, cancelled ones excluded), newest first. The SOAP note is included only once finalized; attachments are listed by name (downloads stay limited to the visit's own doctor).
// @Tags			examinations
// @Produce		json
// @Security		BearerAuth
// @Param			visit_id	path		string	true	"Visit ID"
// @Param			page		query		int		false	"Page number"				default(1)
// @Param			limit		query		int		false	"Items per page (max 100)"	default(20)
// @Success		200			{object}	utils.PaginatedResponse{data=[]HistoryItem}
// @Failure		400			{object}	utils.ErrorResponse
// @Failure		401			{object}	utils.ErrorResponse
// @Failure		403			{object}	utils.ErrorResponse
// @Failure		404			{object}	utils.ErrorResponse
// @Failure		409			{object}	utils.ErrorResponse
// @Router			/api/v1/examinations/{visit_id}/history [get]
func (h *Handler) History(c *gin.Context) {
	visitID, actor, ok := h.request(c)
	if !ok {
		return
	}
	page, limit := utils.ParsePagination(c)
	entries, total, err := h.service.History(c.Request.Context(), visitID, page, limit, actor)
	if err != nil {
		h.fail(c, err)
		return
	}
	items := make([]HistoryItem, len(entries))
	for i, e := range entries {
		items[i] = toHistoryItem(e)
	}
	utils.SuccessPaginated(c, "Visit history retrieved successfully", items, utils.NewPagination(page, limit, total))
}
