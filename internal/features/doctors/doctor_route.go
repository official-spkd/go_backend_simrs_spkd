package doctors

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts the doctor routes. Reads need any authenticated user (dropdowns);
// writes additionally need the writeOnly middleware.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, authenticated, writeOnly gin.HandlerFunc) {
	g := rg.Group("/doctors", authenticated)
	g.GET("", h.List)
	g.GET("/:id", h.Get)

	w := g.Group("", writeOnly)
	w.POST("", h.Create)
	w.PUT("/:id", h.Update)
	w.DELETE("/:id", h.Delete)
}
