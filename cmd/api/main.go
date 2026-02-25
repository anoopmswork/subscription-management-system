package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"subscription-management-system/internal/config"
	"subscription-management-system/internal/customer"
	"subscription-management-system/internal/database"
	"subscription-management-system/internal/http/api"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer pool.Close()

	if cfg.AutoMigrate {
		if err := database.RunMigrations(context.Background(), pool); err != nil {
			log.Fatalf("failed to run migrations: %v", err)
		}
	}

	repo := customer.NewPostgresRepository(pool)
	service := customer.NewService(repo)
	handler := api.NewHandler(service)

	app := fiber.New(fiber.Config{AppName: "subscription-management-system-api"})
	app.Use(requestid.New())
	app.Use(recover.New())
	app.Use(logger.New())

	handler.Register(app)

	address := ":" + cfg.Port
	go func() {
		log.Printf("server starting on %s (env=%s)", address, cfg.Environment)
		if err := app.Listen(address); err != nil {
			log.Fatalf("server stopped: %v", err)
		}
	}()

	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, syscall.SIGINT, syscall.SIGTERM)
	<-shutdownSignal

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}
