package workflow

import (
	"fmt"
	"log"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/microservices/workflow/internal/model"
	"github.com/emoss08/trenova/microservices/workflow/internal/workflow/steps/delayshipmentworkflow"
	"github.com/emoss08/trenova/microservices/workflow/internal/workflow/types"
	"github.com/hatchet-dev/hatchet/pkg/client/create"
	hatchetTypes "github.com/hatchet-dev/hatchet/pkg/client/types"
	"github.com/hatchet-dev/hatchet/pkg/v1/factory"
	"github.com/hatchet-dev/hatchet/pkg/worker"
	// WorkflowDeclaration is returned by factory
	// "github.com/hatchet-dev/hatchet/pkg/worker" // For HatchetContext and step functions
)

// registerShipmentWorkflows registers all shipment-related workflows.
// This function is typically called by a central RegisterAllWorkflows method in the Registry.
func (r *Registry) registerShipmentWorkflows() {
	r.registerDelayShipmentsWorkflowV1()
	r.registerShipmentUpdatedWorkflowV1()
}

func (r *Registry) registerDelayShipmentsWorkflowV1() {
	delayShipmentsWf := factory.NewWorkflow[types.DelayShipmentsInput, types.DelayShipmentsOutput](
		create.WorkflowCreateOpts[types.DelayShipmentsInput]{
			Name:        "delay-shipments-workflow",
			Description: "Delay shipments based on the shipment control settings",
			OnCron:      []string{"*/10 * * * *"}, // Every 10 minutes
		},
		r.client,
	)

	getShipmentControlsStep := delayShipmentsWf.Task(
		create.WorkflowTask[types.DelayShipmentsInput, types.DelayShipmentsOutput]{
			Name: "get-shipment-controls",
		},
		func(ctx worker.HatchetContext, input types.DelayShipmentsInput) (any, error) {
			stepFn := delayshipmentworkflow.QueryShipmentControls(r.db)
			return stepFn(ctx, &types.QueryShipmentControlsInput{})
		},
	)

	queryStopsStep := delayShipmentsWf.Task(
		create.WorkflowTask[types.DelayShipmentsInput, types.DelayShipmentsOutput]{
			Name: "query-stops",
			Parents: []create.NamedTask{
				getShipmentControlsStep,
			},
		},
		func(ctx worker.HatchetContext, input types.DelayShipmentsInput) (any, error) {
			var controlsOutput types.QueryShipmentControlsOutput
			if err := ctx.ParentOutput(getShipmentControlsStep, &controlsOutput); err != nil {
				return nil, fmt.Errorf(
					"error getting parent output from get-shipment-controls: %w",
					err,
				)
			}

			stopsInput := &types.QueryStopsInput{StepOutput: controlsOutput}
			stepFn := delayshipmentworkflow.QueryStops(r.db)
			return stepFn(ctx, stopsInput)
		},
	)

	delayShipmentsWf.Task(
		create.WorkflowTask[types.DelayShipmentsInput, types.DelayShipmentsOutput]{
			Name: "delay-shipments",
			Parents: []create.NamedTask{
				queryStopsStep,
			},
		},
		func(ctx worker.HatchetContext, wfInput types.DelayShipmentsInput) (any, error) {
			var stopsOutput types.QueryStopsOutput
			if err := ctx.ParentOutput(queryStopsStep, &stopsOutput); err != nil {
				return nil, fmt.Errorf("error getting parent output from query-stops: %w", err)
			}
			stepInput := &types.DelayShipmentsInput{StepOutput: stopsOutput}

			stepFn := delayshipmentworkflow.DelayShipments(r.db, r.emailClient)
			return stepFn(ctx, stepInput)
		},
	)
}

func (r *Registry) registerShipmentUpdatedWorkflowV1() {
	var maxRuns int32 = 1
	strategy := hatchetTypes.GroupRoundRobin

	shipmentUpdatedWf := factory.NewWorkflow[types.ShipmentUpdatedWorkflowInput, types.ShipmentUpdatedWorkflowOutput](
		create.WorkflowCreateOpts[types.ShipmentUpdatedWorkflowInput]{
			Name:        "shipment-updated-workflow",
			Description: "Handles shipment update events and associated processes",
			OnEvents:    []string{string(model.TypeShipmentUpdated)},
			Concurrency: &hatchetTypes.Concurrency{
				Expression:    "input.EntityID",
				MaxRuns:       &maxRuns,
				LimitStrategy: &strategy,
			},
		},
		r.client,
	)

	processUpdateStep := shipmentUpdatedWf.Task(
		create.WorkflowTask[types.ShipmentUpdatedWorkflowInput, types.ShipmentUpdatedWorkflowOutput]{
			Name: "process-shipment-update",
		},
		func(ctx worker.HatchetContext, workflowMsg types.ShipmentUpdatedWorkflowInput) (any, error) {
			var payload model.ShipmentWorkflowPayload

			payloadBytes, err := sonic.Marshal(workflowMsg.Payload)
			if err != nil {
				return nil, fmt.Errorf("error marshaling workflow message payload: %w", err)
			}
			if err = sonic.Unmarshal(payloadBytes, &payload); err != nil {
				return nil, fmt.Errorf("error unmarshaling shipment workflow payload: %w", err)
			}

			log.Printf("Processing shipment update for %s with status %s",
				payload.ProNumber, payload.Status)

			result := &model.WorkflowResult{
				Success:  true,
				Message:  fmt.Sprintf("Processed shipment %s", payload.ProNumber),
				Status:   payload.Status,
				EntityID: workflowMsg.EntityID,
			}
			return result, nil
		},
	)

	shipmentUpdatedWf.Task(
		create.WorkflowTask[types.ShipmentUpdatedWorkflowInput, types.ShipmentUpdatedWorkflowOutput]{
			Name: "send-notifications",
			Parents: []create.NamedTask{
				processUpdateStep,
			},
		},
		func(ctx worker.HatchetContext, workflowMsg types.ShipmentUpdatedWorkflowInput) (any, error) {
			var prevStepResult model.WorkflowResult

			if err := ctx.ParentOutput(processUpdateStep, &prevStepResult); err != nil {
				return nil, fmt.Errorf(
					"error getting previous step output from process-shipment-update: %w",
					err,
				)
			}

			result := &model.WorkflowResult{
				Success:  true,
				Message:  "Notifications sent for shipment update",
				Status:   prevStepResult.Status,
				EntityID: prevStepResult.EntityID,
			}
			return result, nil
		},
	)
}
