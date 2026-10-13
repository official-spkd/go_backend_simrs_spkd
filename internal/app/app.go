package app

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"gin-quickstart/internal/features/roles"
	"gin-quickstart/pkg/swagger"
)

func New(ctx context.Context, db *gorm.DB) (*gin.Engine, error) {
	if err := db.AutoMigrate(&roles.Role{}); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	roleSvc := roles.NewService(roles.NewRepository(db))
	if err := roleSvc.SeedDefaults(ctx); err != nil {
		return nil, fmt.Errorf("seed roles: %w", err)
	}

	router := gin.Default()
	swagger.Register(router)
	router.GET("/ping", ping)

	v1 := router.Group("/api/v1")
	roles.RegisterRoutes(v1, roles.NewHandler(roleSvc))

	return router, nil
}

// @Summary	Health check
// @Tags		health
// @Produce	json
// @Success	200	{object}	map[string]string
// @Router		/ping [get]
func ping(c *gin.Context) {
	c.JSON(200, gin.H{"message": "pong"})
}
