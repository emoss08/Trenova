package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
)

type ToolExecuteParams struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	Actor          *RequestActor
	IdempotencyKey string
	Params         map[string]any
}

type AgentTool interface {
	Name() string
	Description() string
	ParamSchema() map[string]any
	Reversible() bool
	PermissionResource() permission.Resource
	PermissionOperation() permission.Operation
	RequiresIdempotencyKey() bool
	DefaultAutonomyTier() agent.AutonomyTier
	Execute(ctx context.Context, params ToolExecuteParams) error
}

type AgentToolDescriptor struct {
	Name         string             `json:"name"`
	Description  string             `json:"description"`
	Parameters   map[string]any     `json:"parameters"`
	AutonomyTier agent.AutonomyTier `json:"autonomyTier"`
}

type AgentToolRegistry interface {
	Get(name string) (AgentTool, bool)
	All() []AgentTool
	Descriptors() []AgentToolDescriptor
}
