package services

import (
	"context"

	"github.com/trenova-app/transport/internal/core/domain/audit"
	"github.com/trenova-app/transport/internal/core/domain/permission"
	"github.com/trenova-app/transport/pkg/types/pulid"
)

type LogActionParams struct {
	Resource       permission.Resource
	ResourceID     string
	Action         permission.Action
	CurrentState   map[string]any
	PreviousState  map[string]any
	UserID         pulid.ID
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}

type LogOption func(*audit.Entry) error

type AuditService interface {
	LogAction(params *LogActionParams, opts ...LogOption) error
	Start() error
	Stop(ctx context.Context) error
}
