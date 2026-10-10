package auditports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
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
	PrincipalType  kernelports.PrincipalType
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

const SensitiveFieldOmit SensitiveFieldAction = iota

const SensitiveFieldMask

const SensitiveFieldHash

const SensitiveFieldEncrypt

type SensitiveField struct {
	Name    string
	Path    string
	Action  SensitiveFieldAction
	Pattern string
}

type AuditService interface {
	List(
		ctx context.Context,
		req *ListAuditEntriesRequest,
	) (*pagination.ListResult[*audit.Entry], error)
	ListConnection(
		ctx context.Context,
		req *ListAuditEntriesConnectionRequest,
	) (*pagination.CursorListResult[*audit.Entry], error)
	ListByResourceID(
		ctx context.Context,
		req *ListByResourceIDRequest,
	) (*pagination.ListResult[*audit.Entry], error)
	GetByID(ctx context.Context, req GetAuditEntryByIDOptions) (*audit.Entry, error)
	LogAction(params *LogActionParams, opts ...LogOption) error
	LogActions(entries []BulkLogEntry) error
	RegisterSensitiveFields(resource permission.Resource, fields []SensitiveField) error
}
