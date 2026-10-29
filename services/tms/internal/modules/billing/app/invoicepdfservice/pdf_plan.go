package invoicepdfservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceaction"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoicedocument"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoicevalidation"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *Service) planPDFGeneration(
	ctx context.Context,
	req *invoiceports.InvoicePreviewRequest,
	actor *permissionports.RequestActor,
) (*invoice.Invoice, pulid.ID, error) {
	if s.workflowStarter == nil || !s.workflowStarter.Enabled() {
		return nil, pulid.Nil, errortypes.NewBusinessError(
			"Invoice PDF generation requires workflow processing to be enabled",
		).WithInternal(kernelports.ErrWorkflowStarterDisabled)
	}
	if req == nil {
		return nil, pulid.Nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}
	if actor == nil {
		return nil, pulid.Nil, errortypes.NewValidationError(
			"actor",
			errortypes.ErrRequired,
			"Actor is required",
		)
	}
	userID := invoiceaction.ActorUserID(actor, req.TenantInfo)
	if userID.IsNil() {
		return nil, pulid.Nil, errortypes.NewValidationError(
			"userId",
			errortypes.ErrRequired,
			"User ID is required to generate invoice PDFs",
		)
	}

	entity, err := s.repo.GetByID(ctx, invoiceports.GetInvoiceByIDRequest{
		ID:         req.InvoiceID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, pulid.Nil, err
	}
	if err = invoicevalidation.RefuseVoided(entity, "rendered"); err != nil {
		return nil, pulid.Nil, err
	}

	return entity, userID, nil
}

func (s *Service) PlanPDFGeneration(
	ctx context.Context,
	req *invoiceports.InvoicePreviewRequest,
	actor *permissionports.RequestActor,
) (*invoiceports.InvoicePDFGenerationPlan, error) {
	entity, _, err := s.planPDFGeneration(ctx, req, actor)
	if err != nil {
		return nil, err
	}

	cus, err := s.customerRepo.GetByID(ctx, customerports.GetCustomerByIDRequest{
		ID:         entity.CustomerID,
		TenantInfo: req.TenantInfo,
		CustomerFilterOptions: customerports.CustomerFilterOptions{
			IncludeBillingProfile: true,
		},
	})
	if err != nil {
		return nil, err
	}

	return &invoiceports.InvoicePDFGenerationPlan{
		Invoice:   entity,
		AutoSends: invoicedocument.AutoSendsOnGeneration(cus, entity),
	}, nil
}
