package accountingsynccontract

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingbackfill"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
)

type DispatcherParams struct {
	fx.In

	Workflows kernelports.WorkflowStarter
	Signals   kernelports.WorkflowSignalStarter
}

type SyncDispatcher struct {
	workflows kernelports.WorkflowStarter
	signals   kernelports.WorkflowSignalStarter
}

var _ accountingsyncports.AccountingSyncDispatcher = (*SyncDispatcher)(nil)

func NewSyncDispatcher(p DispatcherParams) *SyncDispatcher {
	return &SyncDispatcher{workflows: p.Workflows, signals: p.Signals}
}

func (d *SyncDispatcher) Kick(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	connectionID pulid.ID,
) error {
	_, err := d.signals.SignalWithStartWorkflow(
		ctx,
		DrainWorkflowID(connectionID),
		DrainSignalName,
		DrainSignal{},
		client.StartWorkflowOptions{
			ID:        DrainWorkflowID(connectionID),
			TaskQueue: temporaltype.IntegrationTaskQueue,
			StaticSummary: "Send queued documents to the accounting system for connection " +
				connectionID.String(),
		},
		DrainAccountingOutboxWorkflowName,
		&DrainPayload{
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			ConnectionID:   connectionID,
		},
	)
	return err
}

func (d *SyncDispatcher) StartBackfill(
	ctx context.Context,
	backfill *accountingbackfill.AccountingBackfill,
) error {
	_, err := d.workflows.StartWorkflow(ctx, client.StartWorkflowOptions{
		ID:                       BackfillWorkflowID(backfill.ID),
		TaskQueue:                temporaltype.IntegrationTaskQueue,
		WorkflowIDReusePolicy:    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
		WorkflowIDConflictPolicy: enums.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
	}, BackfillAccountingWorkflowName, &BackfillPayload{
		OrganizationID: backfill.OrganizationID,
		BusinessUnitID: backfill.BusinessUnitID,
		BackfillID:     backfill.ID,
	})
	return err
}
