package auditservice

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	LC                    fx.Lifecycle
	AuditRepository       repositories.AuditRepository
	AuditBufferRepository repositories.AuditBufferRepository
	Realtime              services.RealtimeService
	Logger                *zap.Logger
	Config                *config.Config
	Metrics               *metrics.Registry
}

type service struct {
	repo       repositories.AuditRepository
	bufferRepo repositories.AuditBufferRepository
	realtime   services.RealtimeService
	logger     *zap.Logger
	config     *config.Config
	sdm        *SensitiveDataManager
	metrics    *metrics.Registry
}

var clientBitToOperation = map[int64]permission.Operation{
	int64(permission.ClientOpRead):      permission.OpRead,
	int64(permission.ClientOpCreate):    permission.OpCreate,
	int64(permission.ClientOpUpdate):    permission.OpUpdate,
	int64(permission.ClientOpDelete):    permission.OpDelete,
	int64(permission.ClientOpExport):    permission.OpExport,
	int64(permission.ClientOpImport):    permission.OpImport,
	int64(permission.ClientOpApprove):   permission.OpApprove,
	int64(permission.ClientOpReject):    permission.OpReject,
	int64(permission.ClientOpAssign):    permission.OpAssign,
	int64(permission.ClientOpUnassign):  permission.OpUnassign,
	int64(permission.ClientOpArchive):   permission.OpArchive,
	int64(permission.ClientOpRestore):   permission.OpRestore,
	int64(permission.ClientOpSubmit):    permission.OpSubmit,
	int64(permission.ClientOpCancel):    permission.OpCancel,
	int64(permission.ClientOpDuplicate): permission.OpDuplicate,
	int64(permission.ClientOpClose):     permission.OpClose,
	int64(permission.ClientOpLock):      permission.OpLock,
	int64(permission.ClientOpUnlock):    permission.OpUnlock,
	int64(permission.ClientOpActivate):  permission.OpActivate,
	int64(permission.ClientOpReopen):    permission.OpReopen,
}

//nolint:gocritic // this is dependency injection
func New(p Params) services.AuditService {
	srv := &service{
		repo:       p.AuditRepository,
		bufferRepo: p.AuditBufferRepository,
		realtime:   p.Realtime,
		logger:     p.Logger.Named("service.audit"),
		config:     p.Config,
		sdm:        NewSensitiveDataManager(p.Config.Security.Encryption),
		metrics:    p.Metrics,
	}

	srv.configureSensitiveDataManager(p.Config.App.Env)
	if err := srv.registerDefaultSensitiveFields(); err != nil {
		p.Logger.Error("failed to register default sensitive fields", zap.Error(err))
	}

	return srv
}

func (s *service) LogAction(params *services.LogActionParams, opts ...services.LogOption) error {
	if params == nil {
		return ErrAuditParamsRequired
	}

	now := timeutils.NowUnix()
	principalType, principalID := resolveAuditPrincipal(params)

	entry := &audit.Entry{
		ID:             pulid.MustNew("ae_"),
		Resource:       params.Resource,
		ResourceID:     params.ResourceID,
		Operation:      params.Operation,
		CurrentState:   params.CurrentState,
		PreviousState:  params.PreviousState,
		UserID:         params.UserID,
		PrincipalType:  string(principalType),
		PrincipalID:    principalID,
		APIKeyID:       params.APIKeyID,
		OrganizationID: params.OrganizationID,
		BusinessUnitID: params.BusinessUnitID,
		Timestamp:      now,
		Category:       audit.CategorySystem,
		Metadata:       make(map[string]any),
		Critical:       params.Critical,
	}

	for _, opt := range opts {
		if err := opt(entry); err != nil {
			return err
		}
	}

	if err := entry.Validate(); err != nil {
		s.logger.Error("invalid audit entry", zap.Error(err))
		return fmt.Errorf("invalid audit entry: %w", err)
	}

	if err := s.sdm.SanitizeEntry(entry); err != nil {
		s.logger.Error("failed to sanitize sensitive data", zap.Error(err))
		return fmt.Errorf("failed to sanitize sensitive data: %w", err)
	}

	if params.Critical {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.repo.InsertAuditEntries(ctx, []*audit.Entry{entry}); err != nil {
			s.logger.Error("failed to insert critical audit entry",
				zap.Error(err),
				zap.String("resource", string(params.Resource)),
			)
			return fmt.Errorf("failed to insert critical audit entry: %w", err)
		}
		s.metrics.Audit.RecordDirectInsert()
		s.publishRealtimeInvalidation(ctx, []*audit.Entry{entry})
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := s.bufferRepo.Push(ctx, entry); err != nil {
		s.logger.Error("failed to push to Redis buffer, falling back to direct insert",
			zap.Error(err),
			zap.String("entryID", entry.ID.String()),
		)
		s.metrics.Audit.RecordBufferPushFailure()

		directCtx, directCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer directCancel()
		if fallbackErr := s.repo.InsertAuditEntries(
			directCtx,
			[]*audit.Entry{entry},
		); fallbackErr != nil {
			s.logger.Error("fallback direct insert also failed", zap.Error(fallbackErr))
			return fmt.Errorf("failed to insert audit entry: %w", fallbackErr)
		}
		s.metrics.Audit.RecordFallbackInsert()
		s.publishRealtimeInvalidation(directCtx, []*audit.Entry{entry})
		return nil
	}

	s.metrics.Audit.RecordBufferPush()
	return nil
}

func (s *service) LogActions(bulkEntries []services.BulkLogEntry) error {
	if len(bulkEntries) == 0 {
		return nil
	}

	criticalEntries, nonCriticalEntries, validCount := s.buildBulkAuditEntries(bulkEntries)

	if validCount == 0 {
		return fmt.Errorf("all %d audit entries failed validation/sanitization", len(bulkEntries))
	}

	s.insertCriticalEntries(criticalEntries)

	s.pushNonCriticalEntries(nonCriticalEntries)

	return nil
}

func (s *service) buildBulkAuditEntries(
	bulkEntries []services.BulkLogEntry,
) (criticalEntries, nonCriticalEntries []*audit.Entry, validCount int) {
	for i, bulkEntry := range bulkEntries {
		if bulkEntry.Params == nil {
			s.logger.Warn("nil audit params in bulk entry, skipping", zap.Int("index", i))
			continue
		}

		principalType, principalID := resolveAuditPrincipal(bulkEntry.Params)
		entry := &audit.Entry{
			ID:             pulid.MustNew("ae_"),
			Resource:       bulkEntry.Params.Resource,
			ResourceID:     bulkEntry.Params.ResourceID,
			Operation:      bulkEntry.Params.Operation,
			CurrentState:   bulkEntry.Params.CurrentState,
			PreviousState:  bulkEntry.Params.PreviousState,
			UserID:         bulkEntry.Params.UserID,
			PrincipalType:  string(principalType),
			PrincipalID:    principalID,
			APIKeyID:       bulkEntry.Params.APIKeyID,
			OrganizationID: bulkEntry.Params.OrganizationID,
			BusinessUnitID: bulkEntry.Params.BusinessUnitID,
			Timestamp:      time.Now().Unix(),
			Category:       audit.CategorySystem,
			Metadata:       make(map[string]any),
			Critical:       bulkEntry.Params.Critical,
		}

		var optErr error
		for _, opt := range bulkEntry.Options {
			if err := opt(entry); err != nil {
				optErr = err
				break
			}
		}
		if optErr != nil {
			s.logger.Warn("failed to apply options to audit entry, skipping",
				zap.Int("index", i),
				zap.Error(optErr),
			)
			continue
		}

		if err := entry.Validate(); err != nil {
			s.logger.Warn("invalid audit entry, skipping",
				zap.Int("index", i),
				zap.Error(err),
			)
			continue
		}

		if err := s.sdm.SanitizeEntry(entry); err != nil {
			s.logger.Warn("failed to sanitize audit entry, skipping",
				zap.Int("index", i),
				zap.Error(err),
			)
			continue
		}

		validCount++
		if entry.Critical {
			criticalEntries = append(criticalEntries, entry)
		} else {
			nonCriticalEntries = append(nonCriticalEntries, entry)
		}
	}

	return criticalEntries, nonCriticalEntries, validCount
}

func resolveAuditPrincipal(params *services.LogActionParams) (services.PrincipalType, pulid.ID) {
	if params.PrincipalType != "" && params.PrincipalID.IsNotNil() {
		return params.PrincipalType, params.PrincipalID
	}

	return services.PrincipalTypeUser, params.UserID
}

func (s *service) insertCriticalEntries(criticalEntries []*audit.Entry) {
	if len(criticalEntries) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.repo.InsertAuditEntries(ctx, criticalEntries); err != nil {
		s.logger.Error("failed to insert critical audit entries",
			zap.Error(err),
			zap.Int("count", len(criticalEntries)),
		)
		return
	}

	s.metrics.Audit.RecordDirectInsert()
	s.publishRealtimeInvalidation(ctx, criticalEntries)
}

func (s *service) pushNonCriticalEntries(nonCriticalEntries []*audit.Entry) {
	if len(nonCriticalEntries) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.bufferRepo.PushBatch(ctx, nonCriticalEntries); err != nil {
		s.logger.Error("failed to push batch to Redis buffer, falling back to direct insert",
			zap.Error(err),
			zap.Int("count", len(nonCriticalEntries)),
		)
		s.metrics.Audit.RecordBufferPushFailure()

		directCtx, directCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer directCancel()

		if fallbackErr := s.repo.
			InsertAuditEntries(
				directCtx,
				nonCriticalEntries,
			); fallbackErr != nil {
			s.logger.Error("fallback direct insert also failed", zap.Error(fallbackErr))
			return
		}

		s.metrics.Audit.RecordFallbackInsert()
		s.publishRealtimeInvalidation(directCtx, nonCriticalEntries)
		return
	}

	s.metrics.Audit.RecordBufferPush()
}

func (s *service) publishRealtimeInvalidation(ctx context.Context, entries []*audit.Entry) {
	if s.realtime == nil || len(entries) == 0 {
		return
	}

	type tenantBatch struct {
		orgID         pulid.ID
		buID          pulid.ID
		actorUserID   pulid.ID
		actorID       pulid.ID
		actorType     services.PrincipalType
		actorAPIKeyID pulid.ID
		record        pulid.ID
		count         int
	}

	tenantBatches := make(map[string]*tenantBatch, len(entries))
	for _, entry := range entries {
		if entry == nil || entry.OrganizationID.IsNil() || entry.BusinessUnitID.IsNil() {
			continue
		}

		key := entry.RealtimeBatchKey()
		batch, ok := tenantBatches[key]
		if !ok {
			batch = &tenantBatch{
				orgID:         entry.OrganizationID,
				buID:          entry.BusinessUnitID,
				actorUserID:   entry.UserID,
				actorID:       entry.PrincipalID,
				actorType:     services.PrincipalType(entry.PrincipalType),
				actorAPIKeyID: entry.APIKeyID,
				record:        entry.ID,
			}
			tenantBatches[key] = batch
		}

		batch.count++
	}

	for _, batch := range tenantBatches {
		action := "created"
		recordID := batch.record
		if batch.count > 1 {
			action = "bulk_created"
			recordID = pulid.ID("")
		}

		if err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
			OrganizationID: batch.orgID,
			BusinessUnitID: batch.buID,
			ActorUserID:    batch.actorUserID,
			ActorType:      batch.actorType,
			ActorID:        batch.actorID,
			ActorAPIKeyID:  batch.actorAPIKeyID,
			Resource:       "audit-logs",
			Action:         action,
			RecordID:       recordID,
		}); err != nil {
			s.logger.Warn(
				"failed to publish audit realtime invalidation",
				zap.Error(err),
				zap.String("organizationID", batch.orgID.String()),
				zap.String("businessUnitID", batch.buID.String()),
				zap.String("action", action),
			)
		}
	}
}

func (s *service) List(
	ctx context.Context,
	req *repositories.ListAuditEntriesRequest,
) (*pagination.ListResult[*audit.Entry], error) {
	log := s.logger.With(
		zap.String("operation", "List"),
		zap.String("buID", req.Filter.TenantInfo.BuID.String()),
		zap.String("userID", req.Filter.TenantInfo.UserID.String()),
	)

	normalizeOperationFilters(req.Filter)

	entities, err := s.repo.List(ctx, req)
	if err != nil {
		log.Error("failed to list audit entries", zap.Error(err))
		return nil, fmt.Errorf("failed to list audit entries: %w", err)
	}

	return entities, nil
}

func (s *service) ListByResourceID(
	ctx context.Context,
	req *repositories.ListByResourceIDRequest,
) (*pagination.ListResult[*audit.Entry], error) {
	log := s.logger.With(
		zap.String("operation", "ListByResourceID"),
		zap.String("resourceID", req.ResourceID.String()),
	)

	normalizeOperationFilters(req.Filter)

	entities, err := s.repo.ListByResourceID(ctx, req)
	if err != nil {
		log.Error("failed to list audit entries by resource id", zap.Error(err))
		return nil, fmt.Errorf("failed to list audit entries by resource id: %w", err)
	}

	return entities, nil
}

func (s *service) GetByID(
	ctx context.Context,
	req repositories.GetAuditEntryByIDOptions,
) (*audit.Entry, error) {
	log := s.logger.With(
		zap.String("operation", "GetByID"),
		zap.String("auditEntryID", req.EntryID.String()),
	)

	entity, err := s.repo.GetByID(ctx, req)
	if err != nil {
		log.Error("failed to get audit entry", zap.Error(err))
		return nil, fmt.Errorf("failed to get audit entry by id: %w", err)
	}

	return entity, nil
}

func (s *service) RegisterSensitiveFields(
	resource permission.Resource,
	fields []services.SensitiveField,
) error {
	return s.sdm.RegisterSensitiveFields(resource, fields)
}

func (s *service) registerDefaultSensitiveFields() error {
	if err := s.sdm.RegisterSensitiveFields(permission.ResourceUser, []services.SensitiveField{
		{Name: "password", Action: services.SensitiveFieldOmit},
		{Name: "hashedPassword", Action: services.SensitiveFieldOmit},
		{Name: "emailAddress", Action: services.SensitiveFieldMask},
		{Name: "address", Action: services.SensitiveFieldMask},
	}); err != nil {
		return err
	}

	if err := s.sdm.RegisterSensitiveFields(
		permission.ResourceOrganization,
		[]services.SensitiveField{
			{Name: "logoUrl", Action: services.SensitiveFieldMask},
			{Name: "taxId", Action: services.SensitiveFieldMask},
		},
	); err != nil {
		return err
	}

	if err := s.sdm.RegisterSensitiveFields(permission.ResourceWorker, []services.SensitiveField{
		{Name: "licenseNumber", Action: services.SensitiveFieldMask},
		{Name: "dateOfBirth", Action: services.SensitiveFieldMask},
		{Path: "profile", Name: "licenseNumber", Action: services.SensitiveFieldMask},
	}); err != nil {
		return err
	}

	return nil
}

func normalizeOperationFilters(opts *pagination.QueryOptions) {
	if opts == nil {
		return
	}

	normalizeOperationFieldFilters(opts.FieldFilters)
	for idx := range opts.FilterGroups {
		normalizeOperationFieldFilters(opts.FilterGroups[idx].Filters)
	}
}

func normalizeOperationFieldFilters(filters []domaintypes.FieldFilter) {
	for idx := range filters {
		if filters[idx].Field != "operation" {
			continue
		}

		filters[idx].Value = normalizeOperationFilterValue(filters[idx].Value)
	}
}

func normalizeOperationFilterValue(value any) any {
	switch v := value.(type) {
	case []string:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if normalized, ok := toOperationString(item); ok {
				out = append(out, normalized)
			}
		}
		return out
	case []int:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if normalized, ok := toOperationString(item); ok {
				out = append(out, normalized)
			}
		}
		return out
	case []int64:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if normalized, ok := toOperationString(item); ok {
				out = append(out, normalized)
			}
		}
		return out
	case []float64:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if normalized, ok := toOperationString(item); ok {
				out = append(out, normalized)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if normalized, ok := toOperationString(item); ok {
				out = append(out, normalized)
			}
		}
		return out
	default:
		if normalized, ok := toOperationString(value); ok {
			return normalized
		}
		return value
	}
}

func toOperationString(value any) (string, bool) {
	switch v := value.(type) {
	case permission.Operation:
		return string(v), true
	case string:
		normalized := strings.ToLower(strings.TrimSpace(v))
		if normalized == "" {
			return "", false
		}

		if numeric, err := strconv.ParseInt(normalized, 10, 64); err == nil {
			if op, ok := clientBitToOperation[numeric]; ok {
				return string(op), true
			}
			return strconv.FormatInt(numeric, 10), true
		}

		return normalized, true
	case int:
		return normalizeOperationFromInt64(int64(v))
	case int8:
		return normalizeOperationFromInt64(int64(v))
	case int16:
		return normalizeOperationFromInt64(int64(v))
	case int32:
		return normalizeOperationFromInt64(int64(v))
	case int64:
		return normalizeOperationFromInt64(v)
	case uint:
		return normalizeOperationFromUint64(uint64(v))
	case uint8:
		return normalizeOperationFromUint64(uint64(v))
	case uint16:
		return normalizeOperationFromUint64(uint64(v))
	case uint32:
		return normalizeOperationFromUint64(uint64(v))
	case uint64:
		return normalizeOperationFromUint64(v)
	case float32:
		if math.Trunc(float64(v)) != float64(v) {
			return strconv.FormatFloat(float64(v), 'f', -1, 32), true
		}
		return normalizeOperationFromInt64(int64(v))
	case float64:
		if math.Trunc(v) != v {
			return strconv.FormatFloat(v, 'f', -1, 64), true
		}
		return normalizeOperationFromInt64(int64(v))
	default:
		return "", false
	}
}

func normalizeOperationFromInt64(value int64) (string, bool) {
	if op, ok := clientBitToOperation[value]; ok {
		return string(op), true
	}

	return strconv.FormatInt(value, 10), true
}

func normalizeOperationFromUint64(value uint64) (string, bool) {
	n, err := intutils.SafeUint64ToInt64(value)
	if err != nil {
		return strconv.FormatUint(value, 10), true
	}

	return normalizeOperationFromInt64(n)
}

func (s *service) configureSensitiveDataManager(environment string) {
	switch environment {
	case "production", "prod":
		s.sdm.SetAutoDetect(true)
		s.sdm.SetMaskStrategy(MaskStrategyStrict)
		s.logger.Info(
			"sensitive data manager configured for production (strict masking, auto-detect ON)",
		)

	case "staging", "stage":
		s.sdm.SetAutoDetect(true)
		s.sdm.SetMaskStrategy(MaskStrategyDefault)
		s.logger.Info(
			"sensitive data manager configured for staging (default masking, auto-detect ON)",
		)

	case "development", "dev":
		s.sdm.SetAutoDetect(true)
		s.sdm.SetMaskStrategy(MaskStrategyPartial)
		s.logger.Info(
			"sensitive data manager configured for development (partial masking, auto-detect ON)",
		)

	case "test", "testing":
		s.sdm.SetAutoDetect(false)
		s.sdm.SetMaskStrategy(MaskStrategyPartial)
		s.logger.Info(
			"sensitive data manager configured for testing (partial masking, auto-detect OFF)",
		)

	default:
		s.sdm.SetAutoDetect(true)
		s.sdm.SetMaskStrategy(MaskStrategyDefault)
		s.logger.Warn(
			"unknown environment, using default configuration",
			zap.String("environment", environment),
		)
	}
}
