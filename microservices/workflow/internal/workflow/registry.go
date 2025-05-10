package workflow

import (
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/microservices/workflow/internal/email"
	"github.com/emoss08/trenova/microservices/workflow/internal/model"
	"github.com/emoss08/trenova/microservices/workflow/internal/workflow/steps/delayshipmentworkflow"
	"github.com/emoss08/trenova/microservices/workflow/internal/workflow/types"
	"github.com/hatchet-dev/hatchet/pkg/client/create"
	hatchetTypes "github.com/hatchet-dev/hatchet/pkg/client/types"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/factory"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
	"github.com/uptrace/bun"
)

// Registry manages workflow registration and execution
type Registry struct {
	client      v1.HatchetClient
	db          *bun.DB
	emailClient *email.Client
}

// NewRegistry creates a new workflow registry
func NewRegistry(db *bun.DB, emailClient *email.Client, client v1.HatchetClient) *Registry {
	return &Registry{
		db:          db,
		emailClient: emailClient,
		client:      client,
	}
}

// RegisterAllWorkflows registers all the available workflows
func (r *Registry) RegisterAllWorkflows() {
	r.registerShipmentWorkflows()
}

// GetAllWorkflows collects and returns all workflow definitions
// This is used when creating the Hatchet worker
func (r *Registry) GetAllWorkflows(client v1.HatchetClient) []workflow.WorkflowBase {
	workflowList := []workflow.WorkflowBase{
		r.createDelayShipmentsWorkflow(client),
		r.createShipmentUpdatedWorkflow(client),
	}
	return workflowList
}

// createDelayShipmentsWorkflow creates the workflow for delaying shipments
func (r *Registry) createDelayShipmentsWorkflow(client v1.HatchetClient) workflow.WorkflowBase {
	delayShipmentsWf := factory.NewWorkflow[types.DelayShipmentsInput, types.DelayShipmentsOutput](
		create.WorkflowCreateOpts[types.DelayShipmentsInput]{
			Name:        "delay-shipments-workflow",
			Description: "Delay shipments based on the shipment control settings",
			OnCron:      []string{"*/10 * * * *"}, // Every 10 minutes
		},
		client,
	)

	// Add all the tasks from the shipment.go implementation
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
				return nil, fmt.Errorf("error getting parent output from get-shipment-controls: %w", err)
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

	return delayShipmentsWf
}

// createShipmentUpdatedWorkflow creates the workflow for handling shipment updates
func (r *Registry) createShipmentUpdatedWorkflow(client v1.HatchetClient) workflow.WorkflowBase {
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
		client,
	)

	// Add tasks from the shipment.go implementation
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
				return nil, fmt.Errorf("error getting previous step output from process-shipment-update: %w", err)
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

	return shipmentUpdatedWf
}
