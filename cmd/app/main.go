package main

import (
	"github.com/gin-gonic/gin"

	"gin-quickstart/pkg/swagger"
)

func main() {
	router := gin.Default()
	swagger.Register(router)
	router.GET("/ping", ping)
	router.Run()
}

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
