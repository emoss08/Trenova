package businesscentral_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/businesscentral"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccountsFilterByModifiedSince(t *testing.T) {
	t.Parallel()

	since := time.Date(2026, 9, 1, 8, 30, 0, 0, time.FixedZone("EST", -5*3600))
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRead(t, r, testCompanyPath+"/accounts")
		assert.Equal(
			t,
			"lastModifiedDateTime gt 2026-09-01T13:30:00Z",
			r.URL.Query().Get("$filter"),
		)
		_, _ = w.Write(fixture(t, "accounts.json"))
	})

	accounts, err := client.Accounts(t.Context(), &since)
	require.NoError(t, err)
	require.Len(t, accounts, 3)
	assert.Equal(t, testAccount, accounts[0].ID)
	assert.Equal(t, "10100", accounts[0].Number)
	assert.Equal(t, businesscentral.CategoryAssets, accounts[0].Category)
	assert.Equal(t, "Cash", accounts[0].SubCategory)
	assert.True(t, accounts[0].DirectPosting)
	assert.True(t, accounts[0].Postable())
	assert.Equal(t, businesscentral.AccountTypeHeading, accounts[1].AccountType)
	assert.False(t, accounts[1].Postable())
	assert.Empty(t, accounts[2].Category, "a blank enum is empty")
	assert.False(t, accounts[2].Postable(), "a blocked account cannot be posted to")
}

func TestAccountsWithoutSinceSendNoFilter(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.False(t, r.URL.Query().Has("$filter"))
		_, _ = w.Write(fixture(t, "accounts.json"))
	})
	_, err := client.Accounts(t.Context(), nil)
	require.NoError(t, err)
}

func TestItems(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRead(t, r, testCompanyPath+"/items")
		_, _ = w.Write(fixture(t, "items.json"))
	})

	items, err := client.Items(t.Context(), nil)
	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, "Linehaul", items[0].DisplayName)
	assert.Equal(t, businesscentral.ItemTypeService, items[0].Type)
	assert.True(t, dec("1250.5").Equal(items[0].UnitPrice))
	assert.True(t, items[1].Blocked)
}

func TestCreateItemDefaultsToService(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertWrite(t, r, http.MethodPost, testCompanyPath+"/items")
		assert.Equal(t, map[string]any{"displayName": "Fuel surcharge", "type": "Service"},
			readJSON(t, r))
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(fixture(t, "create_item.json"))
	})

	item, err := client.CreateItem(
		t.Context(),
		&businesscentral.ItemInput{DisplayName: " Fuel surcharge "},
	)
	require.NoError(t, err)
	assert.Equal(t, "1002", item.Number)

	_, err = client.CreateItem(t.Context(), &businesscentral.ItemInput{DisplayName: " "})
	require.ErrorIs(t, err, businesscentral.ErrNameRequired)
	_, err = client.CreateItem(
		t.Context(),
		&businesscentral.ItemInput{DisplayName: "x", Type: "Gadget"},
	)
	require.ErrorIs(t, err, businesscentral.ErrUnknownType)
}

func TestCreateWithEmptyBodyIsUnexpected(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	_, err := client.CreateItem(t.Context(), &businesscentral.ItemInput{DisplayName: "x"})
	require.ErrorIs(t, err, businesscentral.ErrUnexpectedPayload)
}

func TestCustomersNormaliseBlankValues(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRead(t, r, testCompanyPath+"/customers")
		_, _ = w.Write(fixture(t, "customers.json"))
	})

	customers, err := client.Parties(t.Context(), businesscentral.PartyCustomer, nil)
	require.NoError(t, err)
	require.Len(t, customers, 2)
	first := customers[0]
	assert.Equal(t, testCustomer, first.ID)
	assert.Equal(t, "C00010", first.Number)
	assert.Equal(t, "O'Brien Logistics", first.DisplayName)
	assert.Equal(t, "ap@obrien.example", first.Email)
	assert.Equal(t, "Atlanta", first.City)
	assert.Empty(t, first.Blocked)
	assert.False(t, first.IsBlocked())
	assert.Empty(t, first.PaymentTermsID, "the empty GUID is no payment terms")
	assert.Equal(t, `W/"JzE5OzE4MTs="`, first.ETag)
	assert.Equal(t, businesscentral.BlockedInvoice, customers[1].Blocked)
	assert.True(t, customers[1].IsBlocked())
	assert.Equal(t, "CAD", customers[1].CurrencyCode)
}

func TestVendors(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRead(t, r, testCompanyPath+"/vendors")
		_, _ = w.Write(fixture(t, "vendors.json"))
	})

	vendors, err := client.Parties(t.Context(), businesscentral.PartyVendor, nil)
	require.NoError(t, err)
	require.Len(t, vendors, 1)
	assert.Equal(t, testVendor, vendors[0].ID)
	assert.Equal(t, businesscentral.BlockedPayment, vendors[0].Blocked)

	_, err = client.Parties(t.Context(), businesscentral.PartyKind(9), nil)
	require.ErrorIs(t, err, businesscentral.ErrUnknownKind)
}

func TestFindPartiesQuotesTheName(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRead(t, r, testCompanyPath+"/customers")
		assert.Equal(t, "displayName eq 'O''Brien Logistics'", r.URL.Query().Get("$filter"))
		_, _ = w.Write(fixture(t, "customers.json"))
	})

	found, err := client.FindParties(
		t.Context(),
		businesscentral.PartyCustomer,
		"O'Brien Logistics",
	)
	require.NoError(t, err)
	assert.NotEmpty(t, found)

	_, err = client.FindParties(t.Context(), businesscentral.PartyCustomer, "bad\nname")
	require.ErrorIs(t, err, businesscentral.ErrInvalidFilter)
	_, err = client.FindParties(t.Context(), businesscentral.PartyCustomer, " ")
	require.ErrorIs(t, err, businesscentral.ErrInvalidFilter)
}

func TestPartyByID(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRead(t, r, testCompanyPath+"/customers(55555555-6666-4777-8888-999999999993)")
		_, _ = w.Write(fixture(t, "create_customer.json"))
	})

	party, err := client.Party(t.Context(), businesscentral.PartyCustomer,
		"55555555-6666-4777-8888-999999999993")
	require.NoError(t, err)
	assert.Equal(t, "Northwind Traders", party.DisplayName)

	_, err = client.Party(t.Context(), businesscentral.PartyCustomer, "x') or (1 eq 1")
	require.ErrorIs(t, err, businesscentral.ErrInvalidID)
}

func northwind() businesscentral.PartyInput {
	return businesscentral.PartyInput{
		DisplayName:    "Northwind Traders",
		Email:          "billing@northwind.example",
		AddressLine1:   "500 Harbor Way",
		City:           "Seattle",
		State:          "WA",
		PostalCode:     "98101",
		Country:        "US",
		PaymentTermsID: "66666666-7777-4888-8999-AAAAAAAAAAA1",
	}
}

func TestCreateCustomer(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertWrite(t, r, http.MethodPost, testCompanyPath+"/customers")
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, map[string]any{
			"displayName":    "Northwind Traders",
			"email":          "billing@northwind.example",
			"addressLine1":   "500 Harbor Way",
			"city":           "Seattle",
			"state":          "WA",
			"postalCode":     "98101",
			"country":        "US",
			"paymentTermsId": "66666666-7777-4888-8999-aaaaaaaaaaa1",
		}, readJSON(t, r))
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(fixture(t, "create_customer.json"))
	})

	in := northwind()
	party, err := client.CreateParty(t.Context(), businesscentral.PartyCustomer, &in)
	require.NoError(t, err)
	assert.Equal(t, "C00030", party.Number)
	assert.Equal(t, `W/"JzE5OzE4Mzs="`, party.ETag)
}

func TestCreatePartyValidatesInput(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(http.ResponseWriter, *http.Request) {
		t.Error("invalid input is never sent")
	})
	in := northwind()
	in.City = "A city name that is far longer than thirty characters"
	_, err := client.CreateParty(t.Context(), businesscentral.PartyCustomer, &in)
	require.ErrorIs(t, err, businesscentral.ErrFieldTooLong)

	in = northwind()
	in.DisplayName = ""
	_, err = client.CreateParty(t.Context(), businesscentral.PartyVendor, &in)
	require.ErrorIs(t, err, businesscentral.ErrNameRequired)

	in = northwind()
	in.PaymentTermsID = "net30"
	_, err = client.CreateParty(t.Context(), businesscentral.PartyVendor, &in)
	require.ErrorIs(t, err, businesscentral.ErrInvalidID)
}

func TestUpdatePartySendsIfMatch(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertWrite(t, r, http.MethodPatch, testCompanyPath+"/vendors("+testVendor+")")
		assert.Equal(t, `W/"JzE5OzE="`, r.Header.Get("If-Match"))
		assert.Equal(t, "Northwind Traders", readJSON(t, r)["displayName"])
		_, _ = w.Write(fixture(t, "create_customer.json"))
	})

	party, err := client.UpdateParty(t.Context(), &businesscentral.PartyUpdate{
		Kind: businesscentral.PartyVendor, PartyID: testVendor, ETag: `W/"JzE5OzE="`,
		Input: northwind(),
	})
	require.NoError(t, err)
	assert.Equal(t, "Northwind Traders", party.DisplayName)

	_, err = client.UpdateParty(t.Context(), &businesscentral.PartyUpdate{
		Kind: businesscentral.PartyVendor, PartyID: testVendor, ETag: "W/\"x\r\nX-Evil: 1\"",
		Input: northwind(),
	})
	require.ErrorIs(t, err, businesscentral.ErrInvalidETag)
}

func TestUpdatePartyWithoutETagMatchesAny(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "*", r.Header.Get("If-Match"))
		_, _ = w.Write(fixture(t, "create_customer.json"))
	})
	_, err := client.UpdateParty(t.Context(), &businesscentral.PartyUpdate{
		Kind: businesscentral.PartyCustomer, PartyID: testCustomer, Input: northwind(),
	})
	require.NoError(t, err)
}

func TestPaymentTermsAndMethods(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testCompanyPath + "/paymentTerms":
			_, _ = w.Write(fixture(t, "payment_terms.json"))
		case testCompanyPath + "/paymentMethods":
			_, _ = w.Write(fixture(t, "payment_methods.json"))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	})

	terms, err := client.PaymentTerms(t.Context())
	require.NoError(t, err)
	require.Len(t, terms, 1)
	assert.Equal(t, businesscentral.PaymentTerm{
		ID: "66666666-7777-4888-8999-aaaaaaaaaaa1", Code: "NET30", DisplayName: "Net 30 days",
		DueDateCalculation: "30D",
	}, terms[0])

	methods, err := client.PaymentMethods(t.Context())
	require.NoError(t, err)
	require.Len(t, methods, 2)
	assert.Equal(t, "CHECK", methods[1].Code)
}
