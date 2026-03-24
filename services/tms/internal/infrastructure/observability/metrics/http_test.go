package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewHTTP_Disabled(t *testing.T) {
	t.Parallel()

	m := NewHTTP(nil, zap.NewNop(), false)

	require.NotNil(t, m)
	assert.False(t, m.IsEnabled())
	assert.Nil(t, m.requestsTotal)
	assert.Nil(t, m.requestDuration)
	assert.Nil(t, m.responseSize)
	assert.Nil(t, m.activeRequests)
}

func TestNewHTTP_Enabled(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	m := NewHTTP(registry, zap.NewNop(), true)

	require.NotNil(t, m)
	assert.True(t, m.IsEnabled())
	assert.NotNil(t, m.requestsTotal)
	assert.NotNil(t, m.requestDuration)
	assert.NotNil(t, m.responseSize)
	assert.NotNil(t, m.activeRequests)
}

func TestHTTP_RecordHTTPRequest_Enabled(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	m := NewHTTP(registry, zap.NewNop(), true)

	assert.NotPanics(t, func() {
		m.RecordHTTPRequest("GET", "/api/users", 200, 0.05, 1024)
	})

	gathered, err := registry.Gather()
	require.NoError(t, err)

	found := map[string]bool{}
	for _, mf := range gathered {
		found[mf.GetName()] = true
	}
	assert.True(t, found["trenova_http_requests_total"])
	assert.True(t, found["trenova_http_request_duration_seconds"])
	assert.True(t, found["trenova_http_response_size_bytes"])
}

func TestHTTP_RecordHTTPRequest_Disabled(t *testing.T) {
	t.Parallel()

	m := NewHTTP(nil, zap.NewNop(), false)

	assert.NotPanics(t, func() {
		m.RecordHTTPRequest("POST", "/api/orders", 201, 0.1, 512)
	})
}

func TestHTTP_RecordHTTPRequest_MultipleStatuses(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	m := NewHTTP(registry, zap.NewNop(), true)

	m.RecordHTTPRequest("GET", "/api/items", 200, 0.01, 100)
	m.RecordHTTPRequest("GET", "/api/items", 404, 0.02, 50)
	m.RecordHTTPRequest("POST", "/api/items", 500, 0.5, 200)

	gathered, err := registry.Gather()
	require.NoError(t, err)

	var requestsCount int
	for _, mf := range gathered {
		if mf.GetName() == "trenova_http_requests_total" {
			requestsCount = len(mf.GetMetric())
			break
		}
	}

	assert.GreaterOrEqual(t, requestsCount, 3)
}

func TestHTTP_IncrementActiveRequests_Enabled(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	m := NewHTTP(registry, zap.NewNop(), true)

	assert.NotPanics(t, func() {
		m.IncrementActiveRequests()
		m.IncrementActiveRequests()
	})

	gathered, err := registry.Gather()
	require.NoError(t, err)

	for _, mf := range gathered {
		if mf.GetName() == "trenova_http_active_requests" {
			assert.Equal(t, float64(2), mf.GetMetric()[0].GetGauge().GetValue())
			return
		}
	}
	t.Fatal("trenova_http_active_requests metric not found")
}

func TestHTTP_DecrementActiveRequests_Enabled(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	m := NewHTTP(registry, zap.NewNop(), true)

	m.IncrementActiveRequests()
	m.IncrementActiveRequests()
	m.DecrementActiveRequests()

	gathered, err := registry.Gather()
	require.NoError(t, err)

	for _, mf := range gathered {
		if mf.GetName() == "trenova_http_active_requests" {
			assert.Equal(t, float64(1), mf.GetMetric()[0].GetGauge().GetValue())
			return
		}
	}
	t.Fatal("trenova_http_active_requests metric not found")
}

func TestHTTP_IncrementActiveRequests_Disabled(t *testing.T) {
	t.Parallel()

	m := NewHTTP(nil, zap.NewNop(), false)

	assert.NotPanics(t, func() {
		m.IncrementActiveRequests()
	})
}

func TestHTTP_DecrementActiveRequests_Disabled(t *testing.T) {
	t.Parallel()

	m := NewHTTP(nil, zap.NewNop(), false)

	assert.NotPanics(t, func() {
		m.DecrementActiveRequests()
	})
}

func TestHTTP_GinMiddleware_ReturnsNonNil(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	m := NewHTTP(registry, zap.NewNop(), true)

	handler := m.GinMiddleware()
	assert.NotNil(t, handler)
}

func TestHTTP_GinMiddleware_Disabled_ReturnsNonNil(t *testing.T) {
	t.Parallel()

	m := NewHTTP(nil, zap.NewNop(), false)

	handler := m.GinMiddleware()
	assert.NotNil(t, handler)
}
