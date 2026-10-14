package app

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"gin-quickstart/internal/features/auth"
	"gin-quickstart/internal/features/patients"
	"gin-quickstart/internal/features/roles"
	"gin-quickstart/internal/features/users"
	"gin-quickstart/internal/features/visits"
	"gin-quickstart/internal/platform/config"
	"gin-quickstart/internal/platform/middleware"
	"gin-quickstart/internal/platform/utils"
	"gin-quickstart/pkg/swagger"
)

func New(ctx context.Context, cfg *config.Config, db *gorm.DB) (*gin.Engine, error) {
	if err := migrate(db); err != nil {
		return nil, err
	}

	utils.InitValidation()
	if err := patients.RegisterValidators(); err != nil {
		return nil, fmt.Errorf("register validators: %w", err)
	}
	if err := visits.RegisterValidators(); err != nil {
		return nil, fmt.Errorf("register validators: %w", err)
	}

	tokens := utils.NewTokenManager(cfg.JWT.Secret, cfg.JWT.AccessTTL)
	requiredAuth := middleware.Required(tokens)
	optionalAuth := middleware.Optional(tokens)

	roleSvc := roles.NewService(roles.NewRepository(db))
	if err := roleSvc.SeedDefaults(ctx); err != nil {
		return nil, fmt.Errorf("seed roles: %w", err)
	}

	authSvc := auth.NewService(auth.NewRepository(db), roleSvc, tokens)

	userSvc := users.NewService(users.NewRepository(db), roleSvc)
	patientSvc := patients.NewService(patients.NewRepository(db))
	visitSvc := visits.NewService(visits.NewRepository(db), patientSvc)

	router := gin.Default()
	router.Use(middleware.SecurityHeaders())
	if len(cfg.CORS.AllowedOrigins) > 0 {
		router.Use(middleware.CORS(cfg.CORS.AllowedOrigins))
	}
	swagger.Register(router)
	router.GET("/ping", ping)

	v1 := router.Group("/api/v1")
	auth.RegisterRoutes(v1, auth.NewHandler(authSvc), optionalAuth, requiredAuth)
	superadminOnly := middleware.RequireRoles(roles.Superadmin)
	roles.RegisterRoutes(v1, roles.NewHandler(roleSvc), requiredAuth, superadminOnly)
	users.RegisterRoutes(v1, users.NewHandler(userSvc), requiredAuth, superadminOnly)
	frontOfficeOnly := middleware.RequireRoles(roles.FrontOffice)
	patients.RegisterRoutes(v1, patients.NewHandler(patientSvc), requiredAuth, frontOfficeOnly)
	visits.RegisterRoutes(v1, visits.NewHandler(visitSvc), requiredAuth, frontOfficeOnly)

	return router, nil
}

// @Summary	Health check
// @Tags		health
// @Produce	json
// @Success	200	{object}	utils.Response
// @Router		/ping [get]
func ping(c *gin.Context) {
	utils.Success(c, http.StatusOK, "pong", nil)
}
