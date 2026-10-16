package examinations

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts the doctor's worklist at /examinations and everything about one visit
// under /examinations/{visit_id}; the vital signs and the SOAP note are separate sub-resources.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, middlewares ...gin.HandlerFunc) {
	rg.Group("/examinations", middlewares...).GET("", h.Worklist)

	g := rg.Group("/examinations/:visit_id", middlewares...)
	g.GET("", h.Overview)
	g.GET("/history", h.History)

	g.POST("/call", h.Call)
	g.POST("/recall", h.Recall)
	g.POST("/skip", h.Skip)
	g.POST("/start", h.StartExam)
	g.POST("/close", h.Close)

	g.GET("/vital-signs", h.GetVitals)
	g.PUT("/vital-signs", h.SaveVitals)
	g.GET("/vital-signs/versions", h.ListVitalVersions)
	g.GET("/vital-signs/versions/:version", h.GetVitalVersion)

	g.GET("/soap-note", h.GetSoap)
	g.PUT("/soap-note", h.SaveSoap)
	g.POST("/soap-note/finalize", h.FinalizeSoap)
	g.POST("/soap-note/addenda", h.AddAddendum)
}
