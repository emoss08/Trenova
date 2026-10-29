package imaging

import (
	"context"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config/imagingconfig"
	"github.com/emoss08/trenova/internal/infrastructure/imaging/imagingpb"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type heldWorker struct {
	fakeWorker

	renders atomic.Int32
	started chan struct{}
	release chan struct{}
}

func (w *heldWorker) Render(stream imagingpb.ImagingService_RenderServer) error {
	if w.renders.Add(1) == 1 && w.release != nil {
		close(w.started)
		<-w.release
	}
	return w.fakeWorker.Render(stream)
}

type addresses struct {
	mu   sync.Mutex
	list []string
}

func (a *addresses) set(list ...string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.list = list
}

func (a *addresses) lookup(context.Context) ([]string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.list...), nil
}

func render(ctx context.Context, client *Client) error {
	_, err := client.Render(ctx, &kernelports.ImagingRenderRequest{
		Source:   kernelports.ImagingBytes("application/pdf", []byte("%PDF-1.7")),
		FitWidth: 240,
	}, func(artifact *kernelports.ImagingArtifact) error {
		if artifact.Body != nil {
			_, readErr := io.Copy(io.Discard, artifact.Body)
			return readErr
		}
		return nil
	})
	return err
}

func TestARetiredEndpointFinishesItsCallsBeforeItsConnectionCloses(t *testing.T) {
	t.Parallel()

	leaving := &heldWorker{started: make(chan struct{}), release: make(chan struct{})}
	staying := &heldWorker{}
	addrs := &addresses{}
	addrs.set(serve(t, leaving))

	cfg := &imagingconfig.Config{InitialBackoff: time.Millisecond, MaxBackoff: 5 * time.Millisecond}
	set := &endpointSet{
		cfg:    cfg,
		logger: zap.NewNop(),
		lookup: addrs.lookup,
		dial: func(target string) (*grpc.ClientConn, error) {
			return grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
		},
		stop: make(chan struct{}),
		done: make(chan struct{}),
	}
	close(set.done)
	client := &Client{cfg: cfg, endpoints: set, logger: zap.NewNop()}
	t.Cleanup(func() { _ = client.Close() })
	require.NoError(t, set.resolve(t.Context()))
	retired := set.endpoints[0]

	inFlight := make(chan error, 1)
	go func() { inFlight <- render(t.Context(), client) }()
	select {
	case <-leaving.started:
	case <-time.After(10 * time.Second):
		t.Fatal("the render never reached the leaving endpoint")
	}

	addrs.set(serve(t, staying))
	require.NoError(t, set.resolve(t.Context()))
	require.NoError(t, render(t.Context(), client), "a new call goes to the remaining endpoint")
	assert.Equal(t, int32(1), staying.renders.Load())

	assert.NotEqual(
		t,
		connectivity.Shutdown,
		retired.conn.GetState(),
		"it stays open while a call is in flight",
	)
	close(leaving.release)
	select {
	case err := <-inFlight:
		require.NoError(t, err, "the call in flight on the retired endpoint completes")
	case <-time.After(10 * time.Second):
		t.Fatal("the call in flight never finished")
	}
	assert.Equal(t, int32(1), leaving.renders.Load())
	require.Eventually(t, func() bool {
		return retired.conn.GetState() == connectivity.Shutdown
	}, 5*time.Second, 10*time.Millisecond, "the retired connection closes once its calls end")
}

func TestAConnectionClosingUnderACallIsRetriedUnlessTheCallerCancelled(t *testing.T) {
	t.Parallel()

	closing := status.Error(codes.Canceled, "grpc: the client connection is closing")
	_, retry := retryable(t.Context(), closing)
	assert.True(t, retry, "a connection closing under the call is retried on another endpoint")

	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	_, retry = retryable(cancelled, closing)
	assert.False(t, retry, "the caller's own cancellation is not retried")
}
