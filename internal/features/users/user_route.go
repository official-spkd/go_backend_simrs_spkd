package users

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup, h *Handler, middlewares ...gin.HandlerFunc) {
	g := rg.Group("/users", middlewares...)
	g.GET("", h.List)
	g.GET("/:id", h.Get)
	g.PUT("/:id", h.Update)
	g.PATCH("/:id/role", h.UpdateRole)
	g.PATCH("/:id/status", h.UpdateStatus)
	g.PUT("/:id/password", h.ResetPassword)
	g.DELETE("/:id", h.Delete)
}
