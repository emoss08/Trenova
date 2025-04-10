package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/emoss08/trenova/microservices/workflow/internal/config"
	"github.com/emoss08/trenova/microservices/workflow/internal/consumer"
	"github.com/emoss08/trenova/microservices/workflow/internal/model"
	"github.com/emoss08/trenova/microservices/workflow/internal/workflow"
	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/worker"
	"github.com/joho/godotenv"
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

	// Register workflows
	registry := workflow.NewRegistry(hatchetWorker)
	if err = registry.RegisterAllWorkflows(); err != nil {
		log.Printf("Failed to register workflows: %v", err)
	}

	// Create and configure scheduler
	// schedulerComponent := scheduler.NewScheduler(hatchetClient)
	// schedulerService := service.NewSchedulerService(schedulerComponent)

	// Create and configure RabbitMQ consumer
	rabbitConsumer, err := consumer.NewRabbitMQConsumer(cfg.RabbitMQ)
	if err != nil {
		log.Printf("Failed to create RabbitMQ consumer: %v", err)
	}

	// Create message handler
	handler := consumer.NewHatchetHandler(hatchetClient)

	// Register handlers for different workflow types
	rabbitConsumer.RegisterHandler(model.WorkflowTypeShipmentUpdated, handler.HandleShipmentMessage)

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

	// Schedule an example maintenance task for 1 minute from now
	// maintenanceTime := time.Now().Add(1 * time.Minute)
	// taskID, err := schedulerService.ScheduleShipmentMaintenance(
	// 	ctx,
	// 	"cleanup",
	// 	"org_12345", // Example tenant ID
	// 	100,         // Batch size
	// 	maintenanceTime,
	// )
	// if err != nil {
	// 	log.Printf("Warning: Failed to schedule maintenance task: %v", err)
	// } else {
	// 	log.Printf("Scheduled maintenance task with ID: %s at %s",
	// 		taskID, maintenanceTime.Format(time.RFC3339))
	// }

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
