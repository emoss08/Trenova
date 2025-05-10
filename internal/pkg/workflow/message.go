package workflow

import (
	"time"

	"github.com/emoss08/trenova/pkg/types/pulid"
)

type Message struct {
	ID          string    `json:"id"`
	Type        Type      `json:"type"`
	EntityID    string    `json:"entityId"`
	EntityType  string    `json:"entityType"`
	TenantID    string    `json:"tenantId"` // * Note: Usually the Organization ID
	RequestedAt time.Time `json:"requestedAt"`
	Payload     any       `json:"payload"`
}

// NewMessage creates a new Message instance
func NewMessage(
	workflowType Type,
	entityID pulid.ID,
	entityType string,
	tenantID pulid.ID,
	payload any,
) *Message {
	return &Message{
		ID:          pulid.MustNew("wf_").String(),
		Type:        workflowType,
		EntityID:    entityID.String(),
		EntityType:  entityType,
		TenantID:    tenantID.String(),
		RequestedAt: time.Now(),
		Payload:     payload,
	}
}

// ShipmentWorkflowPayload contains shipment-specific workflow data
type ShipmentWorkflowPayload struct {
	ProNumber string `json:"proNumber"`
	Status    string `json:"status"`
}

// ReadyToBillWorkflowPayload contains ready-to-bill-specific workflow data
type ReadyToBillWorkflowPayload struct {
	ShipmentID string `json:"shipmentId"`
	OrgID      string `json:"orgId"`
	BuID       string `json:"buId"`
}
