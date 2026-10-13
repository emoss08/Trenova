package invoiceadjustmentservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func (s *Service) Approve(
	ctx context.Context,
	req *invoiceadjustmentports.ApproveInvoiceAdjustmentRequest,
	actor *permissionports.RequestActor,
) (*invoiceadjustment.InvoiceAdjustment, error) {
	var result *invoiceadjustment.InvoiceAdjustment
	err := s.db.WithTx(
		ctx,
		kernelports.TxOptions{LockTimeout: 5 * 1000000000},
		func(txCtx context.Context, _ bun.Tx) error {
			if guardErr := s.refuseSelfApproval(txCtx, req, actor); guardErr != nil {
				return guardErr
			}
			executed, execErr := s.executeApprovedAdjustment(
				txCtx,
				req.AdjustmentID,
				&invoiceadjustmentports.InvoiceAdjustmentRequest{
					TenantInfo: req.TenantInfo,
				},
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
		s.logAdjustmentEvent("invoice adjustment approved", result, zap.InfoLevel)
		s.logAudit(result, actor, permission.OpApprove, "Invoice adjustment approved")
	}
	return result, nil
}

// Reject closes an adjustment without it taking effect. It holds the invoice's
// lock, the same one approval takes, so a rejection and an approval of the same
// adjustment cannot both win. An adjustment whose execution failed
// revalidation can be rejected too, which is the only way to close it when the
// condition that failed it will not clear. A rejected void reversal takes the
// void reason it recorded off the invoice, which stays posted.
func (s *Service) Reject(
	ctx context.Context,
	req *invoiceadjustmentports.RejectInvoiceAdjustmentRequest,
	actor *permissionports.RequestActor,
) (*invoiceadjustment.InvoiceAdjustment, error) {
	var (
		updated  *invoiceadjustment.InvoiceAdjustment
		restored *invoice.Invoice
	)
	err := s.db.WithTx(
		ctx,
		kernelports.TxOptions{LockTimeout: 5 * time.Second},
		func(txCtx context.Context, _ bun.Tx) error {
			entity, txErr := s.lockForDecision(txCtx, req.AdjustmentID, req.TenantInfo)
			if txErr != nil {
				return txErr
			}
			if !entity.adjustment.Status.CanBeRejected() {
				return errortypes.NewValidationError(
					"adjustmentId",
					errortypes.ErrInvalidOperation,
					"Only adjustments awaiting approval or whose execution failed may be rejected",
				)
			}

			entity.adjustment.Reject(actor.UserID, timeutils.NowUnix(), req.Reason)
			if updated, txErr = s.repo.UpdateAdjustment(txCtx, entity.adjustment); txErr != nil {
				return txErr
			}

			restored, txErr = s.withdrawVoidRequest(txCtx, updated, entity.invoice)
			return txErr
		},
	)
	if err != nil {
		return nil, err
	}

	s.logAdjustmentEvent("invoice adjustment rejected", updated, zap.InfoLevel)
	s.logAudit(updated, actor, permission.OpReject, "Invoice adjustment rejected")
	s.publishAdjustedInvoices(ctx, actor, restored)
	return updated, nil
}

// refuseSelfApproval keeps whoever submitted an adjustment that needs
// approval from approving it, where the organization asks for a separate
// approver. Retrying one whose execution failed without ever needing approval
// is not an approval and stays open to its submitter.
func (s *Service) refuseSelfApproval(
	ctx context.Context,
	req *invoiceadjustmentports.ApproveInvoiceAdjustmentRequest,
	actor *permissionports.RequestActor,
) error {
	if actor == nil || actor.UserID.IsNil() || s.adjustmentCtrlRepo == nil {
		return nil
	}

	adjustment, err := s.repo.GetByID(ctx, invoiceadjustmentports.GetInvoiceAdjustmentRequest{
		ID:         req.AdjustmentID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return err
	}
	if !adjustment.ApprovalRequired || adjustment.SubmittedByID != actor.UserID {
		return nil
	}

	control, err := s.adjustmentCtrlRepo.GetByOrgID(ctx, adjustment.OrganizationID)
	if errortypes.IsNotFoundError(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !control.RequireSeparateApprover {
		return nil
	}

	return errortypes.NewValidationError(
		"adjustmentId",
		errortypes.ErrInvalidOperation,
		"You submitted this adjustment, so someone else must approve it",
	)
}

type lockedDecision struct {
	adjustment *invoiceadjustment.InvoiceAdjustment
	invoice    *invoice.Invoice
}

// lockForDecision locks the adjustment's invoice and then reads the adjustment,
// so the status it returns cannot change before the caller's transaction ends.
func (s *Service) lockForDecision(
	ctx context.Context,
	adjustmentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*lockedDecision, error) {
	request := invoiceadjustmentports.GetInvoiceAdjustmentRequest{
		ID:         adjustmentID,
		TenantInfo: tenantInfo,
	}
	unlocked, err := s.repo.GetByID(ctx, request)
	if err != nil {
		return nil, err
	}

	locked, err := s.repo.LockInvoiceForUpdate(
		ctx,
		invoiceadjustmentports.LockInvoiceAdjustmentRequest{
			InvoiceID:  unlocked.OriginalInvoiceID,
			TenantInfo: tenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	adjustment, err := s.repo.GetByID(ctx, request)
	if err != nil {
		return nil, err
	}

	return &lockedDecision{adjustment: adjustment, invoice: locked}, nil
}

// withdrawVoidRequest clears the reason and disposition a void recorded on a
// posted invoice before submitting its reversal, once that reversal is
// rejected: the invoice was never voided and must not read as if it were.
func (s *Service) withdrawVoidRequest(
	ctx context.Context,
	adjustment *invoiceadjustment.InvoiceAdjustment,
	entity *invoice.Invoice,
) (*invoice.Invoice, error) {
	if adjustment.Kind != invoiceadjustment.KindFullReversal ||
		entity == nil ||
		entity.Status != invoice.StatusPosted ||
		(entity.VoidReason == "" && entity.VoidDisposition == "") {
		return nil, nil //nolint:nilnil // only a voided full reversal has a void outcome
	}

	entity.VoidReason = ""
	entity.VoidDisposition = ""
	return s.invoiceRepo.Update(ctx, entity)
}

func (s *Service) GetDetail(
	ctx context.Context,
	req *invoiceadjustmentports.GetInvoiceAdjustmentDetailRequest,
) (*invoiceadjustment.InvoiceAdjustment, error) {
	entity, err := s.repo.GetByID(ctx, invoiceadjustmentports.GetInvoiceAdjustmentRequest{
		ID:         req.AdjustmentID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	adjustmentDocs, err := s.documentRepo.GetByResourceID(
		ctx,
		&documentports.GetDocumentsByResourceRequest{
			TenantInfo:          req.TenantInfo,
			ResourceID:          req.AdjustmentID.String(),
			ResourceType:        adjustmentDocumentResourceType,
			IncludeDocumentType: true,
		},
	)
	if err == nil {
		entity.AdjustmentDocuments = adjustmentDocs
	}

	return entity, nil
}

func (s *Service) GetLineage(
	ctx context.Context,
	req *invoiceadjustmentports.GetInvoiceAdjustmentLineageRequest,
) (*invoiceadjustmentports.InvoiceAdjustmentLineage, error) {
	lineage, err := s.repo.GetLineage(ctx, req.CorrectionGroupID, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	return &invoiceadjustmentports.InvoiceAdjustmentLineage{
		CorrectionGroup: lineage.CorrectionGroup,
		Invoices:        lineage.Invoices,
		Adjustments:     lineage.Adjustments,
	}, nil
}

func (s *Service) GetBatch(
	ctx context.Context,
	batchID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*invoiceadjustment.InvoiceAdjustmentBatch, error) {
	return s.repo.GetBatchByID(ctx, invoiceadjustmentports.GetBatchRequest{
		ID:         batchID,
		TenantInfo: tenantInfo,
	})
}

func (s *Service) ListApprovals(
	ctx context.Context,
	req *invoiceadjustmentports.ListApprovalQueueRequest,
) (*pagination.CursorListResult[*invoiceadjustmentports.InvoiceAdjustmentApprovalQueueItem], error) {
	return s.repo.ListApprovalQueue(ctx, req)
}

func (s *Service) ListReconciliationExceptions(
	ctx context.Context,
	filter pagination.QueryOptions, //nolint:gocritic // stable API shape
) (*pagination.ListResult[*invoiceadjustmentports.InvoiceAdjustmentReconciliationQueueItem], error) {
	return s.repo.ListReconciliationQueue(
		ctx,
		invoiceadjustmentports.ListReconciliationQueueRequest{Filter: filter},
	)
}

func (s *Service) ListBatches(
	ctx context.Context,
	filter pagination.QueryOptions, //nolint:gocritic // stable API shape
) (*pagination.ListResult[*invoiceadjustment.InvoiceAdjustmentBatch], error) {
	return s.repo.ListBatchQueue(ctx, invoiceadjustmentports.ListBatchQueueRequest{Filter: filter})
}

func (s *Service) GetOperationsSummary(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*invoiceadjustmentports.InvoiceAdjustmentOperationsSummary, error) {
	return s.repo.GetOperationsSummary(ctx, tenantInfo)
}
