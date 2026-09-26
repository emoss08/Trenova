package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetAccountingInboundChangeRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
	ForUpdate  bool
}

type ListAccountingInboundChangesByExternalIDsRequest struct {
	TenantInfo   pagination.TenantInfo
	ConnectionID pulid.ID
	Kind         accountingsync.InboundChangeKind
	ExternalIDs  []string
}

type ListDetectedAccountingInboundChangesRequest struct {
	TenantInfo     pagination.TenantInfo
	ConnectionID   pulid.ID
	DetectedBefore int64
	Limit          int
}

type ListAccountingInboundChangesConnectionRequest struct {
	Filter       *pagination.QueryOptions
	Cursor       pagination.CursorInfo
	ConnectionID pulid.ID
	Statuses     []accountingsync.InboundChangeStatus
	Kinds        []accountingsync.InboundChangeKind
	Reasons      []accountingsync.InboundChangeReason
	Search       string
}

type SummarizeAccountingInboundChangesRequest struct {
	TenantInfo   pagination.TenantInfo
	ConnectionID pulid.ID
	DecidedSince int64
}

type AccountingInboundSummary struct {
	Detected     int `bun:"detected"`
	Proposed     int `bun:"proposed"`
	AppliedSince int `bun:"applied_since"`
	IgnoredSince int `bun:"ignored_since"`
}

type ListAccountingInboundAttentionRequest struct {
	TenantInfo     pagination.TenantInfo
	ConnectionID   pulid.ID
	DetectedBefore int64
}

type AccountingInboundAttentionGroup struct {
	Reason           accountingsync.InboundChangeReason `bun:"reason"`
	Count            int                                `bun:"count"`
	OldestDetectedAt int64                              `bun:"oldest_detected_at"`
	SampleID         pulid.ID                           `bun:"sample_id"`
}

type AccountingInboundChangeRepository interface {
	Create(
		ctx context.Context,
		entity *accountingsync.AccountingInboundChange,
	) (*accountingsync.AccountingInboundChange, error)
	Update(
		ctx context.Context,
		entity *accountingsync.AccountingInboundChange,
	) (*accountingsync.AccountingInboundChange, error)
	GetByID(
		ctx context.Context,
		req GetAccountingInboundChangeRequest,
	) (*accountingsync.AccountingInboundChange, error)
	ListByExternalIDs(
		ctx context.Context,
		req *ListAccountingInboundChangesByExternalIDsRequest,
	) ([]*accountingsync.AccountingInboundChange, error)
	ListDetected(
		ctx context.Context,
		req *ListDetectedAccountingInboundChangesRequest,
	) ([]*accountingsync.AccountingInboundChange, error)
	ListConnection(
		ctx context.Context,
		req *ListAccountingInboundChangesConnectionRequest,
	) (*pagination.CursorListResult[*accountingsync.AccountingInboundChange], error)
	Summarize(
		ctx context.Context,
		req *SummarizeAccountingInboundChangesRequest,
	) (*AccountingInboundSummary, error)
	ListAttention(
		ctx context.Context,
		req *ListAccountingInboundAttentionRequest,
	) ([]AccountingInboundAttentionGroup, error)
}
