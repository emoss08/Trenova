package billingqueueapprovalservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingapprovalrun"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/jobs/billingqueuejobs/billingqueuecontract"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
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

const (
	workflowIDPrefix   = "billing-queue-approval/"
	maxRunReasonLength = 500
)

type Params struct {
	fx.In

	Repo      billingqueueports.BillingQueueReviewRepository
	Workflows kernelports.WorkflowStarter
	Logger    *zap.Logger
}

type Service struct {
	repo      billingqueueports.BillingQueueReviewRepository
	workflows kernelports.WorkflowStarter
	l         *zap.Logger
}

func New(p Params) billingqueueports.BillingQueueApprovalService {
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
	req *billingqueueports.StartBillingQueueApprovalRequest,
) (*billingapprovalrun.ApprovalRun, error) {
	if req.TenantInfo.UserID.IsNil() {
		return nil, errortypes.NewAuthenticationError("A user is required to approve")
	}
	key := strings.TrimSpace(req.IdempotencyKey)
	if key == "" || len(key) > 100 {
		return nil, errortypes.NewValidationError(
			"idempotencyKey",
			errortypes.ErrInvalid,
			"An idempotency key of up to 100 characters is required",
		)
	}

	if existing, err := s.repo.GetApprovalRunByKey(ctx, req.TenantInfo, key); err == nil {
		return existing, nil
	} else if !errortypes.IsNotFoundError(err) {
		return nil, err
	}

	if req.Action == "" {
		req.Action = billingapprovalrun.RunActionApprove
	}
	if err := validateRunAction(req); err != nil {
		return nil, err
	}

	itemIDs := sliceutils.Dedupe(req.ItemIDs)
	if len(itemIDs) == 0 {
		return nil, errortypes.NewValidationError(
			"itemIds",
			errortypes.ErrRequired,
			"Pick at least one item",
		)
	}
	if len(itemIDs) > billingapprovalrun.MaxApprovalRunItems {
		return nil, errortypes.NewValidationError(
			"itemIds", errortypes.ErrInvalid, "Act on at most 500 items at a time",
		)
	}

	run, err := s.repo.CreateApprovalRun(ctx, &billingqueueports.CreateApprovalRunRequest{
		Run: &billingapprovalrun.ApprovalRun{
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
			RequestedByID:  req.TenantInfo.UserID,
			IdempotencyKey: key,
			Action:         req.Action,
			Params:         req.Params,
			Status:         billingapprovalrun.ApprovalRunScheduled,
			TotalCount:     len(itemIDs),
			CommitAt:       commitAt(req.Action),
		},
		ItemIDs: itemIDs,
	})
	if err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return s.repo.GetApprovalRunByKey(ctx, req.TenantInfo, key)
		}
		return nil, err
	}

	if err = s.startWorkflow(ctx, run); err != nil {
		return nil, err
	}

	return run, nil
}

func (s *Service) startWorkflow(
	ctx context.Context,
	run *billingapprovalrun.ApprovalRun,
) error {
	wf, err := s.workflows.StartWorkflow(ctx,
		client.StartWorkflowOptions{
			ID:                    workflowIDPrefix + run.ID.String(),
			TaskQueue:             temporaltype.TaskQueueBilling.String(),
			WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
		},
		billingqueuecontract.BulkApprovalWorkflowName,
		&billingqueuecontract.ApprovalRunPayload{
			BasePayload: temporaltype.BasePayload{
				OrganizationID: run.OrganizationID,
				BusinessUnitID: run.BusinessUnitID,
				UserID:         run.RequestedByID,
				Timestamp:      timeutils.NowUnix(),
			},
			RunID:  run.ID,
			Action: run.Action,
			Params: run.Params,
		},
	)
	if err == nil {
		if wf != nil {
			if setErr := s.repo.SetApprovalRunWorkflow(
				ctx,
				&billingqueueports.MarkApprovalRunRunningRequest{
					TenantInfo:         tenantOf(run),
					RunID:              run.ID,
					TemporalWorkflowID: wf.GetID(),
					TemporalRunID:      wf.GetRunID(),
				},
			); setErr != nil {
				s.l.Warn("failed to note the approval workflow", zap.Error(setErr))
			}
			run.TemporalWorkflowID = wf.GetID()
			run.TemporalRunID = wf.GetRunID()
		}
		return nil
	}

	s.l.Error("failed to start billing queue approval workflow",
		zap.String("runId", run.ID.String()), zap.Error(err))
	if _, finErr := s.repo.FinalizeApprovalRun(ctx, &billingqueueports.FinalizeApprovalRunRequest{
		TenantInfo:     tenantOf(run),
		RunID:          run.ID,
		Status:         billingapprovalrun.ApprovalRunFailed,
		FailureMessage: "The bulk action could not be queued",
		LeftoverCode:   billingapprovalrun.ApprovalFailureUnexpected,
	}); finErr != nil {
		s.l.Error("failed to mark unqueued approval run as failed", zap.Error(finErr))
	}

	return errortypes.NewBusinessError("The bulk action could not be queued — try again shortly")
}

func (s *Service) Get(
	ctx context.Context,
	req *billingqueueports.BillingQueueApprovalRunRequest,
) (*billingapprovalrun.ApprovalRun, error) {
	return s.repo.GetApprovalRun(ctx, &billingqueueports.GetApprovalRunRequest{
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
	req *billingqueueports.BillingQueueApprovalRunRequest,
) (*billingapprovalrun.ApprovalRun, error) {
	run, err := s.Get(ctx, req)
	if err != nil {
		return nil, err
	}
	if run.RequestedByID != req.TenantInfo.UserID {
		return nil, errortypes.NewAuthorizationError("Only the person who started this can undo it")
	}

	stopped, err := s.repo.RequestApprovalRunUndo(
		ctx,
		&billingqueueports.RequestApprovalRunUndoRequest{
			TenantInfo:    req.TenantInfo,
			RunID:         run.ID,
			RequestedByID: req.TenantInfo.UserID,
		},
	)
	if err != nil {
		return nil, err
	}
	if !stopped {
		if run.CancelRequestedAt != nil {
			return run, nil
		}
		return nil, errortypes.NewBusinessError(
			"These items are already done — it's too late to undo",
		)
	}

	if run.TemporalWorkflowID != "" {
		if sigErr := s.workflows.SignalWorkflow(
			ctx,
			run.TemporalWorkflowID,
			run.TemporalRunID,
			billingqueuecontract.UndoSignalName,
			nil,
		); sigErr != nil {
			s.l.Warn("failed to signal an approval undo",
				zap.String("runId", run.ID.String()), zap.Error(sigErr))
		}
	}

	return s.Get(ctx, req)
}

func tenantOf(run *billingapprovalrun.ApprovalRun) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: run.OrganizationID, BuID: run.BusinessUnitID}
}

func commitAt(action billingapprovalrun.RunAction) int64 {
	now := timeutils.NowUnix()
	if !action.Undoable() {
		return now
	}

	return now + billingapprovalrun.ApprovalUndoWindowSeconds
}

func validateRunAction(req *billingqueueports.StartBillingQueueApprovalRequest) error {
	if !req.Action.IsValid() {
		return errortypes.NewValidationError("action", errortypes.ErrInvalid, "Unknown bulk action")
	}

	multiErr := errortypes.NewMultiError()
	params := &req.Params
	params.Reason = strings.TrimSpace(params.Reason)
	switch req.Action {
	case billingapprovalrun.RunActionAssign:
		if params.BillerID.IsNil() {
			multiErr.Add("params.billerId", errortypes.ErrRequired, "Choose who to assign")
		}
	case billingapprovalrun.RunActionHold:
		if !billingqueue.HoldReasonCode(params.HoldReasonCode).IsValid() {
			multiErr.Add(
				"params.holdReasonCode",
				errortypes.ErrRequired,
				"Say why the items are on hold",
			)
		}
	case billingapprovalrun.RunActionCancel:
		if params.Reason == "" {
			multiErr.Add("params.reason", errortypes.ErrRequired, "Say why the items are canceled")
		}
	case billingapprovalrun.RunActionResolveIssue:
		if params.IssueCode == "" {
			multiErr.Add("params.issueCode", errortypes.ErrRequired, "Name the check to settle")
		}
		if strings.TrimSpace(params.OptionKey) == "" {
			multiErr.Add("params.optionKey", errortypes.ErrRequired, "Name how to settle it")
		}
	case billingapprovalrun.RunActionApprove,
		billingapprovalrun.RunActionPost,
		billingapprovalrun.RunActionRequestPaperwork:
	}
	if len(params.Reason) > maxRunReasonLength {
		multiErr.Add(
			"params.reason",
			errortypes.ErrInvalid,
			"Keep the reason under {0} characters",
			maxRunReasonLength,
		)
	}
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}
