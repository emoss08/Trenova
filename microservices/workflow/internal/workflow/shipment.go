package workflow

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/emoss08/trenova/microservices/workflow/internal/model"
	"github.com/emoss08/trenova/microservices/workflow/internal/workflow/steps/delayshipmentworkflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

// registerShipmentWorkflows registers all shipment-related workflows
func (r *Registry) registerShipmentWorkflows() error {
	// Register shipment updated workflow
	if err := r.registerShipmentUpdatedWorkflow(); err != nil {
		return err
	}

	if err := r.registerDelayShipmentsWorkflow(); err != nil {
		return err
	}

	return nil
}

func (r *Registry) registerDelayShipmentsWorkflow() error {
	return r.worker.RegisterWorkflow(
		&worker.WorkflowJob{
			Name:        "delay-shipments-workflow",
			Description: "Delay shipments based on the shipment control settings",
			On:          worker.Cron("*/10 * * * *"), // Every 10 minutes
			Steps: []*worker.WorkflowStep{
				worker.Fn(delayshipmentworkflow.QueryShipmentControls(r.db)).
					SetName("get-shipment-controls"),
				worker.Fn(delayshipmentworkflow.QueryStops(r.db)).
					SetName("query-stops").
					AddParents("get-shipment-controls"),
				worker.Fn(delayshipmentworkflow.DelayShipments(r.db, r.emailClient)).
					SetName("delay-shipments").
					AddParents("query-stops"),
			},
		},
	)
}

// registerShipmentUpdatedWorkflow registers the workflow for shipment updates
func (r *Registry) registerShipmentUpdatedWorkflow() error {
	return r.worker.RegisterWorkflow(
		&worker.WorkflowJob{
			On:          worker.Events(string(model.TypeShipmentUpdated)),
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
}
