package queues

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts the staff routes behind the given middlewares and the public display route.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, middlewares ...gin.HandlerFunc) {
	counters := rg.Group("/counters", middlewares...)
	counters.GET("", h.ListCounters)
	counters.PATCH("/:id/status", h.SetCounterStatus)
	counters.POST("/:id/next", h.Next)
	counters.POST("/:id/recall", h.Recall)
	counters.POST("/:id/serve", h.Serve)
	counters.POST("/:id/complete", h.Complete)
	counters.POST("/:id/skip", h.Skip)

	rg.Group("/queues", middlewares...).GET("", h.ListQueues)

	rg.GET("/display/queues", h.Display)
}
