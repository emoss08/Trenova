package qboconnector

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/shared/quickbooks"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type countingLimiter struct {
	mu      sync.Mutex
	buckets []restx.Bucket
}

func (l *countingLimiter) Acquire(_ context.Context, bucket restx.Bucket) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.buckets = append(l.buckets, bucket)
	return nil
}

func TestListReferenceTranslatesTermsIntoDomainObjects(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/query", r.URL.Path)
		assert.Contains(t, r.URL.Query().Get("query"), "select * from Term")
		_, _ = w.Write([]byte(`{"QueryResponse":{"Term":[{"Id":"3","Name":"Net 30","Type":"STANDARD","DueDays":30,"Active":true,"SyncToken":"1","MetaData":{"LastUpdatedTime":"2025-01-01T00:00:00Z"}}]}}`))
	}))
	t.Cleanup(server.Close)

	conn := newConnector(t, config.QuickBooksConfig{})
	conn.apiOpts = []quickbooks.Option{quickbooks.WithBaseURL(server.URL)}

	page, err := conn.ListReference(t.Context(), &services.AccountingReferencePageRequest{
		RealmID:       "123",
		AccessToken:   "token",
		Kind:          accountingsync.ReferenceKindTerm,
		StartPosition: 1,
		PageSize:      conn.MaxReferencePageSize(),
	})
	require.NoError(t, err)
	require.Len(t, page.Objects, 1)

	term := page.Objects[0]
	assert.Equal(t, accountingsync.ReferenceKindTerm, term.Kind)
	assert.Equal(t, "3", term.ExternalID)
	assert.Equal(t, "Net 30", term.Name)
	assert.Equal(t, "STANDARD", term.SubType)
	require.NotNil(t, term.DueDays)
	assert.Equal(t, 30, *term.DueDays)
	require.NotNil(t, term.ProviderUpdatedAt)
	assert.Equal(t, int64(1735689600), *term.ProviderUpdatedAt)
	assert.Zero(t, page.NextStart)
}

func TestCreateReferenceCreatesAVendorWithTheRequestID(t *testing.T) {
	t.Parallel()

	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/vendor", r.URL.Path)
		assert.Equal(t, "req-9", r.URL.Query().Get("requestid"))
		raw, _ := io.ReadAll(r.Body)
		assert.NoError(t, sonic.Unmarshal(raw, &body))
		_, _ = w.Write([]byte(`{"Vendor":{"Id":"95","DisplayName":"Swift Haul","Vendor1099":true,"Active":true,"SyncToken":"0"}}`))
	}))
	t.Cleanup(server.Close)

	conn := newConnector(t, config.QuickBooksConfig{})
	conn.apiOpts = []quickbooks.Option{quickbooks.WithBaseURL(server.URL)}

	vendor, err := conn.CreateReference(t.Context(), &services.AccountingCreateReferenceRequest{
		RealmID:     "123",
		AccessToken: "token",
		RequestID:   "req-9",
		Kind:        accountingsync.ReferenceKindVendor,
		Party:       &services.AccountingPartyDraft{DisplayName: "Swift Haul", Is1099: true},
	})
	require.NoError(t, err)
	assert.Equal(t, accountingsync.ReferenceKindVendor, vendor.Kind)
	assert.Equal(t, "95", vendor.ExternalID)
	assert.True(t, vendor.Is1099)
	assert.Equal(t, true, body["Vendor1099"])
}

func TestCreateReferenceRefusesKindsTheBookkeeperOwns(t *testing.T) {
	t.Parallel()

	conn := newConnector(t, config.QuickBooksConfig{})
	for _, kind := range []accountingsync.ReferenceKind{
		accountingsync.ReferenceKindAccount,
		accountingsync.ReferenceKindTerm,
		accountingsync.ReferenceKindPaymentMethod,
	} {
		_, err := conn.CreateReference(t.Context(), &services.AccountingCreateReferenceRequest{
			RealmID:     "123",
			AccessToken: "token",
			RequestID:   "req",
			Kind:        kind,
		})
		require.Error(t, err, kind)
	}
}

func TestConnectorRatesEveryAPICallPerRealm(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"QueryResponse":{}}`))
	}))
	t.Cleanup(server.Close)

	limiter := &countingLimiter{}
	cfg := &config.Config{App: config.AppConfig{WebBaseURL: "https://app.example.com"}}
	conn, err := New(Params{Config: cfg, Logger: zap.NewNop(), Limiter: limiter})
	require.NoError(t, err)
	conn.apiOpts = append(conn.apiOpts, quickbooks.WithBaseURL(server.URL))

	_, err = conn.ListReference(t.Context(), &services.AccountingReferencePageRequest{
		RealmID:       "555",
		AccessToken:   "token",
		Kind:          accountingsync.ReferenceKindAccount,
		StartPosition: 1,
		PageSize:      10,
	})
	require.NoError(t, err)
	require.Len(t, limiter.buckets, 1)
	assert.Equal(t, "quickbooks:realm:555", limiter.buckets[0].Key)
}

func TestSanitizeNameUsesEachKindsLimit(t *testing.T) {
	t.Parallel()

	conn := newConnector(t, config.QuickBooksConfig{})
	long := make([]byte, 300)
	for idx := range long {
		long[idx] = 'a'
	}
	assert.Len(t, conn.SanitizeName(accountingsync.ReferenceKindItem, string(long)), quickbooks.MaxItemNameLength)
	assert.Len(t, conn.SanitizeName(accountingsync.ReferenceKindCustomer, string(long)), 300)
	assert.Equal(t, "Fuel - Surcharge", conn.SanitizeName(accountingsync.ReferenceKindItem, "Fuel:Surcharge"))
}
