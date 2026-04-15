package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	obsmetrics "github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type fakeDBConnection struct {
	healthErr error
}

func (f *fakeDBConnection) DB() *bun.DB {
	return nil
}

func (f *fakeDBConnection) DBForContext(context.Context) bun.IDB {
	return nil
}

func (f *fakeDBConnection) WithTx(
	context.Context,
	ports.TxOptions,
	func(context.Context, bun.Tx) error,
) error {
	return nil
}

func (f *fakeDBConnection) HealthCheck(context.Context) error {
	return f.healthErr
}

func (f *fakeDBConnection) IsHealthy(context.Context) bool {
	return f.healthErr == nil
}

func (f *fakeDBConnection) Close() error {
	return nil
}

func TestMonitoringServer_CustomPathsAndMetricsEnabled(t *testing.T) {
	t.Parallel()

	server, port := newTestMonitoringServer(t, true, nil)
	startTestMonitoringServer(t, server)

	assert.Equal(t, fmt.Sprintf("127.0.0.1:%d", port), server.server.Addr)

	liveBody := mustGET(t, fmt.Sprintf("http://127.0.0.1:%d/internal/livez", port))
	assert.Equal(t, http.StatusOK, liveBody.statusCode)
	assert.Contains(t, liveBody.body, `"status":"alive"`)

	readyBody := mustGET(t, fmt.Sprintf("http://127.0.0.1:%d/internal/readyz", port))
	assert.Equal(t, http.StatusOK, readyBody.statusCode)
	assert.Contains(t, readyBody.body, `"status":"ready"`)

	healthBody := mustGET(t, fmt.Sprintf("http://127.0.0.1:%d/internal/healthz", port))
	assert.Equal(t, http.StatusOK, healthBody.statusCode)
	assert.Contains(t, healthBody.body, `"status":"up"`)

	metricsBody := mustGET(t, fmt.Sprintf("http://127.0.0.1:%d/internal/metricsz", port))
	assert.Equal(t, http.StatusOK, metricsBody.statusCode)
	assert.Contains(t, metricsBody.body, "go_gc_duration_seconds")
}

func TestMonitoringServer_MetricsDisabledAndReadinessFails(t *testing.T) {
	t.Parallel()

	server, port := newTestMonitoringServer(t, false, errors.New("db unavailable"))
	startTestMonitoringServer(t, server)

	metricsBody := mustGET(t, fmt.Sprintf("http://127.0.0.1:%d/internal/metricsz", port))
	assert.Equal(t, http.StatusServiceUnavailable, metricsBody.statusCode)
	assert.Contains(t, metricsBody.body, "Metrics collection is disabled")

	readyBody := mustGET(t, fmt.Sprintf("http://127.0.0.1:%d/internal/readyz", port))
	assert.Equal(t, http.StatusServiceUnavailable, readyBody.statusCode)
	assert.Contains(t, readyBody.body, `"status":"not_ready"`)
	assert.Contains(t, readyBody.body, "db unavailable")

	healthBody := mustGET(t, fmt.Sprintf("http://127.0.0.1:%d/internal/healthz", port))
	assert.Equal(t, http.StatusServiceUnavailable, healthBody.statusCode)
	assert.Contains(t, healthBody.body, `"status":"down"`)

	liveBody := mustGET(t, fmt.Sprintf("http://127.0.0.1:%d/internal/livez", port))
	assert.Equal(t, http.StatusOK, liveBody.statusCode)
}

type httpResponse struct {
	statusCode int
	body       string
}

func newTestMonitoringServer(
	t *testing.T,
	metricsEnabled bool,
	dbErr error,
) (*MonitoringServer, int) {
	t.Helper()

	port := getFreePort(t)
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "127.0.0.1",
		},
		Monitoring: config.MonitoringConfig{
			Metrics: config.MetricsConfig{
				Enabled: metricsEnabled,
				Port:    port,
				Path:    "/internal/metricsz",
			},
			Health: config.HealthConfig{
				Path:          "/internal/healthz",
				ReadinessPath: "/internal/readyz",
				LivenessPath:  "/internal/livez",
			},
		},
	}

	metricsRegistry, err := obsmetrics.NewRegistry(cfg, zap.NewNop())
	require.NoError(t, err)

	server := &MonitoringServer{
		cfg:      cfg,
		logger:   zap.NewNop(),
		metrics:  metricsRegistry,
		database: &fakeDBConnection{healthErr: dbErr},
	}

	mux := http.NewServeMux()
	mux.Handle(cfg.Monitoring.Metrics.Path, metricsRegistry.Handler())
	mux.HandleFunc(cfg.Monitoring.Health.Path, server.handleHealth)
	mux.HandleFunc(cfg.Monitoring.Health.ReadinessPath, server.handleReadiness)
	mux.HandleFunc(cfg.Monitoring.Health.LivenessPath, server.handleLiveness)

	server.server = &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Monitoring.Metrics.Port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return server, port
}

func startTestMonitoringServer(t *testing.T, server *MonitoringServer) {
	t.Helper()

	require.NoError(t, server.Start(context.Background()))
	t.Cleanup(func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		require.NoError(t, server.Stop(shutdownCtx))
	})

	require.Eventually(t, func() bool {
		resp, err := http.Get("http://" + server.server.Addr + server.cfg.Monitoring.Health.LivenessPath)
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 5*time.Second, 50*time.Millisecond)
}

func mustGET(t *testing.T, url string) httpResponse {
	t.Helper()

	resp, err := http.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return httpResponse{
		statusCode: resp.StatusCode,
		body:       string(body),
	}
}

func getFreePort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	require.True(t, ok)
	require.NotZero(t, tcpAddr.Port)

	return tcpAddr.Port
}

func TestMonitoringServer_DatabaseCheckWithoutConnection(t *testing.T) {
	t.Parallel()

	server, _ := newTestMonitoringServer(t, true, nil)
	server.database = nil

	check := server.databaseCheck(context.Background())

	assert.Equal(t, "down", check.Status)
	assert.True(t, strings.Contains(check.Message, "database connection is not configured"))
}

var _ ports.DBConnection = (*fakeDBConnection)(nil)
