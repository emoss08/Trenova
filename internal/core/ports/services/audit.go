package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/types/pulid"
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

// SensitiveField is a field that is considered sensitive and should be masked.
type SensitiveField struct {
	Name    string
	Action  SensitiveFieldAction
	Pattern string // Optional regex pattern for more precise masking
}

type AuditService interface {
	// API functionality
	List(ctx context.Context, opts *ports.LimitOffsetQueryOptions) (*ports.ListResult[*audit.Entry], error)
	ListByResourceID(ctx context.Context, opts repositories.ListByResourceIDRequest) (*ports.ListResult[*audit.Entry], error)
	GetByID(ctx context.Context, opts repositories.GetAuditEntryByIDOptions) (*audit.Entry, error)

	// Core functionality
	LogAction(params *LogActionParams, opts ...LogOption) error
	Start() error
	Stop() error

	// New methods for enhanced functionality
	RegisterSensitiveFields(resource permission.Resource, fields []SensitiveField) error
	SetDefaultField(key string, value any)
	GetServiceStatus() string
}
