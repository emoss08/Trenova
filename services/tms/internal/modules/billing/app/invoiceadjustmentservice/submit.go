package invoiceadjustmentservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceadjustmentengine"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/jobs/invoiceadjustmentjobs/invoiceadjustmentcontract"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

func (s *Service) Submit( //nolint:funlen // legacy workflow
	ctx context.Context,
	req *invoiceadjustmentports.InvoiceAdjustmentRequest,
	actor *permissionports.RequestActor,
) (*invoiceadjustment.InvoiceAdjustment, error) {
	if multiErr := s.validator.ValidateRequest(ctx, req); multiErr != nil {
		return nil, multiErr
	}

	if existing, err := s.repo.GetByIdempotencyKey(
		ctx,
		invoiceadjustmentports.GetInvoiceAdjustmentByIdempotencyRequest{
			IdempotencyKey: req.IdempotencyKey,
			TenantInfo:     req.TenantInfo,
		},
	); err == nil &&
		existing != nil {
		return existing, nil
	}

	computation, err := s.engine.ComputePreview(ctx, req, pulid.ID(""))
	if err != nil {
		return nil, err
	}
	if len(computation.Preview.Errors) > 0 {
		return nil, invoiceadjustmentengine.PreviewErrorsToMultiError(computation.Preview.Errors)
	}

	var result *invoiceadjustment.InvoiceAdjustment
	err = s.db.Transact(
		ctx,
		kernelports.TxOptions{LockTimeout: 5 * 1000000000},
		func(txCtx context.Context) error {
			group, groupErr := s.engine.EnsureCorrectionGroup(txCtx, computation.Invoice)
			if groupErr != nil {
				return groupErr
			}
			if computation.Invoice.CorrectionGroupID.IsNil() {
				computation.Invoice.CorrectionGroupID = group.ID
				if _, groupErr = s.invoiceRepo.Update(txCtx, computation.Invoice); groupErr != nil {
					return groupErr
				}
			}

			now := timeutils.NowUnix()
			adjustment := &invoiceadjustment.InvoiceAdjustment{
				ID:                pulid.MustNew("iadj_"),
				OrganizationID:    req.TenantInfo.OrgID,
				BusinessUnitID:    req.TenantInfo.BuID,
				CorrectionGroupID: group.ID,
				OriginalInvoiceID: computation.Invoice.ID,
				Kind:              req.Kind,
				Status:            invoiceadjustment.StatusApproved,
				ApprovalStatus:    invoiceadjustment.ApprovalStatusNotRequired,
				ReplacementReviewStatus: invoiceadjustmentengine.ReplacementReviewStatus(
					computation.Preview.RequiresReplacementInvoiceReview,
				),
				RebillStrategy:                  req.RebillStrategy,
				Reason:                          req.Reason,
				PolicyReason:                    strings.Join(computation.Preview.Warnings, " | "),
				IdempotencyKey:                  req.IdempotencyKey,
				AccountingDate:                  computation.Preview.AccountingDate,
				CreditTotalAmount:               computation.Preview.CreditTotalAmount,
				RebillTotalAmount:               computation.Preview.RebillTotalAmount,
				NetDeltaAmount:                  computation.Preview.NetDeltaAmount,
				RerateVariancePercent:           computation.Preview.RerateVariancePercent,
				WouldCreateUnappliedCredit:      computation.Preview.WouldCreateUnappliedCredit,
				RequiresReconciliationException: computation.Preview.RequiresReconciliationException,
				ApprovalRequired:                computation.Preview.RequiresApproval,
				SubmittedByID:                   actor.UserID,
				SubmittedAt:                     &now,
				Metadata:                        invoiceadjustmentengine.AdjustmentMetadata(req),
			}
			if computation.Preview.RequiresApproval {
				adjustment.Status = invoiceadjustment.StatusPendingApproval
				adjustment.ApprovalStatus = invoiceadjustment.ApprovalStatusPending
			}
			references, refErr := s.engine.BuildDocumentReferences(
				txCtx,
				adjustment.ID,
				req.AttachmentIDs,
				computation.Invoice,
				req.TenantInfo,
				actor,
				"attachmentIds",
			)
			if refErr != nil {
				return refErr
			}

			snapshots := []*invoiceadjustment.InvoiceAdjustmentSnapshot{{
				OrganizationID: req.TenantInfo.OrgID,
				BusinessUnitID: req.TenantInfo.BuID,
				InvoiceID:      computation.Invoice.ID,
				Kind:           invoiceadjustment.SnapshotKindSubmission,
				Payload:        s.engine.SnapshotPayload(computation.Invoice),
				CreatedByID:    actor.UserID,
			}}

			for _, line := range computation.Lines {
				line.OrganizationID = req.TenantInfo.OrgID
				line.BusinessUnitID = req.TenantInfo.BuID
				line.AdjustmentID = adjustment.ID
			}
			for _, snapshot := range snapshots {
				snapshot.OrganizationID = req.TenantInfo.OrgID
				snapshot.BusinessUnitID = req.TenantInfo.BuID
				snapshot.AdjustmentID = adjustment.ID
			}

			if err = s.repo.CreateAdjustmentArtifacts(
				txCtx,
				invoiceadjustmentports.CreateAdjustmentArtifactsParams{
					Adjustment:         adjustment,
					Lines:              computation.Lines,
					Snapshots:          snapshots,
					DocumentReferences: references,
				},
			); err != nil {
				return err
			}

			if computation.Preview.RequiresApproval {
				result, err = s.repo.GetByID(
					txCtx,
					invoiceadjustmentports.GetInvoiceAdjustmentRequest{
						ID:         adjustment.ID,
						TenantInfo: req.TenantInfo,
					},
				)
				return err
			}

			executed, execErr := s.engine.ExecuteApprovedAdjustment(
				txCtx,
				adjustment.ID,
				req,
				actor,
			)
			if execErr != nil {
				return execErr
			}
			result = executed
			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	if result != nil {
		s.engine.LogAdjustmentEvent("invoice adjustment submitted", result, zap.InfoLevel)
		s.engine.LogAudit(result, actor, permission.OpCreate, "Invoice adjustment submitted")
	}
	return result, nil
}

func (s *Service) BulkPreview(
	ctx context.Context,
	req *invoiceadjustmentports.InvoiceAdjustmentBulkRequest,
	actor *permissionports.RequestActor,
) ([]*invoiceadjustmentports.InvoiceAdjustmentPreview, error) {
	previews := make([]*invoiceadjustmentports.InvoiceAdjustmentPreview, 0, len(req.Items))
	for _, item := range req.Items {
		item.TenantInfo = req.TenantInfo
		preview, err := s.engine.Preview(ctx, item, actor)
		if err != nil {
			return nil, err
		}
		previews = append(previews, preview)
	}
	return previews, nil
}

//nolint:nestif // existing validation flow mirrors business rule nesting
func (s *Service) BulkSubmit( //nolint:funlen // legacy workflow
	ctx context.Context,
	req *invoiceadjustmentports.InvoiceAdjustmentBulkRequest,
	actor *permissionports.RequestActor,
) (*invoiceadjustment.InvoiceAdjustmentBatch, error) {
	if err := validateBulkRequest(req); err != nil {
		return nil, err
	}

	if existing, err := s.repo.GetBatchByIdempotencyKey(
		ctx,
		invoiceadjustmentports.GetBatchByIdempotencyRequest{
			IdempotencyKey: req.IdempotencyKey,
			TenantInfo:     req.TenantInfo,
		},
	); err == nil &&
		existing != nil {
		return existing, nil
	}

	inlineProcessing := len(req.Items) <= batchInlineThreshold
	now := timeutils.NowUnix()
	batch := &invoiceadjustment.InvoiceAdjustmentBatch{
		ID:             pulid.MustNew("iadjb_"),
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		IdempotencyKey: req.IdempotencyKey,
		Status:         invoiceadjustment.BatchStatusQueued,
		TotalCount:     len(req.Items),
		SubmittedByID:  actor.UserID,
		SubmittedAt:    &now,
		Metadata: map[string]any{
			"processingMode": map[bool]string{true: "inline", false: "temporal"}[inlineProcessing],
		},
	}
	if inlineProcessing {
		batch.Status = invoiceadjustment.BatchStatusRunning
	}

	items := make([]*invoiceadjustment.InvoiceAdjustmentBatchItem, 0, len(req.Items))
	for idx, item := range req.Items {
		item.TenantInfo = req.TenantInfo
		items = append(items, &invoiceadjustment.InvoiceAdjustmentBatchItem{
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
			BatchID:        batch.ID,
			InvoiceID:      item.InvoiceID,
			IdempotencyKey: fmt.Sprintf("%s-%d", req.IdempotencyKey, idx),
			Status:         invoiceadjustment.BatchItemStatusPending,
			RequestPayload: jsonutils.MustToJSON(item),
		})
	}

	created, err := s.repo.CreateBatch(ctx, batch, items)
	if err != nil {
		return nil, err
	}

	if inlineProcessing {
		for idx, item := range req.Items {
			created.Items[idx].Status = invoiceadjustment.BatchItemStatusExecuting
			if _, err = s.repo.UpdateBatchItem(ctx, created.Items[idx]); err != nil {
				return nil, err
			}

			adjustment, submitErr := s.Submit(ctx, item, actor)
			created.ProcessedCount++
			if submitErr != nil {
				created.Items[idx].Status = invoiceadjustment.BatchItemStatusFailed
				created.Items[idx].ErrorMessage = submitErr.Error()
				created.FailedCount++
			} else {
				created.Items[idx].AdjustmentID = adjustment.ID
				if adjustment.Status == invoiceadjustment.StatusPendingApproval {
					created.Items[idx].Status = invoiceadjustment.BatchItemStatusPendingApproval
				} else {
					created.Items[idx].Status = invoiceadjustment.BatchItemStatusExecuted
				}
				created.Items[idx].ResultPayload = jsonutils.MustToJSON(adjustment)
				created.SucceededCount++
			}
			if _, err = s.repo.UpdateBatchItem(ctx, created.Items[idx]); err != nil {
				return nil, err
			}
		}

		switch {
		case created.FailedCount == 0:
			created.Status = invoiceadjustment.BatchStatusCompleted
		case created.SucceededCount == 0:
			created.Status = invoiceadjustment.BatchStatusFailed
		default:
			created.Status = invoiceadjustment.BatchStatusPartial
		}

		return s.repo.UpdateBatch(ctx, created)
	}

	if !s.workflowStarter.Enabled() {
		created.Status = invoiceadjustment.BatchStatusFailed
		created.Metadata["workflowError"] = kernelports.ErrWorkflowStarterDisabled.Error()
		if _, err = s.repo.UpdateBatch(ctx, created); err != nil {
			return nil, err
		}
		return nil, kernelports.ErrWorkflowStarterDisabled
	}

	workflowID := fmt.Sprintf(
		"invoice-adjustment-batch-%s-%s-%s",
		req.TenantInfo.OrgID.String(),
		req.TenantInfo.BuID.String(),
		created.ID.String(),
	)
	run, err := s.workflowStarter.StartWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:            workflowID,
			TaskQueue:     temporaltype.TaskQueueBilling.String(),
			StaticSummary: "Invoice adjustment batch " + created.ID.String(),
		},
		invoiceadjustmentcontract.InvoiceAdjustmentBatchWorkflowName,
		&invoiceadjustmentcontract.BatchWorkflowPayload{
			BatchID:         created.ID,
			ItemIDs:         collectBatchItemIDs(created.Items),
			OrganizationID:  req.TenantInfo.OrgID,
			BusinessUnitID:  req.TenantInfo.BuID,
			UserID:          actor.UserID,
			PrincipalType:   actor.PrincipalType,
			PrincipalID:     actor.PrincipalID,
			APIKeyID:        actor.APIKeyID,
			WorkflowStarted: now,
		},
	)
	if err != nil {
		created.Status = invoiceadjustment.BatchStatusFailed
		created.Metadata["workflowError"] = err.Error()
		if _, updateErr := s.repo.UpdateBatch(ctx, created); updateErr != nil {
			return nil, updateErr
		}
		return nil, err
	}

	created.Status = invoiceadjustment.BatchStatusSubmitted
	created.Metadata["workflowId"] = workflowID
	created.Metadata["runId"] = run.GetRunID()
	return s.repo.UpdateBatch(ctx, created)
}

func validateBulkRequest(req *invoiceadjustmentports.InvoiceAdjustmentBulkRequest) error {
	if req == nil {
		return errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Bulk adjustment request is required",
		)
	}
	if strings.TrimSpace(req.IdempotencyKey) == "" {
		return errortypes.NewValidationError(
			"idempotencyKey",
			errortypes.ErrRequired,
			"Idempotency key is required",
		)
	}
	if len(req.Items) == 0 {
		return errortypes.NewValidationError(
			"items",
			errortypes.ErrRequired,
			"At least one bulk adjustment item is required",
		)
	}
	if req.TenantInfo.OrgID.IsNil() {
		return errortypes.NewValidationError(
			"tenantInfo.orgId",
			errortypes.ErrRequired,
			"Organization ID is required",
		)
	}
	if req.TenantInfo.BuID.IsNil() {
		return errortypes.NewValidationError(
			"tenantInfo.buId",
			errortypes.ErrRequired,
			"Business unit ID is required",
		)
	}
	return nil
}

// batchInlineThreshold is how many lines a bulk request may carry before it is
// worth processing each adjustment on its own rather than in one transaction.
const batchInlineThreshold = 25

func collectBatchItemIDs(items []*invoiceadjustment.InvoiceAdjustmentBatchItem) []pulid.ID {
	ids := make([]pulid.ID, 0, len(items))
	for _, item := range items {
		if item != nil {
			ids = append(ids, item.ID)
		}
	}
	return ids
}
