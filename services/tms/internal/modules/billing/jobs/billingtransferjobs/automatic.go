package billingtransferjobs

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/billing/app/billingtransferservice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingtransfer"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/activity"
	"go.uber.org/zap"
)

// AutomaticTransferSweepActivity gives every organization that transfers
// automatically its turn. One organization's failure is logged and the sweep
// moves on, so a misconfigured tenant cannot hold up everyone else's billing.
func (a *Activities) AutomaticTransferSweepActivity(
	ctx context.Context,
) (*AutomaticTransferSweepResult, error) {
	result := new(AutomaticTransferSweepResult)
	if a.controls == nil || a.users == nil || a.transfers == nil {
		return result, nil
	}

	controls, err := a.controls.ListAutomaticTransfer(ctx)
	if err != nil {
		return nil, err
	}
	if len(controls) == 0 {
		return result, nil
	}

	system, err := a.users.GetSystemUser(ctx, "id")
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	for _, control := range controls {
		activity.RecordHeartbeat(ctx, control.OrganizationID.String())
		result.Organizations++

		tenantCtx := dbscope.WithTenant(ctx, dbscope.Tenant{
			OrganizationID: control.OrganizationID,
			BusinessUnitID: control.BusinessUnitID,
		})
		run, startErr := a.transfers.StartAutomatic(
			tenantCtx,
			&billingtransferservice.AutomaticRunRequest{
				Control:      control,
				SystemUserID: system.ID,
				Now:          now,
			},
		)
		if startErr != nil {
			result.Failed++
			a.l.Error("failed to start an automatic billing transfer",
				zap.String("organizationId", control.OrganizationID.String()),
				zap.Error(startErr))
			continue
		}
		if run != nil {
			result.RunsStarted++
		}
	}

	return result, nil
}

const maxAutomaticRunNotices = 25

// notifyAutomaticFinished tells the people who work the billing queue about an
// automatic transfer, since nobody started it to be told. Shipments it could
// not move are always worth a notice; a clean transfer only when billing
// control asked to hear about automatically created invoices. A stopped run was
// stopped by somebody who already knows.
func (a *Activities) notifyAutomaticFinished(
	ctx context.Context,
	run *billingtransfer.BillingTransferRun,
) {
	eventType, priority, title, message, ok := a.automaticWording(ctx, run)
	if !ok {
		return
	}

	correlation := run.ID.String()
	data := runNoticeData(run)
	data["automatic"] = true
	if _, err := a.notification.NotifyPermitted(ctx, notificationservice.NotifyPermittedRequest{
		Tenant: pagination.TenantInfo{
			OrgID: run.OrganizationID,
			BuID:  run.BusinessUnitID,
		},
		Resource:    permission.ResourceBillingQueue,
		Operation:   permission.OpUpdate,
		Limit:       maxAutomaticRunNotices,
		Now:         timeutils.NowUnix(),
		DedupeSince: max(run.QueuedAt-1, 1),
		Notification: notification.Notification{
			EventType:       eventType,
			Priority:        priority,
			Title:           title,
			Message:         message,
			Source:          "billingtransferjobs.AutomaticTransfer",
			CorrelationID:   &correlation,
			Data:            data,
			RelatedEntities: map[string]any{"billingTransferRunId": correlation},
		},
	}); err != nil {
		a.l.Warn("could not tell anyone how an automatic billing transfer went",
			zap.String("runId", correlation), zap.Error(err))
	}
}

func (a *Activities) automaticWording(
	ctx context.Context,
	run *billingtransfer.BillingTransferRun,
) (eventType string, priority notification.Priority, title, message string, ok bool) {
	switch {
	case run.Status == billingtransfer.RunStatusCanceled:
		return "", "", "", "", false
	case run.Status == billingtransfer.RunStatusFailed:
		return eventRunFailed, notification.PriorityHigh,
			"Automatic transfer to billing failed",
			fmt.Sprintf(
				"%d of %d ready shipments moved into the billing queue before the automatic transfer failed. Open the report to retry the rest.",
				run.TransferredCount,
				run.TotalCount,
			), true
	case run.NotTransferredCount > 0:
		return eventRunCompleted, notification.PriorityHigh,
			"Automatic transfer to billing needs attention",
			fmt.Sprintf(
				"%d of %d ready shipments moved into the billing queue automatically. %d could not be moved.",
				run.TransferredCount,
				run.TotalCount,
				run.NotTransferredCount,
			), true
	case run.TransferredCount == 0 || !a.notifyOnAutoInvoices(ctx, run):
		return "", "", "", "", false
	default:
		return eventRunCompleted, notification.PriorityLow,
			"Shipments moved into billing automatically",
			fmt.Sprintf(
				"%d ready shipments moved into the billing queue. Invoice drafts were created for the ones cleared to bill without review.",
				run.TransferredCount,
			), true
	}
}

func (a *Activities) notifyOnAutoInvoices(
	ctx context.Context,
	run *billingtransfer.BillingTransferRun,
) bool {
	if a.controls == nil {
		return false
	}
	control, err := a.controls.GetByOrgID(ctx, run.OrganizationID)
	if err != nil {
		a.l.Warn("could not read billing control for an automatic transfer notice",
			zap.String("runId", run.ID.String()), zap.Error(err))
		return false
	}

	return control.NotifyOnAutoInvoiceCreation
}
