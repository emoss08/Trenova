package services

import (
	"context"
	"time"

	"github.com/emoss08/trenova/shared/pulid"
)

type RealtimeTokenRequest struct {
	KeyName    string `json:"keyName"`
	ClientID   string `json:"clientId"`
	Nonce      string `json:"nonce"`
	MAC        string `json:"mac"`
	Capability string `json:"capability"`
	Timestamp  int64  `json:"timestamp"`
	TTL        int64  `json:"ttl"`
}

type CreateRealtimeTokenRequest struct {
	UserID         pulid.ID `json:"userId"`
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
}

type ResourceInvalidationEvent struct {
	EventID        string    `json:"eventId"`
	OrganizationID string    `json:"organizationId"`
	BusinessUnitID string    `json:"businessUnitId"`
	Type           string    `json:"type"`
	Resource       string    `json:"resource"`
	Action         string    `json:"action"`
	EntityID       string    `json:"entityId,omitempty"`
	Fields         []string  `json:"fields,omitempty"`
	EntityVersion  int64     `json:"entityVersion,omitempty"`
	Entity         any       `json:"entity,omitempty"`
	RecordID       string    `json:"recordId,omitempty"`
	ActorUserID    string    `json:"actorUserId,omitempty"`
	ActorType      string    `json:"actorType,omitempty"`
	ActorID        string    `json:"actorId,omitempty"`
	ActorAPIKeyID  string    `json:"actorApiKeyId,omitempty"`
	OccurredAt     time.Time `json:"occurredAt"`
}

type PublishResourceInvalidationRequest struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	Resource       string
	Action         string
	EventType      string
	Fields         []string
	EntityVersion  int64
	Entity         any
	RecordID       pulid.ID
	ActorUserID    pulid.ID
	ActorType      PrincipalType
	ActorID        pulid.ID
	ActorAPIKeyID  pulid.ID
}

type RealtimeService interface {
	CreateTokenRequest(
		req *CreateRealtimeTokenRequest,
	) (*RealtimeTokenRequest, error)
	PublishResourceInvalidation(
		ctx context.Context,
		req *PublishResourceInvalidationRequest,
	) error
}
