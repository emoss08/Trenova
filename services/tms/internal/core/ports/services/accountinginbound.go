package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type PollAccountingChangesRequest struct {
	TenantInfo   pagination.TenantInfo
	ConnectionID pulid.ID
}

type AccountingChangesPollResult struct {
	Held          bool
	More          bool
	CursorExpired bool
	Payments      int
	Recorded      int
	References    int
}

type EvaluateAccountingInboundRequest struct {
	TenantInfo   pagination.TenantInfo
	ConnectionID pulid.ID
	Limit        int
}

type AccountingInboundEvaluation struct {
	Evaluated  int
	Applied    int
	Proposed   int
	Ignored    int
	Superseded int
	Waiting    bool
	More       bool
}

type ListAccountingInboundChangesRequest struct {
	Filter          *pagination.QueryOptions
	Cursor          pagination.CursorInfo
	IntegrationType integration.Type
	Statuses        []accountingsync.InboundChangeStatus
	Kinds           []accountingsync.InboundChangeKind
	Reasons         []accountingsync.InboundChangeReason
	Search          string
}

type GetAccountingInboundChangeRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
}

type AccountingInboundOverviewRequest struct {
	TenantInfo      pagination.TenantInfo
	IntegrationType integration.Type
}

type AccountingInboundOverview struct {
	ConnectionID  pulid.ID
	Policy        accountingsync.InboundPaymentPolicy
	ChangesReadAt *int64
	ChangesError  string
	Summary       *repositories.AccountingInboundSummary
}

type DecideAccountingInboundChangeRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
	Note       string
}

type AccountingInboundPostingLine struct {
	ObjectType   accountingsync.SyncObjectType
	ObjectID     pulid.ID
	ObjectNumber string
	AmountMinor  int64
	OpenMinor    int64
}

type AccountingInboundApplyPreview struct {
	Change         *accountingsync.AccountingInboundChange
	CanApply       bool
	Blocker        string
	PaidAt         int64
	CashMinor      int64
	UnappliedMinor int64
	Lines          []AccountingInboundPostingLine
}

type AccountingInboundService interface {
	PollChanges(
		ctx context.Context,
		req *PollAccountingChangesRequest,
	) (*AccountingChangesPollResult, error)
	Evaluate(
		ctx context.Context,
		req *EvaluateAccountingInboundRequest,
	) (*AccountingInboundEvaluation, error)
	List(
		ctx context.Context,
		req *ListAccountingInboundChangesRequest,
	) (*pagination.CursorListResult[*accountingsync.AccountingInboundChange], error)
	Get(
		ctx context.Context,
		req *GetAccountingInboundChangeRequest,
	) (*accountingsync.AccountingInboundChange, error)
	Overview(
		ctx context.Context,
		req *AccountingInboundOverviewRequest,
	) (*AccountingInboundOverview, error)
	PreviewApply(
		ctx context.Context,
		req *GetAccountingInboundChangeRequest,
	) (*AccountingInboundApplyPreview, error)
	Apply(
		ctx context.Context,
		req *DecideAccountingInboundChangeRequest,
		actor *RequestActor,
	) (*accountingsync.AccountingInboundChange, error)
	Ignore(
		ctx context.Context,
		req *DecideAccountingInboundChangeRequest,
		actor *RequestActor,
	) (*accountingsync.AccountingInboundChange, error)
}

type AccountingChangePoller interface {
	PollNow(ctx context.Context, tenantInfo pagination.TenantInfo, connectionID pulid.ID) error
}
