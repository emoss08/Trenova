package bcconnector

import (
	"net/http"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/businesscentral"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func referenceResponder(t *testing.T) func(call bcCall) (int, string) {
	return func(call bcCall) (int, string) {
		if call.Method != http.MethodGet {
			return 0, ""
		}
		switch call.Path {
		case "/accounts":
			return http.StatusOK, readFixture(t, "accounts.json")
		case "/items":
			return http.StatusOK, readFixture(t, "items.json")
		case "/customers":
			return http.StatusOK, readFixture(t, "customers.json")
		case "/vendors":
			return http.StatusOK, readFixture(t, "vendors.json")
		case "/paymentTerms":
			return http.StatusOK, readFixture(t, "payment_terms.json")
		}
		return 0, ""
	}
}

func listKind(
	t *testing.T,
	conn *Connector,
	kind accountingsync.ReferenceKind,
) []*accountingsync.AccountingReferenceObject {
	t.Helper()
	page, err := conn.ListReference(t.Context(), &services.AccountingReferencePageRequest{
		RealmID:       testRef().String(),
		AccessToken:   "access-token",
		Kind:          kind,
		StartPosition: 1,
		PageSize:      conn.MaxReferencePageSize(),
	})
	require.NoError(t, err)
	assert.Zero(t, page.NextStart)
	return page.Objects
}

func TestListReferenceReadsEveryKind(t *testing.T) {
	t.Parallel()

	conn := testConnector(t, &fakeBC{respond: referenceResponder(t)})
	assert.Equal(t, businesscentral.MaxPageSize, conn.MaxReferencePageSize())

	accounts := listKind(t, conn, accountingsync.ReferenceKindAccount)
	require.Len(t, accounts, 3)
	assert.Equal(t, testBank, accounts[0].ExternalID)
	assert.Equal(t, "10100", accounts[0].Number)
	assert.Equal(t, accountingsync.AccountClassBank, accounts[0].AccountClass)
	assert.True(t, accounts[0].Active)
	assert.False(t, accounts[1].Active)
	assert.Equal(t, accountingsync.AccountClassIncome, accounts[1].AccountClass)
	assert.False(t, accounts[2].Active)
	assert.Empty(t, accounts[2].AccountClass)

	items := listKind(t, conn, accountingsync.ReferenceKindItem)
	require.Len(t, items, 2)
	assert.Equal(t, "Linehaul", items[0].Name)
	assert.Equal(t, "Service", items[0].ItemType)
	assert.True(t, items[0].Active)
	assert.False(t, items[1].Active)

	customers := listKind(t, conn, accountingsync.ReferenceKindCustomer)
	require.Len(t, customers, 2)
	assert.Equal(t, accountingsync.ReferenceKindCustomer, customers[0].Kind)
	assert.Equal(t, "O'Brien Logistics", customers[0].Name)
	assert.Equal(t, "Atlanta", customers[0].City)
	assert.True(t, customers[0].Active)
	assert.False(t, customers[1].Active)
	assert.Equal(t, "CAD", customers[1].CurrencyCode)

	vendors := listKind(t, conn, accountingsync.ReferenceKindVendor)
	require.Len(t, vendors, 1)
	assert.Equal(t, accountingsync.ReferenceKindVendor, vendors[0].Kind)
	assert.False(t, vendors[0].Active)

	terms := listKind(t, conn, accountingsync.ReferenceKindTerm)
	require.Len(t, terms, 1)
	assert.Equal(t, "Net 30 days", terms[0].Name)
	require.NotNil(t, terms[0].DueDays)
	assert.Equal(t, 30, *terms[0].DueDays)

	assert.Empty(t, listKind(t, conn, accountingsync.ReferenceKindPaymentMethod))
	_, err := conn.ListReference(t.Context(), &services.AccountingReferencePageRequest{
		RealmID: testRef().String(), AccessToken: "x", Kind: "Nope",
	})
	require.Error(t, err)
}

func TestAccountClassOf(t *testing.T) {
	t.Parallel()

	cases := []struct {
		category string
		sub      string
		want     accountingsync.AccountClass
	}{
		{businesscentral.CategoryIncome, "Income, Services", accountingsync.AccountClassIncome},
		{businesscentral.CategoryCostOfGoodsSold, "", accountingsync.AccountClassCostOfSales},
		{businesscentral.CategoryExpense, "Rent Expense", accountingsync.AccountClassExpense},
		{businesscentral.CategoryAssets, "Accounts Receivable", accountingsync.AccountClassReceivable},
		{businesscentral.CategoryAssets, "Cash", accountingsync.AccountClassBank},
		{businesscentral.CategoryAssets, "Bank Accounts", accountingsync.AccountClassBank},
		{businesscentral.CategoryAssets, "Inventory", accountingsync.AccountClassAsset},
		{businesscentral.CategoryLiabilities, "Accounts Payable", accountingsync.AccountClassPayable},
		{businesscentral.CategoryLiabilities, "Long Term Liabilities", accountingsync.AccountClassLiability},
		{businesscentral.CategoryEquity, "Retained Earnings", accountingsync.AccountClassEquity},
		{"", "", ""},
	}
	for _, tc := range cases {
		got := AccountClassOf(&businesscentral.Account{Category: tc.category, SubCategory: tc.sub})
		assert.Equal(t, tc.want, got, tc.category+"/"+tc.sub)
	}
	assert.True(t, accountObjectOf(&businesscentral.Account{
		AccountType: businesscentral.AccountTypePosting, DirectPosting: false,
	}).Active)
	assert.False(t, accountObjectOf(&businesscentral.Account{
		AccountType: businesscentral.AccountTypePosting, Blocked: true,
	}).Active)
	assert.False(t, accountObjectOf(&businesscentral.Account{AccountType: "Heading"}).Active)
}

func TestCreateReferenceLooksForTheNameFirst(t *testing.T) {
	t.Parallel()

	fake := &fakeBC{respond: func(call bcCall) (int, string) {
		switch {
		case call.is(http.MethodPost, "/items"):
			return http.StatusCreated, readFixture(t, "create_item.json")
		case call.is(http.MethodPost, "/customers"):
			return http.StatusCreated, readFixture(t, "create_customer.json")
		case call.is(http.MethodGet, "/customers") && strings.Contains(call.filter(), "Northwind"):
			return http.StatusOK, `{"value":[]}`
		}
		return referenceResponder(t)(call)
	}}
	conn := testConnector(t, fake)
	base := services.AccountingCreateReferenceRequest{
		RealmID: testRef().String(), AccessToken: "x", RequestID: testRequestID,
	}

	itemReq := base
	itemReq.Kind = accountingsync.ReferenceKindItem
	itemReq.Item = &services.AccountingItemDraft{Name: " Fuel\nsurcharge "}
	item, err := conn.CreateReference(t.Context(), &itemReq)
	require.NoError(t, err)
	assert.Equal(t, "44444444-5555-4666-8777-888888888883", item.ExternalID)
	created, ok := fake.find(http.MethodPost, "/items")
	require.True(t, ok)
	assert.Equal(t, "Fuel surcharge", created.Body["displayName"])
	assert.Equal(t, "Service", created.Body["type"])

	itemReq.Item = &services.AccountingItemDraft{Name: "linehaul"}
	_, err = conn.CreateReference(t.Context(), &itemReq)
	require.Error(t, err)
	assert.True(t, conn.IsDuplicateName(err))

	partyReq := base
	partyReq.Kind = accountingsync.ReferenceKindCustomer
	partyReq.Party = &services.AccountingPartyDraft{
		DisplayName: "Northwind Traders", Email: "billing@northwind.example",
		City: strings.Repeat("c", 40),
	}
	customer, err := conn.CreateReference(t.Context(), &partyReq)
	require.NoError(t, err)
	assert.Equal(t, "Northwind Traders", customer.Name)
	assert.Equal(t, accountingsync.ReferenceKindCustomer, customer.Kind)
	posted, ok := fake.find(http.MethodPost, "/customers")
	require.True(t, ok)
	assert.Len(t, posted.Body["city"], 30)

	partyReq.Party = &services.AccountingPartyDraft{DisplayName: "O'Brien Logistics"}
	_, err = conn.CreateReference(t.Context(), &partyReq)
	assert.True(t, conn.IsDuplicateName(err))
	assert.True(t, conn.IsDuplicateName(&businesscentral.APIError{
		Status: 400, Code: businesscentral.CodeEntityWithSameKey,
	}))
	assert.False(t, conn.IsDuplicateName(businesscentral.ErrFieldTooLong))

	for _, kind := range []accountingsync.ReferenceKind{
		accountingsync.ReferenceKindAccount, accountingsync.ReferenceKindTerm, "Nope",
	} {
		other := base
		other.Kind = kind
		_, err = conn.CreateReference(t.Context(), &other)
		require.Error(t, err)
	}
	missing := base
	missing.Kind = accountingsync.ReferenceKindVendor
	_, err = conn.CreateReference(t.Context(), &missing)
	require.ErrorIs(t, err, errPartyDraftRequired)
	missing.Kind = accountingsync.ReferenceKindItem
	_, err = conn.CreateReference(t.Context(), &missing)
	require.ErrorIs(t, err, errItemDraftRequired)

	assert.Equal(t, strings.Repeat("a", 100),
		conn.SanitizeName(accountingsync.ReferenceKindCustomer, strings.Repeat("a", 120)))
}

func TestUpsertCustomerCreatesWhenUnmapped(t *testing.T) {
	t.Parallel()

	fake := &fakeBC{respond: func(call bcCall) (int, string) {
		switch {
		case call.is(http.MethodGet, "/customers"):
			return http.StatusOK, `{"value":[]}`
		case call.is(http.MethodPost, "/customers"):
			return http.StatusCreated, readFixture(t, "create_customer.json")
		}
		return 0, ""
	}}
	conn := testConnector(t, fake)
	result, err := conn.UpsertCustomer(t.Context(), &services.AccountingCustomerDocument{
		Auth:  testAuth(),
		Party: services.AccountingPartyDraft{DisplayName: "Northwind Traders"},
	})
	require.NoError(t, err)
	assert.Equal(t, "55555555-6666-4777-8888-999999999993", result.ExternalID)
	assert.Equal(t, "Northwind Traders", result.DocNumber)
}
