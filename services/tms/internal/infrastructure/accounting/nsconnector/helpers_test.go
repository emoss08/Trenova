package nsconnector

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/shared/netsuite"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	testAccount      = "1234567_SB1"
	testSubsidiary   = "3"
	testRequestID    = "trn-0123456789abcdef0123456789abcdef01234567"
	testVoidRequest  = "trn-fedcba9876543210fedcba9876543210fedcba98"
	testClientID     = "6f1e2d3c4b5a69788796a5b4c3d2e1f00f1e2d3c4b5a69788796a5b4c3d2e1f0"
	testCustomer     = "341"
	testVendor       = "512"
	testInvoice      = "1201"
	testCreditMemo   = "1301"
	testPayment      = "1401"
	testBill         = "1501"
	testVendorCredit = "1601"
	testBillPayment  = "1701"
	testItem         = "77"
	testWriteOffItem = "78"
	testBank         = "10"
	testDeposit      = "11"
	testExpense      = "60"
	testTerm         = "2"
	restPrefix       = "/services/rest"
	suiteQLPath      = "/query/v1/suiteql"
	tokenPath        = "/auth/oauth2/v1/token"
	revokePath       = "/auth/oauth2/v1/revoke"
)

var testNow = time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)

type nsCall struct {
	Method string
	Path   string
	Query  url.Values
	Form   url.Values
	Body   map[string]any
	SQL    string
	Params []any
	Auth   string
}

func (c nsCall) is(method, path string) bool {
	return c.Method == method && c.Path == path
}

func (c nsCall) queries(fragment string) bool {
	return c.Path == suiteQLPath && strings.Contains(c.SQL, fragment)
}

type reply struct {
	status   int
	body     string
	location string
}

type fakeNS struct {
	mu      sync.Mutex
	calls   []nsCall
	server  *httptest.Server
	respond func(call nsCall) reply
}

func (f *fakeNS) handler(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		call := nsCall{
			Method: r.Method,
			Path:   strings.TrimPrefix(r.URL.Path, restPrefix),
			Query:  r.URL.Query(),
			Auth:   r.Header.Get("Authorization"),
		}
		raw, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		switch {
		case strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded"):
			call.Form, err = url.ParseQuery(string(raw))
			assert.NoError(t, err)
		case len(raw) > 0 && raw[0] == '{':
			assert.NoError(t, sonic.Unmarshal(raw, &call.Body))
		}
		if call.Path == suiteQLPath {
			call.SQL, _ = call.Body["q"].(string)
			call.Params, _ = call.Body["params"].([]any)
		}
		f.mu.Lock()
		f.calls = append(f.calls, call)
		f.mu.Unlock()

		out := f.defaultReply(t, call)
		if f.respond != nil {
			if custom := f.respond(call); custom.status != 0 {
				out = custom
			}
		}
		if out.location != "" {
			w.Header().Set("Location", f.server.URL+restPrefix+out.location)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(out.status)
		if out.body != "" {
			_, _ = w.Write([]byte(out.body))
		}
	}
}

func (f *fakeNS) defaultReply(t *testing.T, call nsCall) reply {
	t.Helper()
	switch {
	case call.Path == suiteQLPath:
		return reply{status: http.StatusOK, body: emptyRows}
	case call.Method == http.MethodGet && call.Query.Get("fields") == fieldTranID:
		id := call.Path[strings.LastIndexByte(call.Path, '/')+1:]
		return reply{status: http.StatusOK, body: `{"id":"` + id + `","tranId":"DOC-` + id + `"}`}
	case call.Method == http.MethodPatch, call.Method == http.MethodDelete:
		return reply{status: http.StatusNoContent}
	default:
		return reply{
			status: http.StatusNotFound,
			body:   readFixture(t, "error_404_nonexistent.json"),
		}
	}
}

const emptyRows = `{"links":[],"count":0,"hasMore":false,"offset":0,"totalResults":0,"items":[]}`

func upserted(recordType, id string) reply {
	return reply{status: http.StatusNoContent, location: "/record/v1/" + recordType + "/" + id}
}

func fixtureReply(t *testing.T, status int, name string) reply {
	t.Helper()
	return reply{status: status, body: readFixture(t, name)}
}

func (f *fakeNS) all() []nsCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]nsCall(nil), f.calls...)
}

func (f *fakeNS) writes() []nsCall {
	out := make([]nsCall, 0, 4)
	for _, call := range f.all() {
		if call.Method != http.MethodGet && call.Path != suiteQLPath {
			out = append(out, call)
		}
	}
	return out
}

func (f *fakeNS) find(method, path string) (nsCall, bool) {
	for _, call := range f.all() {
		if call.is(method, path) {
			return call, true
		}
	}
	return nsCall{}, false
}

func (f *fakeNS) count(method, path string) int {
	n := 0
	for _, call := range f.all() {
		if call.is(method, path) {
			n++
		}
	}
	return n
}

func (f *fakeNS) query(fragment string) (nsCall, bool) {
	for _, call := range f.all() {
		if call.queries(fragment) {
			return call, true
		}
	}
	return nsCall{}, false
}

func readFixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return string(data)
}

func newProvider(t *testing.T, cfg config.NetSuiteConfig) *Provider {
	t.Helper()
	provider, err := New(Params{
		Config: &config.Config{
			App:        config.AppConfig{WebBaseURL: "https://app.example.com"},
			Accounting: config.AccountingConfig{NetSuite: cfg},
		},
		Logger: zap.NewNop(),
	})
	require.NoError(t, err)
	return provider
}

func tenantApp() *accountingsyncports.AccountingApp {
	return &accountingsyncports.AccountingApp{
		Source:       accountingsync.AppSourceTenant,
		Environment:  accountingsync.AppEnvironmentProduction,
		ClientID:     testClientID,
		ClientSecret: "tenant-secret",
	}
}

func testConnector(t *testing.T, fake *fakeNS) *Connector {
	t.Helper()
	fake.server = httptest.NewServer(fake.handler(t))
	t.Cleanup(fake.server.Close)

	provider := newProvider(t, config.NetSuiteConfig{})
	provider.oauthOpts = []netsuite.Option{
		netsuite.WithBaseURL(fake.server.URL),
		netsuite.WithLoginURL(fake.server.URL),
	}
	provider.now = func() time.Time { return testNow }
	provider.books = newCurrencyBooks(provider.now)
	conn, err := provider.bind(tenantApp())
	require.NoError(t, err)
	conn.apiOpts = []netsuite.Option{
		netsuite.WithBaseURL(fake.server.URL),
		netsuite.WithRetry(restx.RetryConfig{}),
	}
	return conn
}

func testRef() netsuite.CompanyRef {
	return netsuite.CompanyRef{AccountID: testAccount, SubsidiaryID: testSubsidiary}
}

func testAuth() accountingsyncports.AccountingDocumentAuth {
	return accountingsyncports.AccountingDocumentAuth{
		RealmID:     testRef().String(),
		AccessToken: "access-token",
		CompanyCode: "USD",
		CompanyName: "Acme US",
	}
}

func recordPath(recordType, id string) string {
	return "/record/v1/" + recordType + "/" + id
}

func eidPath(recordType, externalID string) string {
	return recordPath(recordType, "eid:"+externalID)
}

func object(t *testing.T, body map[string]any, field string) map[string]any {
	t.Helper()
	value, ok := body[field].(map[string]any)
	require.True(t, ok, "field %s", field)
	return value
}

func refID(t *testing.T, body map[string]any, field string) string {
	t.Helper()
	id, ok := object(t, body, field)["id"].(string)
	require.True(t, ok, "field %s has no id", field)
	return id
}

func items(t *testing.T, body map[string]any, sublist string) []map[string]any {
	t.Helper()
	raw, ok := object(t, body, sublist)["items"].([]any)
	require.True(t, ok, "sublist %s", sublist)
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		line, isMap := item.(map[string]any)
		require.True(t, isMap)
		out = append(out, line)
	}
	return out
}

func syncErrorOf(t *testing.T, err error) *accountingsync.SyncError {
	t.Helper()
	require.Error(t, err)
	var syncErr *accountingsync.SyncError
	require.ErrorAs(t, err, &syncErr)
	return syncErr
}

func rowsJSON(t *testing.T, hasMore bool, rows ...map[string]any) string {
	t.Helper()
	out, err := sonic.MarshalString(map[string]any{
		"links":        []any{},
		"count":        len(rows),
		"hasMore":      hasMore,
		"offset":       0,
		"totalResults": len(rows),
		"items":        rows,
	})
	require.NoError(t, err)
	return out
}

func stateRow(id string, overrides map[string]any) map[string]any {
	row := map[string]any{
		"id":                     id,
		"type":                   "CustInvc",
		"tranid":                 "DOC-" + id,
		"entity":                 testCustomer,
		"foreigntotal":           "1250.00",
		"foreignamountremaining": "1250.00",
		"voided":                 "F",
		"currency":               "USD",
		"lastmodified":           "2026-10-05 09:00:00",
		"closed":                 "F",
		"alllocked":              "F",
		"arlocked":               "F",
		"aplocked":               "F",
	}
	for key, value := range overrides {
		row[key] = value
	}
	return row
}

func methods(calls []nsCall) []string {
	out := make([]string, 0, len(calls))
	for _, call := range calls {
		out = append(out, call.Method+" "+call.Path)
	}
	return out
}
