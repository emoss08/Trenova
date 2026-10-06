package bcconnector

import (
	"encoding/base64"
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
	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/shared/businesscentral"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	testTenant       = "6e1f2a3b-4c5d-4e6f-8a9b-0c1d2e3f4a5b"
	testCompany      = "a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d"
	testEnv          = "Production"
	testRequestID    = "trn-0123456789abcdef0123456789abcdef01234567"
	testCustomer     = "55555555-6666-4777-8888-999999999991"
	testVendor       = "77777777-8888-4999-8aaa-bbbbbbbbbbb1"
	testInvoice      = "99999999-aaaa-4bbb-8ccc-ddddddddddd1"
	testNewInvoice   = "99999999-aaaa-4bbb-8ccc-ddddddddddd3"
	testCreditMemo   = "bbbbbbbb-cccc-4ddd-8eee-fffffffffff1"
	testCorrective   = "bbbbbbbb-cccc-4ddd-8eee-fffffffffff9"
	testBill         = "cccccccc-dddd-4eee-8fff-000000000001"
	testNewBill      = "cccccccc-dddd-4eee-8fff-000000000002"
	testVendorCredit = "dddddddd-eeee-4fff-8000-111111111111"
	testItem         = "44444444-5555-4666-8777-888888888881"
	testWriteOffItem = "44444444-5555-4666-8777-888888888883"
	testBank         = "33333333-4444-4555-8666-777777777771"
	testExpense      = "33333333-4444-4555-8666-777777777774"
	testJournal      = "eeeeeeee-ffff-4000-8111-222222222221"
	testTerm         = "66666666-7777-4888-8999-aaaaaaaaaaa1"
	testWebURL       = "https://bc.example.com"
	apiSegment       = "/api/v2.0"
)

var testNow = time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)

type bcCall struct {
	Method  string
	Path    string
	Env     string
	Query   url.Values
	IfMatch string
	Body    map[string]any
}

func (c bcCall) is(method, path string) bool {
	return c.Method == method && c.Path == path
}

func (c bcCall) filter() string {
	return c.Query.Get("$filter")
}

type fakeBC struct {
	mu      sync.Mutex
	calls   []bcCall
	respond func(call bcCall) (int, string)
}

func normalizePath(raw string) (path, env string) {
	path = raw
	if idx := strings.Index(path, apiSegment); idx >= 0 {
		parts := strings.Split(strings.Trim(path[:idx], "/"), "/")
		if len(parts) == 3 {
			env = parts[2]
		}
		path = path[idx+len(apiSegment):]
	}
	if strings.HasPrefix(path, companiesSegment) {
		if end := strings.IndexByte(path, ')'); end >= 0 {
			path = path[end+1:]
		}
	}
	return path, env
}

func (f *fakeBC) handler(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		path, env := normalizePath(r.URL.Path)
		call := bcCall{
			Method:  r.Method,
			Path:    path,
			Env:     env,
			Query:   r.URL.Query(),
			IfMatch: r.Header.Get("If-Match"),
		}
		raw, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		if len(raw) > 0 && raw[0] == '{' {
			assert.NoError(t, sonic.Unmarshal(raw, &call.Body))
		}
		f.mu.Lock()
		f.calls = append(f.calls, call)
		f.mu.Unlock()

		status, body := http.StatusNotFound, readFixture(t, "error_404.json")
		if call.is(http.MethodGet, "/generalLedgerSetup") {
			status, body = http.StatusOK, readFixture(t, "general_ledger_setup.json")
		}
		if call.Query.Get("$skip") != "" {
			status, body = http.StatusOK, `{"value":[]}`
		}
		if f.respond != nil {
			if code, text := f.respond(call); code != 0 {
				status, body = code, text
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}
}

func (f *fakeBC) all() []bcCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]bcCall(nil), f.calls...)
}

func (f *fakeBC) writes() []bcCall {
	out := make([]bcCall, 0, 4)
	for _, call := range f.all() {
		if call.Method != http.MethodGet {
			out = append(out, call)
		}
	}
	return out
}

func (f *fakeBC) find(method, path string) (bcCall, bool) {
	for _, call := range f.all() {
		if call.is(method, path) {
			return call, true
		}
	}
	return bcCall{}, false
}

func (f *fakeBC) count(method, path string) int {
	n := 0
	for _, call := range f.all() {
		if call.is(method, path) {
			n++
		}
	}
	return n
}

func readFixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return string(data)
}

func newProvider(t *testing.T, cfg config.BusinessCentralConfig) *Provider {
	t.Helper()
	provider, err := New(Params{
		Config: &config.Config{
			App:        config.AppConfig{WebBaseURL: "https://app.example.com"},
			Accounting: config.AccountingConfig{BusinessCentral: cfg},
		},
		Logger: zap.NewNop(),
	})
	require.NoError(t, err)
	return provider
}

func tenantApp() *services.AccountingApp {
	return &services.AccountingApp{
		Source:       accountingsync.AppSourceTenant,
		Environment:  accountingsync.AppEnvironmentProduction,
		ClientID:     "tenant-id",
		ClientSecret: "tenant-secret",
	}
}

func testConnector(t *testing.T, fake *fakeBC) *Connector {
	t.Helper()
	server := httptest.NewServer(fake.handler(t))
	t.Cleanup(server.Close)

	provider := newProvider(t, config.BusinessCentralConfig{})
	provider.oauthOpts = []businesscentral.Option{businesscentral.WithLoginURL(server.URL)}
	provider.now = func() time.Time { return testNow }
	conn, err := provider.bind(tenantApp())
	require.NoError(t, err)
	conn.apiOpts = []businesscentral.Option{
		businesscentral.WithBaseURL(server.URL),
		businesscentral.WithWebURL(testWebURL),
		businesscentral.WithRetry(restx.RetryConfig{}),
	}
	return conn
}

func testRef() businesscentral.CompanyRef {
	return businesscentral.CompanyRef{
		TenantID:    testTenant,
		Environment: testEnv,
		CompanyID:   testCompany,
	}
}

func testAuth() services.AccountingDocumentAuth {
	return services.AccountingDocumentAuth{
		RealmID:     testRef().String(),
		AccessToken: "access-token",
		CompanyName: "CRONUS USA, Inc.",
	}
}

func accessTokenFor(tenant string) string {
	claims := base64.RawURLEncoding.EncodeToString([]byte(`{"tid":"` + tenant + `"}`))
	return "eyJhbGciOiJSUzI1NiJ9." + claims + ".signature"
}

type docSpec struct {
	id        string
	number    string
	reference string
	status    string
	party     string
	invoiceID string
	currency  string
	total     float64
	remaining float64
	modified  string
	etag      string
}

func (d docSpec) fields() map[string]any {
	fields := map[string]any{
		"id":                      d.id,
		"number":                  d.number,
		"externalDocumentNumber":  d.reference,
		"vendorInvoiceNumber":     d.reference,
		"status":                  d.status,
		"customerId":              d.party,
		"invoiceId":               d.invoiceID,
		"currencyCode":            d.currency,
		"totalAmountIncludingTax": d.total,
		"remainingAmount":         d.remaining,
		"lastModifiedDateTime":    d.modified,
		"@odata.etag":             d.etag,
	}
	if d.modified == "" {
		fields["lastModifiedDateTime"] = "2026-10-01T18:00:00Z"
	}
	return fields
}

func (d docSpec) json(t *testing.T) string {
	t.Helper()
	out, err := sonic.MarshalString(d.fields())
	require.NoError(t, err)
	return out
}

func listJSON(t *testing.T, docs ...docSpec) string {
	t.Helper()
	values := make([]map[string]any, 0, len(docs))
	for _, doc := range docs {
		values = append(values, doc.fields())
	}
	out, err := sonic.MarshalString(map[string]any{"value": values})
	require.NoError(t, err)
	return out
}

func lines(t *testing.T, body map[string]any, field string) []map[string]any {
	t.Helper()
	raw, ok := body[field].([]any)
	require.True(t, ok, "field %s", field)
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
