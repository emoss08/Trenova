package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/emoss08/trenova/microservices/workflow/internal/config"
	"github.com/emoss08/trenova/microservices/workflow/internal/consumer"
	"github.com/emoss08/trenova/microservices/workflow/internal/email"
	"github.com/emoss08/trenova/microservices/workflow/internal/model"
	"github.com/emoss08/trenova/microservices/workflow/internal/workflow"
	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/worker"
	"github.com/joho/godotenv"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
)

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Load configuration
	cfg := config.LoadConfig()

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	// Setup database connection
	db := setupDB(cfg)
	defer db.Close()

	// Create Hatchet client
	hatchetClient, err := client.New()
	if err != nil {
		log.Printf("Failed to create Hatchet client: %v", err)
	}

	// Create and configure Hatchet worker
	hatchetWorker, err := worker.NewWorker(
		worker.WithClient(hatchetClient),
	)
	if err != nil {
		log.Printf("Failed to create Hatchet worker: %v", err)
	}

	// Initialize email client
	emailClient, err := email.NewClient(cfg.RabbitMQ)
	if err != nil {
		log.Printf("Failed to create email client: %v", err)
	} else {
		defer emailClient.Close()
	}

	// Register workflows
	registry := workflow.NewRegistry(hatchetWorker, db, emailClient)
	if err = registry.RegisterAllWorkflows(); err != nil {
		log.Printf("Failed to register workflows: %v", err)
	}

	// Create and configure RabbitMQ consumer
	rabbitConsumer, err := consumer.NewRabbitMQConsumer(cfg.RabbitMQ)
	if err != nil {
		log.Printf("Failed to create RabbitMQ consumer: %v", err)
	}

	// Create message handler
	handler := consumer.NewHatchetHandler(hatchetClient)

	// Register handlers for different workflow types
	rabbitConsumer.RegisterHandler(model.TypeShipmentUpdated, handler.HandleShipmentMessage)

	workerCleanup, err := hatchetWorker.Start()
	if err != nil {
		log.Printf("Failed to start Hatchet worker: %v", err)
	}
	defer func() {
		if err = workerCleanup(); err != nil {
			log.Printf("Error during Hatchet worker cleanup: %v", err)
		}
	}()

	// Start the RabbitMQ consumer
	if err = rabbitConsumer.Start(ctx); err != nil {
		log.Printf("Failed to start RabbitMQ consumer: %v", err)
	}

	log.Printf("Workflow service started. Environment: %s", cfg.Environment)

	// Wait for termination signal
	<-signalCh
	log.Println("Received termination signal, shutting down...")

	// Trigger context cancellation
	cancel()

	// Clean up resources
	if err = rabbitConsumer.Stop(); err != nil {
		log.Printf("Error stopping RabbitMQ consumer: %v", err)
	}

	log.Println("Service shutdown complete")
}

func setupDB(cfg *config.AppConfig) *bun.DB {
	pgconn := pgdriver.NewConnector(
		pgdriver.WithDSN(cfg.DB.DSN()),
		pgdriver.WithTimeout(30*time.Second),
		pgdriver.WithWriteTimeout(30*time.Second),
	)

	sqldb := sql.OpenDB(pgconn)
	sqldb.SetMaxOpenConns(cfg.DB.MaxConnections)
	sqldb.SetMaxIdleConns(cfg.DB.MaxIdleConns)

	db := bun.NewDB(sqldb, pgdialect.New(), bun.WithDiscardUnknownColumns())
	db.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithVerbose(cfg.DB.Debug),
		bundebug.WithEnabled(cfg.DB.Debug),
	))

	// * Ping the database to ensure connection
	if err := db.PingContext(context.Background()); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	return db
}
