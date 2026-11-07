package auditports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const AlertWorkflowName = "AuditAlertWorkflow"

type AlertPayload struct {
	TenantInfo    pagination.TenantInfo `json:"tenantInfo"`
	Kind          audit.AlertKind       `json:"kind"`
	EntryID       pulid.ID              `json:"entryId"`
	Action        string                `json:"action"`
	Resource      permission.Resource   `json:"resource"`
	ResourceID    string                `json:"resourceId"`
	PrincipalID   pulid.ID              `json:"principalId"`
	PrincipalType string                `json:"principalType"`
	ActorName     string                `json:"actorName"`
	EntityRef     string                `json:"entityRef"`
	Comment       string                `json:"comment"`
	Count         int64                 `json:"count"`
	OccurredAt    int64                 `json:"occurredAt"`
}

type AlertPolicyRepository interface {
	ListForTenant(ctx context.Context, tenant pagination.TenantInfo) ([]*audit.AlertPolicy, error)
	Upsert(ctx context.Context, policy *audit.AlertPolicy) (*audit.AlertPolicy, error)
}

type SealComputation struct {
	TenantKey    TenantKey
	FromTS       int64
	ToTS         int64
	RowCount     int64
	ClassDigests map[string]audit.ClassDigest
}

type SealRepository interface {
	GetHead(ctx context.Context, tenant pagination.TenantInfo) (*audit.ChainHead, error)
	ListHeads(ctx context.Context) ([]*audit.ChainHead, error)
	Digest(ctx context.Context, tenant TenantKey, from, to int64) (*SealComputation, error)
	AppendSeal(
		ctx context.Context,
		tenant TenantKey,
		seal func(head *audit.ChainHead) (*audit.Seal, error),
	) (*audit.Seal, error)
	ListSeals(
		ctx context.Context,
		tenant pagination.TenantInfo,
		afterSeq int64,
		limit int,
	) ([]*audit.Seal, error)
	RecordVerification(ctx context.Context, head *audit.ChainHead) error
	DeleteSealsBefore(ctx context.Context, before int64) (int64, error)
}

type ExportRepository interface {
	Create(ctx context.Context, export *audit.Export) (*audit.Export, error)
	Update(ctx context.Context, export *audit.Export) (*audit.Export, error)
	GetByID(ctx context.Context, req GetExportRequest) (*audit.Export, error)
	ListConnection(
		ctx context.Context,
		req *ListExportsRequest,
	) (*pagination.CursorListResult[*audit.Export], error)
	ListExpired(ctx context.Context, now int64, limit int) ([]*audit.Export, error)
}

type GetExportRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
}

type ListExportsRequest struct {
	Filter        *pagination.QueryOptions
	Cursor        pagination.CursorInfo
	RequestedByID pulid.ID
}
