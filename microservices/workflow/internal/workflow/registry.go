package workflow

import (
	"log"

	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type Registry struct {
	worker *worker.Worker
}

func NewRegistry(worker *worker.Worker) *Registry {
	return &Registry{
		worker: worker,
	}
}

// RegisterAllWorkflows registers all supported workflows
func (r *Registry) RegisterAllWorkflows() error {
	// Register shipment workflows
	if err := r.registerShipmentWorkflows(); err != nil {
		log.Printf("Error registering shipment workflows: %v", err)
		return err
	}

	if err := r.registerAliveWorkflow(); err != nil {
		log.Printf("Error registering alive workflow: %v", err)
		return err
	}

	log.Println("All workflows registered successfully")
	return nil
}
