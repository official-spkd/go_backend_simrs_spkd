package main

import (
	"context"
	"log"
	"net"

	"gin-quickstart/internal/app"
	"gin-quickstart/internal/platform/config"
	"gin-quickstart/internal/platform/database"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := database.NewPostgres(cfg.DB)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer database.Close(db)

	router, err := app.New(context.Background(), cfg, db)
	if err != nil {
		log.Fatalf("app: %v", err)
	}

	if err := router.Run(net.JoinHostPort(cfg.AppHost, cfg.AppPort)); err != nil {
		log.Fatalf("server: %v", err)
	}
}
