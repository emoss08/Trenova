package quickbooks_test

import (
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/quickbooks"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChangesSinceAsksForEntitiesSinceTheCursor(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 25, 16, 0, 0, 0, time.UTC)
	since := now.Add(-2 * time.Hour)
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v3/company/"+testRealm+"/cdc", r.URL.Path)
		assert.Equal(t, "Payment,BillPayment,Customer", r.URL.Query().Get("entities"))
		assert.Equal(t, "2026-09-25T14:00:00Z", r.URL.Query().Get("changedSince"))
		assert.Equal(t, "75", r.URL.Query().Get("minorversion"))
		_, _ = w.Write(fixture(t, "cdc_changes.json"))
	})

	set, err := client.ChangesSince(t.Context(), []quickbooks.ChangeEntity{
		quickbooks.ChangePayment,
		quickbooks.ChangeBillPayment,
		quickbooks.ReferenceChangeEntity(quickbooks.KindCustomer),
	}, since, now)
	require.NoError(t, err)

	require.Len(t, set.Payments, 3)
	paid := set.Payments[0]
	assert.Equal(t, "301", paid.ID)
	assert.Equal(t, "58", paid.CustomerID)
	assert.Equal(t, "Acme Foods", paid.CustomerName)
	assert.True(t, decimal.RequireFromString("1500.25").Equal(paid.TotalAmount))
	assert.True(t, decimal.RequireFromString("49.75").Equal(paid.UnappliedAmount))
	assert.Equal(t, "2026-09-24", paid.TxnDate)
	assert.Equal(t, "USD", paid.CurrencyCode)
	assert.Equal(t, "2", paid.MethodID)
	assert.Equal(t, "Check", paid.MethodName)
	assert.Equal(t, "35", paid.DepositAccountID)
	assert.Equal(t, "10442", paid.ReferenceNumber)
	assert.Equal(t, "J Doe", paid.LastModifiedBy)
	assert.Equal(t,
		time.Date(2026, 9, 24, 16, 12, 30, 0, time.UTC).Unix(),
		paid.LastUpdatedAt,
	)
	assert.False(t, paid.Voided)
	assert.False(t, paid.Deleted)
	require.Len(t, paid.Lines, 3)
	assert.Equal(t, "160", paid.Lines[2].TxnID)
	assert.Equal(t, "CreditMemo", paid.Lines[2].TxnType)
	assert.True(t, decimal.RequireFromString("75").Equal(paid.Lines[2].Amount))

	assert.True(t, set.Payments[1].Voided, "a zero payment noted Voided was voided")
	assert.False(t, set.Payments[1].Deleted)
	assert.True(t, set.Payments[2].Deleted)
	assert.Equal(t, "303", set.Payments[2].ID)

	require.Len(t, set.BillPayments, 1)
	bill := set.BillPayments[0]
	assert.Equal(t, "410", bill.ID)
	assert.Equal(t, "5521", bill.DocNumber)
	assert.Equal(t, "77", bill.VendorID)
	assert.Equal(t, "Swift Haulers LLC", bill.VendorName)
	assert.Equal(t, "35", bill.AccountID)
	assert.Equal(t, "Check", bill.PayType)
	require.Len(t, bill.Lines, 1)
	assert.Equal(t, "Bill", bill.Lines[0].TxnType)
	assert.Equal(t, "390", bill.Lines[0].TxnID)

	require.Len(t, set.References, 2)
	assert.Equal(t, quickbooks.KindCustomer, set.References[0].Kind)
	assert.Equal(t, "Acme Foods Inc", set.References[0].Name)
	assert.False(t, set.References[0].Deleted)
	assert.Equal(t, quickbooks.KindVendor, set.References[1].Kind)
	assert.True(t, set.References[1].Deleted)
	assert.Empty(t, set.Full)
}

func TestChangesSinceRefusesACursorOlderThanThirtyDays(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client := newAPIClient(t, func(http.ResponseWriter, *http.Request) {
		calls.Add(1)
	})

	now := time.Date(2026, 9, 25, 16, 0, 0, 0, time.UTC)
	_, err := client.ChangesSince(
		t.Context(),
		[]quickbooks.ChangeEntity{quickbooks.ChangePayment},
		now.Add(-quickbooks.MaxChangeLookback-time.Second),
		now,
	)
	require.ErrorIs(t, err, quickbooks.ErrChangeWindow)
	assert.Zero(t, calls.Load())
}

func TestChangesSinceRejectsUnknownEntities(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("no call expected")
	})
	now := time.Now()

	_, err := client.ChangesSince(t.Context(), nil, now, now)
	require.ErrorIs(t, err, quickbooks.ErrNoChangeEntities)

	_, err = client.ChangesSince(
		t.Context(),
		[]quickbooks.ChangeEntity{"Estimate"},
		now,
		now,
	)
	require.ErrorIs(t, err, quickbooks.ErrUnknownChangeEntity)
}

func TestChangesSinceFlagsAnEntityThatHitTheCap(t *testing.T) {
	t.Parallel()

	var body strings.Builder
	body.WriteString(`{"CDCResponse":[{"QueryResponse":[{"Payment":[`)
	for idx := range quickbooks.MaxChangesPerCall {
		if idx > 0 {
			body.WriteString(",")
		}
		body.WriteString(`{"Id":"` + strconv.Itoa(idx) + `","TotalAmt":1,"Line":[]}`)
	}
	body.WriteString(`]}]}]}`)

	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body.String()))
	})

	now := time.Now()
	set, err := client.ChangesSince(
		t.Context(),
		[]quickbooks.ChangeEntity{quickbooks.ChangePayment},
		now.Add(-time.Hour),
		now,
	)
	require.NoError(t, err)
	assert.Len(t, set.Payments, quickbooks.MaxChangesPerCall)
	assert.Equal(t, []quickbooks.ChangeEntity{quickbooks.ChangePayment}, set.Full)
}

func TestQueryChangedSincePagesByLastUpdatedTime(t *testing.T) {
	t.Parallel()

	since := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/"+testRealm+"/query", r.URL.Path)
		assert.Equal(t,
			"select * from Payment where MetaData.LastUpdatedTime >= '2026-07-01T00:00:00Z'"+
				" orderby MetaData.LastUpdatedTime startposition 1 maxresults 2",
			r.URL.Query().Get("query"),
		)
		_, _ = w.Write(fixture(t, "query_payment_changed.json"))
	})

	set, next, err := client.QueryChangedSince(t.Context(), quickbooks.ChangePayment, since, 1, 2)
	require.NoError(t, err)
	require.Len(t, set.Payments, 2)
	assert.Equal(t, "288", set.Payments[0].ID)
	assert.Equal(t, "120", set.Payments[0].Lines[0].TxnID)
	assert.Equal(t, 3, next, "a full page means there may be more")
}

func TestQueryChangedSinceIncludesInactiveReferences(t *testing.T) {
	t.Parallel()

	since := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t,
			"select * from Vendor where MetaData.LastUpdatedTime >= '2026-07-01T00:00:00Z'"+
				" and Active in (true, false)"+
				" orderby MetaData.LastUpdatedTime startposition 1 maxresults 1000",
			r.URL.Query().Get("query"),
		)
		_, _ = w.Write(fixture(t, "query_empty.json"))
	})

	set, next, err := client.QueryChangedSince(
		t.Context(),
		quickbooks.ReferenceChangeEntity(quickbooks.KindVendor),
		since,
		1,
		quickbooks.MaxQueryResults,
	)
	require.NoError(t, err)
	assert.Empty(t, set.References)
	assert.Zero(t, next)
}
