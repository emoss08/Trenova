package xeroconnector

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/emoss08/trenova/shared/xero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	testTenant     = "4c7e1f20-3b5a-4d8e-9f10-2a3b4c5d6e7f"
	testRequestID  = "trn-0123456789abcdef0123456789abcdef01234567"
	testCustomer   = "bd2270c3-8706-4c11-9cfb-000b551c3f51"
	testVendor     = "a1b2c3d4-0000-4000-8000-000000000001"
	testInvoice    = "fee88eea-f2aa-4a71-a372-33d6d83d3c45"
	testInvoice2   = "fee88eea-f2aa-4a71-a372-33d6d83d3c46"
	testBill       = "0032c6e3-7b1e-4b0f-9c7e-2f6d2b5b8a11"
	testCreditNote = "7f8b6e4a-2d3c-4b1a-9e8f-6a5b4c3d2e1f"
	testShortNote  = "7f8b6e4a-2d3c-4b1a-9e8f-6a5b4c3d2e20"
	testAllocation = "b6c3a2d1-9f8e-4d7c-8b6a-5e4f3d2c1b0a"
	testPayment    = "b26fd49a-cbae-470a-a8f8-bcbc119e0379"
	testPayment2   = "b26fd49a-cbae-470a-a8f8-bcbc119e0380"
	testOldPayment = "b26fd49a-cbae-470a-a8f8-bcbc119e0301"
	testBatch      = "d318c343-208e-49fe-b04a-45642349bcf1"
	testBank       = "562555f2-8cde-4ce9-8203-0363922537a4"
	testRevenue    = "e2bc5c1b-9a60-4f3f-8d4a-3a0f5c1e2d01"
	testWriteOff   = "e2bc5c1b-9a60-4f3f-8d4a-3a0f5c1e2d02"
	testExpense    = "e2bc5c1b-9a60-4f3f-8d4a-3a0f5c1e2d03"
	testUncoded    = "e2bc5c1b-9a60-4f3f-8d4a-3a0f5c1e2d04"
	apiPrefix      = "/api.xro/2.0"
)

const accountsBody = `{"Accounts":[
 {"AccountID":"` + testBank + `","Code":"090","Name":"Business Bank","Type":"BANK","Status":"ACTIVE","Class":"ASSET","BankAccountType":"BANK","UpdatedDateUTC":"/Date(1573755038314+0000)/"},
 {"AccountID":"5040915e-8ce7-4177-8d08-fde416232f18","Code":"610","Name":"Accounts Receivable","Type":"CURRENT","Status":"ACTIVE","Class":"ASSET","SystemAccount":"DEBTORS"},
 {"AccountID":"4281c446-efb4-445d-9b42-c1d3bca1a3e5","Code":"800","Name":"Accounts Payable","Type":"CURRLIAB","Status":"ACTIVE","Class":"LIABILITY","SystemAccount":"CREDITORS"},
 {"AccountID":"` + testRevenue + `","Code":"200","Name":"Freight Revenue","Type":"REVENUE","Status":"ACTIVE","Class":"REVENUE"},
 {"AccountID":"` + testWriteOff + `","Code":"499","Name":"Short Pay Write-off","Type":"EXPENSE","Status":"ACTIVE","Class":"EXPENSE"},
 {"AccountID":"` + testExpense + `","Code":"310","Name":"Purchased Transportation","Type":"DIRECTCOSTS","Status":"ARCHIVED","Class":"EXPENSE"},
 {"AccountID":"` + testUncoded + `","Name":"Clearing","Type":"CURRLIAB","Status":"ACTIVE","Class":"LIABILITY"}
]}`

type xeroCall struct {
	Method        string
	Path          string
	Query         url.Values
	Key           string
	ModifiedSince string
	Tenant        string
	Body          map[string]any
}

type fakeXero struct {
	mu      sync.Mutex
	calls   []xeroCall
	respond func(call xeroCall) (int, string)
}

func (f *fakeXero) handler(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		call := xeroCall{
			Method:        r.Method,
			Path:          strings.TrimPrefix(r.URL.Path, apiPrefix),
			Query:         r.URL.Query(),
			Key:           r.Header.Get("Idempotency-Key"),
			ModifiedSince: r.Header.Get("If-Modified-Since"),
			Tenant:        r.Header.Get(tenantHeader),
		}
		raw, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		if len(raw) > 0 && raw[0] == '{' {
			assert.NoError(t, sonic.Unmarshal(raw, &call.Body))
		}
		f.mu.Lock()
		f.calls = append(f.calls, call)
		f.mu.Unlock()

		status, body := http.StatusNotFound, `{"Type":"NotFound","Message":"not found"}`
		if call.Method == http.MethodGet && call.Path == "/Accounts" && call.ModifiedSince == "" {
			status, body = http.StatusOK, accountsBody
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

func (f *fakeXero) all() []xeroCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]xeroCall(nil), f.calls...)
}

func (f *fakeXero) writes() []xeroCall {
	out := make([]xeroCall, 0, 4)
	for _, call := range f.all() {
		if call.Method != http.MethodGet {
			out = append(out, call)
		}
	}
	return out
}

func (f *fakeXero) find(method, path string) (xeroCall, bool) {
	for _, call := range f.all() {
		if call.Method == method && call.Path == path {
			return call, true
		}
	}
	return xeroCall{}, false
}

func newProvider(t *testing.T, cfg config.XeroConfig) *Provider {
	t.Helper()
	provider, err := New(Params{
		Config: &config.Config{
			App:        config.AppConfig{WebBaseURL: "https://app.example.com"},
			Accounting: config.AccountingConfig{Xero: cfg},
		},
		Logger: zap.NewNop(),
	})
	require.NoError(t, err)
	return provider
}

func tenantApp() *services.AccountingApp {
	return &services.AccountingApp{
		Source:               accountingsync.AppSourceTenant,
		Environment:          accountingsync.AppEnvironmentProduction,
		ClientID:             "tenant-id",
		ClientSecret:         "tenant-secret",
		WebhookVerifierToken: "tenant-webhook-key",
	}
}

func testConnector(t *testing.T, fake *fakeXero) *Connector {
	t.Helper()
	server := httptest.NewServer(fake.handler(t))
	t.Cleanup(server.Close)

	provider := newProvider(t, config.XeroConfig{})
	provider.oauthOpts = []xero.Option{xero.WithIdentityURL(server.URL)}
	conn, err := provider.bind(tenantApp())
	require.NoError(t, err)
	conn.apiOpts = []xero.Option{
		xero.WithBaseURL(server.URL),
		xero.WithRetry(restx.RetryConfig{}),
	}
	return conn
}

func testAuth() services.AccountingDocumentAuth {
	return services.AccountingDocumentAuth{
		RealmID:     testTenant,
		AccessToken: "access-token",
		CompanyCode: "!Ab12C",
	}
}

func lineItems(t *testing.T, body map[string]any, envelope string) []map[string]any {
	t.Helper()
	docs, ok := body[envelope].([]any)
	require.True(t, ok, "envelope %s", envelope)
	require.Len(t, docs, 1)
	doc, ok := docs[0].(map[string]any)
	require.True(t, ok)
	raw, ok := doc["LineItems"].([]any)
	require.True(t, ok)
	lines := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		line, isMap := item.(map[string]any)
		require.True(t, isMap)
		lines = append(lines, line)
	}
	return lines
}

func firstDoc(t *testing.T, body map[string]any, envelope string) map[string]any {
	t.Helper()
	docs, ok := body[envelope].([]any)
	require.True(t, ok, "envelope %s", envelope)
	require.NotEmpty(t, docs)
	doc, ok := docs[0].(map[string]any)
	require.True(t, ok)
	return doc
}
