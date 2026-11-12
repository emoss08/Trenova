package billingqueuejobs

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingapprovalrun"
	"github.com/emoss08/trenova/internal/modules/billing/jobs/billingqueuejobs/billingqueuecontract"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
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

	ReviewRepo billingqueueports.BillingQueueReviewRepository
	QueueRepo  billingqueueports.BillingQueueRepository
	Review     billingqueueports.BillingQueueReviewService
	Rechecks   billingqueueports.RecheckRequester
	Assist     billingqueueports.BillingQueueAssistService
	Generator  billingqueueports.NovaGenerator
	Realtime   realtimeports.RealtimeService
	Logger     *zap.Logger
}

type Activities struct {
	repo      billingqueueports.BillingQueueReviewRepository
	queueRepo billingqueueports.BillingQueueRepository
	review    billingqueueports.BillingQueueReviewService
	rechecks  billingqueueports.RecheckRequester
	assist    billingqueueports.BillingQueueAssistService
	generator billingqueueports.NovaGenerator
	realtime  realtimeports.RealtimeService
	l         *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		repo:      p.ReviewRepo,
		queueRepo: p.QueueRepo,
		review:    p.Review,
		rechecks:  p.Rechecks,
		assist:    p.Assist,
		generator: p.Generator,
		realtime:  p.Realtime,
		l:         p.Logger.Named("billing-queue-activities"),
	}
}

// StartApprovalRunActivity closes the undo window. It reports false when the
// run was undone first, in which case nothing is approved.
func (a *Activities) StartApprovalRunActivity(
	ctx context.Context,
	payload *billingqueuecontract.ApprovalRunPayload,
) (bool, error) {
	info := activity.GetInfo(ctx)
	started, err := a.repo.MarkApprovalRunRunning(
		ctx,
		&billingqueueports.MarkApprovalRunRunningRequest{
			TenantInfo:         payload.Tenant(),
			RunID:              payload.RunID,
			TemporalWorkflowID: info.WorkflowExecution.ID,
			TemporalRunID:      info.WorkflowExecution.RunID,
		},
	)
	if err != nil {
		return false, err
	}
	a.publishRunChanged(ctx, payload.Tenant(), payload.RunID, payload.UserID)

	return started, nil
}

func (a *Activities) ListApprovalRunItemsActivity(
	ctx context.Context,
	payload *billingqueuecontract.ApprovalRunPayload,
) ([]pulid.ID, error) {
	items, err := a.repo.ListPendingApprovalRunItems(ctx, payload.Tenant(), payload.RunID)
	if err != nil {
		return nil, err
	}
	ids := make([]pulid.ID, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ItemID)
	}

	return ids, nil
}

// ApplyRunItemActivity applies the run's action to one item and records what
// happened. A refusal is an outcome, not an error; only a fault comes back as
// an error, for Temporal to retry.
func (a *Activities) ApplyRunItemActivity(
	ctx context.Context,
	payload *ApplyRunItemPayload,
) (*ApplyRunItemResult, error) {
	tenant := payload.tenant()
	actor := &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    payload.UserID,
		UserID:         payload.UserID,
		OrganizationID: payload.OrganizationID,
		BusinessUnitID: payload.BusinessUnitID,
	}

	record, err := a.applyRunItem(ctx, payload, tenant, actor)
	if err != nil {
		return nil, err
	}
	record.RunID = payload.RunID
	record.ItemID = payload.ItemID

	if recErr := a.repo.RecordApprovalRunItem(ctx, &billingqueueports.RecordApprovalRunItemRequest{
		TenantInfo: tenant,
		Item:       record,
	}); recErr != nil {
		return nil, recErr
	}
	a.publishRunChanged(ctx, tenant, payload.RunID, payload.UserID)

	return &ApplyRunItemResult{Status: record.Status, FailureCode: record.FailureCode}, nil
}

func (a *Activities) applyRunItem(
	ctx context.Context,
	payload *ApplyRunItemPayload,
	tenant pagination.TenantInfo,
	actor *permissionports.RequestActor,
) (*billingapprovalrun.ApprovalRunItem, error) {
	if payload.Action == billingapprovalrun.RunActionApprove {
		return a.approveRunItem(ctx, payload, tenant, actor)
	}

	err := a.review.ApplyBulkAction(ctx, &billingqueueports.ApplyBulkActionRequest{
		ItemID:     payload.ItemID,
		TenantInfo: tenant,
		Action:     payload.Action,
		Params:     payload.Params,
	}, actor)
	switch {
	case err == nil:
		return &billingapprovalrun.ApprovalRunItem{
			Status: billingapprovalrun.ApprovalItemSucceeded,
		}, nil
	case errors.Is(err, billingqueueports.ErrBulkActionNotApplicable):
		return &billingapprovalrun.ApprovalRunItem{
			Status:       billingapprovalrun.ApprovalItemSkipped,
			FailureCode:  billingapprovalrun.ApprovalFailureNotApplies,
			ErrorMessage: err.Error(),
		}, nil
	case isRefusal(err):
		return &billingapprovalrun.ApprovalRunItem{
			Status:       billingapprovalrun.ApprovalItemFailed,
			FailureCode:  billingapprovalrun.ApprovalFailureNotReady,
			ErrorMessage: err.Error(),
		}, nil
	default:
		return nil, err
	}
}

// approveRunItem approves one item if, read fresh, it is still ready.
// Recording is idempotent, and a retry after an approval that went through
// but failed to record finds the item already approved: on a retry that is
// taken as this run's own approval rather than somebody else's.
func (a *Activities) approveRunItem(
	ctx context.Context,
	payload *ApplyRunItemPayload,
	tenant pagination.TenantInfo,
	actor *permissionports.RequestActor,
) (*billingapprovalrun.ApprovalRunItem, error) {
	outcome, err := a.review.ApproveIfReady(ctx, &billingqueueports.ApproveIfReadyRequest{
		ItemID:         payload.ItemID,
		TenantInfo:     tenant,
		AssignApprover: payload.Params.AssignApprover,
	}, actor)
	record := &billingapprovalrun.ApprovalRunItem{}
	switch {
	case err != nil && isRefusal(err):
		record.Status = billingapprovalrun.ApprovalItemFailed
		record.FailureCode = billingapprovalrun.ApprovalFailureNotReady
		record.ErrorMessage = err.Error()
	case err != nil:
		return nil, err
	case outcome.Approved:
		record.Status = billingapprovalrun.ApprovalItemSucceeded
		record.InvoiceID = outcome.InvoiceID
		record.InvoiceNumber = outcome.InvoiceNumber
	case outcome.FailureCode == billingapprovalrun.ApprovalFailureAlreadyDone &&
		activity.GetInfo(ctx).Attempt > 1:
		record.Status = billingapprovalrun.ApprovalItemSucceeded
	case outcome.FailureCode == billingapprovalrun.ApprovalFailureAlreadyDone ||
		outcome.FailureCode == billingapprovalrun.ApprovalFailureOnHold:
		record.Status = billingapprovalrun.ApprovalItemSkipped
		record.FailureCode = outcome.FailureCode
		record.ErrorMessage = outcome.Reason
	default:
		record.Status = billingapprovalrun.ApprovalItemFailed
		record.FailureCode = outcome.FailureCode
		record.ErrorMessage = outcome.Reason
	}

	return record, nil
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
	if err := a.repo.RecordApprovalRunItem(ctx, &billingqueueports.RecordApprovalRunItemRequest{
		TenantInfo: tenant,
		Item: &billingapprovalrun.ApprovalRunItem{
			RunID:        payload.RunID,
			ItemID:       payload.ItemID,
			Status:       billingapprovalrun.ApprovalItemFailed,
			FailureCode:  billingapprovalrun.ApprovalFailureUnexpected,
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
	run, err := a.repo.FinalizeApprovalRun(ctx, &billingqueueports.FinalizeApprovalRunRequest{
		TenantInfo:     tenant,
		RunID:          payload.RunID,
		Status:         payload.Status,
		FailureMessage: payload.FailureMessage,
		LeftoverCode:   payload.LeftoverCode,
	})
	if err != nil {
		return nil, temporal.NewApplicationError(
			"the approval run could not be closed",
			"FINALIZE",
			err,
		)
	}
	a.publishRunChanged(ctx, tenant, payload.RunID, payload.UserID)

	return &ApprovalRunResult{
		RunID:          run.ID,
		Status:         run.Status,
		SucceededCount: run.SucceededCount,
		FailedCount:    run.FailedCount,
		SkippedCount:   run.SkippedCount,
	}, nil
}

func (a *Activities) publishRunChanged(
	ctx context.Context,
	tenant pagination.TenantInfo,
	runID pulid.ID,
	userID pulid.ID,
) {
	if err := a.realtime.PublishResourceInvalidation(
		ctx,
		&realtimeports.PublishResourceInvalidationRequest{
			OrganizationID: tenant.OrgID,
			BusinessUnitID: tenant.BuID,
			Resource:       invalidationResource,
			Action:         "updated",
			RecordID:       runID,
			ActorUserID:    userID,
		},
	); err != nil {
		a.l.Warn("failed to publish approval run progress",
			zap.String("runId", runID.String()), zap.Error(err))
	}
}
