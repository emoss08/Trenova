package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/datausageports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const dataUsageMeasuredAt = int64(1_794_830_400)

type fakeDataUsage struct {
	usage *datausageports.Usage
	calls int
}

func (f *fakeDataUsage) Usage(context.Context) *datausageports.Usage {
	f.calls++
	return f.usage
}

func serveDataUsage(
	t *testing.T,
	usage *fakeDataUsage,
	method, token string,
) *httptest.ResponseRecorder {
	t.Helper()
	handler := api.NewManagedHandler(&api.ManagedHandlerParams{
		Token:     managedToken,
		Logger:    zap.NewNop(),
		DataUsage: usage,
	})
	req := httptest.NewRequestWithContext(t.Context(), method, "/managed/v1/data-usage", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func bytesOf(n int64) *int64 { return &n }

func TestManagedDataUsageAnswersEveryMeasuredFigure(t *testing.T) {
	t.Parallel()

	usage := &fakeDataUsage{usage: &datausageports.Usage{
		MeasuredAt:     dataUsageMeasuredAt,
		DatabaseBytes:  bytesOf(123),
		DocumentsBytes: bytesOf(456),
		SearchBytes:    bytesOf(789),
	}}

	rec := serveDataUsage(t, usage, http.MethodGet, managedToken)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	assert.JSONEq(
		t,
		`{"measuredAt":"2026-11-16T12:00:00Z","databaseBytes":123,"documentsBytes":456,"searchBytes":789}`,
		rec.Body.String(),
	)
}

func TestManagedDataUsageLeavesOutWhatWasNotMeasured(t *testing.T) {
	t.Parallel()

	usage := &fakeDataUsage{usage: &datausageports.Usage{
		MeasuredAt:    dataUsageMeasuredAt,
		DatabaseBytes: bytesOf(0),
	}}

	rec := serveDataUsage(t, usage, http.MethodGet, managedToken)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"measuredAt":"2026-11-16T12:00:00Z","databaseBytes":0}`, rec.Body.String(),
		"a measured zero is sent, an unknown figure is left out")

	usage.usage = &datausageports.Usage{MeasuredAt: dataUsageMeasuredAt}
	rec = serveDataUsage(t, usage, http.MethodGet, managedToken)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"measuredAt":"2026-11-16T12:00:00Z"}`, rec.Body.String())
}

func TestManagedDataUsageRefusesAWrongToken(t *testing.T) {
	t.Parallel()

	usage := &fakeDataUsage{usage: &datausageports.Usage{
		MeasuredAt:    dataUsageMeasuredAt,
		DatabaseBytes: bytesOf(123),
	}}

	for name, token := range map[string]string{"no token": "", "a wrong token": "not-the-token"} {
		rec := serveDataUsage(t, usage, http.MethodGet, token)
		assert.Equal(t, http.StatusUnauthorized, rec.Code, name)
		assert.NotContains(t, rec.Body.String(), "databaseBytes", name)
	}
	assert.Zero(t, usage.calls, "nothing is read for a caller that is not the operator")
}

func TestManagedDataUsageIsReadOnly(t *testing.T) {
	t.Parallel()

	rec := serveDataUsage(t, &fakeDataUsage{}, http.MethodPut, managedToken)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestManagedDataUsageIsNotServedWhenManagedModeIsOff(t *testing.T) {
	t.Parallel()

	off := &recordedLifecycle{}
	api.StartManagedServer(api.ManagedServerParams{
		Config: &managedconfig.Config{Enabled: false},
		Logger: zap.NewNop(),
		LC:     off,
	})
	assert.Empty(
		t,
		off.hooks,
		"a self-hosted instance opens no operator listener, so no data usage route",
	)
}
