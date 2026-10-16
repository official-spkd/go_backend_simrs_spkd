package cashier

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"gin-quickstart/internal/platform/utils"
)

// @Summary		Bill worklist
// @Description	Cashier only. Every visit of the date that is not cancelled, with its bill status and totals; a visit with nothing billed yet is a draft priced from its registration and doctor. visit.visit_status tells ready to bill (Completed) from still at the clinic (Registered, Served) and paid (Paid).
// @Tags			cashier
// @Produce		json
// @Security		BearerAuth
// @Param			date	query		string	false	"Visit date (YYYY-MM-DD), default today"
// @Param			page	query		int		false	"Page number"				default(1)
// @Param			limit	query		int		false	"Items per page (max 100)"	default(20)
// @Success		200		{object}	utils.PaginatedResponse{data=[]SummaryResponse}
// @Failure		400		{object}	utils.ErrorResponse
// @Failure		401		{object}	utils.ErrorResponse
// @Failure		403		{object}	utils.ErrorResponse
// @Router			/api/v1/cashier/bills/worklist [get]
func (h *Handler) Worklist(c *gin.Context) {
	var req WorklistRequest
	if !utils.BindQuery(c, &req) {
		return
	}
	date := utils.Today()
	if req.Date != "" {
		date, _ = time.Parse(time.DateOnly, req.Date) // already validated by the binding
	}
	page, limit := utils.ParsePagination(c)

	views, total, err := h.service.Worklist(c.Request.Context(), date, page, limit)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.SuccessPaginated(c, "Bills retrieved successfully", toSummaryResponses(views), utils.NewPagination(page, limit, total))
}

// @Summary		List invoices and receipts
// @Description	Cashier only. Finalized and paid bills, most recent first. Search by patient name, medical record number, invoice or receipt number.
// @Tags			cashier
// @Produce		json
// @Security		BearerAuth
// @Param			search	query		string	false	"Search by patient, medical record, invoice or receipt number"
// @Param			status	query		string	false	"Only final (awaiting payment) or paid bills"	Enums(final, paid)
// @Param			page	query		int		false	"Page number"				default(1)
// @Param			limit	query		int		false	"Items per page (max 100)"	default(20)
// @Success		200		{object}	utils.PaginatedResponse{data=[]SummaryResponse}
// @Failure		400		{object}	utils.ErrorResponse
// @Failure		401		{object}	utils.ErrorResponse
// @Failure		403		{object}	utils.ErrorResponse
// @Router			/api/v1/cashier/bills [get]
func (h *Handler) ListDocuments(c *gin.Context) {
	var req DocumentListRequest
	if !utils.BindQuery(c, &req) {
		return
	}
	q := DocumentQuery{Search: strings.TrimSpace(req.Search), Status: req.Status}
	q.Page, q.Limit = utils.ParsePagination(c)

	views, total, err := h.service.ListDocuments(c.Request.Context(), q)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.SuccessPaginated(c, "Bills retrieved successfully", toSummaryResponses(views), utils.NewPagination(q.Page, q.Limit, total))
}

// @Summary		Get a visit's bill
// @Description	Cashier only. While draft, the lines and totals are provisional; a final bill's lines are frozen.
// @Tags			cashier
// @Produce		json
// @Security		BearerAuth
// @Param			visit_id	path		string	true	"Visit ID"
// @Success		200			{object}	utils.Response{data=BillResponse}
// @Failure		400			{object}	utils.ErrorResponse
// @Failure		401			{object}	utils.ErrorResponse
// @Failure		403			{object}	utils.ErrorResponse
// @Failure		404			{object}	utils.ErrorResponse
// @Router			/api/v1/cashier/bills/{visit_id} [get]
func (h *Handler) GetBill(c *gin.Context) {
	visitID, ok := utils.ParseUUIDParam(c, "visit_id")
	if !ok {
		return
	}
	view, err := h.service.Get(c.Request.Context(), visitID)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Bill retrieved successfully", toBillResponse(view))
}

// @Summary		Set the payment scheme
// @Description	Cashier only, draft bills only. Sets who pays: umum (the patient), bpjs (the patient pays only the class-upgrade excess) or asuransi (the insurer pays up to the coverage limit). Recorded in the bill's audit trail.
// @Tags			cashier
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			visit_id	path		string			true	"Visit ID"
// @Param			body		body		SchemeRequest	true	"Scheme"
// @Success		200			{object}	utils.Response{data=BillResponse}
// @Failure		400			{object}	utils.ErrorResponse
// @Failure		401			{object}	utils.ErrorResponse
// @Failure		403			{object}	utils.ErrorResponse
// @Failure		404			{object}	utils.ErrorResponse
// @Failure		409			{object}	utils.ErrorResponse
// @Router			/api/v1/cashier/bills/{visit_id}/scheme [put]
func (h *Handler) SetScheme(c *gin.Context) {
	visitID, actor, ok := h.request(c, "visit_id")
	if !ok {
		return
	}
	var req SchemeRequest
	if !utils.BindJSON(c, &req) {
		return
	}
	view, err := h.service.SetScheme(c.Request.Context(), visitID, req, actor)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Payment scheme updated successfully", toBillResponse(view))
}

// @Summary		Set the discount
// @Description	Cashier only, draft bills only. A discount needs a reason of at least 15 characters and cannot exceed the subtotal; amount 0 removes it. Recorded in the bill's audit trail.
// @Tags			cashier
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			visit_id	path		string			true	"Visit ID"
// @Param			body		body		DiscountRequest	true	"Discount"
// @Success		200			{object}	utils.Response{data=BillResponse}
// @Failure		400			{object}	utils.ErrorResponse
// @Failure		401			{object}	utils.ErrorResponse
// @Failure		403			{object}	utils.ErrorResponse
// @Failure		404			{object}	utils.ErrorResponse
// @Failure		409			{object}	utils.ErrorResponse
// @Router			/api/v1/cashier/bills/{visit_id}/discount [put]
func (h *Handler) SetDiscount(c *gin.Context) {
	visitID, actor, ok := h.request(c, "visit_id")
	if !ok {
		return
	}
	var req DiscountRequest
	if !utils.BindJSON(c, &req) {
		return
	}
	view, err := h.service.SetDiscount(c.Request.Context(), visitID, req, actor)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Discount updated successfully", toBillResponse(view))
}

// @Summary		Add a manual item
// @Description	Cashier only, draft bills only. Adds a line under "other"; it needs a reason of at least 15 characters. Recorded in the bill's audit trail.
// @Tags			cashier
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			visit_id	path		string		true	"Visit ID"
// @Param			body		body		ItemRequest	true	"Item"
// @Success		201			{object}	utils.Response{data=BillResponse}
// @Failure		400			{object}	utils.ErrorResponse
// @Failure		401			{object}	utils.ErrorResponse
// @Failure		403			{object}	utils.ErrorResponse
// @Failure		404			{object}	utils.ErrorResponse
// @Failure		409			{object}	utils.ErrorResponse
// @Router			/api/v1/cashier/bills/{visit_id}/items [post]
func (h *Handler) AddItem(c *gin.Context) {
	visitID, actor, ok := h.request(c, "visit_id")
	if !ok {
		return
	}
	var req ItemRequest
	if !utils.BindJSON(c, &req) {
		return
	}
	view, err := h.service.AddItem(c.Request.Context(), visitID, req, actor)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusCreated, "Item added successfully", toBillResponse(view))
}

// @Summary		Finalize the bill
// @Description	Cashier only, and only once the visit is Completed (the doctor finalized the SOAP note). Freezes the lines and issues the invoice number (INV/YYYYMMDD/NNNN).
// @Tags			cashier
// @Produce		json
// @Security		BearerAuth
// @Param			visit_id	path		string	true	"Visit ID"
// @Success		200			{object}	utils.Response{data=BillResponse}
// @Failure		400			{object}	utils.ErrorResponse
// @Failure		401			{object}	utils.ErrorResponse
// @Failure		403			{object}	utils.ErrorResponse
// @Failure		404			{object}	utils.ErrorResponse
// @Failure		409			{object}	utils.ErrorResponse
// @Router			/api/v1/cashier/bills/{visit_id}/finalize [post]
func (h *Handler) Finalize(c *gin.Context) {
	visitID, actor, ok := h.request(c, "visit_id")
	if !ok {
		return
	}
	view, err := h.service.Finalize(c.Request.Context(), visitID, actor)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Bill finalized successfully", toBillResponse(view))
}

// @Summary		Pay the bill
// @Description	Cashier only, final bills only. The payments must cover the remaining patient portion, which the server recomputes; each part is capped to what is left, and cash change is computed from received. Send no payments when the patient portion is 0. Issues the receipt number (KW/YYYYMMDD/NNNN), records the payments in the cashier's open shift (opening one if needed), and moves the visit to Paid.
// @Tags			cashier
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			visit_id	path		string			true	"Visit ID"
// @Param			body		body		PaymentRequest	true	"Payments"
// @Success		201			{object}	utils.Response{data=BillResponse}
// @Failure		400			{object}	utils.ErrorResponse
// @Failure		401			{object}	utils.ErrorResponse
// @Failure		403			{object}	utils.ErrorResponse
// @Failure		404			{object}	utils.ErrorResponse
// @Failure		409			{object}	utils.ErrorResponse
// @Router			/api/v1/cashier/bills/{visit_id}/payments [post]
func (h *Handler) Pay(c *gin.Context) {
	visitID, actor, ok := h.request(c, "visit_id")
	if !ok {
		return
	}
	var req PaymentRequest
	if !utils.BindJSON(c, &req) {
		return
	}
	view, err := h.service.Pay(c.Request.Context(), visitID, req, actor)
	if err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusCreated, "Payment recorded successfully", toBillResponse(view))
}

// @Summary		Record a print
// @Description	Cashier only, for finalized or paid bills. Records that the invoice was printed or saved as PDF, in the bill's audit trail.
// @Tags			cashier
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			visit_id	path		string			true	"Visit ID"
// @Param			body		body		PrintRequest	true	"Print"
// @Success		201			{object}	utils.Response
// @Failure		400			{object}	utils.ErrorResponse
// @Failure		401			{object}	utils.ErrorResponse
// @Failure		403			{object}	utils.ErrorResponse
// @Failure		404			{object}	utils.ErrorResponse
// @Router			/api/v1/cashier/bills/{visit_id}/prints [post]
func (h *Handler) RecordPrint(c *gin.Context) {
	visitID, actor, ok := h.request(c, "visit_id")
	if !ok {
		return
	}
	var req PrintRequest
	if !utils.BindJSON(c, &req) {
		return
	}
	if err := h.service.RecordPrint(c.Request.Context(), visitID, req, actor); err != nil {
		h.fail(c, err)
		return
	}
	utils.Success(c, http.StatusCreated, "Print recorded successfully", nil)
}
