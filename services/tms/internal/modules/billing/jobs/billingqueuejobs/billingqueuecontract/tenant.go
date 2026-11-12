package billingqueuecontract

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	TenantWorkflowName       = "BillingQueueTenantWorkflow"
	RecheckSweepWorkflowName = "BillingQueueRecheckSweepWorkflow"
	TenantSignalName         = "billing-queue-tenant"
	tenantWorkflowIDPrefix   = "billing-queue-tenant-"
	tenantSummary            = "Run the billing queue's due checks and write its brief"
)

// TenantPayload carries one tenant's billing queue drain across
// continue-as-new: whether a brief is still owed and when the last one was
// written, so the brief keeps its own pace while checks run every round.
type TenantPayload struct {
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	Iterations     int      `json:"iterations"`
	BriefPending   bool     `json:"briefPending"`
	LastBriefAtNs  int64    `json:"lastBriefAtNs"`
}

func (p *TenantPayload) Tenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: p.OrganizationID, BuID: p.BusinessUnitID}
}

// TenantSignal wakes the drain. The items to check are the ones marked due in
// the database, never carried here; Brief asks for the brief to be rewritten
// even when no item is due.
type TenantSignal struct {
	Brief bool `json:"brief"`
}

func TenantWorkflowID(tenantInfo pagination.TenantInfo) string {
	return tenantWorkflowIDPrefix + tenantInfo.OrgID.String() + "-" + tenantInfo.BuID.String()
}

type TenantRequesterParams struct {
	fx.In

	Signals kernelports.WorkflowSignalStarter
	Queue   billingqueueports.BillingQueueRepository
	Logger  *zap.Logger
}

// TenantRequester is the one way into a tenant's billing queue drain: item
// rechecks and brief rewrites both wake the same workflow.
type TenantRequester struct {
	signals kernelports.WorkflowSignalStarter
	queue   billingqueueports.BillingQueueRepository
	l       *zap.Logger
}

var (
	_ billingqueueports.RecheckRequester = (*TenantRequester)(nil)
	_ billingqueueports.BriefRequester   = (*TenantRequester)(nil)
)

func NewTenantRequester(p TenantRequesterParams) *TenantRequester {
	return &TenantRequester{
		signals: p.Signals,
		queue:   p.Queue,
		l:       p.Logger.Named("billing-queue-tenant-requester"),
	}
}

func (r *TenantRequester) RequestRecheck(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	itemIDs ...pulid.ID,
) error {
	if tenantInfo.OrgID.IsNil() || tenantInfo.BuID.IsNil() {
		return nil
	}
	ids := slices.DeleteFunc(slices.Clone(itemIDs), pulid.ID.IsNil)
	if len(ids) == 0 {
		return nil
	}

	if err := r.queue.MarkRecheckDue(ctx, &billingqueueports.MarkRecheckDueRequest{
		TenantInfo: tenantInfo,
		ItemIDs:    ids,
		At:         timeutils.NowUnix(),
	}); err != nil {
		return err
	}

	kernelports.AfterCommit(ctx, func(committed context.Context) {
		r.signal(committed, tenantInfo, TenantSignal{})
	})

	return nil
}

func (r *TenantRequester) WakeTenant(ctx context.Context, tenantInfo pagination.TenantInfo) {
	if tenantInfo.OrgID.IsNil() || tenantInfo.BuID.IsNil() {
		return
	}

	r.signal(ctx, tenantInfo, TenantSignal{})
}

// RequestBrief is asked from a read that found the stored brief stale, so it
// signals at once: there is no write to wait on, and a read-only context
// would never run an after-commit hook.
func (r *TenantRequester) RequestBrief(ctx context.Context, tenantInfo pagination.TenantInfo) {
	if tenantInfo.OrgID.IsNil() || tenantInfo.BuID.IsNil() {
		return
	}

	r.signal(ctx, tenantInfo, TenantSignal{Brief: true})
}

func (r *TenantRequester) signal(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	signal TenantSignal,
) {
	id := TenantWorkflowID(tenantInfo)
	_, err := r.signals.SignalWithStartWorkflow(
		ctx,
		id,
		TenantSignalName,
		signal,
		client.StartWorkflowOptions{
			ID:            id,
			TaskQueue:     temporaltype.TaskQueueBilling.String(),
			StaticSummary: tenantSummary,
			Priority:      temporal.Priority{FairnessKey: tenantInfo.OrgID.String()},
		},
		TenantWorkflowName,
		&TenantPayload{OrganizationID: tenantInfo.OrgID, BusinessUnitID: tenantInfo.BuID},
	)
	if err != nil {
		r.l.Warn("could not wake the billing queue drain; the recheck sweep will",
			zap.String("organizationId", tenantInfo.OrgID.String()),
			zap.String("businessUnitId", tenantInfo.BuID.String()),
			zap.Error(err))
	}
}

// ItemPayload names one queue item for the per-item activities the bulk Nova
// generation runs.
type ItemPayload struct {
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	ItemID         pulid.ID `json:"itemId"`
}

func (p *ItemPayload) Tenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: p.OrganizationID, BuID: p.BusinessUnitID}
}
