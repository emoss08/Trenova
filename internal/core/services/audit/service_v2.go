package audit

import (
	"context"
	"maps"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/fiber/v2"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/atomic"
	"go.uber.org/fx"
)

// ServiceV2Params holds the dependencies for the audit service
type ServiceV2Params struct {
	fx.In

	LC               fx.Lifecycle
	AuditRepository  repositories.AuditRepository
	StreamingService services.StreamingService
	PermService      services.PermissionService
	Logger           *logger.Logger
	Config           *config.Manager
}

// ServiceV2 is the improved audit service implementation
type ServiceV2 struct {
	repo          repositories.AuditRepository
	ps            services.PermissionService
	ss            services.StreamingService
	logger        *zerolog.Logger
	config        *config.AuditConfig
	queue         *EntryQueue
	processor     *BatchProcessor
	sdm           *SensitiveDataManagerV2
	mu            sync.RWMutex
	isRunning     atomic.Bool
	serviceState  atomic.String
	defaultFields map[string]any
	metrics       struct {
		totalEntries   atomic.Int64
		failedEntries  atomic.Int64
		processedBatch atomic.Int64
		startTime      time.Time
	}
}

// NewServiceV2 creates a new improved audit service
//
//nolint:gocritic // dependency injection
func NewServiceV2(p ServiceV2Params) services.AuditService {
	log := p.Logger.With().Str("service", "audit_v2").Logger()
	cfg := p.Config.Audit()

	processor := NewBatchProcessor(p.AuditRepository, p.Logger)

	queueConfig := QueueConfig{
		BufferSize:   cfg.BufferSize,
		BatchSize:    cfg.BatchSize,
		FlushTimeout: time.Duration(cfg.FlushInterval) * time.Second,
		Workers:      cfg.Workers,
	}

	queue := NewEntryQueue(queueConfig, processor, p.Logger)

	srv := &ServiceV2{
		repo:          p.AuditRepository,
		ps:            p.PermService,
		ss:            p.StreamingService,
		logger:        &log,
		config:        cfg,
		queue:         queue,
		processor:     processor,
		sdm:           NewSensitiveDataManagerV2(),
		defaultFields: make(map[string]any),
	}

	srv.serviceState.Store(string(ServiceStateInitializing))
	srv.metrics.startTime = time.Now()

	if err := srv.validateConfig(cfg); err != nil {
		log.Error().Err(err).Msg("invalid audit configuration, using defaults")
		srv.applyDefaultConfig()
	}

	srv.configureSensitiveDataManager(p.Config.Get().App.Environment)

	if err := srv.registerDefaultSensitiveFields(); err != nil {
		log.Error().Err(err).Msg("failed to register default sensitive fields")
	}

	srv.defaultFields["auditVersion"] = AuditVersionTag
	srv.defaultFields["environment"] = p.Config.Get().App.Environment

	p.LC.Append(fx.Hook{
		OnStart: func(context.Context) error {
			return srv.Start()
		},
		OnStop: func(context.Context) error {
			return srv.Stop()
		},
	})

	return srv
}

// LogAction logs an audit action
func (s *ServiceV2) LogAction(params *services.LogActionParams, opts ...services.LogOption) error {
	// * Check if service is running
	if !s.isRunning.Load() {
		s.logger.Warn().Msg("attempt to log audit entry while service is stopped")
		return ErrServiceStopped
	}

	// * Create new entry (no object pooling to avoid memory corruption)
	entry := &audit.Entry{
		ID:             pulid.MustNew("ae_"),
		Resource:       params.Resource,
		ResourceID:     params.ResourceID,
		Action:         params.Action,
		CurrentState:   s.deepCopyMap(params.CurrentState),
		PreviousState:  s.deepCopyMap(params.PreviousState),
		UserID:         params.UserID,
		OrganizationID: params.OrganizationID,
		BusinessUnitID: params.BusinessUnitID,
		Timestamp:      time.Now().Unix(),
		Category:       DefaultAuditCategory,
		Metadata:       s.copyDefaultFields(),
		Critical:       params.Critical,
	}

	// * Apply options
	for _, opt := range opts {
		if err := opt(entry); err != nil {
			return eris.Wrap(err, "failed to apply audit option")
		}
	}

	// * Validate entry
	if err := entry.Validate(); err != nil {
		s.logger.Error().Err(err).Msg("invalid audit entry")
		return eris.Wrap(ErrInvalidEntry, err.Error())
	}

	// * Sanitize sensitive data
	if err := s.sdm.SanitizeEntry(entry); err != nil {
		s.logger.Error().Err(err).Msg("failed to sanitize sensitive data")
		return eris.Wrap(ErrSanitizationFailed, err.Error())
	}

	// * Enqueue entry
	var err error
	if params.Critical {
		// * Critical entries get longer timeout
		err = s.queue.EnqueueWithTimeout(entry, 5*time.Second)
	} else {
		err = s.queue.Enqueue(entry)
	}

	if err != nil {
		s.metrics.failedEntries.Inc()
		if eris.Is(err, ErrQueueFull) {
			s.logger.Warn().
				Str("resource", string(params.Resource)).
				Str("action", string(params.Action)).
				Bool("critical", params.Critical).
				Msg("audit queue full, entry dropped")
		}
		return err
	}

	s.metrics.totalEntries.Inc()
	return nil
}

// Start starts the audit service
func (s *ServiceV2) Start() error {
	if !s.isRunning.CompareAndSwap(false, true) {
		s.logger.Warn().Msg("audit service is already running")
		return nil
	}

	s.serviceState.Store(string(ServiceStateRunning))

	workers := s.config.Workers
	if workers == 0 {
		workers = 2
	}
	s.queue.Start(workers)

	s.logger.Info().
		Int("buffer_size", s.config.BufferSize).
		Int("batch_size", s.config.BatchSize).
		Int("flush_interval", s.config.FlushInterval).
		Int("workers", workers).
		Str("audit_version", AuditVersionTag).
		Msg("🚀 Audit service v2 started")

	return nil
}

// Stop stops the audit service
func (s *ServiceV2) Stop() error {
	// * Ensure we're running
	if !s.isRunning.CompareAndSwap(true, false) {
		s.logger.Warn().Msg("audit service is already stopped")
		return nil
	}

	s.serviceState.Store(string(ServiceStateStopping))
	s.logger.Info().Msg("stopping audit service")

	// * Stop the queue with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.queue.Stop(ctx); err != nil {
		s.logger.Error().Err(err).Msg("error stopping audit queue")
		s.serviceState.Store(string(ServiceStateDegraded))
		return err
	}

	s.serviceState.Store(string(ServiceStateStopped))

	// * Log final metrics
	s.logger.Info().
		Int64("total_entries", s.metrics.totalEntries.Load()).
		Int64("failed_entries", s.metrics.failedEntries.Load()).
		Int64("processed_batches", s.metrics.processedBatch.Load()).
		Str("uptime", time.Since(s.metrics.startTime).String()).
		Msg("audit service stopped successfully")

	return nil
}

// List lists audit entries
func (s *ServiceV2) List(
	ctx context.Context,
	opts *ports.LimitOffsetQueryOptions,
) (*ports.ListResult[*audit.Entry], error) {
	log := s.logger.With().
		Str("operation", "List").
		Str("buID", opts.TenantOpts.BuID.String()).
		Str("userID", opts.TenantOpts.UserID.String()).
		Logger()

	// * Check permissions
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

	// * List entries
	entities, err := s.repo.List(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to list audit entries")
		return nil, eris.Wrap(err, "list audit entries")
	}

	return entities, nil
}

// ListByResourceID lists audit entries by resource ID
func (s *ServiceV2) ListByResourceID(
	ctx context.Context,
	opts repositories.ListByResourceIDRequest,
) (*ports.ListResult[*audit.Entry], error) {
	log := s.logger.With().
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

// GetByID gets an audit entry by ID
func (s *ServiceV2) GetByID(
	ctx context.Context,
	opts repositories.GetAuditEntryByIDOptions,
) (*audit.Entry, error) {
	log := s.logger.With().
		Str("operation", "GetByID").
		Str("auditEntryID", opts.ID.String()).
		Logger()

	// TODO: Check permissions

	entity, err := s.repo.GetByID(ctx, opts)
	if err != nil {
		log.Error().Err(err).Str("auditEntryID", opts.ID.String()).Msg("failed to get audit entry")
		return nil, eris.Wrap(err, "get audit entry by id")
	}

	return entity, nil
}

// LiveStream streams audit entries
func (s *ServiceV2) LiveStream(
	c *fiber.Ctx,
	dataFetcher func(ctx context.Context, reqCtx *appctx.RequestContext) ([]*audit.Entry, error),
	timestampExtractor func(entry *audit.Entry) int64,
) error {
	streamDataFetcher := func(ctx context.Context, reqCtx *appctx.RequestContext) (any, error) {
		entries, err := dataFetcher(ctx, reqCtx)
		if err != nil {
			return nil, err
		}

		result := make([]any, len(entries))
		for i, entry := range entries {
			result[i] = entry
		}

		return result, nil
	}

	streamTimestampExtractor := func(item any) int64 {
		if entry, ok := item.(*audit.Entry); ok {
			return timestampExtractor(entry)
		}
		return timeutils.NowUnix()
	}

	return s.ss.StreamData(c, "audit", streamDataFetcher, streamTimestampExtractor)
}

// RegisterSensitiveFields registers sensitive fields for a resource
func (s *ServiceV2) RegisterSensitiveFields(
	resource permission.Resource,
	fields []services.SensitiveField,
) error {
	return s.sdm.RegisterSensitiveFields(resource, fields)
}

// SetDefaultField sets a default field that will be included in all audit entries
func (s *ServiceV2) SetDefaultField(key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.defaultFields[key] = value
}

// GetServiceStatus returns the current status of the audit service
func (s *ServiceV2) GetServiceStatus() string {
	return s.serviceState.Load()
}

// validateConfig validates the audit configuration
func (s *ServiceV2) validateConfig(cfg *config.AuditConfig) error {
	if cfg.BufferSize < MinBufferSize {
		return eris.Wrapf(ErrInvalidConfig, "buffer size too small (min: %d)", MinBufferSize)
	}
	if cfg.FlushInterval < MinFlushInterval {
		return eris.Wrapf(ErrInvalidConfig, "flush interval too small (min: %d)", MinFlushInterval)
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 50
	}
	if cfg.Workers <= 0 {
		cfg.Workers = 2
	}
	return nil
}

// applyDefaultConfig applies default configuration
func (s *ServiceV2) applyDefaultConfig() {
	if s.config.BufferSize < MinBufferSize {
		s.config.BufferSize = 1000
	}
	if s.config.FlushInterval < MinFlushInterval {
		s.config.FlushInterval = 10
	}
	if s.config.BatchSize <= 0 {
		s.config.BatchSize = 50
	}
	if s.config.Workers <= 0 {
		s.config.Workers = 2
	}
}

// copyDefaultFields returns a copy of default fields
func (s *ServiceV2) copyDefaultFields() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]any, len(s.defaultFields))
	maps.Copy(result, s.defaultFields)
	return result
}

// deepCopyMap creates a deep copy of a map to avoid memory corruption
func (s *ServiceV2) deepCopyMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}

	// * Use JSON marshaling for deep copy to avoid reference issues
	data, err := sonic.Marshal(m)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to marshal map for deep copy, using shallow copy")
		// * Fallback to shallow copy
		result := make(map[string]any, len(m))
		maps.Copy(result, m)
		return result
	}

	var result map[string]any
	if err = sonic.Unmarshal(data, &result); err != nil {
		s.logger.Error().Err(err).Msg("failed to unmarshal map for deep copy, using shallow copy")
		// * Fallback to shallow copy
		res := make(map[string]any, len(m))
		maps.Copy(res, m)
		return result
	}

	return result
}

// registerDefaultSensitiveFields registers default sensitive fields
func (s *ServiceV2) registerDefaultSensitiveFields() error {
	// * User-related sensitive data
	if err := s.sdm.RegisterSensitiveFields(permission.ResourceUser, []services.SensitiveField{
		{Name: "password", Action: services.SensitiveFieldOmit},
		{Name: "hashedPassword", Action: services.SensitiveFieldOmit},
		{Name: "emailAddress", Action: services.SensitiveFieldMask},
		{Name: "address", Action: services.SensitiveFieldMask},
	}); err != nil {
		return err
	}

	// * Organization data
	if err := s.sdm.RegisterSensitiveFields(permission.ResourceOrganization, []services.SensitiveField{
		{Name: "logoUrl", Action: services.SensitiveFieldMask},
		{Name: "taxId", Action: services.SensitiveFieldMask},
	}); err != nil {
		return err
	}

	// * Worker sensitive data
	if err := s.sdm.RegisterSensitiveFields(permission.ResourceWorker, []services.SensitiveField{
		{Name: "licenseNumber", Action: services.SensitiveFieldMask},
		{Name: "dateOfBirth", Action: services.SensitiveFieldMask},
		{Path: "profile", Name: "licenseNumber", Action: services.SensitiveFieldMask},
	}); err != nil {
		return err
	}

	return nil
}

// configureSensitiveDataManager configures the sensitive data manager based on environment
//
// Configuration Strategy:
// - Production: Auto-detection ON, Strict masking (maximum security)
// - Staging: Auto-detection ON, Default masking (balanced)
// - Development: Auto-detection ON, Partial masking (easier debugging)
// - Testing: Auto-detection OFF, Partial masking (predictable tests)
func (s *ServiceV2) configureSensitiveDataManager(environment string) {
	switch environment {
	case "production", "prod":
		// * Production: Maximum security
		s.sdm.SetAutoDetect(true)                 // Prevent accidental data exposure
		s.sdm.SetMaskStrategy(MaskStrategyStrict) // Show minimal information
		s.logger.Info().
			Msg("sensitive data manager configured for production (strict masking, auto-detect ON)")

	case "staging", "stage":
		// * Staging: Balanced approach
		s.sdm.SetAutoDetect(true)                  // Still protect sensitive data
		s.sdm.SetMaskStrategy(MaskStrategyDefault) // Balanced masking
		s.logger.Info().
			Msg("sensitive data manager configured for staging (default masking, auto-detect ON)")

	case "development", "dev":
		// * Development: Easier debugging
		s.sdm.SetAutoDetect(true)                  // Protect against accidental commits
		s.sdm.SetMaskStrategy(MaskStrategyPartial) // Show more for debugging
		s.logger.Info().
			Msg("sensitive data manager configured for development (partial masking, auto-detect ON)")

	case "test", "testing":
		// * Testing: Predictable behavior
		s.sdm.SetAutoDetect(false)                 // Only configured fields are masked
		s.sdm.SetMaskStrategy(MaskStrategyPartial) // Easier test assertions
		s.logger.Info().
			Msg("sensitive data manager configured for testing (partial masking, auto-detect OFF)")

	default:
		// * Unknown environment: Default to secure
		s.sdm.SetAutoDetect(true)
		s.sdm.SetMaskStrategy(MaskStrategyDefault)
		s.logger.Warn().
			Str("environment", environment).
			Msg("unknown environment, using default configuration")
	}
}

// SetSensitiveDataMaskStrategy allows runtime configuration of masking strategy
// This is useful for:
// - Temporarily enabling more verbose masking for debugging
// - Adjusting security levels based on compliance requirements
// - A/B testing different masking approaches
func (s *ServiceV2) SetSensitiveDataMaskStrategy(strategy MaskStrategy) {
	s.sdm.SetMaskStrategy(strategy)
	s.logger.Info().Int("strategy", int(strategy)).Msg("updated sensitive data mask strategy")
}

// SetSensitiveDataAutoDetect enables/disables automatic sensitive data detection
// Use cases:
// - Disable during data migration to preserve original values
// - Enable for enhanced security in production
// - Disable for performance optimization if all fields are explicitly configured
func (s *ServiceV2) SetSensitiveDataAutoDetect(enabled bool) {
	s.sdm.SetAutoDetect(enabled)
	s.logger.Info().Bool("enabled", enabled).Msg("updated sensitive data auto-detection")
}

// ClearSensitiveDataCache clears the regex pattern cache
// Use this when:
// - Memory usage is a concern (rarely needed)
// - Patterns have been updated and you need fresh compilation
// - During application restart/reload scenarios
func (s *ServiceV2) ClearSensitiveDataCache() {
	s.sdm.ClearCache()
	s.logger.Info().Msg("cleared sensitive data regex cache")
}
