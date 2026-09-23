package resolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvoiceDaysPastDue(t *testing.T) {
	t.Parallel()

	due := int64(1_700_000_000)
	open := func() *invoice.Invoice {
		return &invoice.Invoice{
			Status:           invoice.StatusPosted,
			BillType:         billingqueue.BillTypeInvoice,
			TotalAmountMinor: 10000,
			DueDate:          &due,
		}
	}

	assert.Nil(t, invoiceDaysPastDue(nil, due+10*secondsPerDay))

	paid := open()
	paid.AppliedAmountMinor = 10000
	assert.Nil(
		t,
		invoiceDaysPastDue(paid, due+10*secondsPerDay),
		"nothing owed means nothing overdue",
	)

	memo := open()
	memo.BillType = billingqueue.BillTypeCreditMemo
	assert.Nil(t, invoiceDaysPastDue(memo, due+10*secondsPerDay), "credit memos are never past due")

	noDue := open()
	noDue.DueDate = nil
	assert.Nil(t, invoiceDaysPastDue(noDue, due+10*secondsPerDay))

	assert.Nil(t, invoiceDaysPastDue(open(), due-1), "not yet due")
	assert.Nil(
		t,
		invoiceDaysPastDue(open(), due+secondsPerDay-1),
		"less than a whole day late reads as current",
	)

	oneDay := invoiceDaysPastDue(open(), due+secondsPerDay)
	require.NotNil(t, oneDay)
	assert.Equal(t, 1, *oneDay)

	tenDays := invoiceDaysPastDue(open(), due+10*secondsPerDay+3600)
	require.NotNil(t, tenDays)
	assert.Equal(t, 10, *tenDays, "partial days are not counted")
}

func TestCreateMemoRequestFromInput(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	customerID := pulid.MustNew("cus_")
	referenceID := pulid.MustNew("inv_")
	accessorialID := pulid.MustNew("acc_")
	invoiceDate := 1_700_000_000
	autoPost := true
	kind := invoice.MemoKindLateCharge
	memo := "Approved by AR"
	quantity := "2"

	req, err := createMemoRequestFromInput(&gqlmodel.CreateMemoInput{
		CustomerID:         customerID.String(),
		BillType:           billingqueue.BillTypeDebitMemo,
		ReferenceInvoiceID: ptr(referenceID.String()),
		Reason:             "Returned cheque fee",
		InvoiceDate:        &invoiceDate,
		Memo:               &memo,
		MemoKind:           &kind,
		AutoPost:           &autoPost,
		Lines: []*gqlmodel.MemoLineInput{
			{Description: "Returned cheque", Amount: "35.00"},
			{
				Description:         "Detention",
				Amount:              "12.5",
				Quantity:            &quantity,
				AccessorialChargeID: ptr(accessorialID.String()),
			},
		},
	}, tenantInfo)

	require.NoError(t, err)
	assert.Equal(t, tenantInfo, req.TenantInfo)
	assert.Equal(t, customerID, req.CustomerID)
	assert.Equal(t, billingqueue.BillTypeDebitMemo, req.BillType)
	assert.Equal(t, referenceID, req.ReferenceInvoiceID)
	assert.Equal(t, "Returned cheque fee", req.Reason)
	assert.Equal(t, int64(invoiceDate), req.InvoiceDate)
	assert.Equal(t, "Approved by AR", req.Memo)
	assert.Equal(t, invoice.MemoKindLateCharge, req.MemoKind)
	assert.True(t, req.AutoPost)
	require.Len(t, req.Lines, 2)
	assert.True(t, req.Lines[0].Amount.Equal(decimal.NewFromInt(35)))
	assert.True(t, req.Lines[0].Quantity.IsZero(), "quantity is left for the service to default")
	assert.True(t, req.Lines[1].Quantity.Equal(decimal.NewFromInt(2)))
	assert.Equal(t, accessorialID, req.Lines[1].AccessorialChargeID)
}

func TestCreateMemoRequestFromInputRejectsBadIDs(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_")}

	_, err := createMemoRequestFromInput(
		&gqlmodel.CreateMemoInput{CustomerID: "nope", BillType: billingqueue.BillTypeCreditMemo},
		tenantInfo,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid customer")

	_, err = createMemoRequestFromInput(&gqlmodel.CreateMemoInput{
		CustomerID:         pulid.MustNew("cus_").String(),
		BillType:           billingqueue.BillTypeCreditMemo,
		ReferenceInvoiceID: ptr("nope"),
	}, tenantInfo)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid reference invoice")

	req, err := createMemoRequestFromInput(&gqlmodel.CreateMemoInput{
		CustomerID:         pulid.MustNew("cus_").String(),
		BillType:           billingqueue.BillTypeCreditMemo,
		ReferenceInvoiceID: ptr(""),
	}, tenantInfo)
	require.NoError(t, err)
	assert.True(t, req.ReferenceInvoiceID.IsNil(), "an empty reference means none")
	assert.False(t, req.AutoPost)
	assert.Equal(t, int64(0), req.InvoiceDate)
}

func TestMemoLinesFromInputCollectsEveryLineError(t *testing.T) {
	t.Parallel()

	_, err := memoLinesFromInput([]*gqlmodel.MemoLineInput{
		nil,
		{Description: "Bad amount", Amount: "twelve"},
		{Description: "Bad quantity", Amount: "1", Quantity: ptr("many")},
		{Description: "Bad accessorial", Amount: "1", AccessorialChargeID: ptr("acc")},
	})

	require.Error(t, err)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	fields := make([]string, 0, len(multiErr.Errors))
	for _, fieldErr := range multiErr.Errors {
		fields = append(fields, fieldErr.Field)
	}
	assert.ElementsMatch(t, []string{
		"lines[0].description",
		"lines[1].amount",
		"lines[2].quantity",
		"lines[3].accessorialChargeId",
	}, fields)
}

func ptr[T any](v T) *T {
	return &v
}
