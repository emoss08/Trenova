package accountingsynccontract

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountinginboundports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
)

type RefresherParams struct {
	fx.In

	Workflows kernelports.WorkflowStarter
}

type ReferenceRefresher struct {
	workflows kernelports.WorkflowStarter
}

var _ accountingconnectionports.AccountingReferenceRefresher = (*ReferenceRefresher)(nil)

func NewReferenceRefresher(p RefresherParams) *ReferenceRefresher {
	return &ReferenceRefresher{workflows: p.Workflows}
}

func (r *ReferenceRefresher) RequestReferenceRefresh(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	connectionID pulid.ID,
) error {
	_, err := r.workflows.StartWorkflow(ctx, client.StartWorkflowOptions{
		ID:                       ReferenceWorkflowID(connectionID),
		TaskQueue:                temporaltype.IntegrationTaskQueue,
		WorkflowIDReusePolicy:    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
		WorkflowIDConflictPolicy: enums.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
	}, RefreshAccountingReferenceWorkflowName, &RefreshReferencePayload{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		ConnectionID:   connectionID,
	})
	return err
}

type ChangesPollerParams struct {
	fx.In

	Signals kernelports.WorkflowSignalStarter
}

type ChangesPoller struct {
	signals kernelports.WorkflowSignalStarter
}

var _ accountinginboundports.AccountingChangePoller = (*ChangesPoller)(nil)

func NewChangesPoller(p ChangesPollerParams) *ChangesPoller {
	return &ChangesPoller{signals: p.Signals}
}

func (p *ChangesPoller) PollNow(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	connectionID pulid.ID,
) error {
	_, err := p.signals.SignalWithStartWorkflow(
		ctx,
		ChangesWorkflowID(connectionID),
		ChangesSignalName,
		ChangesSignal{},
		client.StartWorkflowOptions{
			ID:        ChangesWorkflowID(connectionID),
			TaskQueue: temporaltype.IntegrationTaskQueue,
			StaticSummary: "Read changes from the accounting system for connection " +
				connectionID.String(),
		},
		PollAccountingChangesWorkflowName,
		&ChangesPayload{
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			ConnectionID:   connectionID,
		},
	)
	return err
}
