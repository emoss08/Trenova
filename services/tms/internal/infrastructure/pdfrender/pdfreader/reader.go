package pdfreader

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/klippa-app/go-pdfium"
	"github.com/klippa-app/go-pdfium/requests"
	"github.com/klippa-app/go-pdfium/responses"
	"github.com/klippa-app/go-pdfium/webassembly"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/experimental"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const wasmPageBytes = 64 << 10

var errReaderClosed = errors.New("pdf reader is closed")

type Params struct {
	fx.In

	Config    *config.Config
	Logger    *zap.Logger
	Lifecycle fx.Lifecycle
}

type Reader struct {
	cfg    *config.PDFReaderConfig
	logger *zap.Logger

	mu     sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc
	pool   pdfium.Pool
	closed bool
}

var _ services.PDFReader = (*Reader)(nil)

func New(p Params) *Reader {
	r := NewReader(p.Config.GetPDFReaderConfig(), p.Logger)
	p.Lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go r.warm()
			return nil
		},
		OnStop: func(context.Context) error {
			return r.Close()
		},
	})
	return r
}

func NewReader(cfg *config.PDFReaderConfig, logger *zap.Logger) *Reader {
	if cfg == nil {
		cfg = &config.PDFReaderConfig{}
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Reader{
		cfg:    cfg,
		logger: logger.Named("pdfrender.pdfreader"),
		ctx:    ctx,
		cancel: cancel,
	}
}

func (r *Reader) Open(ctx context.Context, data []byte) (services.PDFDocument, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("open pdf: %w: empty input", services.ErrPDFUnreadable)
	}

	pool, err := r.ensurePool()
	if err != nil {
		return nil, err
	}

	acquireCtx, cancel := context.WithTimeout(ctx, r.cfg.GetAcquireTimeout())
	instance, err := pool.GetInstanceWithContext(acquireCtx)
	cancel()
	if err != nil {
		return nil, fmt.Errorf("acquire pdf reader instance: %w", err)
	}

	doc := &document{instance: instance, maxPixels: r.cfg.GetMaxRenderPixels()}

	var opened *responses.OpenDocument
	if err = doc.guard(ctx, "open pdf", func() error {
		var openErr error
		opened, openErr = instance.OpenDocument(&requests.OpenDocument{File: &data})
		return openErr
	}); err != nil {
		doc.release()
		return nil, err
	}
	doc.handle = opened.Document
	doc.opened = true

	var count *responses.FPDF_GetPageCount
	if err = doc.guard(ctx, "count pages", func() error {
		var countErr error
		count, countErr = instance.FPDF_GetPageCount(
			&requests.FPDF_GetPageCount{Document: doc.handle},
		)
		return countErr
	}); err != nil {
		return nil, errors.Join(err, doc.Close())
	}
	doc.pages = count.PageCount

	return doc, nil
}

func (r *Reader) warm() {
	if _, err := r.ensurePool(); err != nil && !errors.Is(err, errReaderClosed) {
		r.logger.Error("PDF reader failed to start; PDF features will retry on use", zap.Error(err))
	}
}

func (r *Reader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil
	}
	r.closed = true

	var err error
	if r.pool != nil {
		err = r.pool.Close()
		r.pool = nil
	}
	r.cancel()

	return err
}

func (r *Reader) ensurePool() (pdfium.Pool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil, errReaderClosed
	}
	if r.pool != nil {
		return r.pool, nil
	}

	instances := r.cfg.GetMaxInstances()
	limitPages := r.cfg.GetMemoryLimitMB() * (1 << 20) / wasmPageBytes
	memoryPages := uint32(limitPages) //nolint:gosec // validation caps the limit at 4096 MB
	logWriter := zap.NewStdLog(r.logger).Writer()

	pool, err := webassembly.Init(webassembly.Config{
		Context:  r.ctx,
		MinIdle:  0,
		MaxIdle:  instances,
		MaxTotal: instances,
		FSConfig: wazero.NewFSConfig(),
		RuntimeConfig: wazero.NewRuntimeConfig().
			WithCoreFeatures(api.CoreFeaturesV2 | experimental.CoreFeaturesExceptionHandling).
			WithCloseOnContextDone(true).
			WithMemoryLimitPages(memoryPages),
		Stdout: logWriter,
		Stderr: logWriter,
	})
	if err != nil {
		return nil, fmt.Errorf("initialize pdfium: %w", err)
	}

	r.logger.Info(
		"PDF reader ready",
		zap.Int("maxInstances", instances),
		zap.Int("memoryLimitMb", r.cfg.GetMemoryLimitMB()),
	)
	r.pool = pool

	return pool, nil
}
