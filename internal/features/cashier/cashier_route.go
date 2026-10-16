package cashier

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts everything under /cashier: bills are addressed by their visit, because a
// visit has at most one bill and the cashier opens it from the visit worklist. cashierOnly guards
// every route except the shift list, which shiftReaders (cashier and management) may read.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, auth, cashierOnly, shiftReaders gin.HandlerFunc) {
	g := rg.Group("/cashier", auth)

	bills := g.Group("/bills", cashierOnly)
	bills.GET("", h.ListDocuments)
	bills.GET("/worklist", h.Worklist)
	bills.GET("/:visit_id", h.GetBill)
	bills.PUT("/:visit_id/scheme", h.SetScheme)
	bills.PUT("/:visit_id/discount", h.SetDiscount)
	bills.POST("/:visit_id/items", h.AddItem)
	bills.POST("/:visit_id/finalize", h.Finalize)
	bills.POST("/:visit_id/payments", h.Pay)
	bills.POST("/:visit_id/prints", h.RecordPrint)

	shifts := g.Group("/shifts")
	shifts.GET("", shiftReaders, h.ListShifts)
	shifts.GET("/current", cashierOnly, h.CurrentShift)
	shifts.POST("/:id/close", cashierOnly, h.CloseShift)
}
