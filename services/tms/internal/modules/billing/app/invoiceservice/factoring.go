package invoiceservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/factoringcompanyports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// assignFactoring records on a posting invoice the factor its customer's
// invoices are assigned to, so the invoice keeps naming that factor after
// the customer moves. A customer assigned to an inactive factor cannot post:
// its invoice would name a payee the organization no longer uses.
func (s *Service) assignFactoring(
	ctx context.Context,
	entity *invoice.Invoice,
	tenantInfo pagination.TenantInfo,
) error {
	if s.customerRepo == nil || s.factoringRepo == nil || entity.CustomerID.IsNil() {
		return nil
	}
	cus, err := s.customerRepo.GetByID(ctx, customerports.GetCustomerByIDRequest{
		ID:         entity.CustomerID,
		TenantInfo: tenantInfo,
		CustomerFilterOptions: customerports.CustomerFilterOptions{
			IncludeBillingProfile: true,
		},
	})
	if err != nil {
		return err
	}
	factorID := assignedFactorID(cus)
	if factorID == nil {
		entity.FactoringCompanyID = nil
		return nil
	}
	factor, err := s.factoringRepo.GetByID(
		ctx,
		factoringcompanyports.GetFactoringCompanyByIDRequest{
			ID:         *factorID,
			TenantInfo: tenantInfo,
		},
	)
	if err != nil {
		return err
	}
	if !factor.IsActive() {
		return errortypes.NewBusinessError(
			"{0}'s invoices are assigned to {1}, which is inactive. Assign an active factoring company in the customer's billing profile, or stop factoring the customer, before posting.",
			cus.DisplayName(),
			factor.Name,
		)
	}
	entity.FactoringCompanyID = factorID
	return nil
}

// resolveFactor is the factor an invoice is remitted to: the one it was
// assigned to when it posted, or, for a draft, the one its customer's
// invoices are assigned to now.
func resolveFactor(
	ctx context.Context,
	repo factoringcompanyports.FactoringCompanyRepository,
	entity *invoice.Invoice,
	cus *customer.Customer,
	tenantInfo pagination.TenantInfo,
) (*factoringcompany.FactoringCompany, error) {
	if repo == nil || entity == nil {
		return nil, nil //nolint:nilnil // factoring is not wired in this process
	}
	factorID := entity.FactoringCompanyID
	if entity.Status == invoice.StatusDraft {
		factorID = assignedFactorID(cus)
	}
	if factorID == nil || factorID.IsNil() {
		return nil, nil //nolint:nilnil // the invoice has no factor
	}
	factor, err := repo.GetByID(ctx, factoringcompanyports.GetFactoringCompanyByIDRequest{
		ID:           *factorID,
		TenantInfo:   tenantInfo,
		IncludeState: true,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, nil //nolint:nilnil // a factor deleted since assignment is treated as none
		}
		return nil, err
	}
	return factor, nil
}

func assignedFactorID(cus *customer.Customer) *pulid.ID {
	if cus == nil || cus.BillingProfile == nil || cus.BillingProfile.FactoringCompanyID == nil ||
		cus.BillingProfile.FactoringCompanyID.IsNil() {
		return nil
	}
	id := *cus.BillingProfile.FactoringCompanyID
	return &id
}

// factorRemitPDFAddressBlock is where an assigned invoice is paid.
func factorRemitPDFAddressBlock(factor *factoringcompany.FactoringCompany) invoicePDFAddressBlock {
	return invoicePDFAddressBlock{Name: factor.Name, Lines: factor.RemitLines()}
}
