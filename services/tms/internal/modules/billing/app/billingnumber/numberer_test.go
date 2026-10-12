package billingnumber_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/app/billingnumber"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks/customerportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type recordingGenerator struct {
	testutil.TestSequenceGenerator
	requests []*seqgen.GenerateRequest
}

func (g *recordingGenerator) Generate(_ context.Context, req *seqgen.GenerateRequest) (string, error) {
	g.requests = append(g.requests, req)
	return req.Format.Prefix + "-0001", nil
}

func (g *recordingGenerator) GenerateInvoiceNumber(
	context.Context, pulid.ID, pulid.ID, string, string,
) (string, error) {
	return "INV-0001", nil
}

type staticFormats struct{ format *tenant.SequenceFormat }

func (f staticFormats) GetFormat(
	context.Context, tenant.SequenceType, pulid.ID, pulid.ID,
) (*tenant.SequenceFormat, error) {
	return f.format, nil
}

func payerWithPrefix(prefix string) *customer.Customer {
	return &customer.Customer{BillingProfile: &customer.CustomerBillingProfile{
		InvoiceNumberFormat:   customer.InvoiceNumberFormatCustomPrefix,
		CustomerInvoicePrefix: prefix,
	}}
}

func TestInvoiceNumberTakesThePayersPrefix(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	payerID := pulid.MustNew("cus_")
	base := &tenant.SequenceFormat{Prefix: "INV"}
	customers := customerportsmock.NewMockCustomerRepository(t)
	customers.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req customerports.GetCustomerByIDRequest) bool {
			return req.ID == payerID && req.IncludeBillingProfile
		})).
		Return(payerWithPrefix("ACME"), nil).
		Once()
	generator := &recordingGenerator{}

	number, err := billingnumber.New(generator, staticFormats{format: base}, customers).Next(
		t.Context(),
		&billingnumber.Request{TenantInfo: tenantInfo, BillType: billingqueue.BillTypeInvoice, PayerID: payerID},
	)

	require.NoError(t, err)
	assert.Equal(t, "ACME-0001", number)
	assert.Equal(t, "INV", base.Prefix, "the tenant's format is never mutated")
}

func TestInvoiceNumberFallsBackToTheTenantDefault(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	customers := customerportsmock.NewMockCustomerRepository(t)
	customers.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("Customer not found")).
		Once()
	generator := &recordingGenerator{}
	numberer := billingnumber.New(generator, staticFormats{format: &tenant.SequenceFormat{Prefix: "INV"}}, customers)

	number, err := numberer.Next(t.Context(), &billingnumber.Request{
		TenantInfo: tenantInfo, BillType: billingqueue.BillTypeInvoice, PayerID: pulid.MustNew("cus_"),
	})
	require.NoError(t, err)
	assert.Equal(t, "INV-0001", number)

	number, err = numberer.Next(t.Context(), &billingnumber.Request{
		TenantInfo: tenantInfo,
		BillType:   billingqueue.BillTypeInvoice,
		Profile:    &customer.CustomerBillingProfile{InvoiceNumberFormat: customer.InvoiceNumberFormatDefault},
	})
	require.NoError(t, err)
	assert.Equal(t, "INV-0001", number)
	assert.Empty(t, generator.requests)
}

func TestForItemRenumbersForTheItemsPayer(t *testing.T) {
	t.Parallel()

	item := &billingqueue.BillingQueueItem{
		OrganizationID:   pulid.MustNew("org_"),
		BusinessUnitID:   pulid.MustNew("bu_"),
		BillToCustomerID: pulid.MustNew("cus_"),
		BillType:         billingqueue.BillTypeInvoice,
	}
	customers := customerportsmock.NewMockCustomerRepository(t)
	customers.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req customerports.GetCustomerByIDRequest) bool {
			return req.ID == item.BillToCustomerID && req.TenantInfo.OrgID == item.OrganizationID
		})).
		Return(payerWithPrefix("GLOBEX"), nil).
		Once()

	number, err := billingnumber.New(
		&recordingGenerator{}, staticFormats{format: &tenant.SequenceFormat{Prefix: "INV"}}, customers,
	).ForItem(t.Context(), item)

	require.NoError(t, err)
	assert.Equal(t, "GLOBEX-0001", number)
}
