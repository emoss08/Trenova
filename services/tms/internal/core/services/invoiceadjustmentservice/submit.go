package invoiceadjustmentservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs/invoiceadjustmentjobs"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

func (s *Service) SubmitDraft(
	ctx context.Context,
	req *servicesports.GetInvoiceAdjustmentDetailRequest,
	actor *servicesports.RequestActor,
) (*invoiceadjustment.InvoiceAdjustment, error) {
	entity, draftReq, err := s.loadDraftRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	if entity.Status != invoiceadjustment.StatusDraft {
		return nil, errortypes.NewValidationError(
			"adjustmentId",
			errortypes.ErrInvalidOperation,
			"Only draft adjustments may be submitted",
		)
	}

	if multiErr := s.validator.ValidateRequest(ctx, draftReq); multiErr != nil {
		return nil, multiErr
	}

	computation, err := s.computePreview(ctx, draftReq, entity.ID)
	if err != nil {
		return nil, err
	}
	if len(computation.preview.Errors) > 0 {
		return nil, previewErrorsToMultiError(computation.preview.Errors)
	}

	now := timeutils.NowUnix()
	entity.Kind = draftReq.Kind
	entity.RebillStrategy = draftReq.RebillStrategy
	entity.Reason = draftReq.Reason
	entity.PolicyReason = strings.Join(computation.preview.Warnings, " | ")
	entity.AccountingDate = computation.preview.AccountingDate
	entity.CreditTotalAmount = computation.preview.CreditTotalAmount
	entity.RebillTotalAmount = computation.preview.RebillTotalAmount
	entity.NetDeltaAmount = computation.preview.NetDeltaAmount
	entity.RerateVariancePercent = computation.preview.RerateVariancePercent
	entity.WouldCreateUnappliedCredit = computation.preview.WouldCreateUnappliedCredit
	entity.RequiresReconciliationException = computation.preview.RequiresReconciliationException
	entity.ApprovalRequired = computation.preview.RequiresApproval
	entity.ReplacementReviewStatus = replacementReviewStatus(
		computation.preview.RequiresReplacementInvoiceReview,
	)
	entity.SubmittedByID = actor.UserID
	entity.SubmittedAt = &now
	if entity.Metadata == nil {
		entity.Metadata = make(map[string]any, 2)
	}
	entity.Metadata["draft"] = false
	entity.Metadata["attachmentIds"] = draftReq.AttachmentIDs
	entity.Status = invoiceadjustment.StatusApproved
	entity.ApprovalStatus = invoiceadjustment.ApprovalStatusNotRequired
	if computation.preview.RequiresApproval {
		entity.Status = invoiceadjustment.StatusPendingApproval
		entity.ApprovalStatus = invoiceadjustment.ApprovalStatusPending
	}

	snapshots := []*invoiceadjustment.InvoiceAdjustmentSnapshot{{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		AdjustmentID:   entity.ID,
		InvoiceID:      computation.invoice.ID,
		Kind:           invoiceadjustment.SnapshotKindSubmission,
		Payload:        s.snapshotPayload(computation.invoice),
		CreatedByID:    actor.UserID,
	}}

	var executed *invoiceadjustment.InvoiceAdjustment
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		if _, txErr := s.repo.UpdateAdjustment(txCtx, entity); txErr != nil {
			return txErr
		}
		if txErr := s.repo.CreateAdjustmentArtifacts(
			txCtx,
			repositories.CreateAdjustmentArtifactsParams{Snapshots: snapshots},
		); txErr != nil {
			return txErr
		}
		if entity.Status == invoiceadjustment.StatusPendingApproval {
			return nil
		}
		var txErr error
		executed, txErr = s.executeApprovedAdjustment(txCtx, entity.ID, draftReq, actor)
		return txErr
	})
	if err != nil {
		return nil, err
	}

	if entity.Status == invoiceadjustment.StatusPendingApproval {
		return s.GetDetail(ctx, req)
	}
	return executed, nil
}

func (s *Service) Preview(
	ctx context.Context,
	req *servicesports.InvoiceAdjustmentRequest,
	_ *servicesports.RequestActor,
) (*servicesports.InvoiceAdjustmentPreview, error) {
	if multiErr := s.validator.ValidateRequest(ctx, req); multiErr != nil {
		return nil, multiErr
	}

	computation, err := s.computePreview(ctx, req, pulid.ID(""))
	if err != nil {
		return nil, err
	}

	return computation.preview, nil
}

func (s *Service) Submit( //nolint:funlen // legacy workflow
	ctx context.Context,
	req *servicesports.InvoiceAdjustmentRequest,
	actor *servicesports.RequestActor,
) (*invoiceadjustment.InvoiceAdjustment, error) {
	if multiErr := s.validator.ValidateRequest(ctx, req); multiErr != nil {
		return nil, multiErr
	}

	if existing, err := s.repo.GetByIdempotencyKey(
		ctx,
		repositories.GetInvoiceAdjustmentByIdempotencyRequest{
			IdempotencyKey: req.IdempotencyKey,
			TenantInfo:     req.TenantInfo,
		},
	); err == nil &&
		existing != nil {
		return existing, nil
	}

	computation, err := s.computePreview(ctx, req, pulid.ID(""))
	if err != nil {
		return nil, err
	}
	if len(computation.preview.Errors) > 0 {
		return nil, previewErrorsToMultiError(computation.preview.Errors)
	}

	var result *invoiceadjustment.InvoiceAdjustment
	err = s.db.WithTx(
		ctx,
		ports.TxOptions{LockTimeout: 5 * 1000000000},
		func(txCtx context.Context, _ bun.Tx) error {
			group, groupErr := s.ensureCorrectionGroup(txCtx, computation.invoice)
			if groupErr != nil {
				return groupErr
			}
			if computation.invoice.CorrectionGroupID.IsNil() {
				computation.invoice.CorrectionGroupID = group.ID
				if _, groupErr = s.invoiceRepo.Update(txCtx, computation.invoice); groupErr != nil {
					return groupErr
				}
			}

			now := timeutils.NowUnix()
			adjustment := &invoiceadjustment.InvoiceAdjustment{
				ID:                pulid.MustNew("iadj_"),
				OrganizationID:    req.TenantInfo.OrgID,
				BusinessUnitID:    req.TenantInfo.BuID,
				CorrectionGroupID: group.ID,
				OriginalInvoiceID: computation.invoice.ID,
				Kind:              req.Kind,
				Status:            invoiceadjustment.StatusApproved,
				ApprovalStatus:    invoiceadjustment.ApprovalStatusNotRequired,
				ReplacementReviewStatus: replacementReviewStatus(
					computation.preview.RequiresReplacementInvoiceReview,
				),
				RebillStrategy:                  req.RebillStrategy,
				Reason:                          req.Reason,
				PolicyReason:                    strings.Join(computation.preview.Warnings, " | "),
				IdempotencyKey:                  req.IdempotencyKey,
				AccountingDate:                  computation.preview.AccountingDate,
				CreditTotalAmount:               computation.preview.CreditTotalAmount,
				RebillTotalAmount:               computation.preview.RebillTotalAmount,
				NetDeltaAmount:                  computation.preview.NetDeltaAmount,
				RerateVariancePercent:           computation.preview.RerateVariancePercent,
				WouldCreateUnappliedCredit:      computation.preview.WouldCreateUnappliedCredit,
				RequiresReconciliationException: computation.preview.RequiresReconciliationException,
				ApprovalRequired:                computation.preview.RequiresApproval,
				SubmittedByID:                   actor.UserID,
				SubmittedAt:                     &now,
				Metadata: map[string]any{
					"attachmentIds": req.AttachmentIDs,
				},
			}
			if computation.preview.RequiresApproval {
				adjustment.Status = invoiceadjustment.StatusPendingApproval
				adjustment.ApprovalStatus = invoiceadjustment.ApprovalStatusPending
			}
			references, refErr := s.buildDocumentReferences(
				txCtx,
				adjustment.ID,
				req.AttachmentIDs,
				computation.invoice,
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
				InvoiceID:      computation.invoice.ID,
				Kind:           invoiceadjustment.SnapshotKindSubmission,
				Payload:        s.snapshotPayload(computation.invoice),
				CreatedByID:    actor.UserID,
			}}

			for _, line := range computation.lines {
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
				repositories.CreateAdjustmentArtifactsParams{
					Adjustment:         adjustment,
					Lines:              computation.lines,
					Snapshots:          snapshots,
					DocumentReferences: references,
				},
			); err != nil {
				return err
			}

			if computation.preview.RequiresApproval {
				result, err = s.repo.GetByID(txCtx, repositories.GetInvoiceAdjustmentRequest{
					ID:         adjustment.ID,
					TenantInfo: req.TenantInfo,
				})
				return err
			}

			executed, execErr := s.executeApprovedAdjustment(txCtx, adjustment.ID, req, actor)
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
		s.logAdjustmentEvent("invoice adjustment submitted", result, zap.InfoLevel)
		s.logAudit(result, actor, permission.OpCreate, "Invoice adjustment submitted")
	}
	return result, nil
}

func (s *Service) BulkPreview(
	ctx context.Context,
	req *servicesports.InvoiceAdjustmentBulkRequest,
	actor *servicesports.RequestActor,
) ([]*servicesports.InvoiceAdjustmentPreview, error) {
	previews := make([]*servicesports.InvoiceAdjustmentPreview, 0, len(req.Items))
	for _, item := range req.Items {
		item.TenantInfo = req.TenantInfo
		preview, err := s.Preview(ctx, item, actor)
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
	req *servicesports.InvoiceAdjustmentBulkRequest,
	actor *servicesports.RequestActor,
) (*invoiceadjustment.InvoiceAdjustmentBatch, error) {
	if err := validateBulkRequest(req); err != nil {
		return nil, err
	}

	if existing, err := s.repo.GetBatchByIdempotencyKey(
		ctx,
		repositories.GetBatchByIdempotencyRequest{
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
		created.Metadata["workflowError"] = servicesports.ErrWorkflowStarterDisabled.Error()
		if _, err = s.repo.UpdateBatch(ctx, created); err != nil {
			return nil, err
		}
		return nil, servicesports.ErrWorkflowStarterDisabled
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
		invoiceadjustmentjobs.InvoiceAdjustmentBatchWorkflowName,
		&invoiceadjustmentjobs.BatchWorkflowPayload{
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

func validateBulkRequest(req *servicesports.InvoiceAdjustmentBulkRequest) error {
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

func replacementReviewStatus(required bool) invoiceadjustment.ReplacementReviewStatus {
	if required {
		return invoiceadjustment.ReplacementReviewStatusRequired
	}
	return invoiceadjustment.ReplacementReviewStatusNotRequired
}
