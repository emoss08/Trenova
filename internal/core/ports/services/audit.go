package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/fiber/v2"
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

// SensitiveField defines a sensitive field and the action to take.
type SensitiveField struct {
	Name    string               // Name of the field
	Path    string               // Optional dot-separated path to the field relative to current context. e.g. "parent.child"
	Action  SensitiveFieldAction // Action to take
	Pattern string               // Optional regex pattern to match field values
}

type AuditService interface {
	// API functionality
	List(
		ctx context.Context,
		opts *ports.LimitOffsetQueryOptions,
	) (*ports.ListResult[*audit.Entry], error)
	ListByResourceID(
		ctx context.Context,
		opts repositories.ListByResourceIDRequest,
	) (*ports.ListResult[*audit.Entry], error)
	GetByID(ctx context.Context, opts repositories.GetAuditEntryByIDOptions) (*audit.Entry, error)
	LiveStream(
		c *fiber.Ctx,
		dataFetcher func(ctx context.Context, reqCtx *appctx.RequestContext) ([]*audit.Entry, error),
		timestampExtractor func(entry *audit.Entry) int64,
	) error

	// Core functionality
	LogAction(params *LogActionParams, opts ...LogOption) error
	Start() error
	Stop() error

	// New methods for enhanced functionality
	RegisterSensitiveFields(resource permission.Resource, fields []SensitiveField) error
	SetDefaultField(key string, value any)
	GetServiceStatus() string
}
