package visits

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup, h *Handler, middlewares ...gin.HandlerFunc) {
	g := rg.Group("/visits", middlewares...)
	g.POST("", h.Create)
	g.GET("", h.List)
	g.GET("/:id", h.Get)
}
