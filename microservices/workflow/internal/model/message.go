package model

import "time"

type Type string

const (
	// * Shipment Workflow
	TypeShipmentUpdated = ("shipment_updated")
)

// * Message is the message that is sent to the workflow service
//
// ! Should match the same message struct in `internal/pkg/workflow/message.go`
type Message struct {
	ID          string    `json:"id"`
	Type        Type      `json:"type"`
	EntityID    string    `json:"entityId"`
	EntityType  string    `json:"entityType"`
	TenantID    string    `json:"tenantId"`
	RequestedAt time.Time `json:"requestedAt"`
	Payload     any       `json:"payload"`
}
