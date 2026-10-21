package imaging

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/imaging/imagingpb"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const priorityKey = "x-trenova-priority"

type Params struct {
	fx.In

	Config    *config.Config
	Logger    *zap.Logger
	Lifecycle fx.Lifecycle
}

type Client struct {
	cfg       *config.ImagingConfig
	endpoints *endpointSet
	logger    *zap.Logger
}

var _ kernelports.Imaging = (*Client)(nil)

func New(p Params) (kernelports.Imaging, error) {
	client, err := Dial(&p.Config.Imaging, p.Logger)
	if err != nil {
		return nil, err
	}
	p.Lifecycle.Append(fx.Hook{
		OnStop: func(context.Context) error {
			return client.Close()
		},
	})
	return client, nil
}

func Dial(cfg *config.ImagingConfig, logger *zap.Logger) (*Client, error) {
	if logger == nil {
		logger = zap.NewNop()
	}
	creds, err := transportCredentials(cfg)
	if err != nil {
		return nil, err
	}
	options := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                cfg.GetKeepaliveTime(),
			Timeout:             cfg.GetKeepaliveTimeout(),
			PermitWithoutStream: true,
		}),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(cfg.GetMaxMessageBytes()),
			grpc.MaxCallSendMsgSize(cfg.GetMaxMessageBytes()),
		),
	}
	named := logger.Named("imaging")
	endpoints, err := newEndpointSet(cfg, named, func(target string) (*grpc.ClientConn, error) {
		conn, dialErr := grpc.NewClient(target, options...)
		if dialErr != nil {
			return nil, fmt.Errorf("connect to the imaging service at %s: %w", target, dialErr)
		}
		return conn, nil
	})
	if err != nil {
		return nil, err
	}

	return &Client{cfg: cfg, endpoints: endpoints, logger: named}, nil
}

func (c *Client) Close() error {
	return c.endpoints.close()
}

func transportCredentials(cfg *config.ImagingConfig) (credentials.TransportCredentials, error) {
	if !cfg.TLS.Enabled {
		return insecure.NewCredentials(), nil
	}
	pair, err := tls.LoadX509KeyPair(cfg.TLS.CertFile, cfg.TLS.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("load the imaging client certificate: %w", err)
	}
	authority, err := os.ReadFile(cfg.TLS.CAFile)
	if err != nil {
		return nil, fmt.Errorf("read the imaging CA: %w", err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(authority) {
		return nil, errors.New("the imaging CA file holds no certificate")
	}

	return credentials.NewTLS(&tls.Config{
		MinVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{pair},
		RootCAs:      roots,
		ServerName:   cfg.ServerName,
	}), nil
}

func withPriority(ctx context.Context, priority kernelports.ImagingPriority) context.Context {
	value := "batch"
	if priority == kernelports.ImagingInteractive {
		value = "interactive"
	}
	return metadata.AppendToOutgoingContext(ctx, priorityKey, value)
}

func (c *Client) call(
	ctx context.Context,
	deadline time.Duration,
	priority kernelports.ImagingPriority,
	attempt func(ctx context.Context, rpc imagingpb.ImagingServiceClient, progressed *bool) error,
) error {
	ctx, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()
	ctx = withPriority(ctx, priority)

	backoff := c.cfg.GetInitialBackoff()
	tried := make(map[string]struct{}, c.cfg.GetMaxAttempts())
	var last error
	for try := 1; try <= c.cfg.GetMaxAttempts(); try++ {
		target, err := c.endpoints.pick(ctx, tried)
		if err != nil {
			last = err
			break
		}
		tried[target.address] = struct{}{}
		progressed := false
		err = attempt(ctx, target.rpc, &progressed)
		target.record(err, &c.cfg.Breaker, time.Now())
		if err == nil {
			return nil
		}
		last = err
		wait, retry := retryable(err)
		if !retry || progressed || try == c.cfg.GetMaxAttempts() {
			break
		}
		if wait <= 0 {
			wait = backoff
		}
		wait += time.Duration(rand.Int64N(int64(wait/4) + 1)) //nolint:gosec // jitter, not a secret
		backoff = min(backoff*2, c.cfg.GetMaxBackoff())
		c.logger.Debug(
			"imaging call refused; retrying",
			zap.Int("attempt", try),
			zap.Duration("wait", wait),
		)

		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return mapError(ctx.Err())
		case <-timer.C:
		}
	}

	return mapError(last)
}

func retryable(err error) (time.Duration, bool) {
	st, ok := status.FromError(err)
	if !ok {
		return 0, false
	}
	if code := st.Code(); code != codes.ResourceExhausted && code != codes.Unavailable {
		return 0, false
	}
	for _, detail := range st.Details() {
		if info, isRetry := detail.(*errdetails.RetryInfo); isRetry && info.GetRetryDelay() != nil {
			return info.GetRetryDelay().AsDuration(), true
		}
	}
	return 0, true
}

func mapError(err error) error {
	if err == nil {
		return nil
	}
	var page *kernelports.ImagingPageError
	if errors.As(err, &page) {
		return err
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("%w: %w", kernelports.ErrImagingTimeout, err)
	}
	if errors.Is(err, context.Canceled) {
		return err
	}
	st, ok := status.FromError(err)
	if !ok {
		return err
	}
	return fmt.Errorf("%w: %s", sentinel(reasonOf(err), st.Code()), st.Message())
}

func reasonOf(err error) string {
	st, ok := status.FromError(err)
	if !ok {
		return ""
	}
	for _, detail := range st.Details() {
		if info, isInfo := detail.(*errdetails.ErrorInfo); isInfo {
			return info.GetReason()
		}
	}
	return ""
}

func sentinel(reason string, code codes.Code) error {
	switch reason {
	case "INVALID_DOCUMENT", "INVALID_REQUEST":
		return kernelports.ErrImagingInvalidDocument
	case "ENCRYPTED":
		return kernelports.ErrImagingEncrypted
	case "TOO_LARGE":
		return kernelports.ErrImagingTooLarge
	case "PIXEL_BUDGET":
		return kernelports.ErrImagingPixelBudget
	case "UNSUPPORTED_FORMAT":
		return kernelports.ErrImagingUnsupported
	case "PAGE_OUT_OF_RANGE":
		return kernelports.ErrImagingPageOutOfRange
	case "TIMEOUT":
		return kernelports.ErrImagingTimeout
	case "RESOURCE_EXHAUSTED", "DRAINING":
		return kernelports.ErrImagingUnavailable
	}
	return codeSentinel(code)
}

func codeSentinel(code codes.Code) error {
	if code == codes.DeadlineExceeded {
		return kernelports.ErrImagingTimeout
	}
	if code == codes.Unavailable || code == codes.ResourceExhausted {
		return kernelports.ErrImagingUnavailable
	}
	if code == codes.InvalidArgument {
		return kernelports.ErrImagingInvalidDocument
	}
	return kernelports.ErrImagingInternal
}

func pageError(err *imagingpb.PageError) *kernelports.ImagingPageError {
	if err == nil {
		return nil
	}
	reason := imagingpb.ErrorReason_name[int32(err.GetReason())]
	return &kernelports.ImagingPageError{
		Reason:  sentinel(trimReason(reason), codes.Unknown),
		Message: err.GetMessage(),
	}
}

func trimReason(name string) string {
	const prefix = "ERROR_REASON_"
	if len(name) > len(prefix) && name[:len(prefix)] == prefix {
		return name[len(prefix):]
	}
	return name
}

func document(source kernelports.ImagingSource) *imagingpb.DocumentHeader {
	return &imagingpb.DocumentHeader{
		ContentType:  source.ContentType,
		DeclaredSize: sizeOf(source),
	}
}

func sendSource(
	ctx context.Context,
	source kernelports.ImagingSource,
	chunkBytes int,
	buffer []byte,
	send func(chunk []byte) error,
) error {
	if source.Open == nil {
		return fmt.Errorf("%w: the source cannot be opened", kernelports.ErrImagingInvalidDocument)
	}
	body, err := source.Open(ctx)
	if err != nil {
		return fmt.Errorf("open the document for imaging: %w", err)
	}
	defer body.Close()

	limited := io.LimitReader(body, source.Size+1)
	var sent int64
	for {
		n, readErr := io.ReadFull(limited, buffer[:chunkBytes])
		if n > 0 {
			sent += int64(n)
			if sent > source.Size {
				return fmt.Errorf(
					"%w: the document is longer than its declared size",
					kernelports.ErrImagingInvalidDocument,
				)
			}
			if err = send(buffer[:n]); err != nil {
				return err
			}
		}
		if errors.Is(readErr, io.EOF) || errors.Is(readErr, io.ErrUnexpectedEOF) {
			break
		}
		if readErr != nil {
			return fmt.Errorf("read the document for imaging: %w", readErr)
		}
	}
	if sent != source.Size {
		return fmt.Errorf(
			"%w: the document is shorter than its declared size",
			kernelports.ErrImagingInvalidDocument,
		)
	}
	return nil
}

func formatOf(format kernelports.ImagingFormat) imagingpb.ImageFormat {
	if format == kernelports.ImagingJPEG {
		return imagingpb.ImageFormat_IMAGE_FORMAT_JPEG
	}
	if format == kernelports.ImagingPNG {
		return imagingpb.ImageFormat_IMAGE_FORMAT_PNG
	}
	return imagingpb.ImageFormat_IMAGE_FORMAT_WEBP
}

func toUint32s(values []int) []uint32 {
	out := make([]uint32, 0, len(values))
	for _, v := range values {
		out = append(out, u32(v))
	}
	return out
}

func sizeOf(source kernelports.ImagingSource) uint64 {
	if source.Size <= 0 {
		return 0
	}
	return uint64(source.Size)
}
