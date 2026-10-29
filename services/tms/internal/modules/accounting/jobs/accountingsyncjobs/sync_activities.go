package accountingsyncjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/jobs/accountingsyncjobs/accountingsynccontract"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/activity"
	"go.uber.org/zap"
)

type (
	DrainBatchResult   = accountingsyncports.AccountingSyncDrainResult
	BackfillStepResult = accountingsyncports.AccountingBackfillStepResult
)

func syncError(err error) error {
	if err == nil {
		return nil
	}
	if errortypes.IsNotFoundError(err) || errortypes.IsBusinessError(err) ||
		errortypes.IsError(err) {
		return temporaltype.NewNonRetryableError(err.Error(), err).ToTemporalError()
	}
	return err
}

func (a *Activities) DrainAccountingOutboxActivity(
	ctx context.Context,
	payload *accountingsynccontract.DrainPayload,
) (*DrainBatchResult, error) {
	result, err := a.drainer.Drain(ctx, &accountingsyncports.DrainAccountingSyncRequest{
		TenantInfo:   payload.TenantInfo(),
		ConnectionID: payload.ConnectionID,
		Limit:        drainBatchLimit,
		Lease:        drainLease,
		Heartbeat:    func() { activity.RecordHeartbeat(ctx) },
	})
	if err != nil {
		return result, syncError(err)
	}
	return result, nil
}

func (a *Activities) KickDueAccountingSyncActivity(ctx context.Context) (*KickDueResult, error) {
	due, err := a.sweeper.ListDueConnections(ctx, dueConnectionsPerKick)
	if err != nil {
		return nil, err
	}
	result := &KickDueResult{Due: len(due)}
	for _, conn := range due {
		tenant := pagination.TenantInfo{OrgID: conn.OrganizationID, BuID: conn.BusinessUnitID}
		if kickErr := a.dispatcher.Kick(
			dbscope.WithTenant(ctx, tenant.DBTenant()),
			tenant,
			conn.ConnectionID,
		); kickErr != nil {
			a.l.Warn("failed to wake an accounting sender",
				zap.String("connectionId", conn.ConnectionID.String()), zap.Error(kickErr))
			continue
		}
		result.Kicked++
	}
	return result, nil
}

func (a *Activities) AccountingSafetyNetActivity(
	ctx context.Context,
) (*SafetyNetSweepResult, error) {
	result := new(SafetyNetSweepResult)
	afterID := pulid.Nil
	for {
		conns, err := a.connRepo.ListActive(
			ctx,
			accountingconnectionports.ListActiveAccountingConnectionsRequest{
				AfterID: afterID,
				Limit:   safetyNetConnectionsPage,
			},
		)
		if err != nil {
			return result, err
		}
		for _, conn := range conns {
			afterID = conn.ID
			if !conn.IsSyncing() {
				continue
			}
			result.Connections++
			tenant := pagination.TenantInfo{OrgID: conn.OrganizationID, BuID: conn.BusinessUnitID}
			found, netErr := a.sweeper.SafetyNet(
				dbscope.WithTenant(ctx, tenant.DBTenant()),
				accountingsyncports.AccountingSyncConnectionRef{
					TenantInfo:   tenant,
					ConnectionID: conn.ID,
				},
			)
			activity.RecordHeartbeat(ctx, result.Connections)
			if netErr != nil {
				result.Failed++
				a.l.Warn("accounting safety net failed for a connection",
					zap.String("connectionId", conn.ID.String()), zap.Error(netErr))
				continue
			}
			result.Found += found.Found
			result.Queued += found.Queued
		}
		if len(conns) < safetyNetConnectionsPage {
			break
		}
	}
	if result.Queued > 0 {
		a.l.Warn("accounting safety net queued documents a posting path missed",
			zap.Int("queued", result.Queued))
	}
	return result, nil
}

func (a *Activities) PurgeAccountingSyncHistoryActivity(ctx context.Context) (*PurgeResult, error) {
	purged, err := a.sweeper.PurgeHistory(ctx)
	if err != nil {
		return nil, err
	}
	return &PurgeResult{
		PayloadsCleared: purged.PayloadsCleared,
		AttemptsDeleted: purged.AttemptsDeleted,
	}, nil
}

func (a *Activities) AccountingBackfillStepActivity(
	ctx context.Context,
	payload *accountingsynccontract.BackfillPayload,
) (*BackfillStepResult, error) {
	result, err := a.backfills.BackfillStep(ctx, payload.TenantInfo(), payload.BackfillID)
	if err != nil {
		return nil, syncError(err)
	}
	return result, nil
}
