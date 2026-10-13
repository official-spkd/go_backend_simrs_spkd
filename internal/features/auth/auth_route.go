package auth

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup, h *Handler, optionalAuth, requiredAuth gin.HandlerFunc) {
	g := rg.Group("/auth")
	g.POST("/login", h.Login)
	g.POST("/register", optionalAuth, h.Register)
	g.GET("/me", requiredAuth, h.Me)
}
