package billingqueueapprovalservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs/billingqueuejobs"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const workflowIDPrefix = "billing-queue-approval/"

type Params struct {
	fx.In

	Repo      repositories.BillingQueueReviewRepository
	Workflows services.WorkflowStarter
	Logger    *zap.Logger
}

type Service struct {
	repo      repositories.BillingQueueReviewRepository
	workflows services.WorkflowStarter
	l         *zap.Logger
}

func New(p Params) services.BillingQueueApprovalService {
	return &Service{
		repo:      p.Repo,
		workflows: p.Workflows,
		l:         p.Logger.Named("service.billing-queue-approval"),
	}
}

// Start records a bulk approval and hands it to the worker, which waits out
// the undo window before approving anything.
//
// The client sends one idempotency key per press of Approve. A request that
// arrives twice — a retry, a double click — finds the run the first one made
// and returns it, so the same items are never approved by two runs.
func (s *Service) Start(
	ctx context.Context,
	req *services.StartBillingQueueApprovalRequest,
) (*billingqueue.ApprovalRun, error) {
	if req.TenantInfo.UserID.IsNil() {
		return nil, errortypes.NewAuthenticationError("A user is required to approve")
	}
	key := strings.TrimSpace(req.IdempotencyKey)
	if key == "" || len(key) > 100 {
		return nil, errortypes.NewValidationError(
			"idempotencyKey", errortypes.ErrInvalid, "An idempotency key of up to 100 characters is required",
		)
	}

	if existing, err := s.repo.GetApprovalRunByKey(ctx, req.TenantInfo, key); err == nil {
		return existing, nil
	} else if !errortypes.IsNotFoundError(err) {
		return nil, err
	}

	itemIDs := sliceutils.Dedupe(req.ItemIDs)
	if len(itemIDs) == 0 {
		return nil, errortypes.NewValidationError("itemIds", errortypes.ErrRequired, "Pick at least one item")
	}
	if len(itemIDs) > billingqueue.MaxApprovalRunItems {
		return nil, errortypes.NewValidationError(
			"itemIds", errortypes.ErrInvalid, "Approve at most 500 items at a time",
		)
	}
	if !s.workflows.Enabled() {
		return nil, errortypes.NewBusinessError(
			"Approving in bulk is temporarily unavailable — the background worker is not connected",
		)
	}

	run, err := s.repo.CreateApprovalRun(ctx, &repositories.CreateApprovalRunRequest{
		Run: &billingqueue.ApprovalRun{
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
			RequestedByID:  req.TenantInfo.UserID,
			IdempotencyKey: key,
			Status:         billingqueue.ApprovalRunScheduled,
			TotalCount:     len(itemIDs),
			CommitAt:       timeutils.NowUnix() + billingqueue.ApprovalUndoWindowSeconds,
		},
		ItemIDs: itemIDs,
	})
	if err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return s.repo.GetApprovalRunByKey(ctx, req.TenantInfo, key)
		}
		return nil, err
	}

	if err = s.startWorkflow(ctx, run, req.AssignApprover); err != nil {
		return nil, err
	}

	return run, nil
}

func (s *Service) startWorkflow(
	ctx context.Context,
	run *billingqueue.ApprovalRun,
	assignApprover bool,
) error {
	wf, err := s.workflows.StartWorkflow(ctx,
		client.StartWorkflowOptions{
			ID:                    workflowIDPrefix + run.ID.String(),
			TaskQueue:             temporaltype.TaskQueueBilling.String(),
			WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
		},
		billingqueuejobs.BulkApprovalWorkflow,
		&billingqueuejobs.ApprovalRunPayload{
			BasePayload: temporaltype.BasePayload{
				OrganizationID: run.OrganizationID,
				BusinessUnitID: run.BusinessUnitID,
				UserID:         run.RequestedByID,
				Timestamp:      timeutils.NowUnix(),
			},
			RunID:          run.ID,
			AssignApprover: assignApprover,
		},
	)
	if err == nil {
		if wf != nil {
			if setErr := s.repo.SetApprovalRunWorkflow(ctx, &repositories.MarkApprovalRunRunningRequest{
				TenantInfo:         tenantOf(run),
				RunID:              run.ID,
				TemporalWorkflowID: wf.GetID(),
				TemporalRunID:      wf.GetRunID(),
			}); setErr != nil {
				s.l.Warn("failed to note the approval workflow", zap.Error(setErr))
			}
			run.TemporalWorkflowID = wf.GetID()
			run.TemporalRunID = wf.GetRunID()
		}
		return nil
	}

	s.l.Error("failed to start billing queue approval workflow",
		zap.String("runId", run.ID.String()), zap.Error(err))
	if _, finErr := s.repo.FinalizeApprovalRun(ctx, &repositories.FinalizeApprovalRunRequest{
		TenantInfo:     tenantOf(run),
		RunID:          run.ID,
		Status:         billingqueue.ApprovalRunFailed,
		FailureMessage: "The approval could not be queued",
		LeftoverCode:   billingqueue.ApprovalFailureUnexpected,
	}); finErr != nil {
		s.l.Error("failed to mark unqueued approval run as failed", zap.Error(finErr))
	}

	return errortypes.NewBusinessError("The approval could not be queued — try again shortly")
}

func (s *Service) Get(
	ctx context.Context,
	req *services.BillingQueueApprovalRunRequest,
) (*billingqueue.ApprovalRun, error) {
	return s.repo.GetApprovalRun(ctx, &repositories.GetApprovalRunRequest{
		TenantInfo:   req.TenantInfo,
		RunID:        req.RunID,
		IncludeItems: true,
	})
}

// Undo takes a bulk approval back inside its window. The row is written first
// and is what the workflow obeys; the signal only lets it stop without
// waiting for the window to close.
func (s *Service) Undo(
	ctx context.Context,
	req *services.BillingQueueApprovalRunRequest,
) (*billingqueue.ApprovalRun, error) {
	run, err := s.Get(ctx, req)
	if err != nil {
		return nil, err
	}
	if run.RequestedByID != req.TenantInfo.UserID {
		return nil, errortypes.NewAuthorizationError("Only the person who approved can undo it")
	}

	stopped, err := s.repo.RequestApprovalRunUndo(ctx, &repositories.RequestApprovalRunUndoRequest{
		TenantInfo:    req.TenantInfo,
		RunID:         run.ID,
		RequestedByID: req.TenantInfo.UserID,
	})
	if err != nil {
		return nil, err
	}
	if !stopped {
		if run.CancelRequestedAt != nil {
			return run, nil
		}
		return nil, errortypes.NewBusinessError("These invoices are already approved — it's too late to undo")
	}

	if run.TemporalWorkflowID != "" {
		if sigErr := s.workflows.SignalWorkflow(
			ctx, run.TemporalWorkflowID, run.TemporalRunID, billingqueuejobs.UndoSignalName, nil,
		); sigErr != nil {
			s.l.Warn("failed to signal an approval undo",
				zap.String("runId", run.ID.String()), zap.Error(sigErr))
		}
	}

	return s.Get(ctx, req)
}

func tenantOf(run *billingqueue.ApprovalRun) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: run.OrganizationID, BuID: run.BusinessUnitID}
}
