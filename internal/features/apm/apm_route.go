package apm

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts the kiosk routes. They have no login, so each group gets a rate limit:
// a tight one for the patient lookup (it confirms whether an identifier exists), a tight one for
// creating patients (it writes records), and a looser one for the rest.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, lookupLimit, createLimit, generalLimit gin.HandlerFunc) {
	g := rg.Group("/apm")

	g.GET("/patients/lookup", lookupLimit, h.LookupPatient)
	g.POST("/patients", createLimit, h.RegisterPatient)

	open := g.Group("", generalLimit)
	open.GET("/polis", h.ListPolis)
	open.GET("/doctors", h.ListDoctors)
	open.POST("/visits", h.Register)
}
