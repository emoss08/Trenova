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
	"github.com/emoss08/trenova/internal/pkg/errors"
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

	// Compression constants
	DefaultCompressionLevel     = 6  // Medium compression level (balance between speed and size)
	DefaultCompressionThreshold = 10 // KB - only compress if payload exceeds this size
)

// ServiceState represents the state of the audit service
type ServiceState string

const (
	ServiceStateInitializing ServiceState = "initializing"
	ServiceStateRunning      ServiceState = "running"
	ServiceStateDegraded     ServiceState = "degraded"
	ServiceStateStopping     ServiceState = "stopping"
	ServiceStateStopped      ServiceState = "stopped"
)

type ServiceParams struct {
	fx.In

	LC              fx.Lifecycle
	AuditRepository repositories.AuditRepository
	PermService     services.PermissionService
	Logger          *logger.Logger
	Config          *config.Manager
}

type service struct {
	l             *zerolog.Logger
	buffer        *Buffer
	repo          repositories.AuditRepository
	config        *config.AuditConfig
	ps            services.PermissionService
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

	// Channels for goroutine control - optimized structure
	control struct {
		stopCh chan struct{}  // Single channel for stopping all goroutines
		wg     sync.WaitGroup // Wait group for all goroutines
	}

	// Emergency log handling
	emergencyLog struct {
		ch    chan *audit.Entry
		mutex sync.RWMutex
		count atomic.Int64
	}
}

var entryPool = sync.Pool{
	New: func() any {
		return &audit.Entry{
			Metadata: make(map[string]any),
		}
	},
}

func NewService(p ServiceParams) services.AuditService {
	log := p.Logger.With().Str("service", "audit").Logger()

	cfg := p.Config.Audit()

	auditService := &service{
		repo:          p.AuditRepository,
		l:             &log,
		ps:            p.PermService,
		buffer:        NewBuffer(cfg.BufferSize),
		config:        &p.Config.Get().Audit,
		sdm:           NewSensitiveDataManager(),
		wg:            conc.NewWaitGroup(),
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
		OnStop: func(context.Context) error {
			return auditService.Stop()
		},
	})

	return auditService
}

func (s *service) Stop() error {
	// Ensure we're currently running
	if !s.isRunning.CompareAndSwap(true, false) {
		s.l.Warn().Msg("audit service is already stopped")
		return nil
	}

	s.serviceState.Store(string(ServiceStateStopping))
	s.l.Info().Msg("stopping audit service")

	// Signal all goroutines to stop
	close(s.control.stopCh)

	// Create a timeout context
	ctx, cancel := context.WithTimeout(context.Background(), DefaultFlushTimeout)
	defer cancel()

	// Create a done channel for WaitGroup completion
	done := make(chan struct{})

	// Wait for goroutines to finish in a separate goroutine
	go func() {
		s.control.wg.Wait()
		close(done)
	}()

	// Wait for completion or timeout
	select {
	case <-ctx.Done():
		s.serviceState.Store(string(ServiceStateDegraded))
		s.l.Error().Msg("timeout waiting for workers to stop")
		// continue with shutdown despite timeout
	case <-done:
		s.l.Debug().Msg("all goroutines stopped")
	}

	// Perform final flush
	if err := s.performFinalFlush(ctx); err != nil {
		s.l.Error().Err(err).Msg("final flush failed")
		s.serviceState.Store(string(ServiceStateDegraded))
	}

	s.serviceState.Store(string(ServiceStateStopped))
	s.l.Info().
		Int64("total_entries", s.entryCount.Load()).
		Int64("total_flushes", s.flushCount.Load()).
		Int64("total_errors", s.errorCount.Load()).
		Int64("emergency_entries", s.emergencyLog.count.Load()).
		Str("uptime", time.Since(s.startTime).String()).
		Msg("audit service stopped successfully")

	return nil
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

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				Resource:       permission.ResourceAuditEntry,
				BusinessUnitID: opts.TenantOpts.BuID,
				OrganizationID: opts.TenantOpts.OrgID,
				UserID:         opts.TenantOpts.UserID,
				Action:         permission.ActionRead,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read audit entries")
	}

	entities, err := s.repo.List(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to list audit entries")
		return nil, eris.Wrap(err, "list audit entries")
	}

	return entities, nil
}

func (s *service) ListByResourceID(ctx context.Context, opts repositories.ListByResourceIDRequest) (*ports.ListResult[*audit.Entry], error) {
	log := s.l.With().
		Str("operation", "ListByResourceID").
		Str("resourceID", opts.ResourceID.String()).
		Logger()

	entities, err := s.repo.ListByResourceID(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to list audit entries by resource id")
		return nil, eris.Wrap(err, "list audit entries by resource id")
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

	// Get entry from pool
	entryObj := entryPool.Get()
	entry, ok := entryObj.(*audit.Entry)
	if !ok {
		// Fallback if type assertion fails
		entry = &audit.Entry{
			Metadata: make(map[string]any),
		}
	}

	// Reset and initialize the entry
	*entry = audit.Entry{
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
			// Return to pool if there's an error
			entryPool.Put(entry)
			return eris.Wrap(err, "failed to apply audit option")
		}
	}

	// * Validate the entry
	if err := entry.Validate(); err != nil {
		s.l.Error().Err(err).Msg("invalid audit entry")
		entryPool.Put(entry) // Return to pool
		return eris.Wrap(ErrInvalidEntry, err.Error())
	}

	// * Handle sensitive data
	if err := s.sdm.sanitizeData(entry); err != nil {
		s.l.Error().Err(err).Msg("failed to sanitize sensitive data")
		entryPool.Put(entry) // Return to pool
		return eris.Wrap(ErrSanitizationFailed, err.Error())
	}

	// * Try to add the entry to the buffer, respecting circuit breaker
	if !s.buffer.Add(entry) {
		// * Handle rejection (circuit open) - log to emergency buffer if it's a critical audit
		if params.Critical {
			select {
			case s.emergencyLog.ch <- entry:
				s.l.Warn().Msg("added critical audit to emergency buffer due to circuit breaker")
			default:
				s.l.Error().Msg("emergency buffer full, dropping critical audit entry")
				s.errorCount.Inc()
				entryPool.Put(entry) // Return to pool on rejection
			}
		} else {
			s.l.Warn().Msg("audit entry rejected due to circuit breaker")
			s.errorCount.Inc()
			entryPool.Put(entry) // Return to pool on rejection
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

	// Initialize the control channel
	s.control.stopCh = make(chan struct{})

	// Initialize emergency log channel
	s.emergencyLog.ch = make(chan *audit.Entry, 10)

	// * Update state
	s.serviceState.Store(string(ServiceStateRunning))
	s.startTime = time.Now()

	// * Start the flusher
	s.control.wg.Add(1)
	go func() {
		defer s.control.wg.Done()
		s.startFlusher()
	}()

	// * Start the monitor
	s.control.wg.Add(1)
	go func() {
		defer s.control.wg.Done()
		s.monitor()
	}()

	// * Start the emergency log processor
	s.control.wg.Add(1)
	go func() {
		defer s.control.wg.Done()
		s.processEmergencyLogs()
	}()

	s.l.Info().
		Int("buffer_size", s.config.BufferSize).
		Int("flush_interval", s.config.FlushInterval).
		Str("audit_version", AuditVersionTag).
		Msg("🚀 Audit service initialized")

	return nil
}

// Optimized emergency log processor with retry capability
func (s *service) processEmergencyLogs() {
	const maxRetries = 2 // Maximum retry attempts for emergency log processing

	for {
		select {
		case <-s.control.stopCh:
			s.l.Debug().Msg("emergency log processor stopped")
			return
		case entry := <-s.emergencyLog.ch:
			s.processEmergencyEntry(entry, maxRetries)
		}
	}
}

// processEmergencyEntry handles a single emergency audit entry with retries
func (s *service) processEmergencyEntry(entry *audit.Entry, maxRetries int) {
	success, err := s.attemptInsertWithRetry(entry, maxRetries)

	if success {
		s.emergencyLog.count.Inc()
	} else {
		s.l.Error().Err(err).Msg("failed to process critical audit entry after all retries")
		s.errorCount.Inc()
	}
}

// attemptInsertWithRetry tries to insert an entry with exponential backoff
func (s *service) attemptInsertWithRetry(entry *audit.Entry, maxRetries int) (bool, error) {
	var err error

	for retry := 0; retry <= maxRetries; retry++ {
		// Apply backoff for retries
		if retry > 0 {
			backoffTime := time.Duration(retry*150) * time.Millisecond
			time.Sleep(backoffTime)
		}

		// Try to insert the entry
		ctx, cancel := context.WithTimeout(context.Background(), DefaultWorkerTimeout)
		err = s.repo.InsertAuditEntries(ctx, []*audit.Entry{entry})
		cancel()

		// On success, log and return
		if err == nil {
			if retry > 0 {
				s.l.Debug().Int("retry", retry).Msg("critical audit entry processed after retry")
			} else {
				s.l.Debug().Msg("critical audit entry processed")
			}
			return true, nil
		}
	}

	return false, err
}

// Optimized flusher with adaptive intervals
func (s *service) startFlusher() {
	baseInterval := time.Duration(s.config.FlushInterval) * time.Second
	ticker := time.NewTicker(baseInterval)
	defer ticker.Stop()

	// Track consecutive empty flushes
	emptyFlushes := 0

	for {
		select {
		case <-s.control.stopCh:
			s.l.Debug().Msg("flusher stopped due to service shutdown")
			return
		case <-ticker.C:
			if !s.isRunning.Load() {
				return
			}

			emptyFlushes = s.handleTick(baseInterval, ticker, emptyFlushes)
		}
	}
}

// handleTick processes a single tick of the flusher timer
func (s *service) handleTick(baseInterval time.Duration, ticker *time.Ticker, emptyFlushes int) int {
	// Skip flushing if buffer is empty to reduce load
	if s.buffer.Size() == 0 {
		return s.handleEmptyBuffer(baseInterval, ticker, emptyFlushes)
	}

	// Reset back to normal interval if we were previously inactive
	if emptyFlushes > 3 {
		ticker.Reset(baseInterval)
		s.l.Debug().Msg("reset to normal flush interval due to activity")
	}

	// Attempt to flush and handle result
	s.performFlush()
	return 0 // Reset empty flushes counter
}

// handleEmptyBuffer adjusts ticker interval based on inactivity
func (s *service) handleEmptyBuffer(baseInterval time.Duration, ticker *time.Ticker, emptyFlushes int) int {
	emptyFlushes++
	// After several empty flushes, adjust the interval to reduce system load
	if emptyFlushes > 3 {
		newInterval := min(baseInterval*2, 60*time.Second)
		ticker.Reset(newInterval)
		s.l.Debug().
			Dur("interval", newInterval).
			Msg("increased flush interval due to inactivity")
	}
	return emptyFlushes
}

// performFlush executes the flush operation and handles errors
func (s *service) performFlush() {
	err := s.flushBuffer(context.Background())

	if err != nil && !eris.Is(err, ErrEmptyBuffer) {
		s.l.Error().Err(err).Msg("failed to flush audit entries")
		s.errorCount.Inc()
		s.buffer.RecordFailure()
	} else if !eris.Is(err, ErrEmptyBuffer) {
		// Reset failures on successful flush
		s.buffer.ResetFailures()
	}
}

// Optimized buffer flushing with improved chunk sizing and error handling
func (s *service) flushBuffer(ctx context.Context) error {
	entries := s.buffer.FlushAndReset()
	if len(entries) == 0 {
		return ErrEmptyBuffer
	}

	s.l.Debug().Int("entries", len(entries)).Msg("flushing buffer")

	// * Determine optimal chunk size based on entry count
	// Small batches: use smaller chunks to reduce overhead
	// Large batches: use larger chunks for throughput
	var chunkSize int
	switch {
	case len(entries) < 20:
		chunkSize = 10
	case len(entries) < 100:
		chunkSize = len(entries) / 2
	case len(entries) < 500:
		chunkSize = DefaultChunkSize
	default:
		chunkSize = DefaultChunkSize * 2 // Increase for very large batches
	}

	// * Process entries in chunks for better performance and reliability
	for i := 0; i < len(entries); i += chunkSize {
		// Check if context is cancelled
		if ctx.Err() != nil {
			// Put unprocessed entries back into the buffer
			remainingEntries := entries[i:]
			s.l.Warn().
				Int("remaining", len(remainingEntries)).
				Msg("context cancelled, returning entries to buffer")

			for _, entry := range remainingEntries {
				s.buffer.Add(entry)
			}
			return eris.Wrap(ctx.Err(), "flush interrupted")
		}

		end := min(i+chunkSize, len(entries))
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

// Optimized chunk insertion with exponential backoff
func (s *service) insertChunk(ctx context.Context, entries []*audit.Entry) error {
	var err error
	// Use exponential backoff for retries
	backoffs := []time.Duration{
		100 * time.Millisecond,
		250 * time.Millisecond,
		500 * time.Millisecond,
	}

	for retry := range DefaultMaxRetries {
		// Check if context has been cancelled
		if ctx.Err() != nil {
			return eris.Wrap(ctx.Err(), "context cancelled during insert")
		}

		if err = s.repo.InsertAuditEntries(ctx, entries); err == nil {
			return nil
		}

		// Don't sleep on the last retry
		if retry < DefaultMaxRetries-1 {
			// Use backoff if available, otherwise use a simple formula
			var backoff time.Duration
			if retry < len(backoffs) {
				backoff = backoffs[retry]
			} else {
				backoff = time.Duration(retry+1) * 200 * time.Millisecond
			}

			s.l.Warn().Err(err).Int("retry", retry+1).
				Dur("backoff", backoff).
				Msg("retrying chunk insert")

			time.Sleep(backoff)
		}
	}

	return eris.Wrap(ErrMaxRetriesExceeded, err.Error())
}

// Improved final flush with better emergency entry handling
func (s *service) performFinalFlush(ctx context.Context) error {
	flushCtx, cancel := context.WithTimeout(ctx, DefaultFlushTimeout)
	defer cancel()

	// Attempt to flush the buffer
	err := s.flushBuffer(flushCtx)
	if err != nil && !eris.Is(err, ErrEmptyBuffer) {
		s.l.Error().Err(err).Msg("error during final buffer flush")
		// Continue to try to process emergency entries despite buffer flush error
	}

	// Handle remaining emergency logs
	var emergencyEntries []*audit.Entry

	// Drain channel in a non-blocking way
	drainDone := false
	for !drainDone {
		select {
		case entry, ok := <-s.emergencyLog.ch:
			if !ok {
				drainDone = true
				break
			}
			emergencyEntries = append(emergencyEntries, entry)
		default:
			drainDone = true
		}
	}

	// Close the channel to prevent more entries
	close(s.emergencyLog.ch)

	// Process emergency entries in batches if any were collected
	if len(emergencyEntries) > 0 {
		s.l.Info().Int("count", len(emergencyEntries)).Msg("processing emergency entries during shutdown")

		// Process in batches of 10 for better efficiency
		for i := 0; i < len(emergencyEntries); i += 10 {
			end := min(i+10, len(emergencyEntries))
			batch := emergencyEntries[i:end]

			insertCtx, insertCancel := context.WithTimeout(ctx, DefaultWorkerTimeout)
			insertErr := s.repo.InsertAuditEntries(insertCtx, batch)
			insertCancel()

			if insertErr != nil {
				s.l.Error().Err(insertErr).
					Int("count", len(batch)).
					Msg("failed to insert emergency entries during shutdown")
			} else {
				s.l.Info().Int("count", len(batch)).Msg("flushed emergency entries")
			}
		}
	}

	return nil
}

// Enhanced health check that returns health status
func (s *service) checkServiceHealth() bool {
	bufferSize := s.buffer.Size()
	state := s.buffer.GetState()
	errorCount := s.errorCount.Load()

	// Calculate buffer utilization
	utilization := float64(bufferSize) / float64(s.buffer.limit) * 100

	// * Log the current status
	s.l.Debug().
		Int("buffer_size", bufferSize).
		Float64("buffer_utilization_pct", utilization).
		Int64("entries_processed", s.entryCount.Load()).
		Int64("flushes", s.flushCount.Load()).
		Int64("errors", errorCount).
		Str("circuit_state", circuitStateToString(state)).
		Bool("is_running", s.isRunning.Load()).
		Str("service_state", s.serviceState.Load()).
		Msg("audit service health check")

	isHealthy := true

	// * Check for degraded state conditions
	if errorCount > 5 && state == CircuitOpen {
		if s.serviceState.Load() != string(ServiceStateDegraded) {
			s.serviceState.Store(string(ServiceStateDegraded))
			s.l.Warn().Msg("audit service entering degraded state due to repeated errors")
		}
		isHealthy = false
	} else if s.serviceState.Load() == string(ServiceStateDegraded) && state == CircuitClosed {
		// * Auto-recover from degraded state if circuit is closed again
		s.serviceState.Store(string(ServiceStateRunning))
		s.l.Info().Msg("audit service recovered from degraded state")
		isHealthy = true
	}

	// Warn if buffer utilization is high
	if utilization > 80 {
		s.l.Warn().Float64("utilization_pct", utilization).Msg("audit buffer utilization high")
		isHealthy = false
	}

	return isHealthy
}

// Adaptive health check monitor
func (s *service) monitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Track consecutively healthy checks to reduce frequency when stable
	healthyChecks := 0

	for {
		select {
		case <-s.control.stopCh:
			s.l.Debug().Msg("monitor stopped due to service shutdown")
			return
		case <-ticker.C:
			isHealthy := s.checkServiceHealth()

			// If healthy for a while, check less frequently to reduce overhead
			if isHealthy {
				healthyChecks++
				if healthyChecks > 5 {
					// Less frequent checks when stable
					ticker.Reset(60 * time.Second)
				}
			} else {
				// Reset to more frequent checks when issues detected
				healthyChecks = 0
				ticker.Reset(30 * time.Second)
			}
		}
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
	// User-related sensitive data
	if err := s.sdm.RegisterSensitiveFields(permission.ResourceUser, []services.SensitiveField{
		{Name: "password", Action: services.SensitiveFieldOmit},
		{Name: "hashedPassword", Action: services.SensitiveFieldOmit},
		{Name: "emailAddress", Action: services.SensitiveFieldMask},
		{Name: "address", Action: services.SensitiveFieldMask},
	}); err != nil {
		s.l.Error().Err(err).Msg("failed to register user sensitive fields")
		return err
	}

	// Organization data
	if err := s.sdm.RegisterSensitiveFields(permission.ResourceOrganization, []services.SensitiveField{
		{Name: "logoUrl", Action: services.SensitiveFieldMask},
		{Name: "taxId", Action: services.SensitiveFieldMask},
	}); err != nil {
		s.l.Error().Err(err).Msg("failed to register organization sensitive fields")
		return err
	}

	// Driver sensitive data for FMCSA compliance
	if err := s.sdm.RegisterSensitiveFields(permission.ResourceWorker, []services.SensitiveField{
		{Name: "licenseNumber", Action: services.SensitiveFieldMask},
		{Name: "dateOfBirth", Action: services.SensitiveFieldMask},
		{Path: "profile", Name: "licenseNumber", Action: services.SensitiveFieldMask},
	}); err != nil {
		s.l.Error().Err(err).Msg("failed to register worker sensitive fields")
		return err
	}

	return nil
}
