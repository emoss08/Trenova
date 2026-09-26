package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeInboundReader struct {
	changes []*accountingsync.AccountingInboundChange
	listed  *serviceports.ListAccountingInboundChangesRequest
}

func (f *fakeInboundReader) List(
	_ context.Context,
	req *serviceports.ListAccountingInboundChangesRequest,
) (*pagination.CursorListResult[*accountingsync.AccountingInboundChange], error) {
	f.listed = req
	return &pagination.CursorListResult[*accountingsync.AccountingInboundChange]{
		Items: f.changes,
	}, nil
}

func TestListAccountingInboundChanges_FiltersAndSaysWhatEachPays(t *testing.T) {
	t.Parallel()

	invoiceID := pulid.MustNew("inv_")
	reader := &fakeInboundReader{changes: []*accountingsync.AccountingInboundChange{{
		ID:                 pulid.MustNew("acctic_"),
		Kind:               accountingsync.InboundCustomerPayment,
		Status:             accountingsync.InboundStatusProposed,
		Reason:             accountingsync.InboundReasonOverpayment,
		Resolution:         "Pays 1500.25 USD on INV-1001, which has 1000.00 USD open in Trenova.",
		ExternalNumber:     "10442",
		ExternalURL:        "https://books.example/payment/301",
		ProviderModifiedBy: "J Doe",
		PartyName:          "Acme Foods",
		AmountMinor:        150_025,
		CurrencyCode:       "USD",
		TxnDate:            1_790_208_000,
		Document: accountingsync.InboundDocument{
			ReferenceNumber: "10442",
			MethodName:      "Check",
			Lines: []*accountingsync.InboundLine{
				{
					DocumentKind:       accountingsync.InboundDocInvoice,
					DocumentExternalID: "145",
					AmountMinor:        150_025,
					ObjectType:         accountingsync.SyncObjectInvoice,
					ObjectID:           invoiceID,
					ObjectNumber:       "INV-1001",
					OpenMinor:          100_000,
				},
				{
					DocumentKind:       accountingsync.InboundDocOther,
					DocumentExternalID: "9",
					AmountMinor:        1_000,
				},
			},
		},
	}}}
	params := accountingQueryParams("QuickBooksOnline")
	params.Params["status"] = []any{"Proposed"}
	params.Params["kind"] = []any{"CustomerPayment"}
	params.Params["reason"] = []any{"Overpayment", "PeriodNotOpen"}
	params.Params["search"] = " Acme "
	params.Params["limit"] = 500.0

	out, err := newListAccountingInboundChangesTool(reader).Query(t.Context(), params)
	require.NoError(t, err)
	result, ok := out.(*accountingInboundResult)
	require.True(t, ok)

	require.NotNil(t, reader.listed)
	assert.Equal(t, []accountingsync.InboundChangeStatus{accountingsync.InboundStatusProposed},
		reader.listed.Statuses)
	assert.Equal(t, []accountingsync.InboundChangeKind{accountingsync.InboundCustomerPayment},
		reader.listed.Kinds)
	assert.Equal(t, []accountingsync.InboundChangeReason{
		accountingsync.InboundReasonOverpayment,
		accountingsync.InboundReasonPeriodNotOpen,
	}, reader.listed.Reasons)
	assert.Equal(t, "Acme", reader.listed.Search)
	assert.Equal(t, inboundMaxLimit, reader.listed.Cursor.Limit)
	assert.Equal(t, params.OrganizationID, reader.listed.TenantInfo.OrgID)
	assert.Equal(t, accountingInboundPath, result.PagePath)

	require.Len(t, result.Changes, 1)
	row := result.Changes[0]
	assert.Equal(t, "1500.25 USD", row.Amount)
	assert.Equal(t, "Overpayment", row.Reason)
	assert.False(t, row.CanApply, "an overpayment cannot be applied until it matches")
	assert.True(t, row.CanIgnore)
	require.Len(t, row.Pays, 2)
	assert.Equal(t, invoiceID.String(), row.Pays[0].DocumentID)
	assert.Equal(t, "INV-1001", row.Pays[0].DocumentNumber)
	assert.Equal(t, "1000.00 USD", row.Pays[0].OpenInTrenova)
	assert.Empty(t, row.Pays[1].DocumentID, "a document Trenova did not send has no Trenova id")
	assert.Empty(t, row.Pays[1].OpenInTrenova)
}

func TestListAccountingInboundChanges_RejectsUnknownValues(t *testing.T) {
	t.Parallel()

	params := accountingQueryParams("QuickBooksOnline")
	params.Params["status"] = []any{"Pending"}

	_, err := newListAccountingInboundChangesTool(&fakeInboundReader{}).Query(t.Context(), params)
	require.Error(t, err)
}
