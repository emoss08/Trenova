package quickbooks_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/quickbooks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListReferenceQueriesActiveAndInactiveAndPages(t *testing.T) {
	t.Parallel()

	var queries []string
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v3/company/"+testRealm+"/query", r.URL.Path)
		assert.Equal(t, "75", r.URL.Query().Get("minorversion"))
		queries = append(queries, r.URL.Query().Get("query"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture(t, "query_account_page1.json"))
	})

	page, err := client.ListReference(t.Context(), quickbooks.KindAccount, 1, 2)
	require.NoError(t, err)

	require.Len(t, queries, 1)
	assert.Equal(t,
		"select * from Account where Active in (true, false) startposition 1 maxresults 2",
		queries[0],
	)
	require.Len(t, page.Objects, 2)
	assert.Equal(t, 3, page.NextStart)

	ar := page.Objects[0]
	assert.Equal(t, quickbooks.KindAccount, ar.Kind)
	assert.Equal(t, "84", ar.ID)
	assert.Equal(t, "Accounts Receivable (A/R)", ar.Name)
	assert.Equal(t, "1200", ar.Number)
	assert.Equal(t, "Asset", ar.Classification)
	assert.Equal(t, "Accounts Receivable", ar.AccountType)
	assert.Equal(t, "AccountsReceivable", ar.AccountSubType)
	assert.True(t, ar.Active)
	assert.Equal(t, "USD", ar.CurrencyCode)
	assert.Equal(t, "0", ar.SyncToken)
	assert.Equal(t, int64(1738434600), ar.LastUpdatedAt)

	income := page.Objects[1]
	assert.Equal(t, "Income:Freight Income", income.FullyQualifiedName)
	assert.Equal(t, "78", income.ParentID)
	assert.Empty(t, income.Number)
}

func TestListReferenceStopsWhenAPageComesBackShort(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(fixture(t, "query_account_page2.json"))
	})

	page, err := client.ListReference(t.Context(), quickbooks.KindAccount, 3, 2)
	require.NoError(t, err)
	require.Len(t, page.Objects, 1)
	assert.False(t, page.Objects[0].Active)
	assert.Zero(t, page.NextStart)
}

func TestListReferenceReadsAnEmptyResult(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(fixture(t, "query_empty.json"))
	})

	page, err := client.ListReference(t.Context(), quickbooks.KindVendor, 1, 1000)
	require.NoError(t, err)
	assert.Empty(t, page.Objects)
	assert.Zero(t, page.NextStart)
}

func TestListReferenceReadsEveryKind(t *testing.T) {
	t.Parallel()

	cases := []struct {
		kind    quickbooks.ReferenceKind
		fixture string
		check   func(t *testing.T, objects []quickbooks.ReferenceObject)
	}{
		{
			kind:    quickbooks.KindItem,
			fixture: "query_item.json",
			check: func(t *testing.T, objects []quickbooks.ReferenceObject) {
				require.Len(t, objects, 2)
				assert.Equal(t, "DET", objects[0].Number)
				assert.Equal(t, "Service", objects[0].ItemType)
				assert.Equal(t, "79", objects[0].IncomeAccountID)
				assert.Equal(t, "Detention at shipper or consignee", objects[0].Description)
				assert.Equal(t, "Category", objects[1].ItemType)
			},
		},
		{
			kind:    quickbooks.KindCustomer,
			fixture: "query_customer.json",
			check: func(t *testing.T, objects []quickbooks.ReferenceObject) {
				require.Len(t, objects, 1)
				c := objects[0]
				assert.Equal(t, "Peak Distributing LLC", c.Name)
				assert.Equal(t, "Peak Distributing", c.CompanyName)
				assert.Equal(t, "ap@peakdistributing.example", c.Email)
				assert.Equal(t, "100 Main St", c.AddressLine1)
				assert.Equal(t, "Denver", c.City)
				assert.Equal(t, "CO", c.State)
				assert.Equal(t, "80202", c.PostalCode)
			},
		},
		{
			kind:    quickbooks.KindVendor,
			fixture: "query_vendor.json",
			check: func(t *testing.T, objects []quickbooks.ReferenceObject) {
				require.Len(t, objects, 1)
				assert.Equal(t, "Swift Haul Inc", objects[0].Name)
				assert.Equal(t, "MC-123456", objects[0].Number)
				assert.True(t, objects[0].Is1099)
			},
		},
		{
			kind:    quickbooks.KindTerm,
			fixture: "query_term.json",
			check: func(t *testing.T, objects []quickbooks.ReferenceObject) {
				require.Len(t, objects, 2)
				require.NotNil(t, objects[0].DueDays)
				assert.Equal(t, 30, *objects[0].DueDays)
				require.NotNil(t, objects[1].DueDays)
				assert.Equal(t, 0, *objects[1].DueDays)
			},
		},
		{
			kind:    quickbooks.KindPaymentMethod,
			fixture: "query_payment_method.json",
			check: func(t *testing.T, objects []quickbooks.ReferenceObject) {
				require.Len(t, objects, 1)
				assert.Equal(t, "Check", objects[0].Name)
				assert.Equal(t, "NON_CREDIT_CARD", objects[0].PaymentMethodType)
			},
		},
	}

	for _, tc := range cases {
		t.Run(string(tc.kind), func(t *testing.T) {
			t.Parallel()
			client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
				assert.True(t, strings.HasPrefix(
					r.URL.Query().Get("query"),
					"select * from "+string(tc.kind)+" where Active in (true, false)",
				))
				_, _ = w.Write(fixture(t, tc.fixture))
			})
			page, err := client.ListReference(t.Context(), tc.kind, 1, 1000)
			require.NoError(t, err)
			for idx := range page.Objects {
				assert.Equal(t, tc.kind, page.Objects[idx].Kind)
			}
			tc.check(t, page.Objects)
		})
	}
}

func TestListReferenceRefusesAnUnknownKindAndBadPaging(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client := newAPIClient(t, func(http.ResponseWriter, *http.Request) { calls.Add(1) })

	_, err := client.ListReference(t.Context(), quickbooks.ReferenceKind("Invoice; drop"), 1, 10)
	require.ErrorIs(t, err, quickbooks.ErrUnknownReferenceKind)
	_, err = client.ListReference(t.Context(), quickbooks.KindAccount, 0, 10)
	require.ErrorIs(t, err, quickbooks.ErrInvalidPaging)
	_, err = client.ListReference(t.Context(), quickbooks.KindAccount, 1, 1001)
	require.ErrorIs(t, err, quickbooks.ErrInvalidPaging)
	assert.Zero(t, calls.Load())
}

func TestCreateItemSendsTheRequestIDAndReadsTheItemBack(t *testing.T) {
	t.Parallel()

	var body map[string]any
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v3/company/"+testRealm+"/item", r.URL.Path)
		assert.Equal(t, "req-lumper-1", r.URL.Query().Get("requestid"))
		assert.Equal(t, "75", r.URL.Query().Get("minorversion"))
		raw, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.NoError(t, sonic.Unmarshal(raw, &body))
		_, _ = w.Write(fixture(t, "create_item.json"))
	})

	item, err := client.CreateItem(t.Context(), "req-lumper-1", quickbooks.ItemDraft{
		Name:            "Lumper Fee",
		Description:     "Lumper service",
		Sku:             "LUMP",
		IncomeAccountID: "79",
	})
	require.NoError(t, err)

	assert.Equal(t, "Lumper Fee", body["Name"])
	assert.Equal(t, "Service", body["Type"])
	assert.Equal(t, "LUMP", body["Sku"])
	assert.Equal(t, map[string]any{"value": "79"}, body["IncomeAccountRef"])

	assert.Equal(t, quickbooks.KindItem, item.Kind)
	assert.Equal(t, "150", item.ID)
	assert.Equal(t, "79", item.IncomeAccountID)
}

func TestCreateCustomerAndVendorSendTheParty(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		path   string
		reply  string
		create func(ctx context.Context, c *quickbooks.Client, draft *quickbooks.PartyDraft) (*quickbooks.ReferenceObject, error)
		kind   quickbooks.ReferenceKind
	}{
		{
			name:  "customer",
			path:  "/customer",
			reply: `{"Customer":{"Id":"70","DisplayName":"Acme Freight","Active":true,"SyncToken":"0"}}`,
			create: func(ctx context.Context, c *quickbooks.Client, d *quickbooks.PartyDraft) (*quickbooks.ReferenceObject, error) {
				return c.CreateCustomer(ctx, "req-1", d)
			},
			kind: quickbooks.KindCustomer,
		},
		{
			name:  "vendor",
			path:  "/vendor",
			reply: `{"Vendor":{"Id":"95","DisplayName":"Acme Freight","Vendor1099":true,"Active":true,"SyncToken":"0"}}`,
			create: func(ctx context.Context, c *quickbooks.Client, d *quickbooks.PartyDraft) (*quickbooks.ReferenceObject, error) {
				return c.CreateVendor(ctx, "req-1", d)
			},
			kind: quickbooks.KindVendor,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var body map[string]any
			client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/v3/company/"+testRealm+tc.path, r.URL.Path)
				assert.Equal(t, "req-1", r.URL.Query().Get("requestid"))
				raw, _ := io.ReadAll(r.Body)
				assert.NoError(t, sonic.Unmarshal(raw, &body))
				_, _ = w.Write([]byte(tc.reply))
			})

			created, err := tc.create(t.Context(), client, &quickbooks.PartyDraft{
				DisplayName:  "Acme Freight",
				CompanyName:  "Acme Freight",
				Email:        "ap@acme.example",
				AddressLine1: "1 Dock St",
				City:         "Omaha",
				State:        "NE",
				PostalCode:   "68102",
				Country:      "US",
				Is1099:       true,
			})
			require.NoError(t, err)
			assert.Equal(t, tc.kind, created.Kind)
			assert.Equal(t, "Acme Freight", body["DisplayName"])
			assert.Equal(t, map[string]any{"Address": "ap@acme.example"}, body["PrimaryEmailAddr"])
			assert.Equal(t, map[string]any{
				"Line1":                  "1 Dock St",
				"City":                   "Omaha",
				"CountrySubDivisionCode": "NE",
				"PostalCode":             "68102",
				"Country":                "US",
			}, body["BillAddr"])
			if tc.kind == quickbooks.KindVendor {
				assert.Equal(t, true, body["Vendor1099"])
			} else {
				assert.NotContains(t, body, "Vendor1099")
			}
		})
	}
}

func TestCreateOmitsEmptyOptionalFields(t *testing.T) {
	t.Parallel()

	var body map[string]any
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		assert.NoError(t, sonic.Unmarshal(raw, &body))
		_, _ = w.Write([]byte(`{"Customer":{"Id":"70","DisplayName":"Solo","Active":true}}`))
	})

	_, err := client.CreateCustomer(t.Context(), "req-2", &quickbooks.PartyDraft{DisplayName: "Solo"})
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"DisplayName": "Solo"}, body)
}

func TestCreateRejectsBadInputWithoutCallingIntuit(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client := newAPIClient(t, func(http.ResponseWriter, *http.Request) { calls.Add(1) })

	_, err := client.CreateItem(t.Context(), "", quickbooks.ItemDraft{Name: "Lumper", IncomeAccountID: "79"})
	require.ErrorIs(t, err, quickbooks.ErrRequestIDRequired)
	_, err = client.CreateItem(t.Context(), strings.Repeat("r", 51), quickbooks.ItemDraft{Name: "Lumper", IncomeAccountID: "79"})
	require.ErrorIs(t, err, quickbooks.ErrRequestIDRequired)
	_, err = client.CreateItem(t.Context(), "req", quickbooks.ItemDraft{Name: "Bad:Name", IncomeAccountID: "79"})
	require.ErrorIs(t, err, quickbooks.ErrInvalidName)
	_, err = client.CreateItem(t.Context(), "req", quickbooks.ItemDraft{Name: "Lumper"})
	require.ErrorIs(t, err, quickbooks.ErrIncomeAccountRequired)
	_, err = client.CreateVendor(t.Context(), "req", &quickbooks.PartyDraft{DisplayName: " "})
	require.ErrorIs(t, err, quickbooks.ErrInvalidName)
	assert.Zero(t, calls.Load())
}

func TestCreateReportsADuplicateName(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write(fixture(t, "fault_duplicate_name.json"))
	})

	_, err := client.CreateItem(t.Context(), "req-3", quickbooks.ItemDraft{Name: "Detention", IncomeAccountID: "79"})
	require.Error(t, err)
	assert.True(t, quickbooks.IsDuplicateName(err))
	assert.False(t, quickbooks.IsTransient(err))
}

func TestSanitizeNameMakesAnyTextAcceptable(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Fuel - Surcharge", quickbooks.SanitizeName("Fuel:Surcharge", 100))
	assert.Equal(t, "Line one line two", quickbooks.SanitizeName("Line one\n\tline two", 100))
	assert.Equal(t, "abcde", quickbooks.SanitizeName("abcdefgh", 5))
	assert.Equal(t, "", quickbooks.SanitizeName("  ", 100))
}
