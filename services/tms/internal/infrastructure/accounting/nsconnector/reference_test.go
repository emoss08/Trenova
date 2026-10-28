package nsconnector

import (
	"net/http"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func listRequest(
	kind accountingsync.ReferenceKind,
	realm string,
) *accountingconnectorports.AccountingReferencePageRequest {
	return &accountingconnectorports.AccountingReferencePageRequest{
		RealmID:       realm,
		AccessToken:   "access-token",
		Kind:          kind,
		StartPosition: 1,
		PageSize:      1000,
	}
}

func referenceResponder(t *testing.T) func(call nsCall) reply {
	return func(call nsCall) reply {
		switch {
		case call.queries("FROM account a"):
			return fixtureReply(t, http.StatusOK, "accounts.json")
		case call.queries("FROM item i"):
			return fixtureReply(t, http.StatusOK, "items.json")
		case call.queries("FROM customer c"):
			return fixtureReply(t, http.StatusOK, "customers.json")
		case call.queries("FROM vendor v"):
			return fixtureReply(t, http.StatusOK, "vendors.json")
		case call.queries("FROM term tm"):
			return fixtureReply(t, http.StatusOK, "terms.json")
		}
		return reply{}
	}
}

func TestListReferenceReadsAccountsKeysetByID(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{respond: referenceResponder(t)}
	conn := testConnector(t, fake)
	assert.Equal(t, 1000, conn.MaxReferencePageSize())
	page, err := conn.ListReference(
		t.Context(),
		listRequest(accountingsync.ReferenceKindAccount, testRef().String()),
	)
	require.NoError(t, err)
	require.Len(t, page.Objects, 4)
	assert.Equal(t, 96, page.NextStart)

	bank := page.Objects[0]
	assert.Equal(t, "10", bank.ExternalID)
	assert.Equal(t, "Operating Account", bank.Name)
	assert.Equal(t, "Cash : Operating Account", bank.FullyQualifiedName)
	assert.Equal(t, "1000", bank.Number)
	assert.Equal(t, accountingsync.AccountClassBank, bank.AccountClass)
	assert.Equal(t, "9", bank.ParentExternalID)
	assert.True(t, bank.Active)
	require.NotNil(t, bank.ProviderUpdatedAt)
	assert.Equal(t, accountingsync.AccountClassIncome, page.Objects[1].AccountClass)
	assert.Equal(t, accountingsync.AccountClassCostOfSales, page.Objects[2].AccountClass)
	assert.False(t, page.Objects[3].Active, "a statistical account is not a line target")

	call, ok := fake.query("FROM account a")
	require.True(t, ok)
	assert.Contains(t, call.SQL, "a.id >= ?")
	assert.True(t, strings.HasSuffix(call.SQL, "ORDER BY a.id"))
	assert.Equal(t, []any{float64(1)}, call.Params)
	assert.Equal(t, "1000", call.Query.Get("limit"))
}

func TestListReferenceFiltersPartiesAndItemsBySubsidiary(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{respond: referenceResponder(t)}
	conn := testConnector(t, fake)

	items, err := conn.ListReference(
		t.Context(),
		listRequest(accountingsync.ReferenceKindItem, testRef().String()),
	)
	require.NoError(t, err)
	require.Len(t, items.Objects, 2)
	assert.Zero(t, items.NextStart)
	assert.Equal(t, "Linehaul", items.Objects[0].Name)
	assert.Equal(t, "40", items.Objects[0].IncomeAccountExternalID)
	assert.Equal(t, "SHORTPAY", items.Objects[1].Name)
	assert.False(t, items.Objects[1].Active)

	customers, err := conn.ListReference(
		t.Context(),
		listRequest(accountingsync.ReferenceKindCustomer, testRef().String()),
	)
	require.NoError(t, err)
	require.Len(t, customers.Objects, 2)
	assert.Equal(t, "Acme Freight", customers.Objects[0].Name)
	assert.Equal(t, "USD", customers.Objects[0].CurrencyCode)
	assert.Equal(t, "Jane Shipper", customers.Objects[1].Name)
	assert.False(t, customers.Objects[1].Active)

	vendors, err := conn.ListReference(
		t.Context(),
		listRequest(accountingsync.ReferenceKindVendor, testRef().String()),
	)
	require.NoError(t, err)
	require.Len(t, vendors.Objects, 1)
	assert.True(t, vendors.Objects[0].Is1099)
	assert.Equal(t, accountingsync.ReferenceKindVendor, vendors.Objects[0].Kind)

	for fragment, table := range map[string]string{
		"FROM item i":     "itemsubsidiarymap",
		"FROM customer c": "customersubsidiaryrelationship",
		"FROM vendor v":   "vendorsubsidiaryrelationship",
	} {
		call, ok := fake.query(fragment)
		require.True(t, ok, fragment)
		assert.Contains(t, call.SQL, table)
		assert.Equal(t, []any{float64(1), testSubsidiary}, call.Params, fragment)
	}
	itemCall, _ := fake.query("FROM item i")
	assert.Contains(t, itemCall.SQL, "i.itemtype IN ('Service', 'NonInvtPart', 'OthCharge')")

	plain := &fakeNS{respond: referenceResponder(t)}
	conn = testConnector(t, plain)
	_, err = conn.ListReference(
		t.Context(),
		listRequest(accountingsync.ReferenceKindCustomer, testAccount),
	)
	require.NoError(t, err)
	call, ok := plain.query("FROM customer c")
	require.True(t, ok)
	assert.NotContains(t, call.SQL, "customersubsidiaryrelationship")
	assert.Equal(t, []any{float64(1)}, call.Params)
}

func TestListReferenceReadsTermsAndSkipsPaymentMethods(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{respond: referenceResponder(t)}
	conn := testConnector(t, fake)
	terms, err := conn.ListReference(
		t.Context(),
		listRequest(accountingsync.ReferenceKindTerm, testRef().String()),
	)
	require.NoError(t, err)
	require.Len(t, terms.Objects, 2)
	require.NotNil(t, terms.Objects[0].DueDays)
	assert.Equal(t, 30, *terms.Objects[0].DueDays)
	assert.Nil(t, terms.Objects[1].DueDays)

	methods, err := conn.ListReference(
		t.Context(),
		listRequest(accountingsync.ReferenceKindPaymentMethod, testRef().String()),
	)
	require.NoError(t, err)
	assert.Empty(t, methods.Objects)
	_, err = conn.ListReference(
		t.Context(),
		listRequest(accountingsync.ReferenceKind("Bogus"), testRef().String()),
	)
	require.Error(t, err)
	assert.Equal(t, 1, len(fake.all()))
}

func TestAccountClassOfCoversNetSuiteAccountTypes(t *testing.T) {
	t.Parallel()

	for accountType, want := range map[string]accountingsync.AccountClass{
		"Bank":         accountingsync.AccountClassBank,
		"AcctRec":      accountingsync.AccountClassReceivable,
		"UnbilledRec":  accountingsync.AccountClassReceivable,
		"AcctPay":      accountingsync.AccountClassPayable,
		"OthCurrAsset": accountingsync.AccountClassAsset,
		"FixedAsset":   accountingsync.AccountClassAsset,
		"CredCard":     accountingsync.AccountClassLiability,
		"LongTermLiab": accountingsync.AccountClassLiability,
		"Equity":       accountingsync.AccountClassEquity,
		"OthIncome":    accountingsync.AccountClassOtherIncome,
		"Expense":      accountingsync.AccountClassExpense,
		"OthExpense":   accountingsync.AccountClassOtherExpense,
		"NonPosting":   "",
		"Stat":         "",
	} {
		assert.Equal(t, want, AccountClassOf(accountType), accountType)
	}
}

func createRequest(
	kind accountingsync.ReferenceKind,
) *accountingconnectorports.AccountingCreateReferenceRequest {
	return &accountingconnectorports.AccountingCreateReferenceRequest{
		RealmID:     testRef().String(),
		AccessToken: "access-token",
		RequestID:   testRequestID,
		Kind:        kind,
	}
}

func TestCreateCustomerUpsertsByRequestIDWithTheSubsidiary(t *testing.T) {
	t.Parallel()

	created := false
	fake := &fakeNS{respond: func(call nsCall) reply {
		switch {
		case call.is(http.MethodPut, eidPath(recordCustomer, testRequestID)):
			created = true
			return upserted(recordCustomer, "900")
		case call.queries("FROM customer c") && created:
			return reply{status: http.StatusOK, body: rowsJSON(t, false, map[string]any{
				"id": "900", "entityid": "CUST-900", "companyname": "Northwind Logistics",
				"isinactive": "F", "externalid": testRequestID,
			})}
		}
		return reply{}
	}}
	conn := testConnector(t, fake)
	req := createRequest(accountingsync.ReferenceKindCustomer)
	req.Party = &accountingconnectorports.AccountingPartyDraft{
		DisplayName:  "Northwind",
		CompanyName:  "Northwind Logistics",
		Email:        "ap@northwind.example",
		AddressLine1: "1 Main St",
		City:         "Dallas",
		State:        "TX",
		PostalCode:   "75201",
		Country:      "us",
	}
	obj, err := conn.CreateReference(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, "900", obj.ExternalID)
	assert.Equal(t, "Northwind Logistics", obj.Name)
	assert.Equal(t, accountingsync.ReferenceKindCustomer, obj.Kind)

	check, ok := fake.query("UPPER(c.entityid)")
	require.True(t, ok)
	assert.Equal(t, []any{"Northwind Logistics", "Northwind Logistics"}, check.Params)
	put, ok := fake.find(http.MethodPut, eidPath(recordCustomer, testRequestID))
	require.True(t, ok)
	assert.Equal(t, "Northwind Logistics", put.Body["companyName"])
	assert.Equal(t, false, put.Body["isPerson"])
	assert.Equal(t, testSubsidiary, refID(t, put.Body, "subsidiary"))
	address := items(t, put.Body, "addressBook")[0]["addressBookAddress"].(map[string]any)
	assert.Equal(t, "Dallas", address["city"])
	assert.Equal(t, "US", address["country"].(map[string]any)["id"])
}

func TestCreatePartyRefusesADuplicateNameButResumesItsOwnRetry(t *testing.T) {
	t.Parallel()

	owner := "trn-someone-else"
	fake := &fakeNS{respond: func(call nsCall) reply {
		switch {
		case call.queries("UPPER(v.entityid)"):
			return reply{status: http.StatusOK, body: rowsJSON(t, false, map[string]any{
				"id": "512", "entityid": "Swift Carriers LLC", "externalid": owner,
			})}
		case call.is(http.MethodPut, eidPath(recordVendor, testRequestID)):
			return upserted(recordVendor, "512")
		case call.queries("FROM vendor v"):
			return fixtureReply(t, http.StatusOK, "vendors.json")
		}
		return reply{}
	}}
	conn := testConnector(t, fake)
	req := createRequest(accountingsync.ReferenceKindVendor)
	req.Party = &accountingconnectorports.AccountingPartyDraft{
		DisplayName: "Swift Carriers LLC",
		Is1099:      true,
	}
	_, err := conn.CreateReference(t.Context(), req)
	require.ErrorIs(t, err, errDuplicateName)
	assert.True(t, conn.IsDuplicateName(err))
	assert.Equal(t, accountingsync.SyncErrorDuplicate, conn.ClassifyDocumentError(err).Category)
	assert.Zero(t, fake.count(http.MethodPut, eidPath(recordVendor, testRequestID)))

	owner = testRequestID
	obj, err := conn.CreateReference(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, "512", obj.ExternalID)
	put, ok := fake.find(http.MethodPut, eidPath(recordVendor, testRequestID))
	require.True(t, ok)
	assert.Equal(t, true, put.Body["is1099Eligible"])
}

func TestCreateItemFallsBackToANonInventorySaleItem(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{respond: func(call nsCall) reply {
		switch {
		case call.is(http.MethodPut, eidPath(recordServiceItem, testRequestID)):
			return fixtureReply(t, http.StatusBadRequest, "error_invalid_request.json")
		case call.is(http.MethodPut, eidPath(recordNonInvItem, testRequestID)):
			return upserted(recordNonInvItem, "88")
		case call.queries("i.id = ?"):
			return reply{status: http.StatusOK, body: rowsJSON(t, false, map[string]any{
				"id": "88", "itemid": "Detention", "itemtype": "NonInvtPart", "isinactive": "F",
			})}
		}
		return reply{}
	}}
	conn := testConnector(t, fake)
	req := createRequest(accountingsync.ReferenceKindItem)
	req.Item = &accountingconnectorports.AccountingItemDraft{
		Name:            "  Detention\n",
		Description:     "Detention charges",
		IncomeAccountID: "40",
	}
	obj, err := conn.CreateReference(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, "88", obj.ExternalID)
	assert.Equal(t, "Detention", obj.Name)

	put, ok := fake.find(http.MethodPut, eidPath(recordNonInvItem, testRequestID))
	require.True(t, ok)
	assert.Equal(t, "Detention", put.Body["itemId"])
	assert.Equal(t, "40", refID(t, put.Body, "incomeAccount"))
	assert.Equal(t, testSubsidiary, items(t, put.Body, "subsidiary")[0]["id"])
	assert.Equal(t, 1, fake.count(http.MethodPut, eidPath(recordServiceItem, testRequestID)))
}

func TestCreateReferenceRefusesKindsNetSuiteDoesNotTakeFromTrenova(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{}
	conn := testConnector(t, fake)
	for _, kind := range []accountingsync.ReferenceKind{
		accountingsync.ReferenceKindAccount,
		accountingsync.ReferenceKindTerm,
		accountingsync.ReferenceKindPaymentMethod,
		accountingsync.ReferenceKind("Bogus"),
	} {
		_, err := conn.CreateReference(t.Context(), createRequest(kind))
		require.Error(t, err, kind)
	}
	_, err := conn.CreateReference(t.Context(), createRequest(accountingsync.ReferenceKindItem))
	require.ErrorIs(t, err, errItemDraftRequired)
	_, err = conn.CreateReference(t.Context(), createRequest(accountingsync.ReferenceKindCustomer))
	require.ErrorIs(t, err, errPartyDraftRequired)
	req := createRequest(accountingsync.ReferenceKindCustomer)
	req.Party = &accountingconnectorports.AccountingPartyDraft{DisplayName: " "}
	_, err = conn.CreateReference(t.Context(), req)
	require.ErrorIs(t, err, errNameRequired)
	assert.Equal(t, accountingsync.SyncErrorValidation, conn.ClassifyDocumentError(err).Category)
	assert.Empty(t, fake.all())
}

func TestSanitizeNameFitsNetSuiteLimits(t *testing.T) {
	t.Parallel()

	conn := testConnector(t, &fakeNS{})
	long := strings.Repeat("x", 120)
	assert.Len(t, conn.SanitizeName(accountingsync.ReferenceKindItem, long), maxItemNameLength)
	assert.Len(t, conn.SanitizeName(accountingsync.ReferenceKindCustomer, long), maxPartyNameLength)
	assert.Equal(t, "a b", conn.SanitizeName(accountingsync.ReferenceKindVendor, " a\tb "))
}

func TestUpsertCustomerAndVendorPatchExistingRecords(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{respond: func(call nsCall) reply {
		if call.is(http.MethodPut, eidPath(recordVendor, testRequestID)) {
			return upserted(recordVendor, "513")
		}
		return reply{}
	}}
	conn := testConnector(t, fake)
	result, err := conn.UpsertCustomer(
		t.Context(),
		&accountingconnectorports.AccountingCustomerDocument{
			Auth:       testAuth(),
			RequestID:  testRequestID,
			ExternalID: testCustomer,
			Party: accountingconnectorports.AccountingPartyDraft{
				DisplayName: "Acme Freight",
				City:        "Austin",
			},
		},
	)
	require.NoError(t, err)
	assert.Equal(t, testCustomer, result.ExternalID)
	assert.Equal(t, "Acme Freight", result.DocNumber)
	patch, ok := fake.find(http.MethodPatch, recordPath(recordCustomer, testCustomer))
	require.True(t, ok)
	assert.Equal(t, "addressBook", patch.Query.Get("replace"))
	assert.NotContains(t, patch.Body, "subsidiary")

	result, err = conn.UpsertVendor(t.Context(), &accountingconnectorports.AccountingVendorDocument{
		Auth:      testAuth(),
		RequestID: testRequestID,
		Party:     accountingconnectorports.AccountingPartyDraft{DisplayName: "Rapid Haul"},
	})
	require.NoError(t, err)
	assert.Equal(t, "513", result.ExternalID)
	put, ok := fake.find(http.MethodPut, eidPath(recordVendor, testRequestID))
	require.True(t, ok)
	assert.Equal(t, testSubsidiary, refID(t, put.Body, "subsidiary"))
	assert.Equal(t, false, put.Body["is1099Eligible"])
}
