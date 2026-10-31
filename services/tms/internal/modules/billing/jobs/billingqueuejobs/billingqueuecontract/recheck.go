package billingqueuecontract

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	RecheckWorkflowName      = "BillingQueueRecheckWorkflow"
	RecheckSweepWorkflowName = "BillingQueueRecheckSweepWorkflow"
	RecheckSignalName        = "billing-queue-recheck"
	recheckWorkflowIDPrefix  = "billing-queue-recheck-"
)

type RecheckPayload struct {
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	ItemID         pulid.ID `json:"itemId"`
	Iterations     int      `json:"iterations"`
}

func (p *RecheckPayload) Tenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: p.OrganizationID, BuID: p.BusinessUnitID}
}

type RecheckSignal struct{}

func RecheckWorkflowID(itemID pulid.ID) string {
	return recheckWorkflowIDPrefix + itemID.String()
}

type RecheckRequesterParams struct {
	fx.In

	Signals kernelports.WorkflowSignalStarter `optional:"true"`
	Logger  *zap.Logger
}

type RecheckRequester struct {
	signals kernelports.WorkflowSignalStarter
	l       *zap.Logger
}

var _ billingqueueports.RecheckRequester = (*RecheckRequester)(nil)

func NewRecheckRequester(p RecheckRequesterParams) billingqueueports.RecheckRequester {
	return &RecheckRequester{
		signals: p.Signals,
		l:       p.Logger.Named("billing-queue-recheck-requester"),
	}
}

func (r *RecheckRequester) RequestRecheck(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	itemIDs ...pulid.ID,
) {
	if r.signals == nil || tenantInfo.OrgID.IsNil() || len(itemIDs) == 0 {
		return
	}

	kernelports.AfterCommit(ctx, func(committed context.Context) {
		for _, itemID := range itemIDs {
			r.signal(committed, tenantInfo, itemID)
		}
	})
}

func (r *RecheckRequester) signal(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	itemID pulid.ID,
) {
	if itemID.IsNil() {
		return
	}

	_, err := r.signals.SignalWithStartWorkflow(
		ctx,
		RecheckWorkflowID(itemID),
		RecheckSignalName,
		RecheckSignal{},
		client.StartWorkflowOptions{
			ID:            RecheckWorkflowID(itemID),
			TaskQueue:     temporaltype.TaskQueueBilling.String(),
			StaticSummary: "Run the billing checks on a queue item",
			Priority:      temporal.Priority{FairnessKey: tenantInfo.OrgID.String()},
		},
		RecheckWorkflowName,
		&RecheckPayload{
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			ItemID:         itemID,
		},
	)
	if err != nil {
		r.l.Warn("could not request a billing queue recheck",
			zap.String("billingQueueItemId", itemID.String()),
			zap.Error(err))
	}
}
