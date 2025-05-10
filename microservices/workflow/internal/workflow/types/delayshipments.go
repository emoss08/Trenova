package types

import "github.com/emoss08/trenova/microservices/workflow/internal/model"

type ShipmentControlResults struct {
	OrganizationID              string `json:"organizationId" bun:"organization_id"`
	AutoDelayShipments          bool   `json:"autoDelayShipments" bun:"auto_delay_shipments"`
	AutoDelayShipmentsThreshold int16  `json:"autoDelayShipmentsThreshold" bun:"auto_delay_shipments_threshold"`
}

type QueryShipmentControlsInput struct{}

type QueryShipmentControlsOutput struct {
	// * The organizations that have `auto_delay_shipments` set to true
	Organizations []ShipmentControlResults `json:"organizations"`
}

type QueryStopsInput struct {
	StepOutput QueryShipmentControlsOutput `json:"stepOutput"`
}

type QueryStopsOutput struct {
	PastDueStops []StopResults `json:"pastDueStops"`
}

type StopResults struct {
	StopID         string `json:"stopId" bun:"stop_id"`
	OrganizationID string `json:"organizationId" bun:"organization_id"`
	ShipmentMoveID string `json:"shipmentMoveId" bun:"shipment_move_id"`
}

type DelayShipmentsInput struct {
	StepOutput QueryStopsOutput `json:"stepOutput"`
}

type DelayShipmentsOutput struct {
	DelayedShipments int `json:"delayedShipments"`
}

// ShipmentUpdatedWorkflowInput is an alias for model.Message for the shipment updated workflow.
type ShipmentUpdatedWorkflowInput = model.Message

// ShipmentUpdatedWorkflowOutput is an alias for model.WorkflowResult for the shipment updated workflow.
type ShipmentUpdatedWorkflowOutput = model.WorkflowResult
