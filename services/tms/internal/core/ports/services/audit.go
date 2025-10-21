package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
)

type LogActionParams struct {
	Resource       permission.Resource
	ResourceID     string
	Operation      permission.Operation
	CurrentState   map[string]any
	PreviousState  map[string]any
	UserID         pulid.ID
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	Critical       bool
}

type LogOption func(*audit.Entry) error

type SensitiveFieldAction int

const (
	SensitiveFieldOmit SensitiveFieldAction = iota
	SensitiveFieldMask
	SensitiveFieldHash
	SensitiveFieldEncrypt // New action for field-level encryption
)

type SensitiveField struct {
	Name    string               // Name of the field
	Path    string               // Optional dot-separated path to the field relative to current context. e.g. "parent.child"
	Action  SensitiveFieldAction // Action to take
	Pattern string               // Optional regex pattern to match field values
}

type AuditService interface {
	List(
		ctx context.Context,
		opts *pagination.QueryOptions,
	) (*pagination.ListResult[*audit.Entry], error)
	ListByResourceID(
		ctx context.Context,
		opts repositories.ListByResourceIDRequest,
	) (*pagination.ListResult[*audit.Entry], error)
	GetByID(ctx context.Context, opts repositories.GetAuditEntryByIDOptions) (*audit.Entry, error)
	// LiveStream(c *gin.Context) error

	LogAction(params *LogActionParams, opts ...LogOption) error

	RegisterSensitiveFields(resource permission.Resource, fields []SensitiveField) error
}
