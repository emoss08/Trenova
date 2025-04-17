package workflow

import (
	"log"

	"github.com/emoss08/trenova/microservices/workflow/internal/email"
	"github.com/hatchet-dev/hatchet/pkg/worker"
	"github.com/uptrace/bun"
)

// Registry handles the registration of workflows
type Registry struct {
	worker      *worker.Worker
	db          *bun.DB
	emailClient *email.Client
}

// NewRegistry creates a new workflow registry
func NewRegistry(worker *worker.Worker, db *bun.DB, emailClient *email.Client) *Registry {
	return &Registry{
		worker:      worker,
		db:          db,
		emailClient: emailClient,
	}
}

// RegisterAllWorkflows registers all workflows
func (r *Registry) RegisterAllWorkflows() error {
	// Register shipment workflows
	if err := r.registerShipmentWorkflows(); err != nil {
		log.Printf("Error registering shipment workflows: %v", err)
		return err
	}

	return nil
}
