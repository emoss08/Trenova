package accountinginboundports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetAccountingInboundChangeQuery struct {
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
	Detected     int
	Proposed     int
	AppliedSince int
	IgnoredSince int
}

type ListAccountingInboundAttentionRequest struct {
	TenantInfo     pagination.TenantInfo
	ConnectionID   pulid.ID
	DetectedBefore int64
}

type AccountingInboundAttentionGroup struct {
	Reason           accountingsync.InboundChangeReason
	Count            int
	OldestDetectedAt int64
	SampleID         pulid.ID
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
		req GetAccountingInboundChangeQuery,
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
