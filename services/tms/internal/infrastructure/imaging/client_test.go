package imaging

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/imaging/imagingpb"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
)

type fakeWorker struct {
	imagingpb.UnimplementedImagingServiceServer

	mu        sync.Mutex
	probes    atomic.Int32
	refuse    int32
	priority  []string
	received  [][]byte
	pageCount uint32
}

func refusal(code codes.Code, reason string, retry time.Duration) error {
	st, err := status.New(code, "refused").WithDetails(
		&errdetails.ErrorInfo{Reason: reason, Domain: "imaging.trenova.app"},
		&errdetails.RetryInfo{RetryDelay: durationpb.New(retry)},
	)
	if err != nil {
		return status.Error(code, "refused")
	}
	return st.Err()
}

func (w *fakeWorker) Probe(stream imagingpb.ImagingService_ProbeServer) error {
	n := w.probes.Add(1)
	md, _ := metadata.FromIncomingContext(stream.Context())
	w.mu.Lock()
	w.priority = append(w.priority, md.Get(priorityKey)...)
	w.mu.Unlock()

	first, err := stream.Recv()
	if err != nil {
		return err
	}
	if first.GetHeader() == nil {
		return status.Error(codes.InvalidArgument, "no header")
	}
	if n <= w.refuse {
		return refusal(codes.ResourceExhausted, "RESOURCE_EXHAUSTED", 10*time.Millisecond)
	}
	var body []byte
	for {
		msg, recvErr := stream.Recv()
		if errors.Is(recvErr, io.EOF) {
			break
		}
		if recvErr != nil {
			return recvErr
		}
		body = append(body, msg.GetChunk()...)
	}
	if string(body) == "encrypted" {
		return refusal(codes.InvalidArgument, "INVALID_DOCUMENT", 0)
	}
	w.mu.Lock()
	w.received = append(w.received, body)
	w.mu.Unlock()
	return stream.SendAndClose(&imagingpb.ProbeResponse{
		Kind:      imagingpb.DocumentKind_DOCUMENT_KIND_PDF,
		PageCount: max(w.pageCount, 1),
	})
}

func (w *fakeWorker) Render(stream imagingpb.ImagingService_RenderServer) error {
	for {
		_, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
	}
	page := []byte("0123456789")
	if err := stream.Send(&imagingpb.RenderResponse{Part: &imagingpb.RenderResponse_Page{
		Page: &imagingpb.ArtifactHeader{Index: 0, ContentType: "image/webp", WidthPx: 10, HeightPx: 12, Size: 10},
	}}); err != nil {
		return err
	}
	for i, piece := range [][]byte{page[:4], page[4:]} {
		if err := stream.Send(&imagingpb.RenderResponse{Part: &imagingpb.RenderResponse_Chunk{
			Chunk: &imagingpb.ArtifactChunk{Data: piece, Last: i == 1},
		}}); err != nil {
			return err
		}
	}
	if err := stream.Send(&imagingpb.RenderResponse{Part: &imagingpb.RenderResponse_PageError{
		PageError: &imagingpb.PageError{PageIndex: 1, Reason: imagingpb.ErrorReason_ERROR_REASON_PAGE_OUT_OF_RANGE},
	}}); err != nil {
		return err
	}
	return stream.Send(&imagingpb.RenderResponse{Part: &imagingpb.RenderResponse_Summary{
		Summary: &imagingpb.RenderSummary{PageCount: 1, Rendered: 1},
	}})
}

func serve(t *testing.T, worker *fakeWorker) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer()
	imagingpb.RegisterImagingServiceServer(server, worker)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
	return listener.Addr().String()
}

func dial(t *testing.T, addresses ...string) *Client {
	t.Helper()
	cfg := &config.ImagingConfig{
		Endpoint:       "passthrough:///" + addresses[0],
		ChunkBytes:     16 << 10,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     5 * time.Millisecond,
	}
	client, err := Dial(cfg, zap.NewNop())
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestAProbeStreamsItsSourceInChunksAndCarriesItsPriority(t *testing.T) {
	t.Parallel()

	worker := &fakeWorker{pageCount: 3}
	client := dial(t, serve(t, worker))
	body := make([]byte, 50<<10)
	for i := range body {
		body[i] = byte(i)
	}

	probe, err := client.Probe(t.Context(), &kernelports.ImagingProbeRequest{
		Source:   kernelports.ImagingBytes("application/pdf", body),
		Priority: kernelports.ImagingInteractive,
	})
	require.NoError(t, err)
	assert.Equal(t, 3, probe.PageCount)
	require.Len(t, worker.received, 1)
	assert.Equal(t, body, worker.received[0])
	assert.Equal(t, []string{"interactive"}, worker.priority)
}

func TestARefusalForCapacityIsRetriedAfterItsRetryDelay(t *testing.T) {
	t.Parallel()

	worker := &fakeWorker{refuse: 2}
	client := dial(t, serve(t, worker))

	_, err := client.Probe(t.Context(), &kernelports.ImagingProbeRequest{
		Source: kernelports.ImagingBytes("application/pdf", []byte("%PDF-1.7")),
	})
	require.NoError(t, err)
	assert.Equal(t, int32(3), worker.probes.Load(), "two refusals, then the call that went through")
}

func TestRetriesStopAtTheAttemptLimit(t *testing.T) {
	t.Parallel()

	worker := &fakeWorker{refuse: 100}
	client := dial(t, serve(t, worker))

	_, err := client.Probe(t.Context(), &kernelports.ImagingProbeRequest{
		Source: kernelports.ImagingBytes("application/pdf", []byte("%PDF-1.7")),
	})
	require.ErrorIs(t, err, kernelports.ErrImagingUnavailable)
	assert.Equal(t, int32(4), worker.probes.Load())
}

func TestAnInvalidDocumentIsNotRetriedAndMapsToItsSentinel(t *testing.T) {
	t.Parallel()

	worker := &fakeWorker{}
	client := dial(t, serve(t, worker))

	_, err := client.Probe(t.Context(), &kernelports.ImagingProbeRequest{
		Source: kernelports.ImagingBytes("application/pdf", []byte("encrypted")),
	})
	require.ErrorIs(t, err, kernelports.ErrImagingInvalidDocument)
	assert.Equal(t, int32(1), worker.probes.Load())
}

func TestASourceLongerThanDeclaredIsRefusedBeforeItIsSent(t *testing.T) {
	t.Parallel()

	client := dial(t, serve(t, &fakeWorker{}))
	source := kernelports.ImagingBytes("application/pdf", []byte("%PDF-1.7 body"))
	source.Size = 4

	_, err := client.Probe(t.Context(), &kernelports.ImagingProbeRequest{Source: source})
	require.ErrorIs(t, err, kernelports.ErrImagingInvalidDocument)
}

func TestRenderedPagesStreamAsReadersAndPageErrorsArriveInBand(t *testing.T) {
	t.Parallel()

	client := dial(t, serve(t, &fakeWorker{}))
	var pages []string
	var pageErrors []error
	summary, err := client.Render(t.Context(), &kernelports.ImagingRenderRequest{
		Source:   kernelports.ImagingBytes("application/pdf", []byte("%PDF-1.7")),
		FitWidth: 240,
	}, func(artifact *kernelports.ImagingArtifact) error {
		if artifact.Err != nil {
			pageErrors = append(pageErrors, artifact.Err)
			return nil
		}
		data, readErr := io.ReadAll(artifact.Body)
		require.NoError(t, readErr)
		pages = append(pages, string(data))
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, summary.Rendered)
	assert.Equal(t, []string{"0123456789"}, pages)
	require.Len(t, pageErrors, 1)
	assert.ErrorIs(t, pageErrors[0], kernelports.ErrImagingPageOutOfRange)
}

func TestAnUnreadPageBodyIsDrainedSoTheStreamCarriesOn(t *testing.T) {
	t.Parallel()

	client := dial(t, serve(t, &fakeWorker{}))
	summary, err := client.Render(t.Context(), &kernelports.ImagingRenderRequest{
		Source:   kernelports.ImagingBytes("application/pdf", []byte("%PDF-1.7")),
		FitWidth: 240,
	}, func(*kernelports.ImagingArtifact) error { return nil })
	require.NoError(t, err)
	assert.Equal(t, 1, summary.PageCount)
}

func TestTheBreakerTakesAFailingEndpointOutOfRotation(t *testing.T) {
	t.Parallel()

	breaker := &config.ImagingBreakerConfig{ConsecutiveFailures: 2, OpenFor: time.Minute}
	e := &endpoint{address: "a"}
	now := time.Now()
	unavailable := status.Error(codes.Unavailable, "gone")

	e.record(unavailable, breaker, now)
	assert.True(t, e.available(now), "one failure is not enough")
	e.record(unavailable, breaker, now)
	assert.False(t, e.available(now), "two in a row open the breaker")
	assert.True(t, e.available(now.Add(2*time.Minute)), "it closes again after openFor")

	e.record(nil, breaker, now)
	e.record(status.Error(codes.InvalidArgument, "bad document"), breaker, now)
	e.record(status.Error(codes.InvalidArgument, "bad document"), breaker, now)
	assert.True(t, e.available(now), "a bad document says nothing about the endpoint")

	draining := refusal(codes.Unavailable, "DRAINING", time.Second)
	e.record(draining, breaker, now)
	assert.False(t, e.available(now), "a draining worker is taken out at once")
}

func TestPickSkipsEndpointsAlreadyTried(t *testing.T) {
	t.Parallel()

	set := &endpointSet{static: true, endpoints: []*endpoint{{address: "a"}, {address: "b"}}}
	first, err := set.pick(t.Context(), map[string]struct{}{})
	require.NoError(t, err)
	second, err := set.pick(t.Context(), map[string]struct{}{first.address: {}})
	require.NoError(t, err)
	assert.NotEqual(t, first.address, second.address)
}

func TestWithoutCapacityTheErrorSaysTheServiceIsUnavailable(t *testing.T) {
	t.Parallel()

	err := mapError(context.DeadlineExceeded)
	assert.ErrorIs(t, err, kernelports.ErrImagingTimeout)
	assert.ErrorIs(t, mapError(refusal(codes.ResourceExhausted, "RESOURCE_EXHAUSTED", time.Second)),
		kernelports.ErrImagingUnavailable)
	assert.ErrorIs(
		t,
		mapError(refusal(codes.FailedPrecondition, "ENCRYPTED", 0)),
		kernelports.ErrImagingEncrypted,
	)
}
