package invoiceservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/factoringcompanyports"
	"github.com/emoss08/trenova/internal/testutil/mocks/customerportsmock"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeFactors struct {
	factoringcompanyports.FactoringCompanyRepository
	byID map[pulid.ID]*factoringcompany.FactoringCompany
}

func (f *fakeFactors) GetByID(
	_ context.Context,
	req factoringcompanyports.GetFactoringCompanyByIDRequest,
) (*factoringcompany.FactoringCompany, error) {
	if factor, ok := f.byID[req.ID]; ok {
		return factor, nil
	}
	return nil, errortypes.NewNotFoundError("FactoringCompany not found")
}

func factoredCustomer(factorID *pulid.ID) *customer.Customer {
	return &customer.Customer{
		ID:             pulid.MustNew("cus_"),
		Name:           "Halstead Grocery",
		BillingProfile: &customer.CustomerBillingProfile{FactoringCompanyID: factorID},
	}
}

func factoringService(
	t *testing.T,
	cus *customer.Customer,
	factors *fakeFactors,
) *Service {
	t.Helper()
	customers := customerportsmock.NewMockCustomerRepository(t)
	customers.EXPECT().GetByID(mock.Anything, mock.Anything).RunAndReturn(
		func(_ context.Context, req customerports.GetCustomerByIDRequest) (*customer.Customer, error) {
			assert.True(t, req.IncludeBillingProfile)
			return cus, nil
		}).Maybe()
	return &Service{l: zap.NewNop(), customerRepo: customers, factoringRepo: factors}
}

func TestAssignFactoringRecordsTheCustomersFactor(t *testing.T) {
	t.Parallel()

	factor := &factoringcompany.FactoringCompany{
		ID:     pulid.MustNew("fac_"),
		Name:   "RTS Financial",
		Status: domaintypes.StatusActive,
	}
	cus := factoredCustomer(&factor.ID)
	svc := factoringService(t, cus, &fakeFactors{byID: map[pulid.ID]*factoringcompany.FactoringCompany{
		factor.ID: factor,
	}})
	entity := &invoice.Invoice{CustomerID: cus.ID}

	require.NoError(t, svc.assignFactoring(t.Context(), entity, pagination.TenantInfo{}))

	require.NotNil(t, entity.FactoringCompanyID)
	assert.Equal(t, factor.ID, *entity.FactoringCompanyID)
}

func TestAssignFactoringClearsItForACustomerNoLongerFactored(t *testing.T) {
	t.Parallel()

	stale := pulid.MustNew("fac_")
	cus := factoredCustomer(nil)
	svc := factoringService(t, cus, &fakeFactors{})
	entity := &invoice.Invoice{CustomerID: cus.ID, FactoringCompanyID: &stale}

	require.NoError(t, svc.assignFactoring(t.Context(), entity, pagination.TenantInfo{}))

	assert.Nil(t, entity.FactoringCompanyID)
}

func TestAssignFactoringRefusesAnInactiveFactor(t *testing.T) {
	t.Parallel()

	factor := &factoringcompany.FactoringCompany{
		ID:     pulid.MustNew("fac_"),
		Name:   "RTS Financial",
		Status: domaintypes.StatusInactive,
	}
	cus := factoredCustomer(&factor.ID)
	svc := factoringService(t, cus, &fakeFactors{byID: map[pulid.ID]*factoringcompany.FactoringCompany{
		factor.ID: factor,
	}})
	entity := &invoice.Invoice{CustomerID: cus.ID}

	err := svc.assignFactoring(t.Context(), entity, pagination.TenantInfo{})

	var business *errortypes.BusinessError
	require.ErrorAs(t, err, &business)
	assert.Nil(t, entity.FactoringCompanyID)
}

func TestResolveFactorUsesThePostedAssignmentOrTheDraftsCustomer(t *testing.T) {
	t.Parallel()

	posted := &factoringcompany.FactoringCompany{ID: pulid.MustNew("fac_"), Name: "Posted Factor"}
	current := &factoringcompany.FactoringCompany{ID: pulid.MustNew("fac_"), Name: "Current Factor"}
	factors := &fakeFactors{byID: map[pulid.ID]*factoringcompany.FactoringCompany{
		posted.ID:  posted,
		current.ID: current,
	}}
	cus := factoredCustomer(&current.ID)

	got, err := resolveFactor(t.Context(), factors, &invoice.Invoice{
		Status:             invoice.StatusPosted,
		FactoringCompanyID: &posted.ID,
	}, cus, pagination.TenantInfo{})
	require.NoError(t, err)
	assert.Equal(t, posted, got, "a posted invoice keeps the factor it posted with")

	got, err = resolveFactor(t.Context(), factors, &invoice.Invoice{
		Status: invoice.StatusPosted,
	}, cus, pagination.TenantInfo{})
	require.NoError(t, err)
	assert.Nil(t, got, "an invoice posted before the customer was factored is not factored")

	got, err = resolveFactor(t.Context(), factors, &invoice.Invoice{
		Status: invoice.StatusDraft,
	}, cus, pagination.TenantInfo{})
	require.NoError(t, err)
	assert.Equal(t, current, got, "a draft shows the factor it will post with")
}

func TestFactoredInvoicePDFRemitsToTheFactorWithItsNotice(t *testing.T) {
	t.Parallel()

	factor := &factoringcompany.FactoringCompany{
		ID:                     pulid.MustNew("fac_"),
		Name:                   "RTS Financial",
		AddressLine1:           "PO Box 840267",
		City:                   "Dallas",
		PostalCode:             "75284",
		RemittanceInstructions: "ACH routing 111000614",
	}
	entity := &invoice.Invoice{Number: "INV-1", CurrencyCode: "USD"}

	data := buildInvoicePDFData(entity, &invoiceDeliveryProfile{Factor: factor})

	assert.Equal(t, "RTS Financial", data.RemitTo.Name)
	assert.Equal(t, []string{"PO Box 840267", "Dallas, 75284", "ACH routing 111000614"}, data.RemitTo.Lines)
	assert.Contains(t, data.NoticeOfAssignment, "payable only to, RTS Financial")

	unfactored := buildInvoicePDFData(entity, &invoiceDeliveryProfile{})
	assert.Empty(t, unfactored.NoticeOfAssignment)

	email := emailContext(&EmailContextParams{Entity: entity, Profile: &invoiceDeliveryProfile{Factor: factor}})
	assert.Contains(t, email.RemittanceInstructions, "Remit to RTS Financial")
	assert.Equal(t, factor.Notice(), email.NoticeOfAssignment)
}
