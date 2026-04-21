package invoiceadjustmentservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/shipmentcommercial"
	"github.com/emoss08/trenova/internal/core/temporaljobs/invoiceadjustmentjobs"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	batchInlineThreshold           = 25
	adjustmentDocumentResourceType = "invoice_adjustment"
)

type Params struct {
	fx.In

	Logger             *zap.Logger
	DB                 ports.DBConnection
	Repo               repositories.InvoiceAdjustmentRepository
	InvoiceRepo        repositories.InvoiceRepository
	CustomerRepo       repositories.CustomerRepository
	BillingQueueRepo   repositories.BillingQueueRepository
	ShipmentRepo       repositories.ShipmentRepository
	ShipmentCtrlRepo   repositories.ShipmentControlRepository
	BillingCtrlRepo    repositories.BillingControlRepository
	AdjustmentCtrlRepo repositories.InvoiceAdjustmentControlRepository
	AccountingRepo     repositories.AccountingControlRepository
	JournalRepo        repositories.JournalPostingRepository
	FiscalPeriodRepo   repositories.FiscalPeriodRepository
	DocumentRepo       repositories.DocumentRepository
	Validator          *Validator
	AuditService       servicesports.AuditService
	WorkflowStarter    servicesports.WorkflowStarter
	Commercial         *shipmentcommercial.Calculator
	Generator          servicesports.InvoiceAdjustGenerator
	SequenceGenerator  seqgen.Generator
}

type Service struct {
	l                  *zap.Logger
	db                 ports.DBConnection
	repo               repositories.InvoiceAdjustmentRepository
	invoiceRepo        repositories.InvoiceRepository
	customerRepo       repositories.CustomerRepository
	billingQueueRepo   repositories.BillingQueueRepository
	shipmentRepo       repositories.ShipmentRepository
	shipmentCtrlRepo   repositories.ShipmentControlRepository
	billingCtrlRepo    repositories.BillingControlRepository
	adjustmentCtrlRepo repositories.InvoiceAdjustmentControlRepository
	accountingRepo     repositories.AccountingControlRepository
	journalRepo        repositories.JournalPostingRepository
	fiscalPeriodRepo   repositories.FiscalPeriodRepository
	documentRepo       repositories.DocumentRepository
	validator          *Validator
	auditService       servicesports.AuditService
	workflowStarter    servicesports.WorkflowStarter
	commercial         *shipmentcommercial.Calculator
	generator          servicesports.InvoiceAdjustGenerator
	sequenceGenerator  seqgen.Generator
}

type previewComputation struct {
	invoice           *invoice.Invoice
	correctionGroupID pulid.ID
	control           *tenant.InvoiceAdjustmentControl
	accountingControl *tenant.AccountingControl
	preview           *servicesports.InvoiceAdjustmentPreview
	lines             []*invoiceadjustment.InvoiceAdjustmentLine
	creditLineItems   []*invoice.InoviceLine
	replacementLines  []*invoice.InoviceLine
}

type supportingDocumentRequirementResolution struct {
	CustomerPolicy customer.InvoiceAdjustmentSupportingDocumentPolicy
	Required       bool
	Source         invoiceadjustment.SupportingDocumentPolicySource
}

func New(p Params) servicesports.InvoiceAdjustmentService {
	return &Service{
		l:                  p.Logger.Named("service.invoice-adjustment"),
		db:                 p.DB,
		repo:               p.Repo,
		invoiceRepo:        p.InvoiceRepo,
		customerRepo:       p.CustomerRepo,
		billingQueueRepo:   p.BillingQueueRepo,
		shipmentRepo:       p.ShipmentRepo,
		shipmentCtrlRepo:   p.ShipmentCtrlRepo,
		billingCtrlRepo:    p.BillingCtrlRepo,
		adjustmentCtrlRepo: p.AdjustmentCtrlRepo,
		accountingRepo:     p.AccountingRepo,
		journalRepo:        p.JournalRepo,
		fiscalPeriodRepo:   p.FiscalPeriodRepo,
		documentRepo:       p.DocumentRepo,
		validator:          p.Validator,
		auditService:       p.AuditService,
		workflowStarter:    p.WorkflowStarter,
		commercial:         p.Commercial,
		generator:          p.Generator,
		sequenceGenerator:  p.SequenceGenerator,
	}
}

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
		return s.repo.ReplaceDocumentReferences(txCtx, repositories.ReplaceDocumentReferencesRequest{
			AdjustmentID: entity.ID,
			TenantInfo:   req.TenantInfo,
			References:   references,
		})
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

	computation, err := s.computePreview(ctx, draftReq, true, entity.ID)
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

	if _, err = s.repo.UpdateAdjustment(ctx, entity); err != nil {
		return nil, err
	}
	if err = s.repo.CreateAdjustmentArtifacts(ctx, repositories.CreateAdjustmentArtifactsParams{
		Snapshots: snapshots,
	}); err != nil {
		return nil, err
	}

	if entity.Status == invoiceadjustment.StatusPendingApproval {
		return s.GetDetail(ctx, req)
	}

	return s.executeApprovedAdjustment(ctx, entity.ID, draftReq, actor)
}

func (s *Service) Preview(
	ctx context.Context,
	req *servicesports.InvoiceAdjustmentRequest,
	_ *servicesports.RequestActor,
) (*servicesports.InvoiceAdjustmentPreview, error) {
	if multiErr := s.validator.ValidateRequest(ctx, req); multiErr != nil {
		return nil, multiErr
	}

	computation, err := s.computePreview(ctx, req, false, pulid.ID(""))
	if err != nil {
		return nil, err
	}

	return computation.preview, nil
}

func (s *Service) Submit(
	ctx context.Context,
	req *servicesports.InvoiceAdjustmentRequest,
	actor *servicesports.RequestActor,
) (*invoiceadjustment.InvoiceAdjustment, error) {
	if multiErr := s.validator.ValidateRequest(ctx, req); multiErr != nil {
		return nil, multiErr
	}

	if existing, err := s.repo.GetByIdempotencyKey(ctx, repositories.GetInvoiceAdjustmentByIdempotencyRequest{
		IdempotencyKey: req.IdempotencyKey,
		TenantInfo:     req.TenantInfo,
	}); err == nil &&
		existing != nil {
		return existing, nil
	}

	computation, err := s.computePreview(ctx, req, true, pulid.ID(""))
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

			if err = s.repo.CreateAdjustmentArtifacts(txCtx, repositories.CreateAdjustmentArtifactsParams{
				Adjustment:         adjustment,
				Lines:              computation.lines,
				Snapshots:          snapshots,
				DocumentReferences: references,
			}); err != nil {
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

func (s *Service) Approve(
	ctx context.Context,
	req *servicesports.ApproveInvoiceAdjustmentRequest,
	actor *servicesports.RequestActor,
) (*invoiceadjustment.InvoiceAdjustment, error) {
	var result *invoiceadjustment.InvoiceAdjustment
	err := s.db.WithTx(
		ctx,
		ports.TxOptions{LockTimeout: 5 * 1000000000},
		func(txCtx context.Context, _ bun.Tx) error {
			executed, execErr := s.executeApprovedAdjustment(
				txCtx,
				req.AdjustmentID,
				&servicesports.InvoiceAdjustmentRequest{
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
	req *servicesports.RejectInvoiceAdjustmentRequest,
	actor *servicesports.RequestActor,
) (*invoiceadjustment.InvoiceAdjustment, error) {
	entity, err := s.repo.GetByID(ctx, repositories.GetInvoiceAdjustmentRequest{
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
	req *servicesports.GetInvoiceAdjustmentDetailRequest,
) (*invoiceadjustment.InvoiceAdjustment, error) {
	entity, err := s.repo.GetByID(ctx, repositories.GetInvoiceAdjustmentRequest{
		ID:         req.AdjustmentID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	adjustmentDocs, err := s.documentRepo.GetByResourceID(
		ctx,
		&repositories.GetDocumentsByResourceRequest{
			TenantInfo:          req.TenantInfo,
			ResourceID:          req.AdjustmentID.String(),
			ResourceType:        adjustmentDocumentResourceType,
			IncludeDocumentType: true,
		},
	)
	if err == nil {
		entity.AdjustmentDocuments = adjustmentDocs
	}

	sourceInvoice, err := s.invoiceRepo.GetByID(ctx, repositories.GetInvoiceByIDRequest{
		ID:         entity.OriginalInvoiceID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	control, err := s.adjustmentCtrlRepo.GetByOrgID(ctx, req.TenantInfo.OrgID)
	if err != nil {
		return nil, err
	}

	resolution, err := s.resolveSupportingDocumentRequirement(
		ctx,
		sourceInvoice,
		entity.Kind,
		control,
		req.TenantInfo,
	)
	if err != nil {
		return nil, err
	}
	entity.CustomerSupportingDocumentPolicy = resolution.CustomerPolicy
	entity.SupportingDocumentsRequired = resolution.Required
	entity.SupportingDocumentPolicySource = string(resolution.Source)

	return entity, nil
}

func (s *Service) GetLineage(
	ctx context.Context,
	req *servicesports.GetInvoiceAdjustmentLineageRequest,
) (*servicesports.InvoiceAdjustmentLineage, error) {
	lineage, err := s.repo.GetLineage(ctx, req.CorrectionGroupID, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	return &servicesports.InvoiceAdjustmentLineage{
		CorrectionGroup: lineage.CorrectionGroup,
		Invoices:        lineage.Invoices,
		Adjustments:     lineage.Adjustments,
	}, nil
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

func (s *Service) BulkSubmit(
	ctx context.Context,
	req *servicesports.InvoiceAdjustmentBulkRequest,
	actor *servicesports.RequestActor,
) (*invoiceadjustment.InvoiceAdjustmentBatch, error) {
	if err := validateBulkRequest(req); err != nil {
		return nil, err
	}

	if existing, err := s.repo.GetBatchByIdempotencyKey(ctx, repositories.GetBatchByIdempotencyRequest{
		IdempotencyKey: req.IdempotencyKey,
		TenantInfo:     req.TenantInfo,
	}); err == nil &&
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

func (s *Service) GetBatch(
	ctx context.Context,
	batchID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*invoiceadjustment.InvoiceAdjustmentBatch, error) {
	return s.repo.GetBatchByID(ctx, repositories.GetBatchRequest{
		ID:         batchID,
		TenantInfo: tenantInfo,
	})
}

func (s *Service) ListApprovals(
	ctx context.Context,
	filter pagination.QueryOptions,
) (*pagination.ListResult[*repositories.InvoiceAdjustmentApprovalQueueItem], error) {
	return s.repo.ListApprovalQueue(ctx, repositories.ListApprovalQueueRequest{Filter: filter})
}

func (s *Service) ListReconciliationExceptions(
	ctx context.Context,
	filter pagination.QueryOptions,
) (*pagination.ListResult[*repositories.InvoiceAdjustmentReconciliationQueueItem], error) {
	return s.repo.ListReconciliationQueue(
		ctx,
		repositories.ListReconciliationQueueRequest{Filter: filter},
	)
}

func (s *Service) ListBatches(
	ctx context.Context,
	filter pagination.QueryOptions,
) (*pagination.ListResult[*invoiceadjustment.InvoiceAdjustmentBatch], error) {
	return s.repo.ListBatchQueue(ctx, repositories.ListBatchQueueRequest{Filter: filter})
}

func (s *Service) GetOperationsSummary(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*repositories.InvoiceAdjustmentOperationsSummary, error) {
	return s.repo.GetOperationsSummary(ctx, tenantInfo)
}

func (s *Service) computePreview(
	ctx context.Context,
	req *servicesports.InvoiceAdjustmentRequest,
	enforceAttachments bool,
	excludeAdjustmentID pulid.ID,
) (*previewComputation, error) {
	entity, err := s.invoiceRepo.GetByID(ctx, repositories.GetInvoiceByIDRequest{
		ID:         req.InvoiceID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	control, err := s.adjustmentCtrlRepo.GetByOrgID(ctx, req.TenantInfo.OrgID)
	if err != nil {
		return nil, err
	}

	accountingControl, err := s.accountingRepo.GetByOrgID(ctx, req.TenantInfo.OrgID)
	if err != nil {
		return nil, err
	}

	preview := &servicesports.InvoiceAdjustmentPreview{
		InvoiceID:                        entity.ID,
		Kind:                             req.Kind,
		RebillStrategy:                   req.RebillStrategy,
		CustomerSupportingDocumentPolicy: customer.InvoiceAdjustmentSupportingDocumentPolicyInherit,
		SupportingDocumentPolicySource: string(
			invoiceadjustment.SupportingDocumentPolicySourceOrganizationControl,
		),
		Warnings: make([]string, 0),
		Errors:   make(map[string][]string),
		Lines: make(
			[]*servicesports.InvoiceAdjustmentPreviewLine,
			0,
			len(entity.Lines),
		),
	}

	if entity.Status != invoice.StatusPosted {
		appendPreviewError(preview, "invoiceId", "Only posted invoices may be adjusted")
	}

	attachmentResolution, err := s.resolveSupportingDocumentRequirement(
		ctx,
		entity,
		req.Kind,
		control,
		req.TenantInfo,
	)
	if err != nil {
		return nil, err
	}
	preview.CustomerSupportingDocumentPolicy = attachmentResolution.CustomerPolicy
	preview.SupportingDocumentsRequired = attachmentResolution.Required
	preview.SupportingDocumentPolicySource = string(attachmentResolution.Source)

	preview.AccountingDate = s.resolveAccountingDate(ctx, entity, control, preview)
	s.validateSettlementPolicy(entity, control, preview)
	if strings.TrimSpace(req.Reason) == "" &&
		control.AdjustmentReasonRequirement == tenant.RequirementPolicyRequired {
		appendPreviewError(preview, "reason", "Adjustment reason is required by policy")
	}
	if enforceAttachments {
		s.validateAttachments(ctx, req, attachmentResolution.Required, preview)
	}

	switch {
	case entity.CorrectionGroupID.IsNotNil():
		preview.CorrectionGroupID = entity.CorrectionGroupID
	case true:
		if group, groupErr := s.repo.GetCorrectionGroupByRootInvoice(ctx, repositories.GetCorrectionGroupByRootInvoiceRequest{
			RootInvoiceID: entity.ID,
			TenantInfo:    req.TenantInfo,
		}); groupErr == nil &&
			group != nil {
			preview.CorrectionGroupID = group.ID
		} else {
			preview.CorrectionGroupID = pulid.MustNew("icg_")
		}
	}

	usageByLine, err := s.repo.GetInvoiceLineCreditUsage(
		ctx,
		repositories.GetInvoiceLineCreditUsageRequest{
			InvoiceID:           entity.ID,
			TenantInfo:          req.TenantInfo,
			ExcludeAdjustmentID: excludeAdjustmentID,
		},
	)
	if err != nil {
		return nil, err
	}

	requestedByLine := make(map[string]*servicesports.InvoiceAdjustmentLineInput, len(req.Lines))
	for _, line := range req.Lines {
		if line == nil {
			continue
		}
		requestedByLine[line.OriginalLineID.String()] = line
	}

	lines := make([]*invoiceadjustment.InvoiceAdjustmentLine, 0, len(entity.Lines))
	creditLines := make([]*invoice.InoviceLine, 0, len(entity.Lines))
	replacementLines := make([]*invoice.InoviceLine, 0, len(entity.Lines))
	fullScope := len(req.Lines) == 0

	for _, sourceLine := range entity.Lines {
		if sourceLine == nil {
			continue
		}
		input, hasInput := requestedByLine[sourceLine.ID.String()]
		if !fullScope && !hasInput {
			continue
		}

		used := usageByLine[sourceLine.ID.String()]
		eligible := sourceLine.Amount.Abs().Sub(used)
		if eligible.IsNegative() {
			eligible = decimal.Zero
		}

		creditAmount := eligible
		creditQuantity := sourceLine.Quantity
		rebillAmount := decimal.Zero
		rebillQuantity := decimal.Zero
		description := sourceLine.Description
		payload := map[string]any{}

		if hasInput {
			if !input.CreditAmount.IsZero() {
				creditAmount = input.CreditAmount
			}
			if !input.CreditQuantity.IsZero() {
				creditQuantity = input.CreditQuantity
			}
			if !input.RebillAmount.IsZero() {
				rebillAmount = input.RebillAmount
			}
			if !input.RebillQuantity.IsZero() {
				rebillQuantity = input.RebillQuantity
			}
			if strings.TrimSpace(input.Description) != "" {
				description = input.Description
			}
			payload = input.ReplacementPayload
		}

		if req.Kind == invoiceadjustment.KindCreditRebill && fullScope &&
			req.RebillStrategy != invoiceadjustment.RebillStrategyRerate {
			rebillAmount = sourceLine.Amount.Abs()
			rebillQuantity = sourceLine.Quantity
		}

		remainingEligibleAmount := eligible.Sub(creditAmount)
		hasEligibilityError := creditAmount.GreaterThan(eligible)
		eligibilityOverageAmount := decimal.Zero
		eligibilityMessage := ""
		if hasEligibilityError {
			eligibilityOverageAmount = creditAmount.Sub(eligible)
			eligibilityMessage = fmt.Sprintf(
				"Line %d exceeds the remaining eligible amount by %s",
				sourceLine.LineNumber,
				eligibilityOverageAmount.StringFixed(4),
			)
			appendPreviewError(preview, "lines", eligibilityMessage)
		}

		linePreview := &servicesports.InvoiceAdjustmentPreviewLine{
			LineNumber:               sourceLine.LineNumber,
			OriginalLineID:           sourceLine.ID,
			Description:              sourceLine.Description,
			EligibleAmount:           eligible,
			AlreadyCreditedAmount:    used,
			RequestedCreditAmount:    creditAmount,
			RequestedRebillAmount:    rebillAmount,
			RemainingEligibleAmount:  remainingEligibleAmount,
			HasEligibilityError:      hasEligibilityError,
			EligibilityOverageAmount: eligibilityOverageAmount,
			EligibilityMessage:       eligibilityMessage,
		}
		preview.Lines = append(preview.Lines, linePreview)

		preview.CreditTotalAmount = preview.CreditTotalAmount.Add(creditAmount)
		preview.RebillTotalAmount = preview.RebillTotalAmount.Add(rebillAmount)

		lines = append(lines, &invoiceadjustment.InvoiceAdjustmentLine{
			OriginalInvoiceID:       entity.ID,
			OriginalLineID:          sourceLine.ID,
			LineNumber:              sourceLine.LineNumber,
			Description:             description,
			CreditQuantity:          creditQuantity,
			CreditAmount:            creditAmount,
			RemainingEligibleAmount: linePreview.RemainingEligibleAmount,
			RebillQuantity:          rebillQuantity,
			RebillAmount:            rebillAmount,
			ReplacementPayload:      payload,
		})

		if creditAmount.GreaterThan(decimal.Zero) {
			unitPrice := creditAmount
			if creditQuantity.GreaterThan(decimal.Zero) {
				unitPrice = creditAmount.Div(creditQuantity)
			}
			creditLines = append(creditLines, &invoice.InoviceLine{
				LineNumber:  sourceLine.LineNumber,
				Type:        sourceLine.Type,
				Description: description,
				Quantity:    creditQuantity,
				UnitPrice:   unitPrice.Neg(),
				Amount:      creditAmount.Neg(),
			})
		}

		if rebillAmount.GreaterThan(decimal.Zero) {
			unitPrice := rebillAmount
			if rebillQuantity.GreaterThan(decimal.Zero) {
				unitPrice = rebillAmount.Div(rebillQuantity)
			}
			replacementLines = append(replacementLines, &invoice.InoviceLine{
				LineNumber:  len(replacementLines) + 1,
				Type:        sourceLine.Type,
				Description: description,
				Quantity:    maxDecimal(rebillQuantity, decimal.NewFromInt(1)),
				UnitPrice:   unitPrice,
				Amount:      rebillAmount,
			})
		}
	}

	if req.Kind == invoiceadjustment.KindCreditRebill &&
		req.RebillStrategy == invoiceadjustment.RebillStrategyRerate {
		rerateLines, rerateTotal, rerateVariance, rerateErr := s.computeRerate(
			ctx,
			entity,
			req.TenantInfo,
		)
		if rerateErr != nil {
			appendPreviewError(preview, "rebillStrategy", rerateErr.Error())
		} else {
			replacementLines = rerateLines
			preview.RebillTotalAmount = rerateTotal
			preview.RerateVariancePercent = rerateVariance
		}
	}

	preview.NetDeltaAmount = preview.RebillTotalAmount.Sub(preview.CreditTotalAmount)
	preview.WouldCreateUnappliedCredit = preview.CreditTotalAmount.GreaterThan(
		entity.OpenBalanceAmount(),
	)
	if req.Kind == invoiceadjustment.KindWriteOff &&
		preview.CreditTotalAmount.GreaterThan(entity.OpenBalanceAmount()) {
		appendPreviewError(
			preview,
			"creditTotalAmount",
			"Write-off amount cannot exceed the invoice open balance",
		)
	}
	s.applyCreditBalancePolicy(preview, entity, control)
	s.applyApprovalPolicy(preview, entity, control)
	s.applyReplacementReviewPolicy(preview, control)
	if preview.CreditTotalAmount.IsZero() {
		appendPreviewError(preview, "lines", "Adjustment must produce a non-zero credit amount")
	}
	if req.Kind == invoiceadjustment.KindWriteOff && entity.OpenBalanceAmount().IsZero() {
		appendPreviewError(preview, "invoiceId", "Write-offs require a remaining invoice balance")
	}
	if preview.RequiresReconciliationException {
		preview.Warnings = append(
			preview.Warnings,
			"Execution will create a reconciliation exception for finance follow-up",
		)
	}

	return &previewComputation{
		invoice:           entity,
		correctionGroupID: preview.CorrectionGroupID,
		control:           control,
		accountingControl: accountingControl,
		preview:           preview,
		lines:             lines,
		creditLineItems:   creditLines,
		replacementLines:  replacementLines,
	}, nil
}

func (s *Service) resolveAccountingDate(
	ctx context.Context,
	entity *invoice.Invoice,
	control *tenant.InvoiceAdjustmentControl,
	preview *servicesports.InvoiceAdjustmentPreview,
) int64 {
	originalDate := entity.InvoiceDate
	if control.AdjustmentAccountingDatePolicy == tenant.AdjustmentAccountingDateAlwaysNextOpen {
		nextOpen := s.resolveNextOpenPeriodDate(
			ctx,
			entity.OrganizationID,
			entity.BusinessUnitID,
			originalDate,
		)
		if nextOpen > 0 {
			return nextOpen
		}

		return timeutils.NowUnix()
	}

	period, err := s.fiscalPeriodRepo.GetPeriodByDate(ctx, repositories.GetPeriodByDateRequest{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
		Date:  originalDate,
	})
	if err == nil && period != nil && period.Status == fiscalperiod.StatusOpen {
		return originalDate
	}

	switch control.ClosedPeriodAdjustmentPolicy {
	case tenant.ClosedPeriodAdjustmentPolicyDisallow:
		appendPreviewError(
			preview,
			"accountingDate",
			"Closed-period adjustments are disallowed by policy",
		)
	case tenant.ClosedPeriodAdjustmentPolicyRequireReopen:
		appendPreviewError(
			preview,
			"accountingDate",
			"The accounting period must be reopened before this adjustment can be posted",
		)
	case tenant.ClosedPeriodAdjustmentPolicyPostInNextOpenPeriodWithApproval:
		preview.RequiresApproval = true
		preview.Warnings = append(
			preview.Warnings,
			"Adjustment will post in the next open period and requires approval",
		)
	}

	nextOpen := s.resolveNextOpenPeriodDate(
		ctx,
		entity.OrganizationID,
		entity.BusinessUnitID,
		originalDate,
	)
	if nextOpen > 0 {
		return nextOpen
	}

	return timeutils.NowUnix()
}

func (s *Service) resolveNextOpenPeriodDate(
	ctx context.Context,
	orgID, buID pulid.ID,
	fallback int64,
) int64 {
	period, err := s.fiscalPeriodRepo.GetPeriodByDate(ctx, repositories.GetPeriodByDateRequest{
		OrgID: orgID,
		BuID:  buID,
		Date:  timeutils.NowUnix(),
	})
	if err == nil && period != nil && period.Status == fiscalperiod.StatusOpen {
		return maxInt64(period.StartDate, fallback)
	}
	return timeutils.NowUnix()
}

func (s *Service) validateSettlementPolicy(
	entity *invoice.Invoice,
	control *tenant.InvoiceAdjustmentControl,
	preview *servicesports.InvoiceAdjustmentPreview,
) {
	switch entity.SettlementStatus {
	case invoice.SettlementStatusPaid:
		switch control.PaidInvoiceAdjustmentPolicy {
		case tenant.AdjustmentEligibilityDisallow:
			appendPreviewError(preview, "invoiceId", "Paid invoices cannot be adjusted by policy")
		case tenant.AdjustmentEligibilityAllowWithApproval:
			preview.RequiresApproval = true
		}
	case invoice.SettlementStatusPartiallyPaid:
		switch control.PartiallyPaidInvoiceAdjustmentPolicy {
		case tenant.AdjustmentEligibilityDisallow:
			appendPreviewError(
				preview,
				"invoiceId",
				"Partially paid invoices cannot be adjusted by policy",
			)
		case tenant.AdjustmentEligibilityAllowWithApproval:
			preview.RequiresApproval = true
		}
	}

	if entity.DisputeStatus == invoice.DisputeStatusDisputed {
		switch control.DisputedInvoiceAdjustmentPolicy {
		case tenant.AdjustmentEligibilityDisallow:
			appendPreviewError(
				preview,
				"invoiceId",
				"Disputed invoices cannot be adjusted by policy",
			)
		case tenant.AdjustmentEligibilityAllowWithApproval:
			preview.RequiresApproval = true
		}
	}

	if entity.SettlementStatus != invoice.SettlementStatusUnpaid {
		preview.RequiresReconciliationException = true
	}
}

func (s *Service) validateAttachments(
	ctx context.Context,
	req *servicesports.InvoiceAdjustmentRequest,
	required bool,
	preview *servicesports.InvoiceAdjustmentPreview,
) {
	attachmentCount := len(req.AttachmentIDs)
	if req.AdjustmentID.IsNotNil() {
		if docs, err := s.documentRepo.GetByResourceID(ctx, &repositories.GetDocumentsByResourceRequest{
			TenantInfo:          req.TenantInfo,
			ResourceID:          req.AdjustmentID.String(),
			ResourceType:        adjustmentDocumentResourceType,
			IncludeDocumentType: false,
		}); err == nil {
			for _, doc := range docs {
				if doc != nil && doc.Status != document.StatusArchived {
					attachmentCount++
				}
			}
		}
	}
	fieldName := supportingDocumentFieldName(req.AdjustmentID)
	if required && attachmentCount == 0 {
		appendPreviewError(
			preview,
			fieldName,
			"Supporting documents are required for this adjustment by policy",
		)
		return
	}
	if len(req.AttachmentIDs) == 0 {
		return
	}

	docs, err := s.documentRepo.GetByIDs(ctx, repositories.BulkDeleteDocumentRequest{
		IDs:        req.AttachmentIDs,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		appendPreviewError(preview, fieldName, "Failed to validate supporting documents")
		return
	}
	if len(docs) != len(req.AttachmentIDs) {
		appendPreviewError(preview, fieldName, "One or more supporting documents are invalid")
		return
	}
	for _, doc := range docs {
		if doc == nil || doc.Status == document.StatusArchived {
			appendPreviewError(
				preview,
				fieldName,
				"Archived supporting documents cannot be used for adjustments",
			)
			return
		}
	}
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

func (s *Service) buildDraftLines(
	sourceInvoice *invoice.Invoice,
	adjustmentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) []*invoiceadjustment.InvoiceAdjustmentLine {
	lines := make([]*invoiceadjustment.InvoiceAdjustmentLine, 0, len(sourceInvoice.Lines))
	for _, line := range sourceInvoice.Lines {
		lines = append(lines, &invoiceadjustment.InvoiceAdjustmentLine{
			OrganizationID:          tenantInfo.OrgID,
			BusinessUnitID:          tenantInfo.BuID,
			AdjustmentID:            adjustmentID,
			OriginalInvoiceID:       sourceInvoice.ID,
			OriginalLineID:          line.ID,
			LineNumber:              line.LineNumber,
			Description:             line.Description,
			CreditQuantity:          line.Quantity,
			CreditAmount:            line.Amount.Abs(),
			RemainingEligibleAmount: line.Amount.Abs(),
			RebillQuantity:          line.Quantity,
			RebillAmount:            line.Amount.Abs(),
			ReplacementPayload:      map[string]any{},
		})
	}
	return lines
}

func (s *Service) buildAdjustmentLines(
	adjustmentID pulid.ID,
	invoiceID pulid.ID,
	inputs []*servicesports.InvoiceAdjustmentLineInput,
	tenantInfo pagination.TenantInfo,
) []*invoiceadjustment.InvoiceAdjustmentLine {
	lines := make([]*invoiceadjustment.InvoiceAdjustmentLine, 0, len(inputs))
	for idx, line := range inputs {
		if line == nil {
			continue
		}
		lines = append(lines, &invoiceadjustment.InvoiceAdjustmentLine{
			OrganizationID:          tenantInfo.OrgID,
			BusinessUnitID:          tenantInfo.BuID,
			AdjustmentID:            adjustmentID,
			OriginalInvoiceID:       invoiceID,
			OriginalLineID:          line.OriginalLineID,
			LineNumber:              idx + 1,
			Description:             line.Description,
			CreditQuantity:          line.CreditQuantity,
			CreditAmount:            line.CreditAmount,
			RemainingEligibleAmount: line.CreditAmount,
			RebillQuantity:          line.RebillQuantity,
			RebillAmount:            line.RebillAmount,
			ReplacementPayload:      line.ReplacementPayload,
		})
	}
	return lines
}

func (s *Service) buildDocumentReferences(
	ctx context.Context,
	adjustmentID pulid.ID,
	documentIDs []pulid.ID,
	sourceInvoice *invoice.Invoice,
	tenantInfo pagination.TenantInfo,
	actor *servicesports.RequestActor,
	fieldName string,
) ([]*invoiceadjustment.InvoiceAdjustmentDocumentReference, error) {
	if len(documentIDs) == 0 {
		return nil, nil
	}

	docs, err := s.documentRepo.GetByIDs(ctx, repositories.BulkDeleteDocumentRequest{
		IDs:        documentIDs,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, errortypes.NewValidationError(
			fieldName,
			errortypes.ErrInvalidOperation,
			"One or more supporting documents are invalid",
		)
	}
	if len(docs) != len(documentIDs) {
		return nil, errortypes.NewValidationError(
			fieldName,
			errortypes.ErrInvalidOperation,
			"One or more supporting documents are invalid",
		)
	}

	now := timeutils.NowUnix()
	refs := make([]*invoiceadjustment.InvoiceAdjustmentDocumentReference, 0, len(docs))
	for _, doc := range docs {
		if doc == nil || doc.Status == document.StatusArchived {
			return nil, errortypes.NewValidationError(
				fieldName,
				errortypes.ErrInvalidOperation,
				"Archived supporting documents cannot be used for adjustments",
			)
		}
		if !strings.EqualFold(doc.ResourceType, "shipment") ||
			doc.ResourceID != sourceInvoice.ShipmentID.String() {
			return nil, errortypes.NewValidationError(
				fieldName,
				errortypes.ErrInvalidOperation,
				"Supporting evidence must reference documents from the source shipment",
			)
		}

		refs = append(refs, &invoiceadjustment.InvoiceAdjustmentDocumentReference{
			OrganizationID:       tenantInfo.OrgID,
			BusinessUnitID:       tenantInfo.BuID,
			AdjustmentID:         adjustmentID,
			DocumentID:           doc.ID,
			SelectedByID:         actor.UserID,
			SelectedAt:           &now,
			SnapshotFileName:     doc.FileName,
			SnapshotOriginalName: doc.OriginalName,
			SnapshotFileType:     doc.FileType,
			SnapshotResourceType: doc.ResourceType,
			SnapshotResourceID:   doc.ResourceID,
		})
	}

	return refs, nil
}

func supportingDocumentFieldName(adjustmentID pulid.ID) string {
	if adjustmentID.IsNotNil() {
		return "referencedDocumentIds"
	}
	return "attachmentIds"
}

func (s *Service) resolveSupportingDocumentRequirement(
	ctx context.Context,
	sourceInvoice *invoice.Invoice,
	kind invoiceadjustment.Kind,
	control *tenant.InvoiceAdjustmentControl,
	tenantInfo pagination.TenantInfo,
) (*supportingDocumentRequirementResolution, error) {
	_ = kind
	_ = control

	resolution := &supportingDocumentRequirementResolution{
		CustomerPolicy: customer.InvoiceAdjustmentSupportingDocumentPolicyInherit,
		Required:       false,
		Source:         invoiceadjustment.SupportingDocumentPolicySourceDefaultOptional,
	}

	entity, err := s.customerRepo.GetByID(ctx, repositories.GetCustomerByIDRequest{
		ID:         sourceInvoice.CustomerID,
		TenantInfo: tenantInfo,
		CustomerFilterOptions: repositories.CustomerFilterOptions{
			IncludeBillingProfile: true,
		},
	})
	if err != nil {
		return nil, err
	}

	if entity.BillingProfile == nil {
		return resolution, nil
	}

	policy := entity.BillingProfile.InvoiceAdjustmentSupportingDocumentPolicy
	if policy == "" {
		policy = customer.InvoiceAdjustmentSupportingDocumentPolicyInherit
	}
	resolution.CustomerPolicy = policy

	switch policy {
	case customer.InvoiceAdjustmentSupportingDocumentPolicyRequired:
		resolution.Required = true
		resolution.Source = invoiceadjustment.SupportingDocumentPolicySourceCustomerBillingProfile
	case customer.InvoiceAdjustmentSupportingDocumentPolicyOptional:
		resolution.Required = false
		resolution.Source = invoiceadjustment.SupportingDocumentPolicySourceCustomerBillingProfile
	}

	return resolution, nil
}

func (s *Service) applyCreditBalancePolicy(
	preview *servicesports.InvoiceAdjustmentPreview,
	entity *invoice.Invoice,
	control *tenant.InvoiceAdjustmentControl,
) {
	if !preview.WouldCreateUnappliedCredit {
		return
	}
	if preview.Kind == invoiceadjustment.KindWriteOff {
		appendPreviewError(
			preview,
			"creditTotalAmount",
			"Write-offs cannot create unapplied customer credit",
		)
		return
	}

	switch control.CustomerCreditBalancePolicy {
	case tenant.CustomerCreditBalancePolicyDisallow:
		appendPreviewError(
			preview,
			"creditTotalAmount",
			"Policy disallows customer credit balance outcomes",
		)
	case tenant.CustomerCreditBalancePolicyAllowUnappliedCredit:
		switch control.OverCreditPolicy {
		case tenant.OverCreditPolicyBlock:
			appendPreviewError(
				preview,
				"creditTotalAmount",
				"Adjustment would create unapplied customer credit because of payment state",
			)
		case tenant.OverCreditPolicyAllowWithApproval:
			preview.RequiresApproval = true
		}
	}

	if entity.SettlementStatus != invoice.SettlementStatusUnpaid {
		preview.RequiresReconciliationException = true
	}
}

func (s *Service) applyApprovalPolicy(
	preview *servicesports.InvoiceAdjustmentPreview,
	entity *invoice.Invoice,
	control *tenant.InvoiceAdjustmentControl,
) {
	amount := preview.CreditTotalAmount.Abs()
	if preview.Kind == invoiceadjustment.KindWriteOff {
		switch control.WriteOffApprovalPolicy {
		case tenant.WriteOffApprovalPolicyDisallow:
			appendPreviewError(preview, "kind", "Write-offs are disallowed by policy")
		case tenant.WriteOffApprovalPolicyAlwaysRequireApproval:
			preview.RequiresApproval = true
		case tenant.WriteOffApprovalPolicyRequireApprovalAboveThreshold:
			if amount.GreaterThanOrEqual(control.WriteOffApprovalThreshold) {
				preview.RequiresApproval = true
			}
		}
	} else {
		switch control.StandardAdjustmentApprovalPolicy {
		case tenant.ApprovalPolicyAlways:
			preview.RequiresApproval = true
		case tenant.ApprovalPolicyAmountThreshold:
			if amount.GreaterThanOrEqual(control.StandardAdjustmentApprovalThreshold) {
				preview.RequiresApproval = true
			}
		}
	}

	if entity.SettlementStatus != invoice.SettlementStatusUnpaid {
		preview.RequiresReconciliationException = true
	}
}

func (s *Service) applyReplacementReviewPolicy(
	preview *servicesports.InvoiceAdjustmentPreview,
	control *tenant.InvoiceAdjustmentControl,
) {
	if preview.Kind != invoiceadjustment.KindCreditRebill {
		return
	}
	switch control.ReplacementInvoiceReviewPolicy {
	case tenant.ReplacementInvoiceReviewPolicyAlwaysRequireReview:
		preview.RequiresReplacementInvoiceReview = true
	case tenant.ReplacementInvoiceReviewPolicyRequireReviewWhenEconomicTermsChange:
		if preview.RebillStrategy == invoiceadjustment.RebillStrategyRerate {
			preview.RequiresReplacementInvoiceReview = preview.RerateVariancePercent.GreaterThan(
				control.RerateVarianceTolerancePercent,
			)
		} else {
			preview.RequiresReplacementInvoiceReview = !preview.RebillTotalAmount.Equal(preview.CreditTotalAmount)
		}
	}
}

func (s *Service) computeRerate(
	ctx context.Context,
	entity *invoice.Invoice,
	tenantInfo pagination.TenantInfo,
) ([]*invoice.InoviceLine, decimal.Decimal, decimal.Decimal, error) {
	shp, err := s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:         entity.ShipmentID,
		TenantInfo: tenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return nil, decimal.Zero, decimal.Zero, err
	}

	control, err := s.shipmentCtrlRepo.Get(ctx, repositories.GetShipmentControlRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, decimal.Zero, decimal.Zero, err
	}
	if err = s.commercial.Recalculate(ctx, shp, control, tenantInfo.UserID); err != nil {
		return nil, decimal.Zero, decimal.Zero, err
	}

	lines := buildReplacementLinesFromShipment(shp)
	total := shp.TotalChargeAmount.Decimal
	variance := decimal.Zero
	if entity.TotalAmount.GreaterThan(decimal.Zero) && !total.Equal(entity.TotalAmount) {
		variance = total.Sub(entity.TotalAmount).
			Abs().
			Div(entity.TotalAmount).
			Mul(decimal.NewFromInt(100))
	}
	return lines, total, variance, nil
}

func (s *Service) ensureCorrectionGroup(
	ctx context.Context,
	entity *invoice.Invoice,
) (*invoiceadjustment.InvoiceAdjustmentCorrectionGroup, error) {
	rootID := entity.ID
	if entity.CorrectionGroupID.IsNotNil() {
		group, err := s.repo.GetCorrectionGroup(ctx, repositories.GetCorrectionGroupRequest{
			ID: entity.CorrectionGroupID,
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		})
		if err == nil && group != nil {
			return group, nil
		}
	}

	if group, err := s.repo.GetCorrectionGroupByRootInvoice(ctx, repositories.GetCorrectionGroupByRootInvoiceRequest{
		RootInvoiceID: rootID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	}); err == nil &&
		group != nil {
		return group, nil
	}

	return s.repo.CreateCorrectionGroup(ctx, &invoiceadjustment.InvoiceAdjustmentCorrectionGroup{
		OrganizationID:   entity.OrganizationID,
		BusinessUnitID:   entity.BusinessUnitID,
		RootInvoiceID:    rootID,
		CurrentInvoiceID: rootID,
	})
}

func (s *Service) executeApprovedAdjustment(
	ctx context.Context,
	adjustmentID pulid.ID,
	req *servicesports.InvoiceAdjustmentRequest,
	actor *servicesports.RequestActor,
) (*invoiceadjustment.InvoiceAdjustment, error) {
	var tenantInfo pagination.TenantInfo
	if req != nil {
		tenantInfo = req.TenantInfo
	}
	adjustment, err := s.repo.GetByID(ctx, repositories.GetInvoiceAdjustmentRequest{
		ID:         adjustmentID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if tenantInfo.OrgID.IsNil() {
		tenantInfo = pagination.TenantInfo{
			OrgID: adjustment.OrganizationID,
			BuID:  adjustment.BusinessUnitID,
		}
	}
	if adjustment.Status == invoiceadjustment.StatusExecuted {
		return adjustment, nil
	}
	if adjustment.Status == invoiceadjustment.StatusDraft {
		return nil, errortypes.NewValidationError(
			"adjustmentId",
			errortypes.ErrInvalidOperation,
			"Draft adjustments cannot be executed",
		)
	}
	if adjustment.Status == invoiceadjustment.StatusRejected {
		return nil, errortypes.NewValidationError(
			"adjustmentId",
			errortypes.ErrInvalidOperation,
			"Rejected adjustments cannot be executed",
		)
	}

	lockedInvoice, err := s.repo.LockInvoiceForUpdate(
		ctx,
		repositories.LockInvoiceAdjustmentRequest{
			InvoiceID:  adjustment.OriginalInvoiceID,
			TenantInfo: tenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	if req == nil || req.InvoiceID.IsNil() {
		req = &servicesports.InvoiceAdjustmentRequest{
			InvoiceID:      adjustment.OriginalInvoiceID,
			Kind:           adjustment.Kind,
			RebillStrategy: adjustment.RebillStrategy,
			Reason:         adjustment.Reason,
			IdempotencyKey: adjustment.IdempotencyKey,
			TenantInfo:     tenantInfo,
			Lines:          s.requestLinesFromAdjustment(adjustment),
		}
	}
	req.TenantInfo = tenantInfo

	computation, err := s.computePreview(ctx, req, true, adjustment.ID)
	if err != nil {
		return nil, err
	}
	computation.invoice = lockedInvoice
	if len(computation.preview.Errors) > 0 {
		adjustment.Status = invoiceadjustment.StatusExecutionFailed
		adjustment.ExecutionError = previewErrorsToMultiError(computation.preview.Errors).Error()
		s.logAdjustmentEvent(
			"invoice adjustment execution failed revalidation",
			adjustment,
			zap.WarnLevel,
		)
		return s.repo.UpdateAdjustment(ctx, adjustment)
	}

	group, err := s.ensureCorrectionGroup(ctx, lockedInvoice)
	if err != nil {
		return nil, err
	}
	if lockedInvoice.CorrectionGroupID.IsNil() {
		lockedInvoice.CorrectionGroupID = group.ID
		if _, err = s.invoiceRepo.Update(ctx, lockedInvoice); err != nil {
			return nil, err
		}
	}

	now := timeutils.NowUnix()
	if adjustment.ApprovalRequired {
		adjustment.ApprovedByID = actor.UserID
		adjustment.ApprovedAt = &now
		adjustment.ApprovalStatus = invoiceadjustment.ApprovalStatusApproved
	}
	adjustment.Status = invoiceadjustment.StatusExecuting
	adjustment.CorrectionGroupID = group.ID
	adjustment, err = s.repo.UpdateAdjustment(ctx, adjustment)
	if err != nil {
		return nil, err
	}

	creditMemoItem, err := s.createCreditMemoQueueItem(
		ctx,
		adjustment,
		lockedInvoice,
		computation.preview,
	)
	if err != nil {
		return nil, err
	}
	creditMemoInvoice, err := s.createCreditMemoInvoice(
		ctx,
		creditMemoItem,
		adjustment,
		lockedInvoice,
		computation.creditLineItems,
		computation.preview,
		now,
	)
	if err != nil {
		return nil, err
	}
	adjustment.CreditMemoInvoiceID = creditMemoInvoice.ID

	var rebillQueueItem *billingqueue.BillingQueueItem
	var replacementInvoice *invoice.Invoice
	if adjustment.Kind == invoiceadjustment.KindCreditRebill {
		rebillQueueItem, err = s.createReplacementQueueItem(
			ctx,
			adjustment,
			lockedInvoice,
			group,
			computation.replacementLines,
			computation.preview,
		)
		if err != nil {
			return nil, err
		}
		replacementInvoice, err = s.createReplacementDraftInvoice(
			ctx,
			rebillQueueItem,
			adjustment,
			lockedInvoice,
			computation.replacementLines,
			computation.preview,
		)
		if err != nil {
			return nil, err
		}
		adjustment.ReplacementInvoiceID = replacementInvoice.ID
		adjustment.RebillQueueItemID = rebillQueueItem.ID
	}

	var writeOffJournalEntryID pulid.ID
	if adjustment.Kind == invoiceadjustment.KindWriteOff {
		writeOffJournalEntryID, err = s.createWriteOffJournalEntry(
			ctx,
			adjustment,
			lockedInvoice,
			computation.preview,
			actor,
		)
		if err != nil {
			return nil, err
		}
		if adjustment.Metadata == nil {
			adjustment.Metadata = make(map[string]any, 1)
		}
		adjustment.Metadata["writeOffJournalEntryId"] = writeOffJournalEntryID.String()
	}

	if computation.preview.RequiresReconciliationException {
		if _, err = s.db.DBForContext(ctx).
			NewInsert().
			Model(&invoiceadjustment.InvoiceAdjustmentReconciliationException{
				OrganizationID:      adjustment.OrganizationID,
				BusinessUnitID:      adjustment.BusinessUnitID,
				AdjustmentID:        adjustment.ID,
				InvoiceID:           lockedInvoice.ID,
				CreditMemoInvoiceID: creditMemoInvoice.ID,
				Status:              invoiceadjustment.ExceptionStatusOpen,
				Reason:              "Adjustment executed against settled or finance-sensitive invoice state",
				Amount:              computation.preview.CreditTotalAmount,
				Metadata: map[string]any{
					"settlementStatus": lockedInvoice.SettlementStatus,
					"disputeStatus":    lockedInvoice.DisputeStatus,
				},
			}).
			Exec(ctx); err != nil {
			return nil, fmt.Errorf("create reconciliation exception: %w", err)
		}
	}

	if _, err = s.db.DBForContext(ctx).
		NewInsert().
		Model(&invoiceadjustment.InvoiceAdjustmentSnapshot{
			OrganizationID: adjustment.OrganizationID,
			BusinessUnitID: adjustment.BusinessUnitID,
			AdjustmentID:   adjustment.ID,
			InvoiceID:      lockedInvoice.ID,
			Kind:           invoiceadjustment.SnapshotKindExecution,
			CreatedByID:    actor.UserID,
			Payload: map[string]any{
				"sourceInvoice":          s.snapshotPayload(lockedInvoice),
				"creditMemoId":           creditMemoInvoice.ID.String(),
				"rebillQueueItemId":      optionalPulidStringPtr(rebillQueueItem),
				"replacementInvoiceId":   optionalInvoiceIDString(replacementInvoice),
				"writeOffJournalEntryId": optionalIDString(writeOffJournalEntryID),
			},
		}).
		Exec(ctx); err != nil {
		return nil, fmt.Errorf("create execution snapshot: %w", err)
	}

	adjustment.Status = invoiceadjustment.StatusExecuted
	adjustment.ExecutionError = ""
	adjustment.ReplacementReviewStatus = replacementReviewStatus(
		computation.preview.RequiresReplacementInvoiceReview,
	)
	updated, err := s.repo.UpdateAdjustment(ctx, adjustment)
	if err != nil {
		return nil, err
	}

	group.CurrentInvoiceID = lockedInvoice.ID
	if replacementInvoice != nil {
		group.CurrentInvoiceID = replacementInvoice.ID
	}
	if _, err = s.repo.UpdateCorrectionGroup(ctx, group); err != nil {
		return nil, err
	}

	s.logAudit(updated, actor, permission.OpUpdate, "Invoice adjustment executed")
	s.logAdjustmentEvent("invoice adjustment executed", updated, zap.InfoLevel)
	return updated, nil
}

func (s *Service) createCreditMemoQueueItem(
	ctx context.Context,
	adjustment *invoiceadjustment.InvoiceAdjustment,
	sourceInvoice *invoice.Invoice,
	preview *servicesports.InvoiceAdjustmentPreview,
) (*billingqueue.BillingQueueItem, error) {
	number, err := s.generator.GenerateCreditMemoNumber(
		ctx,
		adjustment.OrganizationID,
		adjustment.BusinessUnitID,
		"",
		"",
	)
	if err != nil {
		return nil, err
	}

	return s.billingQueueRepo.Create(ctx, &billingqueue.BillingQueueItem{
		OrganizationID:            adjustment.OrganizationID,
		BusinessUnitID:            adjustment.BusinessUnitID,
		ShipmentID:                sourceInvoice.ShipmentID,
		Number:                    number,
		Status:                    billingqueue.StatusPosted,
		BillType:                  billingqueue.BillTypeCreditMemo,
		IsAdjustmentOrigin:        true,
		SourceInvoiceID:           &sourceInvoice.ID,
		SourceInvoiceAdjustmentID: &adjustment.ID,
		CorrectionGroupID:         &adjustment.CorrectionGroupID,
		RequiresReplacementReview: preview.RequiresReplacementInvoiceReview,
		RerateVariancePercent:     decimal.Zero,
		AdjustmentContext: map[string]any{
			"kind": adjustment.Kind,
		},
	})
}

func (s *Service) createCreditMemoInvoice(
	ctx context.Context,
	item *billingqueue.BillingQueueItem,
	adjustment *invoiceadjustment.InvoiceAdjustment,
	sourceInvoice *invoice.Invoice,
	lines []*invoice.InoviceLine,
	preview *servicesports.InvoiceAdjustmentPreview,
	postedAt int64,
) (*invoice.Invoice, error) {
	subtotal := decimal.Zero
	other := decimal.Zero
	for _, line := range lines {
		if line == nil {
			continue
		}
		if line.Type == invoice.InvoiceLineTypeFreight {
			subtotal = subtotal.Add(line.Amount)
		} else {
			other = other.Add(line.Amount)
		}
	}

	entity := &invoice.Invoice{
		OrganizationID:            sourceInvoice.OrganizationID,
		BusinessUnitID:            sourceInvoice.BusinessUnitID,
		BillingQueueItemID:        item.ID,
		ShipmentID:                sourceInvoice.ShipmentID,
		CustomerID:                sourceInvoice.CustomerID,
		Number:                    item.Number,
		BillType:                  billingqueue.BillTypeCreditMemo,
		Status:                    invoice.StatusPosted,
		PaymentTerm:               sourceInvoice.PaymentTerm,
		CurrencyCode:              sourceInvoice.CurrencyCode,
		InvoiceDate:               preview.AccountingDate,
		DueDate:                   &preview.AccountingDate,
		PostedAt:                  &postedAt,
		ShipmentProNumber:         sourceInvoice.ShipmentProNumber,
		ShipmentBOL:               sourceInvoice.ShipmentBOL,
		ServiceDate:               sourceInvoice.ServiceDate,
		BillToName:                sourceInvoice.BillToName,
		BillToCode:                sourceInvoice.BillToCode,
		BillToAddressLine1:        sourceInvoice.BillToAddressLine1,
		BillToAddressLine2:        sourceInvoice.BillToAddressLine2,
		BillToCity:                sourceInvoice.BillToCity,
		BillToState:               sourceInvoice.BillToState,
		BillToPostalCode:          sourceInvoice.BillToPostalCode,
		BillToCountry:             sourceInvoice.BillToCountry,
		SubtotalAmount:            subtotal,
		OtherAmount:               other,
		TotalAmount:               preview.CreditTotalAmount.Neg(),
		AppliedAmount:             decimal.Zero,
		SettlementStatus:          invoice.SettlementStatusUnpaid,
		DisputeStatus:             invoice.DisputeStatusNone,
		CorrectionGroupID:         adjustment.CorrectionGroupID,
		SourceInvoiceAdjustmentID: adjustment.ID,
		IsAdjustmentArtifact:      true,
		Lines:                     lines,
	}

	return s.invoiceRepo.Create(ctx, entity)
}

func (s *Service) createReplacementQueueItem(
	ctx context.Context,
	adjustment *invoiceadjustment.InvoiceAdjustment,
	sourceInvoice *invoice.Invoice,
	group *invoiceadjustment.InvoiceAdjustmentCorrectionGroup,
	lines []*invoice.InoviceLine,
	preview *servicesports.InvoiceAdjustmentPreview,
) (*billingqueue.BillingQueueItem, error) {
	number, err := s.generator.GenerateInvoiceNumber(
		ctx,
		adjustment.OrganizationID,
		adjustment.BusinessUnitID,
		"",
		"",
	)
	if err != nil {
		return nil, err
	}

	return s.billingQueueRepo.Create(ctx, &billingqueue.BillingQueueItem{
		OrganizationID:            adjustment.OrganizationID,
		BusinessUnitID:            adjustment.BusinessUnitID,
		ShipmentID:                sourceInvoice.ShipmentID,
		Number:                    number,
		Status:                    replacementQueueStatus(preview.RequiresReplacementInvoiceReview),
		BillType:                  billingqueue.BillTypeInvoice,
		IsAdjustmentOrigin:        true,
		SourceInvoiceID:           &sourceInvoice.ID,
		SourceInvoiceAdjustmentID: &adjustment.ID,
		SourceCreditMemoInvoiceID: &adjustment.CreditMemoInvoiceID,
		CorrectionGroupID:         &group.ID,
		RebillStrategy:            string(adjustment.RebillStrategy),
		RequiresReplacementReview: preview.RequiresReplacementInvoiceReview,
		RerateVariancePercent:     preview.RerateVariancePercent,
		AdjustmentContext: map[string]any{
			"replacementLines":   lines,
			"subtotalAmount":     sumInvoiceLines(lines, invoice.InvoiceLineTypeFreight),
			"otherAmount":        sumInvoiceLines(lines, invoice.InvoiceLineTypeAccessorial),
			"totalAmount":        preview.RebillTotalAmount,
			"accountingDate":     preview.AccountingDate,
			"sourceInvoiceId":    sourceInvoice.ID,
			"correctionGroupId":  group.ID,
			"sourceAdjustmentId": adjustment.ID,
		},
	})
}

func (s *Service) snapshotPayload(entity *invoice.Invoice) map[string]any {
	payload := jsonutils.MustToJSON(entity)
	payload["paymentSummary"] = map[string]any{
		"appliedAmount":     entity.AppliedAmount,
		"openBalanceAmount": entity.OpenBalanceAmount(),
		"settlementStatus":  entity.SettlementStatus,
		"disputeStatus":     entity.DisputeStatus,
	}
	return payload
}

func (s *Service) requestLinesFromAdjustment(
	entity *invoiceadjustment.InvoiceAdjustment,
) []*servicesports.InvoiceAdjustmentLineInput {
	lines := make([]*servicesports.InvoiceAdjustmentLineInput, 0, len(entity.Lines))
	for _, line := range entity.Lines {
		if line == nil {
			continue
		}
		lines = append(lines, &servicesports.InvoiceAdjustmentLineInput{
			OriginalLineID:     line.OriginalLineID,
			CreditQuantity:     line.CreditQuantity,
			CreditAmount:       line.CreditAmount,
			RebillQuantity:     line.RebillQuantity,
			RebillAmount:       line.RebillAmount,
			Description:        line.Description,
			ReplacementPayload: line.ReplacementPayload,
		})
	}
	return lines
}

func (s *Service) logAudit(
	entity *invoiceadjustment.InvoiceAdjustment,
	actor *servicesports.RequestActor,
	op permission.Operation,
	comment string,
) {
	if entity == nil {
		return
	}
	if err := s.auditService.LogAction(
		&servicesports.LogActionParams{
			Resource:       permission.ResourceInvoice,
			ResourceID:     entity.ID.String(),
			Operation:      op,
			UserID:         actor.UserID,
			PrincipalType:  actor.PrincipalType,
			PrincipalID:    actor.PrincipalID,
			APIKeyID:       actor.APIKeyID,
			CurrentState:   jsonutils.MustToJSON(entity),
			OrganizationID: entity.OrganizationID,
			BusinessUnitID: entity.BusinessUnitID,
		},
		auditservice.WithComment(comment),
	); err != nil {
		s.l.Warn("failed to log invoice adjustment audit action", zap.Error(err))
	}
}

func (s *Service) logAdjustmentEvent(
	message string,
	entity *invoiceadjustment.InvoiceAdjustment,
	level zapcore.Level,
) {
	if entity == nil {
		return
	}

	fields := []zap.Field{
		zap.String("adjustmentId", entity.ID.String()),
		zap.String("invoiceId", entity.OriginalInvoiceID.String()),
		zap.String("correctionGroupId", entity.CorrectionGroupID.String()),
		zap.String("idempotencyKey", entity.IdempotencyKey),
		zap.String("status", string(entity.Status)),
		zap.String("approvalStatus", string(entity.ApprovalStatus)),
	}
	if entity.ExecutionError != "" {
		fields = append(fields, zap.String("executionError", entity.ExecutionError))
	}

	switch level {
	case zap.WarnLevel:
		s.l.Warn(message, fields...)
	case zap.ErrorLevel:
		s.l.Error(message, fields...)
	default:
		s.l.Info(message, fields...)
	}
}

func appendPreviewError(preview *servicesports.InvoiceAdjustmentPreview, field, message string) {
	preview.Errors[field] = append(preview.Errors[field], message)
}

func previewErrorsToMultiError(errors map[string][]string) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	for field, messages := range errors {
		for _, message := range messages {
			multiErr.Add(field, errortypes.ErrInvalidOperation, message)
		}
	}
	return multiErr
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

func sumInvoiceLines(
	lines []*invoice.InoviceLine,
	lineType invoice.InvoiceLineType,
) decimal.Decimal {
	total := decimal.Zero
	for _, line := range lines {
		if line == nil {
			continue
		}
		if lineType == "" || line.Type == lineType {
			total = total.Add(line.Amount)
		}
	}
	return total
}

func buildReplacementLinesFromShipment(shp *shipment.Shipment) []*invoice.InoviceLine {
	lines := make([]*invoice.InoviceLine, 0, 1+len(shp.AdditionalCharges))
	freight := shp.FreightChargeAmount.Decimal
	lines = append(lines, &invoice.InoviceLine{
		LineNumber:  1,
		Type:        invoice.InvoiceLineTypeFreight,
		Description: "Freight charge",
		Quantity:    decimal.NewFromInt(1),
		UnitPrice:   freight,
		Amount:      freight,
	})
	for idx, charge := range shp.AdditionalCharges {
		if charge == nil {
			continue
		}
		qty := decimal.NewFromInt32(int32(charge.Unit))
		if qty.LessThanOrEqual(decimal.Zero) {
			qty = decimal.NewFromInt(1)
		}
		unitPrice := charge.Amount
		if qty.GreaterThan(decimal.Zero) {
			unitPrice = charge.Amount.Div(qty)
		}
		description := "Accessorial charge"
		if charge.AccessorialCharge != nil &&
			strings.TrimSpace(charge.AccessorialCharge.Description) != "" {
			description = charge.AccessorialCharge.Description
		}
		lines = append(lines, &invoice.InoviceLine{
			LineNumber:  idx + 2,
			Type:        invoice.InvoiceLineTypeAccessorial,
			Description: description,
			Quantity:    qty,
			UnitPrice:   unitPrice,
			Amount:      charge.Amount,
		})
	}
	return lines
}

func optionalPulidStringPtr(item *billingqueue.BillingQueueItem) *string {
	if item == nil {
		return nil
	}
	value := item.ID.String()
	return &value
}

func optionalInvoiceIDString(item *invoice.Invoice) *string {
	if item == nil {
		return nil
	}
	value := item.ID.String()
	return &value
}

func optionalIDString(id pulid.ID) *string {
	if id.IsNil() {
		return nil
	}
	value := id.String()
	return &value
}

func collectBatchItemIDs(items []*invoiceadjustment.InvoiceAdjustmentBatchItem) []pulid.ID {
	ids := make([]pulid.ID, 0, len(items))
	for _, item := range items {
		if item != nil {
			ids = append(ids, item.ID)
		}
	}
	return ids
}

func maxDecimal(a, b decimal.Decimal) decimal.Decimal {
	if a.GreaterThan(b) {
		return a
	}
	return b
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
