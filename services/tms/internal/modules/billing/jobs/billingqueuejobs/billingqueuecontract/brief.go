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
	BriefWorkflowName     = "BillingQueueBriefWorkflow"
	BriefSignalName       = "billing-queue-brief"
	briefWorkflowIDPrefix = "billing-queue-brief-"
)

type BriefPayload struct {
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	Iterations     int      `json:"iterations"`
}

func (p *BriefPayload) Tenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: p.OrganizationID, BuID: p.BusinessUnitID}
}

type BriefSignal struct{}

func BriefWorkflowID(tenantInfo pagination.TenantInfo) string {
	return briefWorkflowIDPrefix + tenantInfo.OrgID.String() + "-" + tenantInfo.BuID.String()
}

type BriefRequesterParams struct {
	fx.In

	Signals kernelports.WorkflowSignalStarter `optional:"true"`
	Logger  *zap.Logger
}

type BriefRequester struct {
	signals kernelports.WorkflowSignalStarter
	l       *zap.Logger
}

var _ billingqueueports.BriefRequester = (*BriefRequester)(nil)

func NewBriefRequester(p BriefRequesterParams) billingqueueports.BriefRequester {
	return &BriefRequester{signals: p.Signals, l: p.Logger.Named("billing-queue-brief-requester")}
}

func (r *BriefRequester) RequestBrief(ctx context.Context, tenantInfo pagination.TenantInfo) {
	if r.signals == nil || tenantInfo.OrgID.IsNil() || tenantInfo.BuID.IsNil() {
		return
	}

	id := BriefWorkflowID(tenantInfo)
	_, err := r.signals.SignalWithStartWorkflow(
		ctx,
		id,
		BriefSignalName,
		BriefSignal{},
		client.StartWorkflowOptions{
			ID:            id,
			TaskQueue:     temporaltype.TaskQueueBilling.String(),
			StaticSummary: "Write the billing queue's brief",
			Priority:      temporal.Priority{FairnessKey: tenantInfo.OrgID.String()},
		},
		BriefWorkflowName,
		&BriefPayload{OrganizationID: tenantInfo.OrgID, BusinessUnitID: tenantInfo.BuID},
	)
	if err != nil {
		r.l.Warn("could not request a billing queue brief",
			zap.String("organizationId", tenantInfo.OrgID.String()),
			zap.Error(err))
	}
}
