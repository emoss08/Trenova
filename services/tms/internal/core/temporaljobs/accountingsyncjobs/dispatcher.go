package accountingsyncjobs

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
)

type DispatcherParams struct {
	fx.In

	Workflows services.WorkflowStarter       `optional:"true"`
	Signals   services.WorkflowSignalStarter `optional:"true"`
}

type SyncDispatcher struct {
	workflows services.WorkflowStarter
	signals   services.WorkflowSignalStarter
}

var _ services.AccountingSyncDispatcher = (*SyncDispatcher)(nil)

func NewSyncDispatcher(p DispatcherParams) *SyncDispatcher {
	return &SyncDispatcher{workflows: p.Workflows, signals: p.Signals}
}

func (d *SyncDispatcher) Kick(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	connectionID pulid.ID,
) error {
	if d.signals == nil {
		return services.ErrWorkflowStarterDisabled
	}
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
	backfill *accountingsync.AccountingBackfill,
) error {
	if d.workflows == nil {
		return errortypes.NewBusinessError(
			"Background work is not available on this server, so a backfill cannot run",
		)
	}
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
	if errors.Is(err, services.ErrWorkflowStarterDisabled) {
		return errortypes.NewBusinessError(
			"Background work is not available on this server, so a backfill cannot run",
		).WithInternal(err)
	}
	return err
}
