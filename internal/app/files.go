package app

import (
	"context"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"gin-quickstart/internal/features/files"
	"gin-quickstart/internal/platform/config"
	"gin-quickstart/internal/platform/middleware"
	"gin-quickstart/internal/platform/storage"
)

// mountFiles switches file upload on when object storage is configured. Every feature that lets
// users attach files passes a target here; the file routes are open to the union of their roles.
func mountFiles(ctx context.Context, cfg *config.Config, db *gorm.DB, v1 *gin.RouterGroup, authenticated gin.HandlerFunc, targets ...files.Target) error {
	if !cfg.Storage.Enabled() {
		log.Println("file upload is off: MINIO_ENDPOINT is not set")
		return nil
	}

	store, err := storage.NewMinIO(ctx, cfg.Storage)
	if err != nil {
		return fmt.Errorf("object storage: %w", err)
	}
	registry, err := files.NewRegistry(targets...)
	if err != nil {
		return err
	}

	service := files.NewService(files.NewRepository(db), registry, store)
	files.RegisterRoutes(v1, files.NewHandler(service), authenticated, middleware.RequireRoles(registry.Roles()...))
	return nil
}
