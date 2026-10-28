package accountinginboundports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountinginbound"
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
	Kind         accountinginbound.InboundChangeKind
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
	Statuses     []accountinginbound.InboundChangeStatus
	Kinds        []accountinginbound.InboundChangeKind
	Reasons      []accountinginbound.InboundChangeReason
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
	Reason           accountinginbound.InboundChangeReason
	Count            int
	OldestDetectedAt int64
	SampleID         pulid.ID
}

type AccountingInboundChangeRepository interface {
	Create(
		ctx context.Context,
		entity *accountinginbound.AccountingInboundChange,
	) (*accountinginbound.AccountingInboundChange, error)
	Update(
		ctx context.Context,
		entity *accountinginbound.AccountingInboundChange,
	) (*accountinginbound.AccountingInboundChange, error)
	GetByID(
		ctx context.Context,
		req GetAccountingInboundChangeQuery,
	) (*accountinginbound.AccountingInboundChange, error)
	ListByExternalIDs(
		ctx context.Context,
		req *ListAccountingInboundChangesByExternalIDsRequest,
	) ([]*accountinginbound.AccountingInboundChange, error)
	ListDetected(
		ctx context.Context,
		req *ListDetectedAccountingInboundChangesRequest,
	) ([]*accountinginbound.AccountingInboundChange, error)
	ListConnection(
		ctx context.Context,
		req *ListAccountingInboundChangesConnectionRequest,
	) (*pagination.CursorListResult[*accountinginbound.AccountingInboundChange], error)
	Summarize(
		ctx context.Context,
		req *SummarizeAccountingInboundChangesRequest,
	) (*AccountingInboundSummary, error)
	ListAttention(
		ctx context.Context,
		req *ListAccountingInboundAttentionRequest,
	) ([]AccountingInboundAttentionGroup, error)
}
