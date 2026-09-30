package invoiceadjustmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func (s *Service) CreateDraft(
	ctx context.Context,
	req *servicesports.CreateDraftInvoiceAdjustmentRequest,
	actor *servicesports.RequestActor,
) (*invoiceadjustment.InvoiceAdjustment, error) {
	entity, err := s.invoiceRepo.GetByID(ctx, repositories.GetInvoiceByIDRequest{
		ID: req.InvoiceID,
		TenantInfo: pagination.TenantInfo{
			OrgID: req.TenantInfo.OrgID,
			BuID:  req.TenantInfo.BuID,
		},
	})
	if err != nil {
		return nil, err
	}

	group, err := s.ensureCorrectionGroup(ctx, entity)
	if err != nil {
		return nil, err
	}
	if entity.CorrectionGroupID.IsNil() {
		entity.CorrectionGroupID = group.ID
		if _, err = s.invoiceRepo.Update(ctx, entity); err != nil {
			return nil, err
		}
	}

	draft := &invoiceadjustment.InvoiceAdjustment{
		ID:                      pulid.MustNew("iadj_"),
		OrganizationID:          req.TenantInfo.OrgID,
		BusinessUnitID:          req.TenantInfo.BuID,
		CorrectionGroupID:       group.ID,
		OriginalInvoiceID:       entity.ID,
		Kind:                    invoiceadjustment.KindCreditOnly,
		Status:                  invoiceadjustment.StatusDraft,
		ApprovalStatus:          invoiceadjustment.ApprovalStatusNotRequired,
		ReplacementReviewStatus: invoiceadjustment.ReplacementReviewStatusNotRequired,
		RebillStrategy:          invoiceadjustment.RebillStrategyCloneExact,
		IdempotencyKey:          pulid.MustNew("iadjkey_").String(),
		AccountingDate:          timeutils.NowUnix(),
		Metadata: map[string]any{
			"draft": true,
		},
	}

	lines := s.buildDraftLines(entity, draft.ID, req.TenantInfo)
	if err = s.repo.CreateAdjustmentArtifacts(ctx, repositories.CreateAdjustmentArtifactsParams{
		Adjustment: draft,
		Lines:      lines,
	}); err != nil {
		return nil, err
	}

	created, err := s.GetDetail(ctx, &servicesports.GetInvoiceAdjustmentDetailRequest{
		AdjustmentID: draft.ID,
		TenantInfo:   req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	s.logAdjustmentEvent("invoice adjustment draft created", created, zap.InfoLevel)
	s.logAudit(created, actor, permission.OpCreate, "Invoice adjustment draft created")

	return created, nil
}

func (s *Service) UpdateDraft(
	ctx context.Context,
	req *servicesports.UpdateDraftInvoiceAdjustmentRequest,
	actor *servicesports.RequestActor,
) (*invoiceadjustment.InvoiceAdjustment, error) {
	entity, err := s.repo.GetByID(ctx, repositories.GetInvoiceAdjustmentRequest{
		ID:         req.AdjustmentID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if entity.Status != invoiceadjustment.StatusDraft {
		return nil, errortypes.NewValidationError(
			"adjustmentId",
			errortypes.ErrInvalidOperation,
			"Only draft adjustments may be updated",
		)
	}

	sourceInvoice, err := s.invoiceRepo.GetByID(ctx, repositories.GetInvoiceByIDRequest{
		ID:         entity.OriginalInvoiceID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	references, err := s.buildDocumentReferences(
		ctx,
		entity.ID,
		req.ReferencedDocumentIDs,
		sourceInvoice,
		req.TenantInfo,
		actor,
		"referencedDocumentIds",
	)
	if err != nil {
		return nil, err
	}

	entity.Kind = req.Kind
	entity.RebillStrategy = req.RebillStrategy
	entity.Reason = req.Reason

	lines := s.buildAdjustmentLines(entity.ID, sourceInvoice.ID, req.Lines, req.TenantInfo)

	if err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		if _, txErr := s.repo.UpdateAdjustment(txCtx, entity); txErr != nil {
			return txErr
		}
		if txErr := s.repo.ReplaceAdjustmentLines(txCtx, repositories.ReplaceAdjustmentLinesRequest{
			AdjustmentID: entity.ID,
			TenantInfo:   req.TenantInfo,
			Lines:        lines,
		}); txErr != nil {
			return txErr
		}
		return s.repo.ReplaceDocumentReferences(
			txCtx,
			repositories.ReplaceDocumentReferencesRequest{
				AdjustmentID: entity.ID,
				TenantInfo:   req.TenantInfo,
				References:   references,
			},
		)
	}); err != nil {
		return nil, err
	}

	return s.GetDetail(ctx, &servicesports.GetInvoiceAdjustmentDetailRequest{
		AdjustmentID: entity.ID,
		TenantInfo:   req.TenantInfo,
	})
}

func (s *Service) PreviewDraft(
	ctx context.Context,
	req *servicesports.GetInvoiceAdjustmentDetailRequest,
	_ *servicesports.RequestActor,
) (*servicesports.InvoiceAdjustmentPreview, error) {
	entity, draftReq, err := s.loadDraftRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	if entity.Status != invoiceadjustment.StatusDraft {
		return nil, errortypes.NewValidationError(
			"adjustmentId",
			errortypes.ErrInvalidOperation,
			"Only draft adjustments may be previewed",
		)
	}

	return s.Preview(ctx, draftReq, nil)
}

func (s *Service) loadDraftRequest(
	ctx context.Context,
	req *servicesports.GetInvoiceAdjustmentDetailRequest,
) (*invoiceadjustment.InvoiceAdjustment, *servicesports.InvoiceAdjustmentRequest, error) {
	entity, err := s.repo.GetByID(ctx, repositories.GetInvoiceAdjustmentRequest{
		ID:         req.AdjustmentID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, nil, err
	}

	lineItems := make([]*servicesports.InvoiceAdjustmentLineInput, 0, len(entity.Lines))
	for _, line := range entity.Lines {
		lineItems = append(lineItems, &servicesports.InvoiceAdjustmentLineInput{
			OriginalLineID:     line.OriginalLineID,
			CreditQuantity:     line.CreditQuantity,
			CreditAmount:       line.CreditAmount,
			RebillQuantity:     line.RebillQuantity,
			RebillAmount:       line.RebillAmount,
			Description:        line.Description,
			ReplacementPayload: line.ReplacementPayload,
		})
	}

	attachmentIDs := make([]pulid.ID, 0, len(entity.DocumentReferences))
	for _, reference := range entity.DocumentReferences {
		attachmentIDs = append(attachmentIDs, reference.DocumentID)
	}

	return entity, &servicesports.InvoiceAdjustmentRequest{
		AdjustmentID:   entity.ID,
		InvoiceID:      entity.OriginalInvoiceID,
		Kind:           entity.Kind,
		RebillStrategy: entity.RebillStrategy,
		Reason:         entity.Reason,
		IdempotencyKey: entity.IdempotencyKey,
		AttachmentIDs:  attachmentIDs,
		Lines:          lineItems,
		TenantInfo:     req.TenantInfo,
	}, nil
}
