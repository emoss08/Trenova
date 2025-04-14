package workflow

import (
	"log"

	"github.com/hatchet-dev/hatchet/pkg/worker"
	"github.com/uptrace/bun"
)

type Registry struct {
	worker *worker.Worker
	db     *bun.DB
}

func NewRegistry(worker *worker.Worker, db *bun.DB) *Registry {
	return &Registry{
		worker: worker,
		db:     db,
	}
}

// RegisterAllWorkflows registers all supported workflows
func (r *Registry) RegisterAllWorkflows() error {
	// * Register shipment workflows
	if err := r.registerShipmentWorkflows(); err != nil {
		log.Printf("Error registering shipment workflows: %v", err)
		return err
	}

	log.Println("All workflows registered successfully")
	return nil
}
