package invoicesendservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/app/invoicedocument"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoicevalidation"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func (s *Service) AutoSendInvoiceAfterPDFGeneration(
	ctx context.Context,
	req *invoiceports.AutoSendInvoiceAfterPDFGenerationRequest,
	actor *permissionports.RequestActor,
) (*invoiceports.InvoiceSendResult, error) {
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

	entity, err := s.repo.GetByID(ctx, invoiceports.GetInvoiceByIDRequest{
		ID:         req.InvoiceID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if err = invoicevalidation.RefuseVoided(entity, "sent"); err != nil {
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
	if !invoicedocument.AutoSendsOnGeneration(cus, entity) {
		return &invoiceports.InvoiceSendResult{Invoice: entity}, nil
	}

	return s.SendFromWorkflow(ctx, &invoiceports.InvoiceSendRequest{
		InvoiceID:  req.InvoiceID,
		TenantInfo: req.TenantInfo,
		BaseURL:    req.BaseURL,
		StartedAt:  req.StartedAt,
	}, actor)
}
