package billingtransferservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/billingtransfer"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/temporaljobs/billingtransferjobs"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.uber.org/zap"
)

// Cancel asks a run to stop.
//
// The database flag is the authority, not the Temporal signal: it survives a
// worker restart and a Temporal outage, it applies to a run no worker has picked
// up yet, and the dialog can show "stopping" the moment it is written. The
// signal only shortens the wait.
//
// The run is deliberately not moved to a terminal state here. The workflow stops
// between batches and finalizes itself, so the batch already in flight finishes
// and writes its outcomes — abandoning it would leave shipments committed to the
// billing queue that the run's own report never mentions.
func (s *Service) Cancel(
	ctx context.Context,
	req *RunRequest,
) (*billingtransfer.BillingTransferRun, error) {
	run, err := s.runRepo.GetByID(ctx, &repositories.GetBillingTransferRunRequest{
		TenantInfo: req.TenantInfo,
		RunID:      req.RunID,
	})
	if err != nil {
		return nil, err
	}

	if run.RequestedByID != req.TenantInfo.UserID {
		return nil, errortypes.NewAuthorizationError(
			"Only the person who started a transfer can stop it",
		)
	}

	if run.Status.IsTerminal() {
		return nil, errortypes.NewBusinessError("This transfer has already finished")
	}

	canceled, err := s.runRepo.RequestCancel(
		ctx,
		&repositories.RequestBillingTransferRunCancelRequest{
			TenantInfo:    req.TenantInfo,
			RunID:         run.ID,
			RequestedByID: req.TenantInfo.UserID,
		},
	)
	if err != nil {
		return nil, err
	}

	if canceled.TemporalWorkflowID != "" {
		if sigErr := s.workflows.SignalWorkflow(
			ctx,
			canceled.TemporalWorkflowID,
			canceled.TemporalRunID,
			billingtransferjobs.CancelSignalName,
			nil,
		); sigErr != nil {
			// Not an error the caller needs to see: the flag is already written
			// and the next batch will read it.
			s.l.Warn("failed to signal a billing transfer cancel",
				zap.String("runId", run.ID.String()), zap.Error(sigErr))
		}
	}

	return canceled, nil
}

// Retry starts a fresh run over the shipments a second attempt could still move.
//
// Every shipment from the source comes across, not only the retried ones:
// outcomes that cannot change keep the answer they already have. That way the
// retry run is one complete report rather than a fragment somebody has to read
// alongside the run before it.
func (s *Service) Retry(
	ctx context.Context,
	req *RunRequest,
) (*billingtransfer.BillingTransferRun, error) {
	log := s.l.With(zap.String("operation", "RetryRun"))

	if !s.workflows.Enabled() {
		return nil, errortypes.NewBusinessError(
			"Transferring to billing is temporarily unavailable — the background worker is not connected",
		)
	}

	source, err := s.runRepo.GetByID(ctx, &repositories.GetBillingTransferRunRequest{
		TenantInfo: req.TenantInfo,
		RunID:      req.RunID,
	})
	if err != nil {
		return nil, err
	}

	if !source.Status.IsTerminal() {
		return nil, errortypes.NewBusinessError(
			"Wait for this transfer to finish before retrying it",
		)
	}

	if source.RetryableCount == 0 && source.SkippedCount == 0 {
		return nil, errortypes.NewBusinessError(
			"Nothing in this transfer can be retried",
		)
	}

	run := &billingtransfer.BillingTransferRun{
		BusinessUnitID:              req.TenantInfo.BuID,
		OrganizationID:              req.TenantInfo.OrgID,
		RequestedByID:               req.TenantInfo.UserID,
		SourceRunID:                 source.ID,
		Status:                      billingtransfer.RunStatusQueued,
		Scope:                       billingtransfer.RunScopeRetry,
		SearchQuery:                 source.SearchQuery,
		ShipmentStatus:              source.ShipmentStatus,
		BillType:                    source.BillType,
		MarkCompletedReadyToInvoice: source.MarkCompletedReadyToInvoice,
		TotalCount:                  source.TotalCount,
		UnmatchedCount:              source.UnmatchedCount,
	}

	multiErr := errortypes.NewMultiError()
	run.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	created, err := s.runRepo.Create(ctx, run)
	if err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, errortypes.NewConflictError(
				"You already have a transfer running — wait for it to finish or stop it first",
			)
		}
		log.Error("failed to create retry billing transfer run", zap.Error(err))
		return nil, err
	}

	seeded, err := s.runRepo.SeedRetryItems(
		ctx,
		&repositories.SeedBillingTransferRetryItemsRequest{
			TenantInfo:  req.TenantInfo,
			RunID:       created.ID,
			SourceRunID: source.ID,
		},
	)
	if err != nil {
		log.Error("failed to seed retry billing transfer run items", zap.Error(err))
		return nil, err
	}

	// The carried-over answers are already settled, so the run starts partway
	// along rather than pretending it has everything left to do.
	created.TotalCount = seeded.TotalCount
	created.ProcessedCount = seeded.TotalCount - seeded.PendingCount
	created.TransferredCount = seeded.TransferredCount
	created.NotTransferredCount = seeded.NotTransferredCount
	created.MarkedReadyToInvoiceCount = seeded.MarkedReadyToInvoiceCount
	if _, err = s.runRepo.Update(ctx, created); err != nil {
		log.Error("failed to seed retry run counters", zap.Error(err))
		return nil, err
	}

	if err = s.startWorkflow(ctx, created); err != nil {
		return nil, err
	}

	return created, nil
}
