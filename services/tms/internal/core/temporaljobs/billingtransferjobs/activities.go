package billingtransferjobs

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/billingtransfer"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	eventRunCompleted = "billing_transfer_run_completed"
	eventRunCanceled  = "billing_transfer_run_canceled"
	eventRunFailed    = "billing_transfer_run_failed"

	// invalidationResource is the key the browser maps back to its billing
	// transfer queries; it must match RESOURCE_QUERY_KEY_MAP on the client.
	invalidationResource = "billing-transfer-run"
)

type ActivitiesParams struct {
	fx.In

	RunRepo             repositories.BillingTransferRunRepository
	ShipmentService     services.ShipmentService
	NotificationService *notificationservice.Service
	RealtimeService     services.RealtimeService `optional:"true"`
	Logger              *zap.Logger
}

type Activities struct {
	runRepo      repositories.BillingTransferRunRepository
	shipments    services.ShipmentService
	notification *notificationservice.Service
	realtime     services.RealtimeService
	l            *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		runRepo:      p.RunRepo,
		shipments:    p.ShipmentService,
		notification: p.NotificationService,
		realtime:     p.RealtimeService,
		l:            p.Logger.Named("billing-transfer-activities"),
	}
}

// PrepareRunActivity opens the run for work.
//
// For a run that named its shipments, the rows are already seeded and this only
// stamps the Temporal identity. For one that named a filter instead, this is
// where the candidates are resolved — deliberately here and not in the request
// that started it, so that resolving up to five thousand shipments never holds
// a browser connection open.
func (a *Activities) PrepareRunActivity(
	ctx context.Context,
	payload *TransferRunPayload,
) (*PreparedRun, error) {
	tenant := payload.tenant()

	run, err := a.runRepo.GetByID(ctx, &repositories.GetBillingTransferRunRequest{
		TenantInfo: tenant,
		RunID:      payload.RunID,
	})
	if err != nil {
		return nil, err
	}

	if run.Status.IsTerminal() {
		return nil, temporal.NewNonRetryableApplicationError(
			"the transfer has already finished",
			ErrTypeTransferValidation,
			nil,
		)
	}

	totalCount := run.TotalCount
	unmatchedCount := run.UnmatchedCount

	if run.Scope == billingtransfer.RunScopeAllMatching {
		candidates, listErr := a.shipments.ListBillingTransferCandidateIDs(
			ctx,
			&services.ListBillingTransferCandidateIDsRequest{
				Filter: &pagination.QueryOptions{
					TenantInfo: tenant,
					Query:      run.SearchQuery,
				},
				Status: run.ShipmentStatus,
			},
		)
		if listErr != nil {
			return nil, temporal.NewApplicationError(
				"the shipments matching this search could not be listed",
				ErrTypeTransferCandidates,
				listErr,
			)
		}

		if _, seedErr := a.runRepo.SeedItems(ctx, &repositories.SeedBillingTransferRunItemsRequest{
			TenantInfo:  tenant,
			RunID:       run.ID,
			ShipmentIDs: candidates.IDs,
		}); seedErr != nil {
			return nil, seedErr
		}

		totalCount = len(candidates.IDs)
		// What the filter matched beyond the cap. Reported rather than silently
		// dropped, because the biller has to know there is a second run to do.
		unmatchedCount = max(candidates.TotalCount-len(candidates.IDs), 0)
	}

	info := activity.GetInfo(ctx)
	updated, err := a.runRepo.MarkRunning(ctx, &repositories.MarkBillingTransferRunRunningRequest{
		TenantInfo:         tenant,
		RunID:              run.ID,
		TemporalWorkflowID: info.WorkflowExecution.ID,
		TemporalRunID:      info.WorkflowExecution.RunID,
		TotalCount:         totalCount,
		UnmatchedCount:     unmatchedCount,
	})
	if err != nil {
		return nil, err
	}

	a.publishRunChanged(ctx, updated)

	return &PreparedRun{
		RunID:          updated.ID,
		RequestedByID:  updated.RequestedByID,
		TotalCount:     updated.TotalCount,
		UnmatchedCount: updated.UnmatchedCount,
		BatchSize:      min(BatchSize, services.MaxBulkTransferToBillingShipments),
	}, nil
}

// ProcessBatchActivity transfers the next batch of shipments still waiting.
//
// Outcomes are written back every few shipments rather than once at the end.
// That is what lets the progress bar move smoothly, and it bounds what a crash
// costs: at worst the last few shipments of a batch are re-attempted, and they
// come back as AlreadyTransferred rather than being transferred twice.
func (a *Activities) ProcessBatchActivity(
	ctx context.Context,
	payload *ProcessBatchPayload,
) (*ProcessBatchResult, error) {
	tenant := payload.tenant()

	run, err := a.runRepo.GetByID(ctx, &repositories.GetBillingTransferRunRequest{
		TenantInfo: tenant,
		RunID:      payload.RunID,
	})
	if err != nil {
		return nil, err
	}

	if run.Status.IsTerminal() || run.IsCancelRequested() {
		return &ProcessBatchResult{CancelRequested: true}, nil
	}

	shipmentIDs, err := a.runRepo.NextPendingShipmentIDs(
		ctx,
		&repositories.NextPendingBillingTransferItemsRequest{
			TenantInfo: tenant,
			RunID:      run.ID,
			Limit:      payload.Limit,
		},
	)
	if err != nil {
		return nil, err
	}
	if len(shipmentIDs) == 0 {
		return &ProcessBatchResult{}, nil
	}

	actor := &services.RequestActor{
		PrincipalType:  services.PrincipalTypeUser,
		PrincipalID:    run.RequestedByID,
		UserID:         run.RequestedByID,
		OrganizationID: run.OrganizationID,
		BusinessUnitID: run.BusinessUnitID,
	}

	result := new(ProcessBatchResult)
	pending := make([]repositories.BillingTransferItemOutcome, 0, flushEvery)
	done := 0

	flush := func() error {
		if len(pending) == 0 {
			return nil
		}

		progress, recErr := a.runRepo.RecordItemOutcomes(
			ctx,
			&repositories.RecordBillingTransferOutcomesRequest{
				TenantInfo: tenant,
				RunID:      run.ID,
				Outcomes:   pending,
			},
		)
		if recErr != nil {
			return recErr
		}

		pending = pending[:0]
		if progress.CancelRequested {
			result.CancelRequested = true
		}
		a.publishRunChanged(ctx, run)

		return nil
	}

	var flushErr error
	_, err = a.shipments.BulkTransferToBilling(ctx, &services.BulkTransferShipmentToBillingRequest{
		ShipmentIDs:                    shipmentIDs,
		BillType:                       run.BillType,
		MarkCompletedReadyToInvoice:    run.MarkCompletedReadyToInvoice,
		SuppressExceptionNotifications: true,
		OnResult: func(outcome services.BulkTransferToBillingResult) {
			if flushErr != nil {
				return
			}

			pending = append(pending, itemOutcomeFrom(outcome))
			done++
			result.Processed++
			if outcome.Success {
				result.Transferred++
			} else {
				result.NotTransferred++
			}

			activity.RecordHeartbeat(ctx, done)

			if len(pending) >= flushEvery {
				flushErr = flush()
			}
		},
	}, actor)
	if flushErr != nil {
		return nil, flushErr
	}
	if err != nil {
		// Whatever was answered for before this failed is still real work; it is
		// recorded before the error is reported so a retry does not redo it.
		if recErr := flush(); recErr != nil {
			a.l.Error("failed to record outcomes after a failed batch", zap.Error(recErr))
		}
		return nil, err
	}

	if err = flush(); err != nil {
		return nil, err
	}

	return result, nil
}

// FinalizeRunActivity closes the run out and tells the person who started it.
// It is idempotent: the workflow allows it ten attempts, and a run that is
// already terminal simply reports the numbers it settled on.
func (a *Activities) FinalizeRunActivity(
	ctx context.Context,
	payload *FinalizeRunPayload,
) (*TransferRunResult, error) {
	tenant := payload.tenant()

	existing, err := a.runRepo.GetByID(ctx, &repositories.GetBillingTransferRunRequest{
		TenantInfo: tenant,
		RunID:      payload.RunID,
	})
	if err != nil {
		return nil, err
	}
	if existing.Status.IsTerminal() {
		return runResultFrom(existing), nil
	}

	run, err := a.runRepo.Finalize(ctx, &repositories.FinalizeBillingTransferRunRequest{
		TenantInfo:     tenant,
		RunID:          payload.RunID,
		Status:         payload.Status,
		FailureMessage: payload.FailureMessage,
	})
	if err != nil {
		return nil, err
	}

	a.notifyFinished(ctx, run)
	a.publishRunChanged(ctx, run)

	return runResultFrom(run), nil
}

// ReconcileZombieRunsActivity is the second net under "a run is never left
// stuck". The workflow finalizes itself on every path it can reach, but a
// worker killed between batches reaches none of them.
func (a *Activities) ReconcileZombieRunsActivity(
	ctx context.Context,
) (*ReconcileZombieRunsResult, error) {
	cutoff := timeutils.NowUnix() - zombieRunCutoffSeconds

	runs, err := a.runRepo.ListStale(ctx, &repositories.ListStaleBillingTransferRunsRequest{
		UpdatedBeforeUnix: cutoff,
		Limit:             zombieRunSweepLimit,
	})
	if err != nil {
		return nil, err
	}

	out := &ReconcileZombieRunsResult{Examined: len(runs)}
	for _, run := range runs {
		activity.RecordHeartbeat(ctx, run.ID.String())

		finalized, finErr := a.runRepo.Finalize(
			ctx,
			&repositories.FinalizeBillingTransferRunRequest{
				TenantInfo: pagination.TenantInfo{
					OrgID: run.OrganizationID,
					BuID:  run.BusinessUnitID,
				},
				RunID:  run.ID,
				Status: billingtransfer.RunStatusFailed,
				FailureMessage: "The transfer stopped responding and was closed out — " +
					"retry the shipments it did not reach",
			},
		)
		if finErr != nil {
			a.l.Error("failed to close out a stalled billing transfer run",
				zap.String("runId", run.ID.String()), zap.Error(finErr))
			continue
		}

		out.Failed++
		a.notifyFinished(ctx, finalized)
		a.publishRunChanged(ctx, finalized)
	}

	return out, nil
}

const (
	// zombieRunCutoffSeconds is how long a run may go without any counter
	// moving before it is treated as dead. A single batch is capped at ten
	// minutes, so anything past thirty has stopped for good.
	zombieRunCutoffSeconds = 30 * 60
	zombieRunSweepLimit    = 100
)

func itemOutcomeFrom(
	result services.BulkTransferToBillingResult,
) repositories.BillingTransferItemOutcome {
	outcome := repositories.BillingTransferItemOutcome{
		ShipmentID:           result.ShipmentID,
		ProNumber:            result.ProNumber,
		MarkedReadyToInvoice: result.MarkedReadyToInvoice,
		ErrorMessage:         result.Error,
		ProcessedAt:          timeutils.NowUnix(),
		MissingRequirements:  make([]billingtransfer.MissingRequirement, 0, len(result.MissingRequirements)),
		ValidationFailures:   make([]billingtransfer.ValidationFailure, 0, len(result.ValidationFailures)),
	}

	if result.Success {
		outcome.Status = billingtransfer.ItemStatusTransferred
	} else {
		outcome.Status = billingtransfer.ItemStatusNotTransferred
		outcome.FailureCode = result.FailureCode
		if outcome.FailureCode == "" {
			// The row's own constraint requires a reason for anything that did
			// not transfer, and "we do not know" is still a reason.
			outcome.FailureCode = billingtransfer.FailureUnexpected
		}
	}

	if result.Item != nil {
		outcome.BillingQueueItemID = result.Item.ID
		outcome.BillingQueueNumber = result.Item.Number
		outcome.BillingQueueStatus = result.Item.Status
	}

	for _, requirement := range result.MissingRequirements {
		outcome.MissingRequirements = append(
			outcome.MissingRequirements,
			billingtransfer.MissingRequirement{
				DocumentTypeID:   requirement.DocumentTypeID,
				DocumentTypeCode: requirement.DocumentTypeCode,
				DocumentTypeName: requirement.DocumentTypeName,
			},
		)
	}
	for _, failure := range result.ValidationFailures {
		outcome.ValidationFailures = append(
			outcome.ValidationFailures,
			billingtransfer.ValidationFailure{
				Field:   failure.Field,
				Code:    failure.Code,
				Message: failure.Message,
			},
		)
	}

	return outcome
}

func runResultFrom(run *billingtransfer.BillingTransferRun) *TransferRunResult {
	return &TransferRunResult{
		RunID:                     run.ID,
		Status:                    run.Status,
		TotalCount:                run.TotalCount,
		ProcessedCount:            run.ProcessedCount,
		TransferredCount:          run.TransferredCount,
		NotTransferredCount:       run.NotTransferredCount,
		SkippedCount:              run.SkippedCount,
		MarkedReadyToInvoiceCount: run.MarkedReadyToInvoiceCount,
		RetryableCount:            run.RetryableCount,
		UnmatchedCount:            run.UnmatchedCount,
	}
}

// notifyFinished is best effort. The run is already recorded; failing to
// announce it must never fail the activity and send the workflow round again.
func (a *Activities) notifyFinished(
	ctx context.Context,
	run *billingtransfer.BillingTransferRun,
) {
	if a.notification == nil {
		return
	}

	eventType, priority, title, message := finishedWording(run)
	targetUserID := run.RequestedByID
	buID := run.BusinessUnitID

	if _, err := a.notification.Create(ctx, &notification.Notification{
		OrganizationID: run.OrganizationID,
		BusinessUnitID: &buID,
		TargetUserID:   &targetUserID,
		Channel:        notification.ChannelUser,
		EventType:      eventType,
		Priority:       priority,
		Title:          title,
		Message:        message,
		Data: map[string]any{
			"runId":               run.ID.String(),
			"status":              string(run.Status),
			"totalCount":          run.TotalCount,
			"transferredCount":    run.TransferredCount,
			"notTransferredCount": run.NotTransferredCount,
			"skippedCount":        run.SkippedCount,
			"retryableCount":      run.RetryableCount,
			"unmatchedCount":      run.UnmatchedCount,
		},
		Source: "billingtransferjobs.FinalizeRun",
	}); err != nil {
		a.l.Error("failed to announce a finished billing transfer",
			zap.String("runId", run.ID.String()), zap.Error(err))
	}
}

func finishedWording(
	run *billingtransfer.BillingTransferRun,
) (eventType string, priority notification.Priority, title, message string) {
	switch run.Status {
	case billingtransfer.RunStatusCanceled:
		return eventRunCanceled, notification.PriorityMedium,
			"Transfer to billing stopped",
			fmt.Sprintf(
				"%d of %d shipments were checked before the transfer was stopped. %d moved into the billing queue.",
				run.ProcessedCount, run.TotalCount, run.TransferredCount,
			)
	case billingtransfer.RunStatusFailed:
		return eventRunFailed, notification.PriorityHigh,
			"Transfer to billing failed",
			fmt.Sprintf(
				"%d of %d shipments moved into the billing queue before the transfer failed.",
				run.TransferredCount, run.TotalCount,
			)
	case billingtransfer.RunStatusCompleted,
		billingtransfer.RunStatusQueued,
		billingtransfer.RunStatusRunning:
	}

	if run.NotTransferredCount == 0 {
		return eventRunCompleted, notification.PriorityMedium,
			"Transfer to billing finished",
			fmt.Sprintf(
				"All %d shipments moved into the billing queue.",
				run.TransferredCount,
			)
	}

	return eventRunCompleted, notification.PriorityMedium,
		"Transfer to billing finished",
		fmt.Sprintf(
			"%d of %d shipments moved into the billing queue. %d need attention.",
			run.TransferredCount, run.TotalCount, run.NotTransferredCount,
		)
}

// publishRunChanged nudges any browser watching this run. It is a hint, not the
// delivery mechanism: the dialog also polls, so a dropped push costs a second
// of latency rather than a stuck progress bar.
func (a *Activities) publishRunChanged(
	ctx context.Context,
	run *billingtransfer.BillingTransferRun,
) {
	if a.realtime == nil || run == nil {
		return
	}

	if err := a.realtime.PublishResourceInvalidation(
		ctx,
		&services.PublishResourceInvalidationRequest{
			OrganizationID: run.OrganizationID,
			BusinessUnitID: run.BusinessUnitID,
			Resource:       invalidationResource,
			Action:         "updated",
			RecordID:       run.ID,
			ActorUserID:    run.RequestedByID,
		},
	); err != nil {
		a.l.Warn("failed to publish billing transfer progress",
			zap.String("runId", run.ID.String()), zap.Error(err))
	}
}

func (p *TransferRunPayload) tenant() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  p.OrganizationID,
		BuID:   p.BusinessUnitID,
		UserID: p.UserID,
	}
}

func (p *ProcessBatchPayload) tenant() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  p.OrganizationID,
		BuID:   p.BusinessUnitID,
		UserID: p.UserID,
	}
}

func (p *FinalizeRunPayload) tenant() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  p.OrganizationID,
		BuID:   p.BusinessUnitID,
		UserID: p.UserID,
	}
}
