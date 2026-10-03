package billingqueuejobs

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	ReviewRepo repositories.BillingQueueReviewRepository
	Review     services.BillingQueueReviewService
	Realtime   services.RealtimeService `optional:"true"`
	Logger     *zap.Logger
}

type Activities struct {
	repo     repositories.BillingQueueReviewRepository
	review   services.BillingQueueReviewService
	realtime services.RealtimeService
	l        *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		repo:     p.ReviewRepo,
		review:   p.Review,
		realtime: p.Realtime,
		l:        p.Logger.Named("billing-queue-approval-activities"),
	}
}

// StartApprovalRunActivity closes the undo window. It reports false when the
// run was undone first, in which case nothing is approved.
func (a *Activities) StartApprovalRunActivity(
	ctx context.Context,
	payload *ApprovalRunPayload,
) (bool, error) {
	info := activity.GetInfo(ctx)
	started, err := a.repo.MarkApprovalRunRunning(ctx, &repositories.MarkApprovalRunRunningRequest{
		TenantInfo:         payload.tenant(),
		RunID:              payload.RunID,
		TemporalWorkflowID: info.WorkflowExecution.ID,
		TemporalRunID:      info.WorkflowExecution.RunID,
	})
	if err != nil {
		return false, err
	}
	a.publishRunChanged(ctx, payload.tenant(), payload.RunID, payload.UserID)

	return started, nil
}

func (a *Activities) ListApprovalRunItemsActivity(
	ctx context.Context,
	payload *ApprovalRunPayload,
) ([]pulid.ID, error) {
	items, err := a.repo.ListPendingApprovalRunItems(ctx, payload.tenant(), payload.RunID)
	if err != nil {
		return nil, err
	}
	ids := make([]pulid.ID, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ItemID)
	}

	return ids, nil
}

// ApproveQueueItemActivity approves one item if, read fresh, it is still
// ready, and records what happened. A refusal is an outcome, not an error.
// Only a fault comes back as an error, for Temporal to retry.
//
// Recording is idempotent, and a retry after an approval that went through
// but failed to record finds the item already approved: on a retry that is
// taken as this run's own approval rather than somebody else's.
func (a *Activities) ApproveQueueItemActivity(
	ctx context.Context,
	payload *ApproveItemPayload,
) (*ApproveItemResult, error) {
	tenant := payload.tenant()
	actor := &services.RequestActor{
		PrincipalType:  services.PrincipalTypeUser,
		PrincipalID:    payload.UserID,
		UserID:         payload.UserID,
		OrganizationID: payload.OrganizationID,
		BusinessUnitID: payload.BusinessUnitID,
	}

	outcome, err := a.review.ApproveIfReady(ctx, &services.BillingQueueItemRequest{
		ItemID:     payload.ItemID,
		TenantInfo: tenant,
	}, actor)
	record := &billingqueue.ApprovalRunItem{RunID: payload.RunID, ItemID: payload.ItemID}
	switch {
	case err != nil && isRefusal(err):
		record.Status = billingqueue.ApprovalItemFailed
		record.FailureCode = billingqueue.ApprovalFailureNotReady
		record.ErrorMessage = err.Error()
	case err != nil:
		return nil, err
	case outcome.Approved:
		record.Status = billingqueue.ApprovalItemApproved
		record.InvoiceID = outcome.InvoiceID
		record.InvoiceNumber = outcome.InvoiceNumber
	case outcome.FailureCode == billingqueue.ApprovalFailureAlreadyDone &&
		activity.GetInfo(ctx).Attempt > 1:
		record.Status = billingqueue.ApprovalItemApproved
	case outcome.FailureCode == billingqueue.ApprovalFailureAlreadyDone ||
		outcome.FailureCode == billingqueue.ApprovalFailureOnHold:
		record.Status = billingqueue.ApprovalItemSkipped
		record.FailureCode = outcome.FailureCode
		record.ErrorMessage = outcome.Reason
	default:
		record.Status = billingqueue.ApprovalItemFailed
		record.FailureCode = outcome.FailureCode
		record.ErrorMessage = outcome.Reason
	}

	if recErr := a.repo.RecordApprovalRunItem(ctx, &repositories.RecordApprovalRunItemRequest{
		TenantInfo: tenant,
		Item:       record,
	}); recErr != nil {
		return nil, recErr
	}
	a.publishRunChanged(ctx, tenant, payload.RunID, payload.UserID)

	return &ApproveItemResult{Status: record.Status, FailureCode: record.FailureCode}, nil
}

// isRefusal is an error that says no rather than one that went wrong: a
// validation the item failed, a rule it broke. Retrying would only say no
// again.
func isRefusal(err error) bool {
	return errortypes.IsError(err) || errortypes.IsBusinessError(err) ||
		errortypes.IsNotFoundError(err) || errortypes.IsConflictError(err) ||
		errortypes.IsVersionMismatchError(err)
}

func (a *Activities) RecordApprovalItemFailureActivity(
	ctx context.Context,
	payload *RecordItemFailurePayload,
) error {
	tenant := pagination.TenantInfo{
		OrgID:  payload.OrganizationID,
		BuID:   payload.BusinessUnitID,
		UserID: payload.UserID,
	}
	if err := a.repo.RecordApprovalRunItem(ctx, &repositories.RecordApprovalRunItemRequest{
		TenantInfo: tenant,
		Item: &billingqueue.ApprovalRunItem{
			RunID:        payload.RunID,
			ItemID:       payload.ItemID,
			Status:       billingqueue.ApprovalItemFailed,
			FailureCode:  billingqueue.ApprovalFailureUnexpected,
			ErrorMessage: payload.Message,
		},
	}); err != nil {
		return err
	}
	a.publishRunChanged(ctx, tenant, payload.RunID, payload.UserID)

	return nil
}

func (a *Activities) FinalizeApprovalRunActivity(
	ctx context.Context,
	payload *FinalizeApprovalRunPayload,
) (*ApprovalRunResult, error) {
	tenant := pagination.TenantInfo{
		OrgID:  payload.OrganizationID,
		BuID:   payload.BusinessUnitID,
		UserID: payload.UserID,
	}
	run, err := a.repo.FinalizeApprovalRun(ctx, &repositories.FinalizeApprovalRunRequest{
		TenantInfo:     tenant,
		RunID:          payload.RunID,
		Status:         payload.Status,
		FailureMessage: payload.FailureMessage,
		LeftoverCode:   payload.LeftoverCode,
	})
	if err != nil {
		return nil, temporal.NewApplicationError("the approval run could not be closed", "FINALIZE", err)
	}
	a.publishRunChanged(ctx, tenant, payload.RunID, payload.UserID)

	return &ApprovalRunResult{
		RunID:         run.ID,
		Status:        run.Status,
		ApprovedCount: run.ApprovedCount,
		FailedCount:   run.FailedCount,
		SkippedCount:  run.SkippedCount,
	}, nil
}

func (a *Activities) publishRunChanged(
	ctx context.Context,
	tenant pagination.TenantInfo,
	runID pulid.ID,
	userID pulid.ID,
) {
	if a.realtime == nil {
		return
	}
	if err := a.realtime.PublishResourceInvalidation(ctx, &services.PublishResourceInvalidationRequest{
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		Resource:       invalidationResource,
		Action:         "updated",
		RecordID:       runID,
		ActorUserID:    userID,
	}); err != nil {
		a.l.Warn("failed to publish approval run progress",
			zap.String("runId", runID.String()), zap.Error(err))
	}
}
