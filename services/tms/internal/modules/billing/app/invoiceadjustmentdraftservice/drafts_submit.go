package invoiceadjustmentdraftservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceadjustmentengine"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/timeutils"
)

func (s *Service) SubmitDraft(
	ctx context.Context,
	req *invoiceadjustmentports.GetInvoiceAdjustmentDetailRequest,
	actor *permissionports.RequestActor,
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

	computation, err := s.engine.ComputePreview(ctx, draftReq, entity.ID)
	if err != nil {
		return nil, err
	}
	if len(computation.Preview.Errors) > 0 {
		return nil, invoiceadjustmentengine.PreviewErrorsToMultiError(computation.Preview.Errors)
	}

	now := timeutils.NowUnix()
	entity.Kind = draftReq.Kind
	entity.RebillStrategy = draftReq.RebillStrategy
	entity.Reason = draftReq.Reason
	entity.PolicyReason = strings.Join(computation.Preview.Warnings, " | ")
	entity.AccountingDate = computation.Preview.AccountingDate
	entity.CreditTotalAmount = computation.Preview.CreditTotalAmount
	entity.RebillTotalAmount = computation.Preview.RebillTotalAmount
	entity.NetDeltaAmount = computation.Preview.NetDeltaAmount
	entity.RerateVariancePercent = computation.Preview.RerateVariancePercent
	entity.WouldCreateUnappliedCredit = computation.Preview.WouldCreateUnappliedCredit
	entity.RequiresReconciliationException = computation.Preview.RequiresReconciliationException
	entity.ApprovalRequired = computation.Preview.RequiresApproval
	entity.ReplacementReviewStatus = invoiceadjustmentengine.ReplacementReviewStatus(
		computation.Preview.RequiresReplacementInvoiceReview,
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
	if computation.Preview.RequiresApproval {
		entity.Status = invoiceadjustment.StatusPendingApproval
		entity.ApprovalStatus = invoiceadjustment.ApprovalStatusPending
	}

	snapshots := []*invoiceadjustment.InvoiceAdjustmentSnapshot{{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		AdjustmentID:   entity.ID,
		InvoiceID:      computation.Invoice.ID,
		Kind:           invoiceadjustment.SnapshotKindSubmission,
		Payload:        s.engine.SnapshotPayload(computation.Invoice),
		CreatedByID:    actor.UserID,
	}}

	var executed *invoiceadjustment.InvoiceAdjustment
	err = s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		if _, txErr := s.repo.UpdateAdjustment(txCtx, entity); txErr != nil {
			return txErr
		}
		if txErr := s.repo.CreateAdjustmentArtifacts(
			txCtx,
			invoiceadjustmentports.CreateAdjustmentArtifactsParams{Snapshots: snapshots},
		); txErr != nil {
			return txErr
		}
		if entity.Status == invoiceadjustment.StatusPendingApproval {
			return nil
		}
		var txErr error
		executed, txErr = s.engine.ExecuteApprovedAdjustment(txCtx, entity.ID, draftReq, actor)
		return txErr
	})
	if err != nil {
		return nil, err
	}

	if entity.Status == invoiceadjustment.StatusPendingApproval {
		return s.engine.GetDetail(ctx, req)
	}
	return executed, nil
}
