package temporaljobs

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/temporalconfig"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/pollwatch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type fakeFrontend struct {
	workflowservice.UnimplementedWorkflowServiceServer
}

func (fakeFrontend) GetSystemInfo(
	context.Context,
	*workflowservice.GetSystemInfoRequest,
) (*workflowservice.GetSystemInfoResponse, error) {
	return &workflowservice.GetSystemInfoResponse{
		Capabilities: &workflowservice.GetSystemInfoResponse_Capabilities{},
	}, nil
}

func (fakeFrontend) DescribeNamespace(
	context.Context,
	*workflowservice.DescribeNamespaceRequest,
) (*workflowservice.DescribeNamespaceResponse, error) {
	return &workflowservice.DescribeNamespaceResponse{}, nil
}

func serveFrontendLater(t *testing.T, addr string, after time.Duration) {
	t.Helper()

	server := grpc.NewServer()
	workflowservice.RegisterWorkflowServiceServer(server, fakeFrontend{})
	checker := health.NewServer()
	checker.SetServingStatus(
		"temporal.api.workflowservice.v1.WorkflowService",
		healthpb.HealthCheckResponse_SERVING,
	)
	healthpb.RegisterHealthServer(server, checker)

	started := make(chan struct{})
	go func() {
		defer close(started)
		time.Sleep(after)
		l, err := net.Listen("tcp", addr)
		if err != nil {
			return
		}
		go func() { _ = server.Serve(l) }()
	}()

	t.Cleanup(func() {
		<-started
		server.Stop()
	})
}

func freeAddress(t *testing.T) string {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := l.Addr().String()
	require.NoError(t, l.Close())

	return addr
}

func engineParams(
	t *testing.T,
	addr string,
	wait time.Duration,
) (AwaitEngineParams, *observer.ObservedLogs) {
	t.Helper()

	c, err := client.NewLazyClient(client.Options{HostPort: addr, Namespace: "default"})
	require.NoError(t, err)
	t.Cleanup(c.Close)

	core, logs := observer.New(zap.InfoLevel)
	return AwaitEngineParams{
		Client:         c,
		TemporalConfig: &temporalconfig.Config{Namespace: "default"},
		AppConfig:      &appconfig.Config{StartupWait: wait},
		Logger:         zap.New(core),
	}, logs
}

func TestAwaitEngine_StartsWhenTemporalComesUpLate(t *testing.T) {
	addr := freeAddress(t)
	p, logs := engineParams(t, addr, time.Hour)
	serveFrontendLater(t, addr, 700*time.Millisecond)

	require.NoError(t, AwaitEngine(p))

	assert.Equal(t, 1, logs.FilterMessage("Waiting for Temporal to answer before starting").Len())
	assert.Equal(t, 1, logs.FilterMessage("Temporal answered; starting").Len())
}

func TestAwaitEngine_FailsOnceTheStartUpWaitIsSpent(t *testing.T) {
	p, _ := engineParams(t, freeAddress(t), 0)

	err := AwaitEngine(p)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Temporal did not answer within the start-up wait")
}

func TestSDKLogger_FailedPollsGoToTheWatchAndOtherWarningsStay(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	logger := zap.New(core)
	lc := fxtest.NewLifecycle(t)
	polls := pollwatch.New(pollwatch.Params{Logger: logger, LC: lc})
	lc.RequireStart()
	t.Cleanup(func() { lc.RequireStop() })
	sdk := NewSDKLogger(SDKLoggerParams{Logger: logger, Polls: polls})

	for range 70 {
		sdk.Warn("Failed to poll for task.", "Error", errors.New("connection refused"))
	}
	sdk.Warn("Worker heartbeat failed.", "Error", errors.New("boom"))

	assert.Equal(t, 1, logs.FilterMessageSnippet("Cannot reach the workflow engine").Len())
	assert.Equal(t, 1, logs.FilterMessage("Worker heartbeat failed.").Len())
	assert.Zero(t, logs.FilterMessage("Failed to poll for task.").Len())
}
