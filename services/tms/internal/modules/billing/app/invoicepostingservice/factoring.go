package invoicepostingservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/app/invoicedocument"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/factoringcompanyports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
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
	if entity.FactoringOverride.Excludes() {
		entity.FactoringCompanyID = nil
		entity.FactoringAgreementID = nil
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
	factorID := invoicedocument.AssignedFactorID(cus)
	if factorID == nil {
		entity.FactoringCompanyID = nil
		entity.FactoringAgreementID = nil
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
	return s.assignAgreement(ctx, entity, tenantInfo, *factorID)
}

func (s *Service) assignAgreement(
	ctx context.Context,
	entity *invoice.Invoice,
	tenantInfo pagination.TenantInfo,
	factorID pulid.ID,
) error {
	entity.FactoringAgreementID = nil
	if s.agreementRepo == nil {
		return nil
	}
	at := timeutils.NowUnix()
	if entity.PostedAt != nil {
		at = *entity.PostedAt
	}
	agreement, err := s.agreementRepo.GetInForce(ctx, factoringports.GetAgreementInForceRequest{
		TenantInfo:         tenantInfo,
		FactoringCompanyID: factorID,
		At:                 at,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil
		}
		return err
	}
	id := agreement.ID
	entity.FactoringAgreementID = &id
	return nil
}
