package schedules

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts the schedule routes. Every route needs both middlewares: schedules are
// master data managed by the restricted role only.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, authenticated, restricted gin.HandlerFunc) {
	g := rg.Group("/schedules", authenticated, restricted)
	g.GET("", h.List)
	g.POST("", h.Create)
	g.GET("/sessions", h.ListSessions)
	g.GET("/exceptions", h.ListExceptions)
	g.POST("/exceptions", h.CreateException)
	g.DELETE("/exceptions/:id", h.DeleteException)
	g.GET("/:id", h.Get)
	g.PUT("/:id", h.Update)
	g.DELETE("/:id", h.Delete)
}

// RegisterAvailabilityRoutes mounts the read-only availability list used by registration.
func RegisterAvailabilityRoutes(rg *gin.RouterGroup, h *Handler, authenticated, allowed gin.HandlerFunc) {
	rg.GET("/availability", authenticated, allowed, h.ListAvailability)
}
