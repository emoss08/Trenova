package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type LogActionParams struct {
	Resource       permission.Resource
	ResourceID     string
	Operation      permission.Operation
	CurrentState   map[string]any
	PreviousState  map[string]any
	UserID         pulid.ID
	PrincipalType  PrincipalType
	PrincipalID    pulid.ID
	APIKeyID       pulid.ID
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	Critical       bool
}

type LogOption func(*audit.Entry) error

type BulkLogEntry struct {
	Params  *LogActionParams
	Options []LogOption
}

type SensitiveFieldAction int

const (
	SensitiveFieldOmit SensitiveFieldAction = iota
	SensitiveFieldMask
	SensitiveFieldHash
	SensitiveFieldEncrypt
)

type SensitiveField struct {
	Name    string
	Path    string
	Action  SensitiveFieldAction
	Pattern string
}

type AuditService interface {
	List(
		ctx context.Context,
		req *repositories.ListAuditEntriesRequest,
	) (*pagination.ListResult[*audit.Entry], error)
	ListByResourceID(
		ctx context.Context,
		req *repositories.ListByResourceIDRequest,
	) (*pagination.ListResult[*audit.Entry], error)
	GetByID(ctx context.Context, req repositories.GetAuditEntryByIDOptions) (*audit.Entry, error)
	LogAction(params *LogActionParams, opts ...LogOption) error
	LogActions(entries []BulkLogEntry) error
	RegisterSensitiveFields(resource permission.Resource, fields []SensitiveField) error
}
