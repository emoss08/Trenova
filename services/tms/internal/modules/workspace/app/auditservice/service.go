package auditservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config/auditconfig"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/auditctx"
	"github.com/emoss08/trenova/pkg/requestmeta"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const writeTimeout = 10 * time.Second

var ErrNoValidEntries = errors.New("no audit entry was valid")

type Params struct {
	fx.In

	AuditRepository  auditports.AuditRepository
	Realtime         realtimeports.RealtimeService
	Logger           *zap.Logger
	Config           *auditconfig.Config
	Metrics          *metrics.Registry
	AgentDefinitions agentdefinitionports.AgentDefinitionRepository
	Alerts           *Alerts `optional:"true"`
}

type service struct {
	repo     auditports.AuditRepository
	realtime realtimeports.RealtimeService
	logger   *zap.Logger
	redactor *Redactor
	metrics  *metrics.Registry
	agents   *agentCredits
	alerts   *Alerts
}

//nolint:gocritic // this is dependency injection
func New(p Params) auditports.AuditService {
	logger := p.Logger.Named("service.audit")
	pepper := ""
	if p.Config != nil {
		pepper = p.Config.FingerprintPepper
	}

	return &service{
		repo:     p.AuditRepository,
		realtime: p.Realtime,
		logger:   logger,
		redactor: NewRedactor(pepper),
		metrics:  p.Metrics,
		agents:   newAgentCredits(p.AgentDefinitions, logger),
		alerts:   p.Alerts,
	}
}

func (s *service) LogAction(
	ctx context.Context,
	params *auditports.LogActionParams,
	opts ...auditports.LogOption,
) error {
	if params == nil {
		return ErrAuditParamsRequired
	}

	entry, err := s.build(ctx, params, opts)
	if err != nil {
		s.logger.Error("invalid audit entry",
			zap.String("resource", string(params.Resource)),
			zap.String("resourceId", params.ResourceID),
			zap.Error(err),
		)
		return fmt.Errorf("invalid audit entry: %w", err)
	}

	return s.record(ctx, []*audit.Entry{entry})
}

func (s *service) LogActions(ctx context.Context, bulkEntries []auditports.BulkLogEntry) error {
	if len(bulkEntries) == 0 {
		return nil
	}

	entries := make([]*audit.Entry, 0, len(bulkEntries))
	var invalid []error
	for i, bulk := range bulkEntries {
		if bulk.Params == nil {
			invalid = append(invalid, fmt.Errorf("entry %d: %w", i, ErrAuditParamsRequired))
			continue
		}
		entry, err := s.build(ctx, bulk.Params, bulk.Options)
		if err != nil {
			invalid = append(invalid, fmt.Errorf("entry %d: %w", i, err))
			continue
		}
		entries = append(entries, entry)
	}

	if len(invalid) > 0 {
		s.logger.Error("audit entries were invalid and are not recorded",
			zap.Int("invalid", len(invalid)),
			zap.Int("total", len(bulkEntries)),
			zap.Error(errors.Join(invalid...)),
		)
	}
	if len(entries) == 0 {
		return fmt.Errorf("%w: %w", ErrNoValidEntries, errors.Join(invalid...))
	}

	return s.record(ctx, entries)
}

func (s *service) build(
	ctx context.Context,
	params *auditports.LogActionParams,
	opts []auditports.LogOption,
) (*audit.Entry, error) {
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
		Timestamp:      timeutils.NowUnix(),
		Category:       audit.CategorySystem,
		Critical:       params.Critical,
	}

	for _, opt := range opts {
		if err := opt(entry); err != nil {
			return nil, err
		}
	}

	enrichFromContext(ctx, entry)

	if err := entry.Validate(); err != nil {
		return nil, err
	}

	s.agents.credit(ctx, entry)
	s.redactor.RedactEntry(entry)
	compactStates(entry)

	return entry, nil
}

func enrichFromContext(ctx context.Context, entry *audit.Entry) {
	if meta, ok := requestmeta.From(ctx); ok {
		if entry.IPAddress == "" {
			entry.IPAddress = meta.ClientIP
		}
		if entry.UserAgent == "" {
			entry.UserAgent = meta.UserAgent
		}
		if entry.CorrelationID == "" {
			entry.CorrelationID = meta.RequestID
		}
	}

	if entry.TraceID == "" {
		if span := trace.SpanContextFromContext(ctx); span.HasTraceID() {
			entry.TraceID = span.TraceID().String()
		}
	}

	source := auditctx.From(ctx)
	if entry.AIRunID == "" && source.AIRunID != "" {
		entry.AIRunID = source.AIRunID
	}
	if entry.Channel == "" {
		entry.Channel = channelOf(ctx, entry, source)
	}
}

func channelOf(ctx context.Context, entry *audit.Entry, source auditctx.Source) audit.Channel {
	if source.Channel != "" {
		return audit.Channel(source.Channel)
	}

	_, fromRequest := requestmeta.From(ctx)
	switch kernelports.PrincipalType(entry.PrincipalType) {
	case permissionports.PrincipalTypeAPIKey:
		return audit.ChannelAPI
	case permissionports.PrincipalTypeAgent:
		return audit.ChannelAgentRun
	case permissionports.PrincipalTypeSystem:
		if fromRequest {
			return audit.ChannelSystem
		}
		return audit.ChannelJob
	default:
		if fromRequest {
			return audit.ChannelWeb
		}
		return audit.ChannelJob
	}
}

func compactStates(entry *audit.Entry) {
	switch entry.Operation { //nolint:exhaustive // only creates and deletes keep a state
	case permission.OpCreate:
		entry.PreviousState = nil
		return
	case permission.OpDelete:
		entry.CurrentState = nil
		return
	}
	if len(entry.Changes) > 0 {
		entry.PreviousState = nil
		entry.CurrentState = nil
	}
}

func resolveAuditPrincipal(
	params *auditports.LogActionParams,
) (kernelports.PrincipalType, pulid.ID) {
	if params.PrincipalType != "" && params.PrincipalID.IsNotNil() {
		return params.PrincipalType, params.PrincipalID
	}

	return permissionports.PrincipalTypeUser, params.UserID
}

func (s *service) record(ctx context.Context, entries []*audit.Entry) error {
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), writeTimeout)
	defer cancel()

	started := time.Now()
	err := s.repo.Insert(writeCtx, entries)
	s.metrics.Audit.RecordWrite(countByClass(entries), err == nil, time.Since(started))
	if err != nil {
		for _, entry := range entries {
			s.logger.Error("audit entry could not be stored; this log line is its only copy",
				zap.Error(err),
				zap.Any("entry", entry),
			)
		}
		return fmt.Errorf("record audit entries: %w", err)
	}

	kernelports.AfterCommit(ctx, func(ctx context.Context) {
		ctx = context.WithoutCancel(ctx)
		s.publishRecordChanges(ctx, entries)
		s.publishRealtimeInvalidation(ctx, entries)
		s.alerts.Observe(ctx, entries)
	})

	return nil
}

func countByClass(entries []*audit.Entry) map[string]int {
	counts := make(map[string]int, len(audit.RetentionClasses))
	for _, entry := range entries {
		counts[string(entry.RetentionClass)]++
	}
	return counts
}
