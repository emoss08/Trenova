package invoiceservicetest

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/internal/modules/parties/ports/factoringcompanyports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type FakeFactors struct {
	factoringcompanyports.FactoringCompanyRepository
	ByID map[pulid.ID]*factoringcompany.FactoringCompany
}

func (f *FakeFactors) GetByID(
	_ context.Context,
	req factoringcompanyports.GetFactoringCompanyByIDRequest,
) (*factoringcompany.FactoringCompany, error) {
	if factor, ok := f.ByID[req.ID]; ok {
		return factor, nil
	}
	return nil, errortypes.NewNotFoundError("FactoringCompany not found")
}

func FactoredCustomer(factorID *pulid.ID) *customer.Customer {
	return &customer.Customer{
		ID:             pulid.MustNew("cus_"),
		Name:           "Halstead Grocery",
		BillingProfile: &customer.CustomerBillingProfile{FactoringCompanyID: factorID},
	}
}
