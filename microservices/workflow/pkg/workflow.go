package pkg

// ShipmentWorkflowPayload contains shipment-specific workflow data
type ShipmentWorkflowPayload struct {
	ProNumber string `json:"proNumber"`
	Status    string `json:"status"`
	// Add other relevant shipment fields
}
