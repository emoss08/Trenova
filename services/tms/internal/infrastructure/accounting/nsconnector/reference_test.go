package nsconnector

import (
	"net/http"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/accounting/nsbinding"
	"github.com/emoss08/trenova/internal/infrastructure/accounting/nsrefsync"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingmapping"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func listRequest(
	kind accountingprovider.ReferenceKind,
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
		listRequest(accountingprovider.ReferenceKindAccount, testRef().String()),
	)
	require.NoError(t, err)
	require.Len(t, page.Objects, 4)
	assert.Equal(t, 96, page.NextStart)

	bank := page.Objects[0]
	assert.Equal(t, "10", bank.ExternalID)
	assert.Equal(t, "Operating Account", bank.Name)
	assert.Equal(t, "Cash : Operating Account", bank.FullyQualifiedName)
	assert.Equal(t, "1000", bank.Number)
	assert.Equal(t, accountingmapping.AccountClassBank, bank.AccountClass)
	assert.Equal(t, "9", bank.ParentExternalID)
	assert.True(t, bank.Active)
	require.NotNil(t, bank.ProviderUpdatedAt)
	assert.Equal(t, accountingmapping.AccountClassIncome, page.Objects[1].AccountClass)
	assert.Equal(t, accountingmapping.AccountClassCostOfSales, page.Objects[2].AccountClass)
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
		listRequest(accountingprovider.ReferenceKindItem, testRef().String()),
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
		listRequest(accountingprovider.ReferenceKindCustomer, testRef().String()),
	)
	require.NoError(t, err)
	require.Len(t, customers.Objects, 2)
	assert.Equal(t, "Acme Freight", customers.Objects[0].Name)
	assert.Equal(t, "USD", customers.Objects[0].CurrencyCode)
	assert.Equal(t, "Jane Shipper", customers.Objects[1].Name)
	assert.False(t, customers.Objects[1].Active)

	vendors, err := conn.ListReference(
		t.Context(),
		listRequest(accountingprovider.ReferenceKindVendor, testRef().String()),
	)
	require.NoError(t, err)
	require.Len(t, vendors.Objects, 1)
	assert.True(t, vendors.Objects[0].Is1099)
	assert.Equal(t, accountingprovider.ReferenceKindVendor, vendors.Objects[0].Kind)

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
		listRequest(accountingprovider.ReferenceKindCustomer, testAccount),
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
		listRequest(accountingprovider.ReferenceKindTerm, testRef().String()),
	)
	require.NoError(t, err)
	require.Len(t, terms.Objects, 2)
	require.NotNil(t, terms.Objects[0].DueDays)
	assert.Equal(t, 30, *terms.Objects[0].DueDays)
	assert.Nil(t, terms.Objects[1].DueDays)

	methods, err := conn.ListReference(
		t.Context(),
		listRequest(accountingprovider.ReferenceKindPaymentMethod, testRef().String()),
	)
	require.NoError(t, err)
	assert.Empty(t, methods.Objects)
	_, err = conn.ListReference(
		t.Context(),
		listRequest(accountingprovider.ReferenceKind("Bogus"), testRef().String()),
	)
	require.Error(t, err)
	assert.Equal(t, 1, len(fake.all()))
}

func TestAccountClassOfCoversNetSuiteAccountTypes(t *testing.T) {
	t.Parallel()

	for accountType, want := range map[string]accountingmapping.AccountClass{
		"Bank":         accountingmapping.AccountClassBank,
		"AcctRec":      accountingmapping.AccountClassReceivable,
		"UnbilledRec":  accountingmapping.AccountClassReceivable,
		"AcctPay":      accountingmapping.AccountClassPayable,
		"OthCurrAsset": accountingmapping.AccountClassAsset,
		"FixedAsset":   accountingmapping.AccountClassAsset,
		"CredCard":     accountingmapping.AccountClassLiability,
		"LongTermLiab": accountingmapping.AccountClassLiability,
		"Equity":       accountingmapping.AccountClassEquity,
		"OthIncome":    accountingmapping.AccountClassOtherIncome,
		"Expense":      accountingmapping.AccountClassExpense,
		"OthExpense":   accountingmapping.AccountClassOtherExpense,
		"NonPosting":   "",
		"Stat":         "",
	} {
		assert.Equal(t, want, nsrefsync.AccountClassOf(accountType), accountType)
	}
}

func createRequest(
	kind accountingprovider.ReferenceKind,
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
		case call.is(http.MethodPut, eidPath(nsrefsync.RecordCustomer, testRequestID)):
			created = true
			return upserted(nsrefsync.RecordCustomer, "900")
		case call.queries("FROM customer c") && created:
			return reply{status: http.StatusOK, body: rowsJSON(t, false, map[string]any{
				"id": "900", "entityid": "CUST-900", "companyname": "Northwind Logistics",
				"isinactive": "F", "externalid": testRequestID,
			})}
		}
		return reply{}
	}}
	conn := testConnector(t, fake)
	req := createRequest(accountingprovider.ReferenceKindCustomer)
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
	assert.Equal(t, accountingprovider.ReferenceKindCustomer, obj.Kind)

	check, ok := fake.query("UPPER(c.entityid)")
	require.True(t, ok)
	assert.Equal(t, []any{"Northwind Logistics", "Northwind Logistics"}, check.Params)
	put, ok := fake.find(http.MethodPut, eidPath(nsrefsync.RecordCustomer, testRequestID))
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
		case call.is(http.MethodPut, eidPath(nsrefsync.RecordVendor, testRequestID)):
			return upserted(nsrefsync.RecordVendor, "512")
		case call.queries("FROM vendor v"):
			return fixtureReply(t, http.StatusOK, "vendors.json")
		}
		return reply{}
	}}
	conn := testConnector(t, fake)
	req := createRequest(accountingprovider.ReferenceKindVendor)
	req.Party = &accountingconnectorports.AccountingPartyDraft{
		DisplayName: "Swift Carriers LLC",
		Is1099:      true,
	}
	_, err := conn.CreateReference(t.Context(), req)
	require.ErrorIs(t, err, nsbinding.ErrDuplicateName)
	assert.True(t, conn.IsDuplicateName(err))
	assert.Equal(t, accountingsync.SyncErrorDuplicate, conn.ClassifyDocumentError(err).Category)
	assert.Zero(t, fake.count(http.MethodPut, eidPath(nsrefsync.RecordVendor, testRequestID)))

	owner = testRequestID
	obj, err := conn.CreateReference(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, "512", obj.ExternalID)
	put, ok := fake.find(http.MethodPut, eidPath(nsrefsync.RecordVendor, testRequestID))
	require.True(t, ok)
	assert.Equal(t, true, put.Body["is1099Eligible"])
}

func TestCreateItemFallsBackToANonInventorySaleItem(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{respond: func(call nsCall) reply {
		switch {
		case call.is(http.MethodPut, eidPath(nsrefsync.RecordServiceItem, testRequestID)):
			return fixtureReply(t, http.StatusBadRequest, "error_invalid_request.json")
		case call.is(http.MethodPut, eidPath(nsrefsync.RecordNonInvItem, testRequestID)):
			return upserted(nsrefsync.RecordNonInvItem, "88")
		case call.queries("i.id = ?"):
			return reply{status: http.StatusOK, body: rowsJSON(t, false, map[string]any{
				"id": "88", "itemid": "Detention", "itemtype": "NonInvtPart", "isinactive": "F",
			})}
		}
		return reply{}
	}}
	conn := testConnector(t, fake)
	req := createRequest(accountingprovider.ReferenceKindItem)
	req.Item = &accountingconnectorports.AccountingItemDraft{
		Name:            "  Detention\n",
		Description:     "Detention charges",
		IncomeAccountID: "40",
	}
	obj, err := conn.CreateReference(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, "88", obj.ExternalID)
	assert.Equal(t, "Detention", obj.Name)

	put, ok := fake.find(http.MethodPut, eidPath(nsrefsync.RecordNonInvItem, testRequestID))
	require.True(t, ok)
	assert.Equal(t, "Detention", put.Body["itemId"])
	assert.Equal(t, "40", refID(t, put.Body, "incomeAccount"))
	assert.Equal(t, testSubsidiary, items(t, put.Body, "subsidiary")[0]["id"])
	assert.Equal(
		t,
		1,
		fake.count(http.MethodPut, eidPath(nsrefsync.RecordServiceItem, testRequestID)),
	)
}

func TestCreateReferenceRefusesKindsNetSuiteDoesNotTakeFromTrenova(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{}
	conn := testConnector(t, fake)
	for _, kind := range []accountingprovider.ReferenceKind{
		accountingprovider.ReferenceKindAccount,
		accountingprovider.ReferenceKindTerm,
		accountingprovider.ReferenceKindPaymentMethod,
		accountingprovider.ReferenceKind("Bogus"),
	} {
		_, err := conn.CreateReference(t.Context(), createRequest(kind))
		require.Error(t, err, kind)
	}
	_, err := conn.CreateReference(t.Context(), createRequest(accountingprovider.ReferenceKindItem))
	require.ErrorIs(t, err, nsbinding.ErrItemDraftRequired)
	_, err = conn.CreateReference(
		t.Context(),
		createRequest(accountingprovider.ReferenceKindCustomer),
	)
	require.ErrorIs(t, err, nsbinding.ErrPartyDraftRequired)
	req := createRequest(accountingprovider.ReferenceKindCustomer)
	req.Party = &accountingconnectorports.AccountingPartyDraft{DisplayName: " "}
	_, err = conn.CreateReference(t.Context(), req)
	require.ErrorIs(t, err, nsbinding.ErrNameRequired)
	assert.Equal(t, accountingsync.SyncErrorValidation, conn.ClassifyDocumentError(err).Category)
	assert.Empty(t, fake.all())
}

func TestSanitizeNameFitsNetSuiteLimits(t *testing.T) {
	t.Parallel()

	conn := testConnector(t, &fakeNS{})
	long := strings.Repeat("x", 120)
	assert.Len(
		t,
		conn.SanitizeName(accountingprovider.ReferenceKindItem, long),
		nsrefsync.MaxItemNameLength,
	)
	assert.Len(
		t,
		conn.SanitizeName(accountingprovider.ReferenceKindCustomer, long),
		nsrefsync.MaxPartyNameLength,
	)
	assert.Equal(t, "a b", conn.SanitizeName(accountingprovider.ReferenceKindVendor, " a\tb "))
}

func TestUpsertCustomerAndVendorPatchExistingRecords(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{respond: func(call nsCall) reply {
		if call.is(http.MethodPut, eidPath(nsrefsync.RecordVendor, testRequestID)) {
			return upserted(nsrefsync.RecordVendor, "513")
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
	patch, ok := fake.find(http.MethodPatch, recordPath(nsrefsync.RecordCustomer, testCustomer))
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
	put, ok := fake.find(http.MethodPut, eidPath(nsrefsync.RecordVendor, testRequestID))
	require.True(t, ok)
	assert.Equal(t, testSubsidiary, refID(t, put.Body, "subsidiary"))
	assert.Equal(t, false, put.Body["is1099Eligible"])
}
