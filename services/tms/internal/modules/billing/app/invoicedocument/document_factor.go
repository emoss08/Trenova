package invoicedocument

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/internal/modules/parties/ports/factoringcompanyports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

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
		factorID = AssignedFactorID(cus)
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

func AssignedFactorID(cus *customer.Customer) *pulid.ID {
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

func AutoSendsOnGeneration(cus *customer.Customer, entity *invoice.Invoice) bool {
	if cus == nil || cus.BillingProfile == nil ||
		!cus.BillingProfile.AutoSendInvoiceOnGeneration ||
		!cus.BillingProfile.EmailInvoiceEnabled {
		return false
	}

	if entity.Status != invoice.StatusPosted {
		return false
	}

	return entity.SendStatus == invoice.SendStatusNotSent ||
		entity.SendStatus == invoice.SendStatusFailed
}
