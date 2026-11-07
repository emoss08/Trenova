package dispatchcoveragecontract

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/dispatchconsoleports"
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
	RefreshWorkflowName     = "DispatchCoverageRefreshWorkflow"
	RefreshSignalName       = "dispatch-coverage-refresh"
	refreshWorkflowIDPrefix = "dispatch-coverage-refresh-"
)

type RefreshPayload struct {
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	Iterations     int      `json:"iterations"`
}

func (p *RefreshPayload) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: p.OrganizationID, BuID: p.BusinessUnitID}
}

type RefreshSignal struct{}

// RefreshWorkflowID is one workflow per organization and business unit: a
// burst of new and uncovered loads re-plans once, not once per load.
func RefreshWorkflowID(tenantInfo pagination.TenantInfo) string {
	return refreshWorkflowIDPrefix + tenantInfo.OrgID.String() + "-" + tenantInfo.BuID.String()
}

type RequesterParams struct {
	fx.In

	Signals kernelports.WorkflowSignalStarter
	Logger  *zap.Logger
}

type Requester struct {
	signals kernelports.WorkflowSignalStarter
	l       *zap.Logger
}

var _ dispatchconsoleports.CoverageRefreshRequester = (*Requester)(nil)

func NewRequester(p RequesterParams) dispatchconsoleports.CoverageRefreshRequester {
	return &Requester{signals: p.Signals, l: p.Logger.Named("dispatch-coverage-requester")}
}

func (r *Requester) RequestRefresh(ctx context.Context, tenantInfo pagination.TenantInfo) {
	if tenantInfo.OrgID.IsNil() || tenantInfo.BuID.IsNil() {
		return
	}

	id := RefreshWorkflowID(tenantInfo)
	_, err := r.signals.SignalWithStartWorkflow(
		ctx,
		id,
		RefreshSignalName,
		RefreshSignal{},
		client.StartWorkflowOptions{
			ID:            id,
			TaskQueue:     temporaltype.TaskQueueSystem.String(),
			StaticSummary: "Plan the organization's uncovered moves again",
			Priority:      temporal.Priority{FairnessKey: tenantInfo.OrgID.String()},
		},
		RefreshWorkflowName,
		&RefreshPayload{
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
		},
	)
	if err != nil {
		r.l.Warn("could not request a coverage refresh",
			zap.String("orgId", tenantInfo.OrgID.String()),
			zap.Error(err))
	}
}
