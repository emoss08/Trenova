package billingtransferservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/billingtransfer"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/temporaljobs/billingtransferjobs"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

const workflowIDPrefix = "billing-transfer-run/"

type StartRunRequest struct {
	TenantInfo pagination.TenantInfo
	Scope      billingtransfer.RunScope

	// ShipmentIDs is what a Selected run transfers. An AllMatching run names a
	// search instead and its shipments are resolved inside the workflow.
	ShipmentIDs []pulid.ID
	SearchQuery string

	ShipmentStatus              shipment.Status
	BillType                    billingqueue.BillType
	MarkCompletedReadyToInvoice bool
}

// Start records the run and hands it to the worker.
//
// The row is written before the workflow is started, and never the other way
// round: a run the browser can see but nothing is working on is recoverable,
// whereas a workflow with no row to write to has nowhere to report.
func (s *Service) Start(
	ctx context.Context,
	req *StartRunRequest,
) (*billingtransfer.BillingTransferRun, error) {
	log := s.l.With(zap.String("operation", "StartRun"))

	if !s.workflows.Enabled() {
		return nil, errortypes.NewBusinessError(
			"Transferring to billing is temporarily unavailable — the background worker is not connected",
		)
	}

	run, err := s.buildRun(req)
	if err != nil {
		return nil, err
	}

	created, err := s.runRepo.Create(ctx, run)
	if err != nil {
		// The partial unique index is what stops a double-click becoming two
		// runs over the same shipments; it deserves a sentence, not a 500.
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, errortypes.NewConflictError(
				"You already have a transfer running — wait for it to finish or stop it first",
			)
		}
		log.Error("failed to create billing transfer run", zap.Error(err))
		return nil, err
	}

	// A Selected run knows its shipments now, so its report is complete and
	// ordered before the worker has even picked the job up.
	if created.Scope == billingtransfer.RunScopeSelected {
		if _, err = s.runRepo.SeedItems(ctx, &repositories.SeedBillingTransferRunItemsRequest{
			TenantInfo:  req.TenantInfo,
			RunID:       created.ID,
			ShipmentIDs: req.ShipmentIDs,
		}); err != nil {
			log.Error("failed to seed billing transfer run items", zap.Error(err))
			return nil, err
		}
	}

	if err = s.startWorkflow(ctx, created); err != nil {
		return nil, err
	}

	return created, nil
}

func (s *Service) buildRun(req *StartRunRequest) (*billingtransfer.BillingTransferRun, error) {
	if req.TenantInfo.UserID.IsNil() {
		return nil, errortypes.NewAuthenticationError("A user is required to start a transfer")
	}

	billType := req.BillType
	if billType == "" {
		billType = billingqueue.BillTypeInvoice
	}

	run := &billingtransfer.BillingTransferRun{
		BusinessUnitID:              req.TenantInfo.BuID,
		OrganizationID:              req.TenantInfo.OrgID,
		RequestedByID:               req.TenantInfo.UserID,
		Status:                      billingtransfer.RunStatusQueued,
		Scope:                       req.Scope,
		SearchQuery:                 req.SearchQuery,
		ShipmentStatus:              req.ShipmentStatus,
		BillType:                    billType,
		MarkCompletedReadyToInvoice: req.MarkCompletedReadyToInvoice,
	}

	switch req.Scope {
	case billingtransfer.RunScopeSelected:
		req.ShipmentIDs = sliceutils.Dedupe(req.ShipmentIDs)
		run.TotalCount = len(req.ShipmentIDs)
	case billingtransfer.RunScopeAllMatching:
		// Counted by the workflow when it resolves the search.
		run.TotalCount = 0
	case billingtransfer.RunScopeRetry:
		return nil, errortypes.NewValidationError(
			"scope", errortypes.ErrInvalidOperation,
			"A retry is started from the transfer it retries",
		)
	default:
		return nil, errortypes.NewValidationError(
			"scope", errortypes.ErrInvalid, "Scope is invalid",
		)
	}

	multiErr := errortypes.NewMultiError()
	run.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return run, nil
}

// startWorkflow hands the run to Temporal. A run that cannot be queued is marked
// Failed rather than left sitting in Queued forever with nothing coming for it.
func (s *Service) startWorkflow(
	ctx context.Context,
	run *billingtransfer.BillingTransferRun,
) error {
	_, err := s.workflows.StartWorkflow(ctx,
		client.StartWorkflowOptions{
			ID:                    workflowIDPrefix + run.ID.String(),
			TaskQueue:             temporaltype.TaskQueueBilling.String(),
			WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
		},
		billingtransferjobs.BulkBillingTransferWorkflow,
		&billingtransferjobs.TransferRunPayload{
			BasePayload: temporaltype.BasePayload{
				OrganizationID: run.OrganizationID,
				BusinessUnitID: run.BusinessUnitID,
				UserID:         run.RequestedByID,
				Timestamp:      timeutils.NowUnix(),
			},
			RunID: run.ID,
		},
	)
	if err == nil {
		return nil
	}

	s.l.Error("failed to start billing transfer workflow",
		zap.String("runId", run.ID.String()), zap.Error(err))

	completedAt := timeutils.NowUnix()
	run.Status = billingtransfer.RunStatusFailed
	run.FailureMessage = "The transfer could not be queued"
	run.CompletedAt = &completedAt
	if _, updateErr := s.runRepo.Update(ctx, run); updateErr != nil {
		s.l.Error("failed to mark unqueued billing transfer run as failed",
			zap.String("runId", run.ID.String()), zap.Error(updateErr))
	}

	return errortypes.NewBusinessError(
		"The transfer could not be queued — try again shortly",
	)
}
