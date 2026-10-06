package bcconnector

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/businesscentral"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var pollNow = time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)

func TestChangeFeedLimitsAndCursorAt(t *testing.T) {
	t.Parallel()

	conn := testConnector(t, &fakeBC{})
	assert.Equal(t, services.AccountingChangeFeedLimits{
		MaxLookback:    0,
		ReportsDeletes: false,
		MaxPerRead:     businesscentral.MaxPageSize,
	}, conn.ChangeFeedLimits())
	at := time.Date(2026, 9, 1, 8, 30, 15, 900, time.FixedZone("x", 3600))
	assert.Equal(t, "2026-09-01T07:30:15Z", conn.ChangeCursorAt(at))
}

func TestChangeCursorRoundTripsAndRejectsForeignCursors(t *testing.T) {
	t.Parallel()

	paging := &changeCursor{
		since:   time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		paging:  true,
		started: pollNow,
		latest:  pollNow.Add(-time.Minute),
		targets: []changeTarget{targetSalesInvoices, targetCustomers},
		index:   1,
	}
	parsed, err := parseChangeCursor(paging.String(), pollNow)
	require.NoError(t, err)
	assert.Equal(t, paging, parsed)

	empty, err := parseChangeCursor("", pollNow)
	require.NoError(t, err)
	assert.Equal(t, pollNow, empty.since)

	plain, err := parseChangeCursor("2026-10-01T00:00:00Z", pollNow)
	require.NoError(t, err)
	assert.False(t, plain.paging)

	for _, bad := range []string{
		"yesterday",
		"p|2026-09-01T00:00:00Z||||0",
		"p|2026-09-01T00:00:00Z|||payments|0",
		"p|2026-09-01T00:00:00Z|||salesInvoices|1",
		"p|2026-09-01T00:00:00Z|||salesInvoices|x",
		"p||||salesInvoices|0",
		"p|2026-09-01T00:00:00Z|||salesInvoices|0|1",
	} {
		_, err = parseChangeCursor(bad, pollNow)
		require.ErrorIs(t, err, errChangeCursor, bad)
	}
}

func TestNextSinceIsTheLatestChangeLessOneSecond(t *testing.T) {
	t.Parallel()

	since := time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC)
	latest := time.Date(2026, 9, 29, 11, 15, 30, 500, time.UTC)
	assert.Equal(t, since, nextSince(since, time.Time{}, time.Time{}))
	assert.Equal(t, time.Date(2026, 9, 29, 11, 15, 29, 0, time.UTC),
		nextSince(since, latest, time.Time{}))
	assert.Equal(t, since, nextSince(since, since, time.Time{}))
	assert.Equal(t, since.Add(30*time.Minute-time.Second),
		nextSince(since, latest, since.Add(30*time.Minute)))
}

func changesResponder(t *testing.T) func(call bcCall) (int, string) {
	return func(call bcCall) (int, string) {
		if call.Method != http.MethodGet {
			return 0, ""
		}
		switch call.Path {
		case "/salesInvoices":
			return http.StatusOK, listJSON(t,
				docSpec{id: testInvoice, status: "Open", modified: "2026-10-06T12:05:00Z"},
				docSpec{id: testNewInvoice, status: "Canceled", modified: "2026-10-06T12:01:00Z"},
			)
		case "/salesCreditMemos", "/purchaseInvoices", "/items", "/vendors":
			return http.StatusOK, `{"value":[]}`
		case "/purchaseCreditMemos":
			return http.StatusOK, listJSON(t, docSpec{
				id: testVendorCredit, status: "Corrective", modified: "2026-10-06T12:02:00Z",
			})
		case "/customers":
			return http.StatusOK, readFixture(t, "customers.json")
		case "/accounts":
			return http.StatusOK, readFixture(t, "accounts.json")
		}
		return 0, ""
	}
}

func changesRequest(cursor string) *services.ReadAccountingChangesRequest {
	return &services.ReadAccountingChangesRequest{
		Auth:         testAuth(),
		Cursor:       cursor,
		Now:          pollNow.Add(10 * time.Minute),
		Payments:     true,
		BillPayments: true,
		Documents:    true,
		ReferenceKinds: []accountingsync.ReferenceKind{
			accountingsync.ReferenceKindAccount,
			accountingsync.ReferenceKindItem,
			accountingsync.ReferenceKindCustomer,
			accountingsync.ReferenceKindVendor,
			accountingsync.ReferenceKindTerm,
		},
	}
}

func TestReadChangesReadsDocumentsAndReferencesButNeverPayments(t *testing.T) {
	t.Parallel()

	fake := &fakeBC{respond: changesResponder(t)}
	conn := testConnector(t, fake)

	page, err := conn.ReadChanges(t.Context(), changesRequest("2026-10-06T12:00:00Z"))
	require.NoError(t, err)
	assert.Empty(t, page.Payments)
	assert.False(t, page.More)
	assert.False(t, page.CursorExpired)

	require.Len(t, page.Documents, 3)
	assert.Equal(t, testInvoice, page.Documents[0].ExternalID)
	assert.Equal(t, services.AccountingChangeUpsert, page.Documents[0].Operation)
	assert.Equal(t, []accountingsync.SyncObjectType{
		accountingsync.SyncObjectInvoice, accountingsync.SyncObjectDebitMemo,
	}, page.Documents[0].ObjectTypes)
	assert.Equal(t, services.AccountingChangeVoid, page.Documents[1].Operation)
	assert.Equal(t, []accountingsync.SyncObjectType{
		accountingsync.SyncObjectCarrierBill, accountingsync.SyncObjectDriverBill,
	}, page.Documents[2].ObjectTypes)

	require.Len(t, page.References, 5)
	kinds := make(map[accountingsync.ReferenceKind]int, 2)
	for _, ref := range page.References {
		kinds[ref.Object.Kind]++
		assert.False(t, ref.Deleted)
	}
	assert.Equal(t, 3, kinds[accountingsync.ReferenceKindAccount])
	assert.Equal(t, 2, kinds[accountingsync.ReferenceKindCustomer])

	assert.Equal(t, "2026-10-06T12:04:59Z", page.NextCursor)
	for _, call := range fake.all() {
		if call.Path == "/generalLedgerSetup" {
			continue
		}
		assert.Equal(t, "lastModifiedDateTime gt 2026-10-06T12:00:00Z", call.filter(), call.Path)
	}
	assert.Zero(t, fake.count(http.MethodGet, "/paymentTerms"))
	assert.Zero(t, fake.count(http.MethodGet, "/customerPayments"))
}

func TestReadChangesPagesAcrossTargetsWhenAReadIsLarge(t *testing.T) {
	t.Parallel()

	many := make([]docSpec, 0, businesscentral.MaxPageSize)
	for idx := range businesscentral.MaxPageSize {
		many = append(many, docSpec{
			id:       fmt.Sprintf("99999999-aaaa-4bbb-8ccc-%012d", idx),
			status:   "Open",
			modified: "2026-10-06T12:03:00Z",
		})
	}
	big := listJSON(t, many...)
	base := changesResponder(t)
	fake := &fakeBC{respond: func(call bcCall) (int, string) {
		if call.Query.Get("$skip") != "" {
			return 0, ""
		}
		if call.is(http.MethodGet, "/salesInvoices") {
			return http.StatusOK, big
		}
		return base(call)
	}}
	conn := testConnector(t, fake)

	first, err := conn.ReadChanges(t.Context(), changesRequest(""))
	require.NoError(t, err)
	assert.True(t, first.More)
	assert.Len(t, first.Documents, businesscentral.MaxPageSize)
	cursor, err := parseChangeCursor(first.NextCursor, pollNow)
	require.NoError(t, err)
	assert.True(t, cursor.paging)
	assert.Equal(t, 1, cursor.index)
	assert.Equal(t, pollNow.Add(10*time.Minute), cursor.started)
	assert.Equal(t, pollNow.Add(10*time.Minute), cursor.since)

	second, err := conn.ReadChanges(t.Context(), changesRequest(first.NextCursor))
	require.NoError(t, err)
	assert.False(t, second.More)
	assert.Len(t, second.Documents, 1)
	assert.Equal(t, "2026-10-06T12:10:00Z", second.NextCursor)
	assert.Equal(t, 2, fake.count(http.MethodGet, "/salesInvoices"))
}

func TestReadChangesWithNothingToReadKeepsTheCursor(t *testing.T) {
	t.Parallel()

	fake := &fakeBC{}
	conn := testConnector(t, fake)
	page, err := conn.ReadChanges(t.Context(), &services.ReadAccountingChangesRequest{
		Auth:     testAuth(),
		Cursor:   "2026-10-01T00:00:00Z",
		Payments: true,
	})
	require.NoError(t, err)
	assert.Equal(t, "2026-10-01T00:00:00Z", page.NextCursor)
	assert.Empty(t, fake.all())

	_, err = conn.ReadChanges(t.Context(), changesRequest("p|bad"))
	require.ErrorIs(t, err, errChangeCursor)
}
