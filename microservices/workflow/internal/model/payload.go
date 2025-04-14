package model

// * ShipmentWorkflowPayload contains shipment-specific workflow data
//
// ! Should match the same payload struct in `internal/pkg/workflow/message.go`
type ShipmentWorkflowPayload struct {
	ProNumber string `json:"proNumber"`
	Status    string `json:"status"`
}

// WorkflowResult represents common return data from workflow steps
type WorkflowResult struct {
	Success     bool     `json:"success"`
	Message     string   `json:"message"`
	Status      string   `json:"status,omitempty"`
	EntityID    string   `json:"entityId,omitempty"`
	NextActions []string `json:"nextActions,omitempty"`
}

type AlivePayload struct{}
