package netsuite_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/netsuite"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testAccount     = "1234567_SB1"
	testAccessToken = "eyJraWQiOiJjLjEyMzQ1NjcifQ.super-secret-access.sig"
	testRecordBase  = "/services/rest/record/v1"
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

func fastRetry() netsuite.Option {
	return netsuite.WithRetry(restx.RetryConfig{
		Enabled:        true,
		MaxAttempts:    3,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     2 * time.Millisecond,
	})
}

func newServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

func newAPIClient(
	t *testing.T,
	handler http.HandlerFunc,
	opts ...netsuite.Option,
) (*netsuite.Client, *httptest.Server) {
	t.Helper()
	server := newServer(t, handler)
	all := append([]netsuite.Option{netsuite.WithBaseURL(server.URL), fastRetry()}, opts...)
	client, err := netsuite.NewAccountClient(netsuite.AccountConfig{
		AccountID:   testAccount,
		AccessToken: testAccessToken,
	}, all...)
	require.NoError(t, err)
	return client, server
}

func serveFixture(t *testing.T, status int, name string) http.HandlerFunc {
	t.Helper()
	body := fixture(t, name)
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.oracle.resource+json; type=error")
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}
}

func readJSON(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	raw, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	var body map[string]any
	require.NoError(t, sonic.Unmarshal(raw, &body))
	return body
}

func assertAPIHeaders(t *testing.T, r *http.Request) {
	t.Helper()
	assert.Equal(t, "Bearer "+testAccessToken, r.Header.Get("Authorization"))
	assert.Equal(t, "application/json", r.Header.Get("Accept"))
}

func assertRead(t *testing.T, r *http.Request, path string) {
	t.Helper()
	assert.Equal(t, http.MethodGet, r.Method)
	assert.Equal(t, path, r.URL.Path)
	assertAPIHeaders(t, r)
	assert.Empty(t, r.Header.Get("Prefer"))
}

func assertWrite(t *testing.T, r *http.Request, method, path string) {
	t.Helper()
	assert.Equal(t, method, r.Method)
	assert.Equal(t, path, r.URL.Path)
	assertAPIHeaders(t, r)
	assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
}

func created(w http.ResponseWriter, location string) {
	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusNoContent)
}

func TestNewAccountClientValidates(t *testing.T) {
	t.Parallel()

	_, err := netsuite.NewAccountClient(netsuite.AccountConfig{AccountID: testAccount})
	require.ErrorIs(t, err, netsuite.ErrAccessTokenRequired)

	_, err = netsuite.NewAccountClient(netsuite.AccountConfig{
		AccountID: "1234567-SB1", AccessToken: testAccessToken,
	})
	require.ErrorIs(t, err, netsuite.ErrInvalidAccountID)

	client, err := netsuite.NewAccountClient(netsuite.AccountConfig{
		AccountID: " 1234567_sb1 ", AccessToken: testAccessToken,
	})
	require.NoError(t, err)
	assert.Equal(t, "1234567_SB1", client.AccountID())
}

func TestUpsertPutsByExternalIDAndReadsTheLocation(t *testing.T) {
	t.Parallel()

	var base string
	client, server := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertWrite(t, r, http.MethodPut, testRecordBase+"/invoice/eid:trn-0a1b2c3d-1")
		body := readJSON(t, r)
		assert.Equal(t, "trn-0a1b2c3d-1", body["externalId"])
		created(w, base+testRecordBase+"/invoice/1201")
	})
	base = server.URL

	id, err := client.Upsert(t.Context(), netsuite.RecordWrite{
		Type:       "invoice",
		ExternalID: "trn-0a1b2c3d-1",
		Body:       map[string]any{"externalId": "trn-0a1b2c3d-1"},
	})
	require.NoError(t, err)
	assert.Equal(t, "1201", id)
}

func TestUpsertAcceptsOKAndCreated(t *testing.T) {
	t.Parallel()

	for _, status := range []int{http.StatusOK, http.StatusCreated} {
		var base string
		client, server := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Location", base+testRecordBase+"/customer/341")
			w.WriteHeader(status)
		})
		base = server.URL
		id, err := client.Upsert(t.Context(), netsuite.RecordWrite{
			Type: "customer", ExternalID: "trn-c1", Body: map[string]any{},
		})
		require.NoError(t, err, status)
		assert.Equal(t, "341", id)
	}
}

func TestUpsertValidatesInput(t *testing.T) {
	t.Parallel()

	client, _ := newAPIClient(t, func(http.ResponseWriter, *http.Request) {
		t.Error("no request is sent")
	})
	body := map[string]any{}
	for _, external := range []string{"", "a b", "trn/1", "eid:x", "a?b", strings.Repeat("a", 256)} {
		_, err := client.Upsert(t.Context(), netsuite.RecordWrite{
			Type: "invoice", ExternalID: external, Body: body,
		})
		require.ErrorIs(t, err, netsuite.ErrInvalidExternalID, external)
	}
	for _, kind := range []string{"", "../invoice", "invoice/1", "1invoice"} {
		_, err := client.Upsert(t.Context(), netsuite.RecordWrite{
			Type: kind, ExternalID: "trn-1", Body: body,
		})
		require.ErrorIs(t, err, netsuite.ErrInvalidRecordType, kind)
	}
	_, err := client.Upsert(t.Context(), netsuite.RecordWrite{Type: "invoice", ExternalID: "x"})
	require.ErrorIs(t, err, netsuite.ErrBodyRequired)
}

func TestUpsertWithoutLocationIsUnexpected(t *testing.T) {
	t.Parallel()

	client, _ := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	_, err := client.Upsert(t.Context(), netsuite.RecordWrite{
		Type: "invoice", ExternalID: "trn-1", Body: map[string]any{},
	})
	require.ErrorIs(t, err, netsuite.ErrUnexpectedPayload)
}

func TestLocationOnAForeignHostIsRefused(t *testing.T) {
	t.Parallel()

	client, _ := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		created(w, "https://evil.example"+testRecordBase+"/invoice/1201")
	})
	_, err := client.Create(t.Context(), "invoice", map[string]any{})
	require.ErrorIs(t, err, netsuite.ErrForeignLocation)

	client, _ = newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		created(w, "/services/rest/query/v1/invoice/1201")
	})
	_, err = client.Create(t.Context(), "invoice", map[string]any{})
	require.ErrorIs(t, err, netsuite.ErrForeignLocation)

	client, _ = newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		created(w, testRecordBase+"/vendorBill/1201")
	})
	_, err = client.Create(t.Context(), "invoice", map[string]any{})
	require.ErrorIs(t, err, netsuite.ErrUnexpectedPayload)
}

func TestCreatePostsAndReadsTheLocation(t *testing.T) {
	t.Parallel()

	var base string
	client, server := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertWrite(t, r, http.MethodPost, testRecordBase+"/vendorBill")
		body := readJSON(t, r)
		assert.Equal(t, "BILL-7", body["tranId"])
		created(w, base+testRecordBase+"/vendorbill/88")
	})
	base = server.URL

	id, err := client.Create(t.Context(), "vendorBill", map[string]any{"tranId": "BILL-7"})
	require.NoError(t, err)
	assert.Equal(t, "88", id)
}

func TestPatchSendsReplace(t *testing.T) {
	t.Parallel()

	client, _ := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertWrite(t, r, http.MethodPatch, testRecordBase+"/invoice/1201")
		assert.Equal(t, "item,apply", r.URL.Query().Get("replace"))
		body := readJSON(t, r)
		assert.Contains(t, body, "item")
		w.WriteHeader(http.StatusNoContent)
	})

	err := client.Patch(t.Context(), netsuite.RecordPatch{
		Type:    "invoice",
		ID:      "1201",
		Body:    map[string]any{"item": map[string]any{"items": []any{}}},
		Replace: []string{"item", "apply"},
	})
	require.NoError(t, err)

	err = client.Patch(t.Context(), netsuite.RecordPatch{
		Type: "invoice", ID: "1201", Body: map[string]any{}, Replace: []string{"item,apply"},
	})
	require.ErrorIs(t, err, netsuite.ErrInvalidFieldName)
}

func TestPatchWithoutReplaceAndByExternalID(t *testing.T) {
	t.Parallel()

	client, _ := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertWrite(t, r, http.MethodPatch, testRecordBase+"/creditMemo/eid:trn-9")
		assert.Empty(t, r.URL.RawQuery)
		w.WriteHeader(http.StatusNoContent)
	})
	require.NoError(t, client.Patch(t.Context(), netsuite.RecordPatch{
		Type: "creditMemo", ID: "eid:trn-9", Body: map[string]any{"memo": "x"},
	}))
}

func TestDelete(t *testing.T) {
	t.Parallel()

	client, _ := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, testRecordBase+"/customerPayment/5001", r.URL.Path)
		assertAPIHeaders(t, r)
		w.WriteHeader(http.StatusNoContent)
	})
	require.NoError(t, client.Delete(t.Context(), "customerPayment", "5001"))

	for _, id := range []string{"", "12a", "eid:", "eid:a b", "../1"} {
		require.Error(t, client.Delete(t.Context(), "customerPayment", id), id)
	}
}

func TestGetExpandsSubResources(t *testing.T) {
	t.Parallel()

	client, _ := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRead(t, r, testRecordBase+"/invoice/1201")
		assert.Equal(t, "true", r.URL.Query().Get("expandSubResources"))
		assert.Equal(t, "tranId,item", r.URL.Query().Get("fields"))
		_, _ = w.Write(fixture(t, "invoice.json"))
	})

	var invoice struct {
		ID     string `json:"id"`
		TranID string `json:"tranId"`
		Entity struct {
			ID string `json:"id"`
		} `json:"entity"`
		Item struct {
			Items []struct {
				Amount float64 `json:"amount"`
			} `json:"items"`
		} `json:"item"`
	}
	err := client.Get(t.Context(), netsuite.RecordRead{
		Type:               "invoice",
		ID:                 "1201",
		ExpandSubResources: true,
		Fields:             []string{"tranId", "item"},
	}, &invoice)
	require.NoError(t, err)
	assert.Equal(t, "1201", invoice.ID)
	assert.Equal(t, "INV-1001", invoice.TranID)
	assert.Equal(t, "341", invoice.Entity.ID)
	require.Len(t, invoice.Item.Items, 1)
	assert.InDelta(t, 1250.0, invoice.Item.Items[0].Amount, 0.001)

	require.ErrorIs(t, client.Get(t.Context(), netsuite.RecordRead{
		Type: "invoice", ID: "1",
	}, nil), netsuite.ErrOutRequired)
}

func TestGetWithoutOptionsSendsNoQuery(t *testing.T) {
	t.Parallel()

	client, _ := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, testRecordBase+"/invoice/eid:trn-1", r.URL.Path)
		assert.Empty(t, r.URL.RawQuery)
		_, _ = w.Write(fixture(t, "invoice.json"))
	})
	var out map[string]any
	require.NoError(t, client.Get(t.Context(), netsuite.RecordRead{
		Type: "invoice", ID: "eid:trn-1",
	}, &out))
	assert.Equal(t, "1201", out["id"])
}

func TestReadsAreRetriedOnServerErrors(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client, _ := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			serveFixture(t, http.StatusInternalServerError, "error_500.json")(w, nil)
			return
		}
		_, _ = w.Write(fixture(t, "invoice.json"))
	})
	var out map[string]any
	require.NoError(t, client.Get(t.Context(), netsuite.RecordRead{Type: "invoice", ID: "1"}, &out))
	assert.Equal(t, int32(2), calls.Load())
}

func TestWritesAreNeverRetried(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	handler := func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		serveFixture(t, http.StatusInternalServerError, "error_500.json")(w, r)
	}
	client, _ := newAPIClient(t, handler)

	_, err := client.Upsert(t.Context(), netsuite.RecordWrite{
		Type: "invoice", ExternalID: "trn-1", Body: map[string]any{},
	})
	require.Error(t, err)
	assert.True(t, netsuite.IsTransient(err))
	_, err = client.Create(t.Context(), "invoice", map[string]any{})
	require.Error(t, err)
	require.Error(t, client.Patch(t.Context(), netsuite.RecordPatch{
		Type: "invoice", ID: "1", Body: map[string]any{},
	}))
	require.Error(t, client.Delete(t.Context(), "invoice", "1"))
	assert.Equal(t, int32(4), calls.Load(), "each write is sent once")
}

func TestTransportErrorOnAWriteIsNotRetried(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client, _ := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		hijacker, ok := w.(http.Hijacker)
		require.True(t, ok)
		conn, _, err := hijacker.Hijack()
		require.NoError(t, err)
		_ = conn.Close()
	})

	_, err := client.Create(t.Context(), "invoice", map[string]any{})
	require.Error(t, err)
	assert.True(t, netsuite.IsTransient(err))
	assert.Equal(t, int32(1), calls.Load())
	assert.NotContains(t, err.Error(), "super-secret-access")
}

func TestLimiterBucketIsPerAccount(t *testing.T) {
	t.Parallel()

	limiter := &recordingLimiter{}
	client, _ := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(fixture(t, "invoice.json"))
	}, netsuite.WithLimiter(limiter, ""))

	var out map[string]any
	require.NoError(t, client.Get(t.Context(), netsuite.RecordRead{Type: "invoice", ID: "1"}, &out))
	require.Len(t, limiter.buckets, 1)
	assert.Equal(t, netsuite.AccountBucket("", testAccount), limiter.buckets[0])
	assert.Equal(t, "netsuite:1234567_sb1", limiter.buckets[0].Key)
	assert.Equal(t, 300, limiter.buckets[0].Limit)
	assert.Equal(t, time.Minute, limiter.buckets[0].Period)
	assert.Equal(t, 2, limiter.buckets[0].Burst)
	assert.NotEqual(t, limiter.buckets[0].Key, netsuite.AccountBucket("", "1234567").Key)
	assert.Equal(t, "acct:7654321", netsuite.AccountBucket("acct", "7654321").Key)
}
