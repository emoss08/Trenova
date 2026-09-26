package invoiceservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type memoPlan struct {
	customer *customer.Customer
	control  *tenant.BillingControl
	lines    []*invoice.InvoiceLine
}

func (s *Service) planMemo(
	ctx context.Context,
	req *servicesports.CreateMemoRequest,
) (*memoPlan, error) {
	if multiErr := validateMemoRequest(req); multiErr != nil {
		return nil, multiErr
	}

	cus, err := s.customerRepo.GetByID(ctx, repositories.GetCustomerByIDRequest{
		ID:         req.CustomerID,
		TenantInfo: req.TenantInfo,
		CustomerFilterOptions: repositories.CustomerFilterOptions{
			IncludeBillingProfile: true,
			IncludeState:          true,
		},
	})
	if err != nil {
		return nil, err
	}
	if err = s.validateMemoReference(ctx, req, cus); err != nil {
		return nil, err
	}

	control, err := s.billingRepo.GetByOrgID(ctx, req.TenantInfo.OrgID)
	if err != nil && !errortypes.IsNotFoundError(err) {
		return nil, err
	}

	lines, err := s.memoLines(ctx, req)
	if err != nil {
		return nil, err
	}

	return &memoPlan{customer: cus, control: control, lines: lines}, nil
}

func memoQueueItem(
	req *servicesports.CreateMemoRequest,
	cus *customer.Customer,
	number string,
) *billingqueue.BillingQueueItem {
	return &billingqueue.BillingQueueItem{
		OrganizationID:   req.TenantInfo.OrgID,
		BusinessUnitID:   req.TenantInfo.BuID,
		BillToCustomerID: cus.ID,
		Status:           billingqueue.StatusApproved,
		BillType:         req.BillType,
		Number:           number,
		AdjustmentContext: map[string]any{
			"memo":     true,
			"memoKind": string(memoKindOrDefault(req.MemoKind)),
		},
	}
}

func (s *Service) PreviewMemo(
	ctx context.Context,
	req *servicesports.CreateMemoRequest,
) (*invoice.Invoice, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}

	plan, err := s.planMemo(ctx, req)
	if err != nil {
		return nil, err
	}

	item := memoQueueItem(req, plan.customer, unnumberedInvoice)
	item.ID = pulid.MustNew("bqi_")
	entity := s.buildMemoEntity(req, item, plan.customer, plan.control, plan.lines)
	if multiErr := s.validator.ValidateCreate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}
	entity.Number = ""

	return entity, nil
}

func (s *Service) planVoid(
	ctx context.Context,
	req *servicesports.VoidInvoiceRequest,
	actor *servicesports.RequestActor,
) (*invoice.Invoice, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}
	if actor == nil {
		return nil, errortypes.NewValidationError(
			"actor",
			errortypes.ErrRequired,
			"Actor is required",
		)
	}
	if multiErr := validateVoidRequest(req); multiErr != nil {
		return nil, multiErr
	}

	entity, err := s.repo.GetByID(ctx, repositories.GetInvoiceByIDRequest{
		ID:         req.InvoiceID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if err = s.refuseVoidWithStandingLateCharges(ctx, entity, req.TenantInfo); err != nil {
		return nil, err
	}

	switch entity.Status {
	case invoice.StatusDraft, invoice.StatusPosted:
		return entity, nil
	case invoice.StatusVoided:
		return nil, errortypes.NewValidationError(
			"invoiceId",
			errortypes.ErrInvalidOperation,
			"Invoice {0} is already voided",
			entity.Number,
		)
	default:
		return nil, errortypes.NewValidationError(
			"invoiceId",
			errortypes.ErrInvalidOperation,
			"Invoice cannot be voided from its current status",
		)
	}
}

func refuseAppliedInvoice(entity *invoice.Invoice) error {
	if entity.AppliedAmountMinor <= 0 {
		return nil
	}

	return errortypes.NewValidationError(
		"invoiceId",
		errortypes.ErrInvalidOperation,
		"Unapply the customer payments and credit memos on this invoice before voiding it",
	)
}

func (s *Service) markDraftVoided(
	ctx context.Context,
	entity *invoice.Invoice,
	req *servicesports.VoidInvoiceRequest,
	actor *servicesports.RequestActor,
	now int64,
) error {
	if err := refuseAppliedInvoice(entity); err != nil {
		return err
	}

	entity.Status = invoice.StatusVoided
	entity.VoidedAt = &now
	entity.VoidedByID = actor.UserID
	entity.VoidReason = strings.TrimSpace(req.Reason)
	entity.VoidDisposition = req.Disposition
	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return multiErr
	}

	return nil
}

func (s *Service) refuseVoidingPosted(entity *invoice.Invoice) error {
	if err := refuseAppliedInvoice(entity); err != nil {
		return err
	}
	if s.adjustmentService == nil {
		return errortypes.NewConflictError(
			"Invoice adjustments are unavailable; a posted invoice cannot be voided",
		)
	}

	return nil
}

func voidReversalRequest(
	entity *invoice.Invoice,
	req *servicesports.VoidInvoiceRequest,
) *servicesports.InvoiceAdjustmentRequest {
	return &servicesports.InvoiceAdjustmentRequest{
		InvoiceID:      entity.ID,
		Kind:           invoiceadjustment.KindFullReversal,
		Reason:         strings.TrimSpace(req.Reason),
		IdempotencyKey: "invoice-void:" + entity.ID.String(),
		TenantInfo:     req.TenantInfo,
	}
}

func (s *Service) PreviewVoid(
	ctx context.Context,
	req *servicesports.VoidInvoiceRequest,
	actor *servicesports.RequestActor,
) (*servicesports.InvoiceVoidPreview, error) {
	entity, err := s.planVoid(ctx, req, actor)
	if err != nil {
		return nil, err
	}

	after := *entity
	preview := &servicesports.InvoiceVoidPreview{Before: entity, After: &after}
	if entity.Status == invoice.StatusDraft {
		if err = s.markDraftVoided(ctx, &after, req, actor, timeutils.NowUnix()); err != nil {
			return nil, err
		}
		return preview, nil
	}

	if err = s.refuseVoidingPosted(entity); err != nil {
		return nil, err
	}
	after.VoidReason = strings.TrimSpace(req.Reason)
	after.VoidDisposition = req.Disposition
	preview.Posted = true
	if preview.Reversal, err = s.adjustmentService.Preview(
		ctx,
		voidReversalRequest(entity, req),
		actor,
	); err != nil {
		return nil, err
	}

	return preview, nil
}
