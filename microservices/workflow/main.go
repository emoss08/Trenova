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
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/worker"
	"github.com/joho/godotenv"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
	"go.uber.org/fx"
)

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Use fx for dependency injection
	app := fx.New(
		// Provide all the constructors
		fx.Provide(
			config.LoadConfig,
			provideDB,
			provideHatchetClient,
			provideHatchetWorker,
			provideEmailClient,
			workflow.NewRegistry,
			provideRabbitMQConsumer,
			provideHatchetHandler,
		),
		// Register lifecycle hooks
		fx.Invoke(setupApplication),
	)

	// Start the application
	if err := app.Start(context.Background()); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}

	// Set up signal handling for graceful shutdown
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	// Wait for termination signal
	<-signalCh
	log.Println("Received termination signal, shutting down...")

	// Stop the application
	if err := app.Stop(context.Background()); err != nil {
		log.Fatalf("Failed to stop application: %v", err)
	}

	log.Println("Workflow service shutdown complete")
}

// provideDB creates and configures the database connection
func provideDB(cfg *config.AppConfig) *bun.DB {
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

	// Ping the database to ensure connection
	if err := db.PingContext(context.Background()); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	return db
}

// provideHatchetClient creates a new Hatchet client
func provideHatchetClient() (v1.HatchetClient, error) {
	client, err := v1.NewHatchetClient()
	if err != nil {
		return nil, err
	}

	return client, nil
}

// provideHatchetWorker creates and configures a Hatchet v1 worker.
// It now returns worker.Worker (interface) instead of *worker.Worker.
func provideHatchetWorker(client v1.HatchetClient, registry *workflow.Registry) (worker.Worker, error) {
	workerName := "workflow-worker"

	// Get workflow definitions from registry
	workflows := registry.GetAllWorkflows(client)

	w, err := client.Worker(
		worker.WorkerOpts{
			Name:      workerName,
			Workflows: workflows, // Pass workflows during worker creation
			Slots:     100,
		},
	)
	if err != nil {
		return nil, err
	}
	return w, nil
}

// provideEmailClient creates a new email client
func provideEmailClient(cfg *config.AppConfig) (*email.Client, error) {
	return email.NewClient(cfg.RabbitMQ)
}

// provideRabbitMQConsumer creates a new RabbitMQ consumer
func provideRabbitMQConsumer(cfg *config.AppConfig) (*consumer.RabbitMQConsumer, error) {
	return consumer.NewRabbitMQConsumer(cfg.RabbitMQ)
}

func provideHatchetHandler(hatchetClient v1.HatchetClient) *consumer.HatchetHandler {
	return consumer.NewHatchetHandler(hatchetClient)
}

// setupApplication registers handlers and initializes services
func setupApplication(
	lc fx.Lifecycle,
	cfg *config.AppConfig,
	hatchetWorker worker.Worker,
	emailClient *email.Client,
	rabbitConsumer *consumer.RabbitMQConsumer,
	handler *consumer.HatchetHandler,
) {
	// Register handlers for different workflow types
	rabbitConsumer.RegisterHandler(model.TypeShipmentUpdated, handler.HandleShipmentMessage)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Start RabbitMQ consumer
			if err := rabbitConsumer.Start(ctx); err != nil {
				return err
			}

			// Start Hatchet worker in a goroutine since StartBlocking will block until context is cancelled
			go func() {
				interruptCtx, cancel := cmdutils.NewInterruptContext()
				defer cancel() // Only cancel when the goroutine exits

				log.Println("Starting Hatchet worker...")
				if err := hatchetWorker.StartBlocking(interruptCtx); err != nil {
					log.Printf("Error in Hatchet worker: %v", err)
				}
				log.Println("Hatchet worker stopped")
			}()

			log.Printf("Workflow service started. Environment: %s", cfg.Environment)
			return nil
		},
		OnStop: func(context.Context) error {
			log.Println("Stopping RabbitMQ consumer")
			if err := rabbitConsumer.Stop(); err != nil {
				log.Printf("Error stopping RabbitMQ consumer: %v", err)
			}

			log.Println("Closing email client")
			if err := emailClient.Close(); err != nil {
				log.Printf("Error closing email client: %v", err)
			}

			// Note: The Hatchet worker will be automatically stopped when the interruptCtx is cancelled
			// in the defer cancel() in the goroutine above

			return nil
		},
	})
}
