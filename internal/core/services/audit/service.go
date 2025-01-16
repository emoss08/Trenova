package audit

import (
	"context"
	"time"

	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/sourcegraph/conc"
	"github.com/trenova-app/transport/internal/core/domain/audit"
	"github.com/trenova-app/transport/internal/core/domain/permission"
	"github.com/trenova-app/transport/internal/core/ports/repositories"
	"github.com/trenova-app/transport/internal/core/ports/services"
	"github.com/trenova-app/transport/internal/pkg/config"
	"github.com/trenova-app/transport/internal/pkg/logger"
	"github.com/trenova-app/transport/internal/pkg/utils/jsonutils"
	"github.com/trenova-app/transport/pkg/types/pulid"
	"go.uber.org/atomic"
	"go.uber.org/fx"
)

const (
	DefaultMaskValue     = "****"
	DefaultFlushTimeout  = 10 * time.Second
	DefaultWorkerTimeout = 5 * time.Second
	DefaultChunkSize     = 100
	DefaultMaxRetries    = 3
)

type ServiceParams struct {
	fx.In

	LC              fx.Lifecycle
	AuditRepository repositories.AuditRepository
	Logger          *logger.Logger
	Config          *config.Manager
}

type service struct {
	l         *zerolog.Logger
	buffer    *Buffer
	repo      repositories.AuditRepository
	config    *config.AuditConfig
	wg        *conc.WaitGroup
	sdm       *SensitiveDataManager
	isRunning atomic.Bool

	// Channels for goroutine control
	stopFlusher chan struct{}
	flusherDone chan struct{}
	stopMonitor chan struct{}
	monitorDone chan struct{}
}

func NewService(p ServiceParams) services.AuditService {
	log := p.Logger.With().Str("service", "audit").Logger()

	cfg := p.Config.Audit()

	auditService := &service{
		repo:        p.AuditRepository,
		l:           &log,
		buffer:      NewBuffer(cfg.BufferSize),
		config:      &p.Config.Get().Audit,
		sdm:         NewSensitiveDataManager(),
		wg:          conc.NewWaitGroup(),
		stopFlusher: make(chan struct{}),
		flusherDone: make(chan struct{}),
		stopMonitor: make(chan struct{}),
		monitorDone: make(chan struct{}),
	}

	if err := auditService.validateConfig(cfg); err != nil {
		log.Error().Err(err).Msg("invalid audit config")
		return nil
	}

	// Register default sensitive fields
	auditService.registerDefaultSensitiveFields()

	// Register the lifecycle hooks
	p.LC.Append(fx.Hook{
		OnStart: func(context.Context) error {
			return auditService.Start()
		},
		OnStop: func(ctx context.Context) error {
			return auditService.Stop(ctx)
		},
	})

	return auditService
}

func (s *service) validateConfig(params *config.AuditConfig) error {
	if params.BufferSize == 0 {
		return ErrBufferSizeNotSet
	}

	if params.FlushInterval == 0 {
		return ErrFlushIntervalNotSet
	}

	return nil
}

func (s *service) LogAction(
	params *services.LogActionParams,
	opts ...services.LogOption,
) error {
	// Create base audit entry
	entry := &audit.Entry{
		Resource:       params.Resource,
		ResourceID:     params.ResourceID,
		Action:         params.Action,
		CurrentState:   params.CurrentState,
		PreviousState:  params.PreviousState,
		UserID:         params.UserID,
		OrganizationID: params.OrganizationID,
		BusinessUnitID: params.BusinessUnitID,
		Timestamp:      time.Now().Unix(),
	}

	for _, opt := range opts {
		if err := opt(entry); err != nil {
			return eris.Wrap(err, "failed to apply audit option")
		}
	}

	if err := entry.Validate(); err != nil {
		s.l.Error().Err(err).Msg("invalid audit entry")
		return eris.Wrap(err, "invalid audit entry")
	}

	s.withSensitiveData(entry)

	// Insert the entry into the buffer
	s.buffer.Add(entry)

	// if the buffer is full flush it in a goroutine
	if s.buffer.IsFull() {
		flushCtx, cancel := context.WithTimeout(context.Background(), DefaultWorkerTimeout)

		s.wg.Go(func() {
			defer cancel()

			if err := s.flushBuffer(flushCtx); err != nil {
				s.l.Error().Err(err).Msg("failed to flush buffer")
			}
		})
	}

	return nil
}

func (s *service) Start() error {
	if !s.isRunning.CompareAndSwap(false, true) {
		s.l.Warn().Msg("audit service is already running")
		return nil
	}

	// Start the flusher
	go func() {
		defer close(s.flusherDone)
		s.startFlusher()
	}()

	// Start the monitor
	go func() {
		defer close(s.monitorDone)
		s.monitor()
	}()

	s.l.Info().
		Int("buffer_size", s.config.BufferSize).
		Int("flush_interval", s.config.FlushInterval).
		Msg("🚀 Audit service initialized")

	return nil
}

func (s *service) Stop(ctx context.Context) error {
	if !s.isRunning.CompareAndSwap(true, false) {
		s.l.Warn().Msg("audit service is already stopped")
		return nil
	}

	s.l.Info().Msg("stopping audit service")

	// Signal goroutines to stop
	close(s.stopFlusher)
	close(s.stopMonitor)

	// Wait for goroutines to finish with timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, DefaultFlushTimeout)
	defer cancel()

	// Create channels to signal completion of each wait
	flusherOK := make(chan struct{})
	monitorOK := make(chan struct{})

	// Wait for flusher
	go func() {
		<-s.flusherDone
		close(flusherOK)
	}()

	// Wait for monitor
	go func() {
		<-s.monitorDone
		close(monitorOK)
	}()

	// Wait for both goroutines or timeout
	select {
	case <-shutdownCtx.Done():
		return eris.Wrap(ErrTimeoutWaitingStop, "workers failed to stop")
	case <-flusherOK:
		s.l.Trace().Msg("flusher stopped")
		select {
		case <-shutdownCtx.Done():
			return eris.Wrap(ErrTimeoutWaitingStop, "monitor failed to stop")
		case <-monitorOK:
			s.l.Trace().Msg("monitor stopped")
		}
	}

	// Perform final flush
	if err := s.performFinalFlush(ctx); err != nil {
		s.l.Error().Err(err).Msg("final flush failed")
		return err
	}

	return nil
}

func (s *service) monitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopMonitor:
			s.l.Trace().Msg("monitor stopped due to service shutdown")
			return
		case <-ticker.C:
			s.checkServiceHealth()
		}
	}
}

func (s *service) checkServiceHealth() {
	bufferSize := s.buffer.Size()
	s.l.Trace().
		Int("buffer_size", bufferSize).
		Bool("is_running", s.isRunning.Load()).
		Msg("health check")
}

func (s *service) startFlusher() {
	ticker := time.NewTicker(time.Duration(s.config.FlushInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopFlusher:
			s.l.Trace().Msg("flusher stopped due to service shutdown")
			return
		case <-ticker.C:
			if !s.isRunning.Load() {
				return
			}

			if err := s.flushBuffer(context.Background()); err != nil && !eris.Is(err, ErrEmptyBuffer) {
				s.l.Error().Err(err).Msg("failed to flush audit entries")
			}
		}
	}
}

func (s *service) flushBuffer(ctx context.Context) error {
	entries := s.buffer.FlushAndReset()
	if len(entries) == 0 {
		return ErrEmptyBuffer
	}

	s.l.Trace().Int("entries", len(entries)).Msg("flushing buffer")

	// Insert the entries into the database
	err := s.insertChunk(ctx, entries)
	if err != nil {
		return eris.Wrap(err, "failed to insert audit entries")
	}

	s.l.Trace().Msg("flushed buffer")

	return nil
}

func (s *service) performFinalFlush(ctx context.Context) error {
	flushCtx, cancel := context.WithTimeout(ctx, DefaultFlushTimeout)
	defer cancel()

	err := s.flushBuffer(flushCtx)
	if err != nil && !eris.Is(err, ErrEmptyBuffer) {
		return eris.Wrap(err, "final flush failed")
	}

	return nil
}

func (s *service) insertChunk(ctx context.Context, entries []*audit.Entry) error {
	var err error
	for retries := 0; retries < DefaultMaxRetries; retries++ {
		if err = s.repo.InsertAuditEntries(ctx, entries); err == nil {
			return nil
		}
		s.l.Warn().Err(err).Int("retry", retries+1).Msg("retrying chunk insert")
		time.Sleep(time.Duration(retries+1) * 100 * time.Millisecond)
	}
	return eris.Wrap(err, "max retries exceeded")
}

func (s *service) withSensitiveData(entry *audit.Entry) {
	fields := s.sdm.GetSensitiveFields(entry.Resource)
	if len(fields) == 0 {
		return
	}

	entry.SensitiveData = true
	sanitizeData(entry, fields)
}

// TODO(Wolfred): We need to move this somewhere else.
func (s *service) registerDefaultSensitiveFields() {
	s.sdm.RegisterSensitiveFields(permission.ResourceUser, []SensitiveField{
		{Name: "password", Action: SensitiveFieldOmit},
		{Name: "email", Action: SensitiveFieldHash},
	})

	s.sdm.RegisterSensitiveFields(permission.ResourceOrganization, []SensitiveField{
		{Name: "logoUrl", Action: SensitiveFieldMask},
	})
}

func WithComment(comment string) services.LogOption {
	return func(entry *audit.Entry) error {
		entry.Comment = comment
		return nil
	}
}

func WithDiff(before, after any) services.LogOption {
	return func(entry *audit.Entry) error {
		opts := jsonutils.DefaultOptions()
		// Customize options as needed
		opts.IgnoreFields = []string{"updated_at", "version"}

		diff, err := jsonutils.JSONDiff(before, after, opts)
		if err != nil {
			return eris.Wrap(err, "failed to compute diff")
		}

		// Convert the structured diff to a simple map[string]any
		changes := make(map[string]any, len(diff))
		for path, change := range diff {
			changes[path] = map[string]any{
				"from":      change.From,
				"to":        change.To,
				"type":      change.Type,
				"fieldType": change.FieldType,
				"path":      change.Path,
			}
		}

		entry.Changes = changes
		return nil
	}
}

func WithMetadata(metadata map[string]any) services.LogOption {
	return func(entry *audit.Entry) error {
		if entry.Metadata == nil {
			entry.Metadata = make(map[string]any)
		}
		for k, v := range metadata {
			entry.Metadata[k] = v
		}
		return nil
	}
}

func WithUserAgent(userAgent string) services.LogOption {
	return func(entry *audit.Entry) error {
		entry.UserAgent = userAgent
		return nil
	}
}

func WithCorrelationID() services.LogOption {
	return func(entry *audit.Entry) error {
		// Automatically generate a correlation ID.
		entry.CorrelationID = pulid.MustNew("corr_")
		return nil
	}
}
