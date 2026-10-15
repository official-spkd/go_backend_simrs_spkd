package examinations

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts everything under one root, /examinations/{visit_id}; the vital signs and
// the SOAP note are separate sub-resources with their own models.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, middlewares ...gin.HandlerFunc) {
	g := rg.Group("/examinations/:visit_id", middlewares...)
	g.GET("", h.Overview)

	g.GET("/vital-signs", h.GetVitals)
	g.PUT("/vital-signs", h.SaveVitals)

	g.GET("/soap-note", h.GetSoap)
	g.PUT("/soap-note", h.SaveSoap)
	g.POST("/soap-note/finalize", h.FinalizeSoap)
	g.POST("/soap-note/addenda", h.AddAddendum)
}
