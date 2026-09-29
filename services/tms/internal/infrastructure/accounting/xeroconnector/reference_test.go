package xeroconnector

import (
	"net/http"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/xero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func listReference(
	t *testing.T,
	conn *Connector,
	kind accountingsync.ReferenceKind,
	start int,
) *services.AccountingReferencePage {
	t.Helper()
	page, err := conn.ListReference(t.Context(), &services.AccountingReferencePageRequest{
		RealmID:       testTenant,
		AccessToken:   "token",
		Kind:          kind,
		StartPosition: start,
		PageSize:      conn.MaxReferencePageSize(),
	})
	require.NoError(t, err)
	return page
}

func TestListAccountsSetsTheNeutralAccountClass(t *testing.T) {
	t.Parallel()

	conn := testConnector(t, &fakeXero{})
	page := listReference(t, conn, accountingsync.ReferenceKindAccount, 1)
	assert.Zero(t, page.NextStart)
	require.Len(t, page.Objects, 7)

	byName := make(map[string]*accountingsync.AccountingReferenceObject, len(page.Objects))
	for _, obj := range page.Objects {
		byName[obj.Name] = obj
	}
	cases := []struct {
		name   string
		class  accountingsync.AccountClass
		number string
		active bool
	}{
		{"Business Bank", accountingsync.AccountClassBank, "090", true},
		{"Accounts Receivable", accountingsync.AccountClassReceivable, "610", true},
		{"Accounts Payable", accountingsync.AccountClassPayable, "800", true},
		{"Freight Revenue", accountingsync.AccountClassIncome, "200", true},
		{"Short Pay Write-off", accountingsync.AccountClassExpense, "499", true},
		{"Purchased Transportation", accountingsync.AccountClassCostOfSales, "310", false},
		{"Clearing", accountingsync.AccountClassLiability, "", true},
	}
	for _, tc := range cases {
		obj, ok := byName[tc.name]
		require.True(t, ok, tc.name)
		assert.Equal(t, accountingsync.ReferenceKindAccount, obj.Kind, tc.name)
		assert.Equal(t, tc.class, obj.AccountClass, tc.name)
		assert.True(t, obj.AccountClass == "" || obj.AccountClass.IsValid(), tc.name)
		assert.Equal(t, tc.number, obj.Number, tc.name)
		assert.Equal(t, tc.active, obj.Active, tc.name)
	}
	assert.Equal(t, "DEBTORS", byName["Accounts Receivable"].AccountSubType)
	assert.Equal(t, testBank, byName["Business Bank"].ExternalID)
	require.NotNil(t, byName["Business Bank"].ProviderUpdatedAt)
}

func TestAccountClassOfEveryXeroType(t *testing.T) {
	t.Parallel()

	cases := map[string]accountingsync.AccountClass{
		"SALES":       accountingsync.AccountClassIncome,
		"OTHERINCOME": accountingsync.AccountClassOtherIncome,
		"OVERHEADS":   accountingsync.AccountClassExpense,
		"DEPRECIATN":  accountingsync.AccountClassExpense,
		"FIXED":       accountingsync.AccountClassAsset,
		"PREPAYMENT":  accountingsync.AccountClassAsset,
		"INVENTORY":   accountingsync.AccountClassAsset,
		"NONCURRENT":  accountingsync.AccountClassAsset,
		"TERMLIAB":    accountingsync.AccountClassLiability,
		"LIABILITY":   accountingsync.AccountClassLiability,
		"EQUITY":      accountingsync.AccountClassEquity,
		"unknown":     "",
	}
	for typ, want := range cases {
		assert.Equal(t, want, AccountClassOf(&xero.Account{Type: typ}), typ)
	}
	assert.Equal(t, accountingsync.AccountClassReceivable,
		AccountClassOf(&xero.Account{Type: "CURRENT", SystemAccount: "DEBTORS"}))
}

func TestListItemsResolvesTheSalesAccount(t *testing.T) {
	t.Parallel()

	fake := &fakeXero{respond: func(call xeroCall) (int, string) {
		if call.Path == "/Items" {
			return http.StatusOK, `{"Items":[
			 {"ItemID":"9a1f0b2e-0000-4000-8000-000000000001","Code":"DET","Name":"Detention","Description":"Detention time","IsSold":true,"SalesDetails":{"AccountCode":"200"}},
			 {"ItemID":"9a1f0b2e-0000-4000-8000-000000000002","Code":"LUMP","IsSold":true}
			]}`
		}
		return 0, ""
	}}
	conn := testConnector(t, fake)
	page := listReference(t, conn, accountingsync.ReferenceKindItem, 1)
	require.Len(t, page.Objects, 2)
	assert.Equal(t, "DET", page.Objects[0].Number)
	assert.Equal(t, testRevenue, page.Objects[0].IncomeAccountExternalID)
	assert.True(t, page.Objects[0].Usable())
	assert.Equal(t, "LUMP", page.Objects[1].Name)
	assert.Empty(t, page.Objects[1].IncomeAccountExternalID)
}

func TestListContactsSplitsCustomersAndSuppliers(t *testing.T) {
	t.Parallel()

	fake := &fakeXero{respond: func(call xeroCall) (int, string) {
		if call.Path != "/Contacts" {
			return 0, ""
		}
		if call.Query.Get("page") == "1" {
			return http.StatusOK, `{"pagination":{"page":1,"pageSize":1000,"pageCount":2,"itemCount":1003},"Contacts":[
			 {"ContactID":"` + testCustomer + `","Name":"Globex","ContactStatus":"ACTIVE","IsCustomer":true,"EmailAddress":"ap@globex.test","Addresses":[{"AddressType":"POBOX","AddressLine1":"PO Box 1","City":"Austin"},{"AddressType":"STREET","AddressLine1":"1 Main","City":"Dallas","Region":"TX","PostalCode":"75001"}]},
			 {"ContactID":"` + testVendor + `","Name":"Hauler","ContactStatus":"ARCHIVED","IsSupplier":true},
			 {"ContactID":"c0ffee00-0000-4000-8000-000000000003","Name":"Brand New","ContactStatus":"ACTIVE"}
			]}`
		}
		return http.StatusOK, `{"pagination":{"page":2,"pageSize":1000,"pageCount":2,"itemCount":1003},"Contacts":[]}`
	}}
	conn := testConnector(t, fake)

	customers := listReference(t, conn, accountingsync.ReferenceKindCustomer, 1)
	assert.Equal(t, 2, customers.NextStart)
	require.Len(t, customers.Objects, 2)
	assert.Equal(t, testCustomer, customers.Objects[0].ExternalID)
	assert.Equal(t, accountingsync.ReferenceKindCustomer, customers.Objects[0].Kind)
	assert.Equal(t, "1 Main", customers.Objects[0].AddressLine1)
	assert.Equal(t, "TX", customers.Objects[0].State)
	assert.Equal(t, "ap@globex.test", customers.Objects[0].Email)
	assert.Equal(t, "Brand New", customers.Objects[1].Name)

	vendors := listReference(t, conn, accountingsync.ReferenceKindVendor, 1)
	require.Len(t, vendors.Objects, 2)
	assert.Equal(t, testVendor, vendors.Objects[0].ExternalID)
	assert.False(t, vendors.Objects[0].Active)
	assert.Equal(t, "Brand New", vendors.Objects[1].Name)

	last := listReference(t, conn, accountingsync.ReferenceKindVendor, 2)
	assert.Zero(t, last.NextStart)
	call, ok := fake.find(http.MethodGet, "/Contacts")
	require.True(t, ok)
	assert.Equal(t, "true", call.Query.Get("includeArchived"))
}

func TestListReferenceHasNoTermsOrPaymentMethods(t *testing.T) {
	t.Parallel()

	fake := &fakeXero{}
	conn := testConnector(t, fake)
	page := listReference(t, conn, accountingsync.ReferenceKindTerm, 1)
	assert.Empty(t, page.Objects)
	assert.Empty(t, fake.all())
}

func TestCreateItemUsesTheChargeCodeAndTheIncomeAccountCode(t *testing.T) {
	t.Parallel()

	fake := &fakeXero{respond: func(call xeroCall) (int, string) {
		if call.Method == http.MethodPut && call.Path == "/Items" {
			return http.StatusOK, `{"Items":[{"ItemID":"9a1f0b2e-0000-4000-8000-000000000009","Code":"DET","Name":"Detention","IsSold":true,"SalesDetails":{"AccountCode":"200"}}]}`
		}
		return 0, ""
	}}
	conn := testConnector(t, fake)

	obj, err := conn.CreateReference(t.Context(), &services.AccountingCreateReferenceRequest{
		RealmID:     testTenant,
		AccessToken: "token",
		RequestID:   testRequestID,
		Kind:        accountingsync.ReferenceKindItem,
		Item: &services.AccountingItemDraft{
			Name:            "Detention",
			Description:     "Detention time",
			Sku:             "DET",
			IncomeAccountID: testRevenue,
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "9a1f0b2e-0000-4000-8000-000000000009", obj.ExternalID)
	assert.Equal(t, testRevenue, obj.IncomeAccountExternalID)

	call, ok := fake.find(http.MethodPut, "/Items")
	require.True(t, ok)
	assert.Equal(t, testRequestID, call.Key)
	item := firstDoc(t, call.Body, "Items")
	assert.Equal(t, "DET", item["Code"])
	assert.Equal(t, map[string]any{"AccountCode": "200"}, item["SalesDetails"])
}

func TestCreateContactsAndDuplicateNames(t *testing.T) {
	t.Parallel()

	fake := &fakeXero{respond: func(call xeroCall) (int, string) {
		if call.Method != http.MethodPut || call.Path != "/Contacts" {
			return 0, ""
		}
		if call.Key == "dup" {
			return http.StatusBadRequest, `{"Type":"ValidationException","Elements":[{"ValidationErrors":[{"Message":"The contact name Globex is already assigned to another contact. The contact name must be unique across all active contacts."}]}]}`
		}
		return http.StatusOK, `{"Contacts":[{"ContactID":"` + testCustomer + `","Name":"Globex","ContactStatus":"ACTIVE"}]}`
	}}
	conn := testConnector(t, fake)

	party := &services.AccountingPartyDraft{
		DisplayName:  "Globex\n Shipping",
		Email:        "ap@globex.test",
		AddressLine1: "1 Main",
		City:         "Dallas",
		State:        "TX",
	}
	obj, err := conn.CreateReference(t.Context(), &services.AccountingCreateReferenceRequest{
		RealmID: testTenant, AccessToken: "token", RequestID: "req-1",
		Kind: accountingsync.ReferenceKindVendor, Party: party,
	})
	require.NoError(t, err)
	assert.Equal(t, accountingsync.ReferenceKindVendor, obj.Kind)
	call, ok := fake.find(http.MethodPut, "/Contacts")
	require.True(t, ok)
	contact := firstDoc(t, call.Body, "Contacts")
	assert.Equal(t, "Globex Shipping", contact["Name"])
	addresses, ok := contact["Addresses"].([]any)
	require.True(t, ok)
	require.Len(t, addresses, 1)

	_, err = conn.CreateReference(t.Context(), &services.AccountingCreateReferenceRequest{
		RealmID: testTenant, AccessToken: "token", RequestID: "dup",
		Kind: accountingsync.ReferenceKindCustomer, Party: party,
	})
	require.Error(t, err)
	assert.True(t, conn.IsDuplicateName(err))
	assert.Equal(t, accountingsync.SyncErrorDuplicate, conn.ClassifyDocumentError(err).Category)

	_, err = conn.CreateReference(t.Context(), &services.AccountingCreateReferenceRequest{
		RealmID: testTenant, AccessToken: "token", RequestID: "x",
		Kind: accountingsync.ReferenceKindAccount,
	})
	require.Error(t, err)
}

func TestSanitizeName(t *testing.T) {
	t.Parallel()

	conn := testConnector(t, &fakeXero{})
	assert.Equal(t, "A B", conn.SanitizeName(accountingsync.ReferenceKindCustomer, " A\t\nB "))
	long := strings.Repeat("x", 80)
	assert.Len(t, conn.SanitizeName(accountingsync.ReferenceKindItem, long), xero.MaxItemNameLength)
	assert.Len(t, conn.SanitizeName(accountingsync.ReferenceKindVendor, long), 80)
	assert.Equal(t, "Line haul", itemCode(&services.AccountingItemDraft{Name: "Line haul"}))
}
