package files

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts the file routes. Which roles may use them follows from the registered
// targets; each target also decides access to its own records.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, middlewares ...gin.HandlerFunc) {
	g := rg.Group("/files", middlewares...)
	g.POST("/uploads", h.RequestUpload)
	g.POST("/:id/complete", h.Complete)
	g.GET("", h.List)
	g.GET("/:id/download-url", h.Download)
	g.DELETE("/:id", h.Delete)
}
