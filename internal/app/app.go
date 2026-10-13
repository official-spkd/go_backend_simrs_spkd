package app

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"gin-quickstart/internal/features/auth"
	"gin-quickstart/internal/features/roles"
	"gin-quickstart/internal/platform/config"
	"gin-quickstart/internal/platform/middleware"
	"gin-quickstart/internal/platform/models"
	"gin-quickstart/internal/platform/utils"
	"gin-quickstart/pkg/swagger"
)

func New(ctx context.Context, cfg *config.Config, db *gorm.DB) (*gin.Engine, error) {
	if err := db.AutoMigrate(&roles.Role{}, &models.User{}); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	tokens := utils.NewTokenManager(cfg.JWT.Secret, cfg.JWT.AccessTTL)
	requiredAuth := middleware.Required(tokens)
	optionalAuth := middleware.Optional(tokens)

	roleSvc := roles.NewService(roles.NewRepository(db))
	if err := roleSvc.SeedDefaults(ctx); err != nil {
		return nil, fmt.Errorf("seed roles: %w", err)
	}

	authSvc := auth.NewService(auth.NewRepository(db), roleSvc, tokens)

	router := gin.Default()
	swagger.Register(router)
	router.GET("/ping", ping)

	v1 := router.Group("/api/v1")
	auth.RegisterRoutes(v1, auth.NewHandler(authSvc), optionalAuth, requiredAuth)
	roles.RegisterRoutes(v1, roles.NewHandler(roleSvc), requiredAuth, middleware.RequireRoles(roles.Superadmin))

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
