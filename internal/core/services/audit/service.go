package audit

import (
	"context"
	"fmt"
	"maps"
	"strings"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/sourcegraph/conc"
	"go.uber.org/atomic"
	"go.uber.org/fx"
)

const (
	DefaultMaskValue     = "****"
	DefaultFlushTimeout  = 10 * time.Second
	DefaultWorkerTimeout = 5 * time.Second
	DefaultChunkSize     = 100
	DefaultMaxRetries    = 3

	// New constants
	AuditVersionTag      = "v1"
	MinBufferSize        = 50
	MinFlushInterval     = 5
	DefaultAuditCategory = "system"
)

// AuditServiceState represents the state of the audit service
type AuditServiceState string

const (
	ServiceStateInitializing AuditServiceState = "initializing"
	ServiceStateRunning      AuditServiceState = "running"
	ServiceStateDegraded     AuditServiceState = "degraded"
	ServiceStateStopping     AuditServiceState = "stopping"
	ServiceStateStopped      AuditServiceState = "stopped"
)

type ServiceParams struct {
	fx.In

	LC              fx.Lifecycle
	AuditRepository repositories.AuditRepository
	Logger          *logger.Logger
	Config          *config.Manager
}

type service struct {
	l             *zerolog.Logger
	buffer        *Buffer
	repo          repositories.AuditRepository
	config        *config.AuditConfig
	wg            *conc.WaitGroup
	sdm           *SensitiveDataManager
	mutex         sync.RWMutex
	isRunning     atomic.Bool
	serviceState  atomic.String
	flushCount    atomic.Int64
	entryCount    atomic.Int64
	errorCount    atomic.Int64
	startTime     time.Time
	defaultFields map[string]any

	// Channels for goroutine control
	stopFlusher  chan struct{}
	flusherDone  chan struct{}
	stopMonitor  chan struct{}
	monitorDone  chan struct{}
	emergencyLog chan *audit.Entry
}

func NewService(p ServiceParams) services.AuditService {
	log := p.Logger.With().Str("service", "audit").Logger()

	cfg := p.Config.Audit()

	auditService := &service{
		repo:          p.AuditRepository,
		l:             &log,
		buffer:        NewBuffer(cfg.BufferSize),
		config:        &p.Config.Get().Audit,
		sdm:           NewSensitiveDataManager(),
		wg:            conc.NewWaitGroup(),
		stopFlusher:   make(chan struct{}),
		flusherDone:   make(chan struct{}),
		stopMonitor:   make(chan struct{}),
		monitorDone:   make(chan struct{}),
		emergencyLog:  make(chan *audit.Entry, 10), // * Small buffer for critical audit events
		defaultFields: make(map[string]any),
	}

	// * Set initial state
	auditService.serviceState.Store(string(ServiceStateInitializing))

	if err := auditService.validateConfig(cfg); err != nil {
		log.Error().Err(err).Msg("invalid audit configuration")
		auditService.applyDefaultConfig()
	}

	// * Register default sensitive fields
	if err := auditService.registerDefaultSensitiveFields(); err != nil {
		log.Error().Err(err).Msg("failed to register default sensitive fields")
	}

	// * Set default fields that should be included in all audit entries
	auditService.defaultFields["auditVersion"] = AuditVersionTag
	auditService.defaultFields["environment"] = p.Config.Get().App.Environment

	// * Register the lifecycle hooks
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

func (s *service) applyDefaultConfig() {
	s.l.Warn().Msg("applying default audit configuration")
	if s.config.BufferSize < MinBufferSize {
		s.config.BufferSize = MinBufferSize
		s.buffer = NewBuffer(MinBufferSize)
	}

	if s.config.FlushInterval < MinFlushInterval {
		s.config.FlushInterval = MinFlushInterval
	}
}

func (s *service) validateConfig(params *config.AuditConfig) error {
	var errs []string

	if params.BufferSize == 0 {
		errs = append(errs, "buffer size is not set")
	} else if params.BufferSize < MinBufferSize {
		errs = append(errs, fmt.Sprintf("buffer size is too small (min: %d)", MinBufferSize))
	}

	if params.FlushInterval == 0 {
		errs = append(errs, "flush interval is not set")
	} else if params.FlushInterval < MinFlushInterval {
		errs = append(errs, fmt.Sprintf("flush interval is too small (min: %d seconds)", MinFlushInterval))
	}

	if len(errs) > 0 {
		return eris.Wrap(ErrInvalidConfig, strings.Join(errs, "; "))
	}

	return nil
}

func (s *service) List(ctx context.Context, opts *ports.LimitOffsetQueryOptions) (*ports.ListResult[*audit.Entry], error) {
	log := s.l.With().
		Str("operation", "List").
		Str("buID", opts.TenantOpts.BuID.String()).
		Str("userID", opts.TenantOpts.UserID.String()).
		Logger()

	// TODO(wolfred): We need to check the permissions here.

	entities, err := s.repo.List(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to list audit entries")
		return nil, eris.Wrap(err, "list audit entries")
	}

	return entities, nil
}

func (s *service) GetByID(ctx context.Context, opts repositories.GetAuditEntryByIDOptions) (*audit.Entry, error) {
	log := s.l.With().
		Str("operation", "GetByID").
		Str("auditEntryID", opts.ID.String()).
		Logger()

	// TODO(wolfred): We need to check the permissions here.

	entity, err := s.repo.GetByID(ctx, opts)
	if err != nil {
		log.Error().Err(err).Str("auditEntryID", opts.ID.String()).Msg("failed to get audit entry")
		return nil, eris.Wrap(err, "get audit entry by id")
	}

	return entity, nil
}

func (s *service) LogAction(
	params *services.LogActionParams,
	opts ...services.LogOption,
) error {
	// * Quick check if the service is running
	if !s.isRunning.Load() {
		s.l.Warn().Msg("attempt to log audit entry while service is stopped")
		return ErrServiceStopped
	}

	// * Create base audit entry
	entry := &audit.Entry{
		ID:             pulid.MustNew("ae_"),
		Resource:       params.Resource,
		ResourceID:     params.ResourceID,
		Action:         params.Action,
		CurrentState:   params.CurrentState,
		PreviousState:  params.PreviousState,
		UserID:         params.UserID,
		OrganizationID: params.OrganizationID,
		BusinessUnitID: params.BusinessUnitID,
		Timestamp:      time.Now().Unix(),
		Category:       DefaultAuditCategory,
		Metadata:       s.copyDefaultFields(),
	}

	// * Apply options
	for _, opt := range opts {
		if err := opt(entry); err != nil {
			return eris.Wrap(err, "failed to apply audit option")
		}
	}

	// * Validate the entry
	if err := entry.Validate(); err != nil {
		s.l.Error().Err(err).Msg("invalid audit entry")
		return eris.Wrap(ErrInvalidEntry, err.Error())
	}

	// * Handle sensitive data
	if err := s.sdm.sanitizeData(entry); err != nil {
		s.l.Error().Err(err).Msg("failed to sanitize sensitive data")
		return eris.Wrap(ErrSanitizationFailed, err.Error())
	}

	// * Try to add the entry to the buffer, respecting circuit breaker
	if !s.buffer.Add(entry) {
		// * Handle rejection (circuit open) - log to emergency buffer if it's a critical audit
		if params.Critical {
			select {
			case s.emergencyLog <- entry:
				s.l.Warn().Msg("added critical audit to emergency buffer due to circuit breaker")
			default:
				s.l.Error().Msg("emergency buffer full, dropping critical audit entry")
				s.errorCount.Inc()
			}
		} else {
			s.l.Warn().Msg("audit entry rejected due to circuit breaker")
			s.errorCount.Inc()
		}
		return nil
	}

	// * Update the entry count
	s.entryCount.Inc()

	// * If the buffer is full, flush it in a goroutine
	if s.buffer.IsFull() {
		flushCtx, cancel := context.WithTimeout(context.Background(), DefaultWorkerTimeout)
		s.wg.Go(func() {
			defer cancel()
			if err := s.flushBuffer(flushCtx); err != nil && !eris.Is(err, ErrEmptyBuffer) {
				s.l.Error().Err(err).Msg("failed to flush buffer")
				s.errorCount.Inc()
			}
		})
	}

	return nil
}

func (s *service) copyDefaultFields() map[string]any {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	result := make(map[string]any, len(s.defaultFields))

	// * Copy the default fields to the result map
	maps.Copy(result, s.defaultFields)

	return result
}

func (s *service) Start() error {
	// * Ensure we're not already running
	if !s.isRunning.CompareAndSwap(false, true) {
		s.l.Warn().Msg("audit service is already running")
		return nil
	}

	// * Update state
	s.serviceState.Store(string(ServiceStateRunning))
	s.startTime = time.Now()

	// * Start the flusher
	go func() {
		defer close(s.flusherDone)
		s.startFlusher()
	}()

	// * Start the monitor
	go func() {
		defer close(s.monitorDone)
		s.monitor()
	}()

	// * Start the emergency log processor
	go s.processEmergencyLogs()

	s.l.Info().
		Int("buffer_size", s.config.BufferSize).
		Int("flush_interval", s.config.FlushInterval).
		Str("audit_version", AuditVersionTag).
		Msg("🚀 Audit service initialized")

	return nil
}

func (s *service) processEmergencyLogs() {
	for {
		select {
		case <-s.stopFlusher: // * Reuse the flusher stop signal
			s.l.Debug().Msg("emergency log processor stopped")
			return
		case entry := <-s.emergencyLog:
			// * Direct insert of critical audit entries, bypassing the buffer
			ctx, cancel := context.WithTimeout(context.Background(), DefaultWorkerTimeout)
			err := s.repo.InsertAuditEntries(ctx, []*audit.Entry{entry})
			cancel()

			if err != nil {
				s.l.Error().Err(err).Msg("failed to insert critical audit entry")
				s.errorCount.Inc()
			} else {
				s.l.Debug().Msg("critical audit entry inserted directly")
			}
		}
	}
}

func (s *service) Stop(ctx context.Context) error {
	// * Ensure we're currently running
	if !s.isRunning.CompareAndSwap(true, false) {
		s.l.Warn().Msg("audit service is already stopped")
		return nil
	}

	s.serviceState.Store(string(ServiceStateStopping))
	s.l.Info().Msg("stopping audit service")

	// * Signal goroutines to stop
	close(s.stopFlusher)
	close(s.stopMonitor)

	// * Wait for goroutines to finish with timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, DefaultFlushTimeout)
	defer cancel()

	// * Create channels to signal completion of each wait
	flusherOK := make(chan struct{})
	monitorOK := make(chan struct{})

	// * Wait for flusher
	go func() {
		<-s.flusherDone
		close(flusherOK)
	}()

	// * Wait for monitor
	go func() {
		<-s.monitorDone
		close(monitorOK)
	}()

	// * Wait for both goroutines or timeout
	select {
	case <-shutdownCtx.Done():
		s.serviceState.Store(string(ServiceStateDegraded))
		return eris.Wrap(ErrTimeoutWaitingStop, "workers failed to stop")
	case <-flusherOK:
		s.l.Debug().Msg("flusher stopped")
		select {
		case <-shutdownCtx.Done():
			s.serviceState.Store(string(ServiceStateDegraded))
			return eris.Wrap(ErrTimeoutWaitingStop, "monitor failed to stop")
		case <-monitorOK:
			s.l.Debug().Msg("monitor stopped")
		}
	}

	// * Perform final flush
	if err := s.performFinalFlush(ctx); err != nil {
		s.l.Error().Err(err).Msg("final flush failed")
		s.serviceState.Store(string(ServiceStateDegraded))
		return err
	}

	s.serviceState.Store(string(ServiceStateStopped))
	s.l.Info().
		Int64("total_entries", s.entryCount.Load()).
		Int64("total_flushes", s.flushCount.Load()).
		Int64("total_errors", s.errorCount.Load()).
		Str("uptime", time.Since(s.startTime).String()).
		Msg("audit service stopped successfully")

	return nil
}

func (s *service) monitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopMonitor:
			s.l.Debug().Msg("monitor stopped due to service shutdown")
			return
		case <-ticker.C:
			s.checkServiceHealth()
		}
	}
}

func (s *service) checkServiceHealth() {
	bufferSize := s.buffer.Size()
	state := s.buffer.GetState()

	// * Log the current status
	s.l.Debug().
		Int("buffer_size", bufferSize).
		Int64("entries_processed", s.entryCount.Load()).
		Int64("flushes", s.flushCount.Load()).
		Int64("errors", s.errorCount.Load()).
		Str("circuit_state", circuitStateToString(state)).
		Bool("is_running", s.isRunning.Load()).
		Str("service_state", s.serviceState.Load()).
		Msg("audit service health check")

	// * Check for degraded state conditions
	if s.errorCount.Load() > 5 && state == CircuitOpen {
		if s.serviceState.Load() != string(ServiceStateDegraded) {
			s.serviceState.Store(string(ServiceStateDegraded))
			s.l.Warn().Msg("audit service entering degraded state due to repeated errors")
		}
	} else if s.serviceState.Load() == string(ServiceStateDegraded) && state == CircuitClosed {
		// * Auto-recover from degraded state if circuit is closed again
		s.serviceState.Store(string(ServiceStateRunning))
		s.l.Info().Msg("audit service recovered from degraded state")
	}
}

func circuitStateToString(state CircuitState) string {
	switch state {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

func (s *service) startFlusher() {
	ticker := time.NewTicker(time.Duration(s.config.FlushInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopFlusher:
			s.l.Debug().Msg("flusher stopped due to service shutdown")
			return
		case <-ticker.C:
			if !s.isRunning.Load() {
				return
			}

			if err := s.flushBuffer(context.Background()); err != nil && !eris.Is(err, ErrEmptyBuffer) {
				s.l.Error().Err(err).Msg("failed to flush audit entries")
				s.errorCount.Inc()
				s.buffer.RecordFailure()
			} else if !eris.Is(err, ErrEmptyBuffer) {
				// Reset failures on successful flush
				s.buffer.ResetFailures()
			}
		}
	}
}

func (s *service) flushBuffer(ctx context.Context) error {
	entries := s.buffer.FlushAndReset()
	if len(entries) == 0 {
		return ErrEmptyBuffer
	}

	s.l.Debug().Int("entries", len(entries)).Msg("flushing buffer")

	// * Process entries in chunks for better performance and reliability
	for i := 0; i < len(entries); i += DefaultChunkSize {
		end := min(i+DefaultChunkSize, len(entries))

		chunk := entries[i:end]

		// * Insert the chunk into the database
		err := s.insertChunk(ctx, chunk)
		if err != nil {
			// * Put unprocessed entries back into the buffer to try again later
			s.l.Error().Err(err).
				Int("chunk_size", len(chunk)).
				Int("remaining", len(entries)-i).
				Msg("failed to insert chunk, returning entries to buffer")

			// * Only put back entries that weren't successfully inserted
			for _, entry := range entries[i:] {
				s.buffer.Add(entry)
			}

			return eris.Wrap(err, "failed to insert audit entries")
		}
	}

	s.flushCount.Inc()
	s.l.Debug().Int("entries", len(entries)).Msg("flushed buffer successfully")

	return nil
}

func (s *service) performFinalFlush(ctx context.Context) error {
	flushCtx, cancel := context.WithTimeout(ctx, DefaultFlushTimeout)
	defer cancel()

	err := s.flushBuffer(flushCtx)
	if err != nil && !eris.Is(err, ErrEmptyBuffer) {
		return eris.Wrap(err, "final flush failed")
	}

	// * Also flush any emergency logs
	close(s.emergencyLog) // * Close channel to signal no more entries
	for entry := range s.emergencyLog {
		if err := s.repo.InsertAuditEntries(ctx, []*audit.Entry{entry}); err != nil {
			s.l.Error().Err(err).Msg("failed to insert critical audit entry during shutdown")
		}
	}

	return nil
}

func (s *service) insertChunk(ctx context.Context, entries []*audit.Entry) error {
	var err error

	for retry := range DefaultMaxRetries {
		if err = s.repo.InsertAuditEntries(ctx, entries); err == nil {
			return nil
		}
		s.l.Warn().Err(err).Int("retry", retry+1).Msg("retrying chunk insert")
		time.Sleep(time.Duration(retry+1) * 100 * time.Millisecond)
	}

	return eris.Wrap(ErrMaxRetriesExceeded, err.Error())
}

// RegisterSensitiveFields registers sensitive fields for a resource
func (s *service) RegisterSensitiveFields(resource permission.Resource, fields []services.SensitiveField) error {
	return s.sdm.RegisterSensitiveFields(resource, fields)
}

// SetDefaultField sets a default field that will be included in all audit entries
func (s *service) SetDefaultField(key string, value any) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.defaultFields[key] = value
}

// GetServiceStatus returns the current status of the audit service
func (s *service) GetServiceStatus() string {
	return s.serviceState.Load()
}

// registerDefaultSensitiveFields registers default sensitive fields
func (s *service) registerDefaultSensitiveFields() error {
	var err error

	// User-related sensitive data
	err = s.sdm.RegisterSensitiveFields(permission.ResourceUser, []services.SensitiveField{
		{Name: "password", Action: services.SensitiveFieldOmit},
		{Name: "hashedPassword", Action: services.SensitiveFieldOmit},
		{Name: "email", Action: services.SensitiveFieldMask},
		{Name: "phone", Action: services.SensitiveFieldMask},
		{Name: "ssn", Action: services.SensitiveFieldMask, Pattern: `^\d{3}-\d{2}-\d{4}$`},
		{Name: "address", Action: services.SensitiveFieldMask},
		{Name: "creditCardNumber", Action: services.SensitiveFieldMask, Pattern: `^(?:\d[ -]*?){13,16}$`},
	})

	// Organization data
	err = s.sdm.RegisterSensitiveFields(permission.ResourceOrganization, []services.SensitiveField{
		{Name: "logoUrl", Action: services.SensitiveFieldMask},
		{Name: "taxId", Action: services.SensitiveFieldMask},
		{Name: "dotNumber", Action: services.SensitiveFieldMask},
	})

	// Financial data
	// err = s.sdm.RegisterSensitiveFields(permission.ResourceFinancial, []SensitiveField{
	// 	{Name: "accountNumber", Action: SensitiveFieldMask},
	// 	{Name: "routingNumber", Action: SensitiveFieldMask},
	// 	{Name: "taxId", Action: SensitiveFieldMask},
	// })

	// Driver sensitive data for FMCSA compliance
	err = s.sdm.RegisterSensitiveFields(permission.ResourceWorker, []services.SensitiveField{
		{Name: "licenseNumber", Action: services.SensitiveFieldMask},
		{Name: "dateOfBirth", Action: services.SensitiveFieldMask},
	})
	if err != nil {
		s.l.Error().Err(err).Msg("failed to register default sensitive fields")
		return err
	}

	return nil
}
