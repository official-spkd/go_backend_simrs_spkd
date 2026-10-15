package queues

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts the routes. Counter actions need authenticated plus counterStaff; the queue
// list needs authenticated plus queueReaders (counter staff and doctors); the display is public.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, authenticated, counterStaff, queueReaders gin.HandlerFunc) {
	counters := rg.Group("/counters", authenticated, counterStaff)
	counters.GET("", h.ListCounters)
	counters.PATCH("/:id/status", h.SetCounterStatus)
	counters.POST("/:id/next", h.Next)
	counters.POST("/:id/recall", h.Recall)
	counters.POST("/:id/serve", h.Serve)
	counters.POST("/:id/complete", h.Complete)
	counters.POST("/:id/skip", h.Skip)

	rg.Group("/queues", authenticated, queueReaders).GET("", h.ListQueues)

	rg.GET("/display/queues", h.Display)
}
