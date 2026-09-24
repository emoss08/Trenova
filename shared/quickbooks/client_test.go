package quickbooks_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/quickbooks"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testRealm       = "9341452431742015"
	testAccessToken = "eyJ.super.secret.access"
)

type recordingLimiter struct {
	mu      sync.Mutex
	buckets []restx.Bucket
}

func (l *recordingLimiter) Acquire(_ context.Context, bucket restx.Bucket) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.buckets = append(l.buckets, bucket)
	return nil
}

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return data
}

func fastRetry() quickbooks.Option {
	return quickbooks.WithRetry(restx.RetryConfig{
		Enabled:        true,
		MaxAttempts:    3,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     2 * time.Millisecond,
	})
}

func newAPIClient(
	t *testing.T,
	handler http.HandlerFunc,
	opts ...quickbooks.Option,
) *quickbooks.Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	all := append([]quickbooks.Option{quickbooks.WithBaseURL(server.URL), fastRetry()}, opts...)
	client, err := quickbooks.New(quickbooks.EnvironmentSandbox, testRealm, testAccessToken, all...)
	require.NoError(t, err)
	return client
}

func TestNewRequiresRealmAndToken(t *testing.T) {
	t.Parallel()

	_, err := quickbooks.New(quickbooks.EnvironmentSandbox, " ", testAccessToken)
	require.ErrorIs(t, err, quickbooks.ErrRealmIDRequired)
	_, err = quickbooks.New(quickbooks.EnvironmentSandbox, testRealm, "")
	require.ErrorIs(t, err, quickbooks.ErrAccessTokenRequired)
}

func TestCompanyInfoSendsBearerAndMinorVersion(t *testing.T) {
	t.Parallel()

	var gotPath, gotAuth, gotMinor string
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotMinor = r.URL.Query().Get("minorversion")
		_, _ = w.Write(fixture(t, "company_info.json"))
	})

	info, err := client.CompanyInfo(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "/v3/company/"+testRealm+"/companyinfo/"+testRealm, gotPath)
	assert.Equal(t, "Bearer "+testAccessToken, gotAuth)
	assert.Equal(t, "75", gotMinor)
	assert.Equal(t, "Acme Freight LLC", info.CompanyName)
	assert.Equal(t, "Acme Freight Holdings LLC", info.LegalName)
	assert.Equal(t, "US", info.Country)
	assert.Equal(t, testRealm, info.RealmID)
}

func TestPreferencesNormalizesCurrencyAndCloseDate(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/"+testRealm+"/preferences", r.URL.Path)
		_, _ = w.Write(fixture(t, "preferences.json"))
	})

	prefs, err := client.Preferences(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "USD", prefs.HomeCurrency)
	assert.False(t, prefs.MultiCurrencyEnabled)
	assert.Equal(t, "2026-06-30", prefs.BooksClosedThrough)
}

func TestFaultIsDecodedAndClassified(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write(fixture(t, "fault_401.json"))
	})

	_, err := client.CompanyInfo(t.Context())
	require.Error(t, err)
	var fault *quickbooks.FaultError
	require.ErrorAs(t, err, &fault)
	assert.Equal(t, "AUTHENTICATION", fault.Type)
	assert.Equal(t, "3200", fault.FirstCode())
	assert.True(t, quickbooks.IsUnauthorized(err))
	assert.False(t, quickbooks.IsTransient(err))
	assert.Equal(t, int32(1), calls.Load())
	assert.NotContains(t, err.Error(), testAccessToken)
}

func TestServerErrorsAreRetriedAndTransient(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	_, err := client.CompanyInfo(t.Context())
	require.Error(t, err)
	assert.True(t, quickbooks.IsTransient(err))
	assert.Equal(t, int32(3), calls.Load())
}

func TestLimiterBucketIsPerRealm(t *testing.T) {
	t.Parallel()

	limiter := &recordingLimiter{}
	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(fixture(t, "company_info.json"))
	}, quickbooks.WithLimiter(limiter, "qbo"))

	_, err := client.CompanyInfo(t.Context())
	require.NoError(t, err)
	require.Len(t, limiter.buckets, 1)
	assert.Equal(t, "qbo:realm:"+testRealm, limiter.buckets[0].Key)
	assert.Equal(t, time.Minute, limiter.buckets[0].Period)
	assert.Less(t, limiter.buckets[0].Limit, 500)
}

func TestParseEnvironment(t *testing.T) {
	t.Parallel()

	env, ok := quickbooks.ParseEnvironment(" Sandbox ")
	require.True(t, ok)
	assert.Equal(t, quickbooks.EnvironmentSandbox, env)
	assert.True(t, strings.Contains(env.APIBaseURL(), "sandbox"))

	env, ok = quickbooks.ParseEnvironment("production")
	require.True(t, ok)
	assert.Equal(t, "https://quickbooks.api.intuit.com", env.APIBaseURL())

	_, ok = quickbooks.ParseEnvironment("staging")
	assert.False(t, ok)
}
