package invoiceadjustmentservice

import (
	"context"

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

func (s *Service) Reject(
	ctx context.Context,
	req *invoiceadjustmentports.RejectInvoiceAdjustmentRequest,
	actor *permissionports.RequestActor,
) (*invoiceadjustment.InvoiceAdjustment, error) {
	entity, err := s.repo.GetByID(ctx, invoiceadjustmentports.GetInvoiceAdjustmentRequest{
		ID:         req.AdjustmentID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if entity.Status != invoiceadjustment.StatusPendingApproval {
		return nil, errortypes.NewValidationError(
			"adjustmentId",
			errortypes.ErrInvalidOperation,
			"Only pending approval adjustments may be rejected",
		)
	}

	now := timeutils.NowUnix()
	entity.Status = invoiceadjustment.StatusRejected
	entity.ApprovalStatus = invoiceadjustment.ApprovalStatusRejected
	entity.RejectedByID = actor.UserID
	entity.RejectedAt = &now
	entity.RejectionReason = req.Reason

	updated, err := s.repo.UpdateAdjustment(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.logAdjustmentEvent("invoice adjustment rejected", updated, zap.InfoLevel)
	s.logAudit(updated, actor, permission.OpReject, "Invoice adjustment rejected")
	return updated, nil
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
