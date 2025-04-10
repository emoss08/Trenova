package workflow

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/emoss08/trenova/microservices/workflow/internal/model"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

// registerShipmentWorkflows registers all shipment-related workflows
func (r *Registry) registerShipmentWorkflows() error {
	// Register shipment updated workflow
	if err := r.registerShipmentUpdatedWorkflow(); err != nil {
		return err
	}

	return nil
}

// registerShipmentUpdatedWorkflow registers the workflow for shipment updates
func (r *Registry) registerShipmentUpdatedWorkflow() error {
	err := r.worker.RegisterWorkflow(
		&worker.WorkflowJob{
			On:          worker.Events(string(model.WorkflowTypeShipmentUpdated)),
			Name:        "shipment-updated-workflow",
			Description: "Handles shipment update events and associated processes",
			Concurrency: worker.Expression("input.entityId"),
			Steps: []*worker.WorkflowStep{
				worker.Fn(func(ctx worker.HatchetContext) (*model.WorkflowResult, error) {
					var msg model.Message
					var payload model.ShipmentWorkflowPayload

					// Extract the workflow message
					if err := ctx.WorkflowInput(&msg); err != nil {
						return nil, fmt.Errorf("error extracting workflow input: %w", err)
					}

					// Convert the payload to bytes
					payloadBytes, err := json.Marshal(msg.Payload)
					if err != nil {
						return nil, fmt.Errorf("error marshaling payload: %w", err)
					}

					// Unmarshal the payload into the specific type
					if err = json.Unmarshal(payloadBytes, &payload); err != nil {
						return nil, fmt.Errorf("error unmarshaling payload: %w", err)
					}

					log.Printf("Processing shipment update for %s with status %s",
						payload.ProNumber, payload.Status)

					return &model.WorkflowResult{
						Success:  true,
						Message:  fmt.Sprintf("Processed shipment %s", payload.ProNumber),
						Status:   payload.Status,
						EntityID: msg.EntityID,
					}, nil
				}).SetName("process-shipment-update"),

				worker.Fn(func(ctx worker.HatchetContext) (*model.WorkflowResult, error) {
					var result model.WorkflowResult
					var msg model.Message

					// Get workflow input for context
					if err := ctx.WorkflowInput(&msg); err != nil {
						return nil, fmt.Errorf("error extracting workflow input: %w", err)
					}

					// Get previous step output
					if err := ctx.StepOutput("process-shipment-update", &result); err != nil {
						return nil, fmt.Errorf("error getting previous step output: %w", err)
					}

					return &model.WorkflowResult{
						Success:  true,
						Message:  "Notifications sent for shipment update",
						Status:   result.Status,
						EntityID: result.EntityID,
					}, nil
				}).SetName("send-notifications").AddParents("process-shipment-update"),
			},
		},
	)

	return err
}

func (r *Registry) registerAliveWorkflow() error {
	err := r.worker.RegisterWorkflow(
		&worker.WorkflowJob{
			On:   worker.Cron("* * * * *"), // Every Minute
			Name: "alive-workflow",
			Steps: []*worker.WorkflowStep{
				worker.Fn(func(worker.HatchetContext) (*model.WorkflowResult, error) {
					log.Println("Alive workflow executed")
					return &model.WorkflowResult{
						Success: true,
						Message: "Alive workflow executed",
					}, nil
				}).SetName("alive-step"),
			},
		},
	)

	return err
}
