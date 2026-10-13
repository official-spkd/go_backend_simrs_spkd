package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"gin-quickstart/internal/platform/config"
	"gin-quickstart/internal/platform/database"
	"gin-quickstart/pkg/swagger"
)

func main() {
	cfg := config.Load()

	db, err := database.NewPostgres(cfg.DB)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer database.Close(db)

	router := gin.Default()
	swagger.Register(router)
	router.GET("/ping", ping)

	if err := router.Run(":" + cfg.AppPort); err != nil {
		log.Fatalf("server: %v", err)
	}
}

// ping godoc
//
//	@Summary	Health check
//	@Tags		health
//	@Produce	json
//	@Success	200	{object}	map[string]string
//	@Router		/ping [get]
func ping(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "pong",
	})
}
