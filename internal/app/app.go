package app

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"gin-quickstart/internal/features/auth"
	"gin-quickstart/internal/features/doctors"
	"gin-quickstart/internal/features/examinations"
	"gin-quickstart/internal/features/patients"
	"gin-quickstart/internal/features/polis"
	"gin-quickstart/internal/features/queues"
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
	if err := examinations.RegisterValidators(); err != nil {
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

	poliSvc := polis.NewService(polis.NewRepository(db))
	doctorSvc := doctors.NewService(doctors.NewRepository(db), poliSvc)
	authSvc := auth.NewService(auth.NewRepository(db), roleSvc, doctorSvc, tokens)

	userSvc := users.NewService(users.NewRepository(db), roleSvc, doctorSvc)
	patientSvc := patients.NewService(patients.NewRepository(db))
	visitSvc := visits.NewService(visits.NewRepository(db), patientSvc, doctorSvc)
	examinationSvc := examinations.NewService(examinations.NewRepository(db), visitSvc, userSvc)
	queueSvc := queues.NewService(queues.NewRepository(db), userSvc)
	if err := queueSvc.SeedCounters(ctx); err != nil {
		return nil, fmt.Errorf("seed counters: %w", err)
	}

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
	doctors.RegisterRoutes(v1, doctors.NewHandler(doctorSvc), requiredAuth, superadminOnly)
	users.RegisterRoutes(v1, users.NewHandler(userSvc), requiredAuth, superadminOnly)
	polis.RegisterRoutes(v1, polis.NewHandler(poliSvc), requiredAuth, superadminOnly)
	frontOfficeOnly := middleware.RequireRoles(roles.FrontOffice)
	patients.RegisterRoutes(v1, patients.NewHandler(patientSvc), requiredAuth, frontOfficeOnly)
	visits.RegisterRoutes(v1, visits.NewHandler(visitSvc), requiredAuth, frontOfficeOnly)
	examinations.RegisterRoutes(v1, examinations.NewHandler(examinationSvc), requiredAuth, middleware.RequireRoles(roles.Doctor))
	if err := mountFiles(ctx, cfg, db, v1, requiredAuth, examinations.NewAttachmentTarget(examinationSvc)); err != nil {
		return nil, err
	}
	queues.RegisterRoutes(v1, queues.NewHandler(queueSvc), requiredAuth, frontOfficeOnly,
		middleware.RequireRoles(roles.FrontOffice, roles.Doctor))

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
