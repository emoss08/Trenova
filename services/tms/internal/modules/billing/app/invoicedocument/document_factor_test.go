package invoicedocument

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/billingtest/invoiceservicetest"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveFactorUsesThePostedAssignmentOrTheDraftsCustomer(t *testing.T) {
	t.Parallel()

	posted := &factoringcompany.FactoringCompany{ID: pulid.MustNew("fac_"), Name: "Posted Factor"}
	current := &factoringcompany.FactoringCompany{ID: pulid.MustNew("fac_"), Name: "Current Factor"}
	factors := &invoiceservicetest.FakeFactors{
		ByID: map[pulid.ID]*factoringcompany.FactoringCompany{
			posted.ID:  posted,
			current.ID: current,
		},
	}
	cus := invoiceservicetest.FactoredCustomer(&current.ID)

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

	data := buildInvoicePDFData(entity, &DeliveryProfile{Factor: factor})

	assert.Equal(t, "RTS Financial", data.RemitTo.Name)
	assert.Equal(
		t,
		[]string{"PO Box 840267", "Dallas, 75284", "ACH routing 111000614"},
		data.RemitTo.Lines,
	)
	assert.Contains(t, data.NoticeOfAssignment, "payable only to, RTS Financial")

	unfactored := buildInvoicePDFData(entity, &DeliveryProfile{})
	assert.Empty(t, unfactored.NoticeOfAssignment)

	email := EmailContext(
		&EmailContextParams{Entity: entity, Profile: &DeliveryProfile{Factor: factor}},
	)
	assert.Contains(t, email.RemittanceInstructions, "Remit to RTS Financial")
	assert.Equal(t, factor.Notice(), email.NoticeOfAssignment)
}
