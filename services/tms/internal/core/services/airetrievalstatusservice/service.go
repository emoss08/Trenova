package airetrievalstatusservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const maxSettingsWriteAttempts = 3

var _ serviceports.AIRetrievalStatusService = (*Service)(nil)

type Params struct {
	fx.In

	Logger     *zap.Logger
	Repo       repositories.AIRetrievalRepository
	Sources    repositories.RetrievalSourceRepository
	Usage      repositories.AIUsageRepository
	Providers  repositories.AIProviderRepository
	Vectorizer serviceports.QueryVectorizer
	Indexer    serviceports.RetrievalIndexer
	Pipeline   serviceports.RetrievalIndexPipeline
	Audit      serviceports.AuditService
	Embeddings serviceports.EmbeddingService `optional:"true"`
}

type Service struct {
	l          *zap.Logger
	repo       repositories.AIRetrievalRepository
	sources    repositories.RetrievalSourceRepository
	usage      repositories.AIUsageRepository
	providers  repositories.AIProviderRepository
	vectorizer serviceports.QueryVectorizer
	indexer    serviceports.RetrievalIndexer
	pipeline   serviceports.RetrievalIndexPipeline
	audit      serviceports.AuditService
	embeddings serviceports.EmbeddingService
	now        func() int64
}

func New(p Params) *Service { //nolint:gocritic // fx param structs are passed by value
	return &Service{
		l:          p.Logger.Named("service.ai-retrieval-status"),
		repo:       p.Repo,
		sources:    p.Sources,
		usage:      p.Usage,
		providers:  p.Providers,
		vectorizer: p.Vectorizer,
		indexer:    p.Indexer,
		pipeline:   p.Pipeline,
		audit:      p.Audit,
		embeddings: p.Embeddings,
		now:        timeutils.NowUnix,
	}
}

func AsService(s *Service) serviceports.AIRetrievalStatusService { return s }

func validateTenant(tenant pagination.TenantInfo) error {
	if tenant.OrgID.IsNil() || tenant.BuID.IsNil() {
		return errortypes.NewAuthorizationError("An organization and business unit are required")
	}

	return nil
}

func validateSourceType(sourceType airetrieval.SourceType) error {
	if sourceType.IsValid() {
		return nil
	}

	return errortypes.NewValidationError(
		"sourceType",
		errortypes.ErrInvalid,
		"Source type must be Memory, Document or InboundMessage",
	)
}

type statusInputs struct {
	availability airetrieval.Availability
	settings     *airetrieval.Settings
	indexing     *repositories.AIUsageCost
	retrieval    *repositories.AIUsageCost
	totals       map[airetrieval.SourceType]int
	active       []repositories.IndexEntryCount
	pending      []repositories.IndexEntryCount
	configured   string
}

func (s *Service) Status(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (*serviceports.AIRetrievalStatus, error) {
	if err := validateTenant(tenant); err != nil {
		return nil, err
	}

	settings, err := s.repo.GetSettings(ctx, tenant)
	if err != nil {
		return nil, fmt.Errorf("read retrieval settings: %w", err)
	}

	in, err := s.gather(ctx, tenant, settings)
	if err != nil {
		return nil, err
	}

	return buildStatus(in, timeutils.MonthStartUTC(s.now())), nil
}

func (s *Service) gather(
	ctx context.Context,
	tenant pagination.TenantInfo,
	settings *airetrieval.Settings,
) (*statusInputs, error) {
	in := &statusInputs{
		settings: settings,
		totals:   make(map[airetrieval.SourceType]int, len(airetrieval.AllSourceTypes())),
	}
	monthStart := timeutils.MonthStartUTC(s.now())
	totals := make([]int, len(airetrieval.AllSourceTypes()))

	group, gctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		availability, err := s.vectorizer.Availability(gctx, tenant)
		if err != nil {
			return fmt.Errorf("check retrieval availability: %w", err)
		}
		in.availability = availability
		return nil
	})
	group.Go(func() error {
		cost, err := s.surfaceCost(gctx, tenant, aiusage.SurfaceIndexing, monthStart)
		in.indexing = cost
		return err
	})
	group.Go(func() error {
		cost, err := s.surfaceCost(gctx, tenant, aiusage.SurfaceRetrieval, monthStart)
		in.retrieval = cost
		return err
	})
	for idx, sourceType := range airetrieval.AllSourceTypes() {
		group.Go(func() error {
			count, err := s.sources.CountSources(gctx, repositories.CountRetrievalSourcesRequest{
				TenantInfo: tenant,
				SourceType: sourceType,
			})
			if err != nil {
				return fmt.Errorf("count %s sources: %w", sourceType, err)
			}
			totals[idx] = count
			return nil
		})
	}
	if settings.HasActiveModel() {
		group.Go(func() error {
			counts, err := s.countEntries(gctx, tenant, settings.ActiveModelKey)
			in.active = counts
			return err
		})
	}
	if settings.HasPendingModel() {
		group.Go(func() error {
			counts, err := s.countEntries(gctx, tenant, settings.PendingModelKey)
			in.pending = counts
			return err
		})
	}
	group.Go(func() error {
		configured, err := s.configuredModelKey(gctx, tenant)
		in.configured = configured
		return err
	})

	if err := group.Wait(); err != nil {
		return nil, err
	}

	for idx, sourceType := range airetrieval.AllSourceTypes() {
		in.totals[sourceType] = totals[idx]
	}

	return in, nil
}

func (s *Service) surfaceCost(
	ctx context.Context,
	tenant pagination.TenantInfo,
	surface aiusage.Surface,
	since int64,
) (*repositories.AIUsageCost, error) {
	cost, err := s.usage.SurfaceCost(ctx, repositories.AIUsageSurfaceCostRequest{
		TenantInfo: tenant,
		Surface:    surface,
		Since:      since,
	})
	if err != nil {
		return nil, fmt.Errorf("sum this month's %s cost: %w", surface, err)
	}
	if cost == nil {
		return &repositories.AIUsageCost{}, nil
	}

	return cost, nil
}

func (s *Service) countEntries(
	ctx context.Context,
	tenant pagination.TenantInfo,
	modelKey string,
) ([]repositories.IndexEntryCount, error) {
	counts, err := s.repo.CountIndexEntries(ctx, repositories.CountIndexEntriesRequest{
		TenantInfo: tenant,
		ModelKey:   modelKey,
	})
	if err != nil {
		return nil, fmt.Errorf("count index entries for %s: %w", modelKey, err)
	}

	return counts, nil
}

func (s *Service) configuredModelKey(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (string, error) {
	if s.embeddings == nil {
		return "", nil
	}

	key, err := s.embeddings.ConfiguredModelKey(ctx, tenant)
	switch {
	case err == nil:
		return key, nil
	case errors.Is(err, serviceports.ErrNoProviderConfigured) || errortypes.IsBusinessError(err):
		return "", nil
	default:
		return "", fmt.Errorf("resolve the configured embedding model: %w", err)
	}
}

func (s *Service) UpdateSettings(
	ctx context.Context,
	req *serviceports.UpdateAIRetrievalSettingsRequest,
) (*serviceports.AIRetrievalStatus, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"input",
			errortypes.ErrRequired,
			"Settings to change are required",
		)
	}
	if err := validateTenant(req.TenantInfo); err != nil {
		return nil, err
	}

	var (
		saved    *airetrieval.Settings
		previous *airetrieval.Settings
		changed  bool
		err      error
	)
	for attempt := range maxSettingsWriteAttempts {
		saved, previous, changed, err = s.writeSettings(ctx, req)
		if err == nil || !errortypes.IsVersionMismatchError(err) ||
			attempt == maxSettingsWriteAttempts-1 {
			break
		}
	}
	if err != nil {
		return nil, err
	}

	if changed {
		s.logSettings(saved, previous, req.Actor)
		if !saved.Paused || saved.PausedByBudget() {
			s.pipeline.Wake(ctx, req.TenantInfo)
		}
	}

	return s.Status(ctx, req.TenantInfo)
}

func (s *Service) writeSettings(
	ctx context.Context,
	req *serviceports.UpdateAIRetrievalSettingsRequest,
) (saved, previous *airetrieval.Settings, changed bool, err error) {
	current, err := s.repo.GetSettings(ctx, req.TenantInfo)
	if err != nil {
		return nil, nil, false, fmt.Errorf("read retrieval settings: %w", err)
	}

	before := *current
	next := *current
	if !ApplySettingsPatch(&next, req, s.now()) {
		return current, &before, false, nil
	}

	multiErr := errortypes.NewMultiError()
	next.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, nil, false, multiErr
	}

	saved, err = s.repo.UpdateSettings(ctx, &next)
	if err != nil {
		return nil, nil, false, err
	}

	return saved, &before, true, nil
}

func ApplySettingsPatch(
	settings *airetrieval.Settings,
	req *serviceports.UpdateAIRetrievalSettingsRequest,
	now int64,
) bool {
	changed := false

	setBool := func(target *bool, value *bool) {
		if value != nil && *target != *value {
			*target = *value
			changed = true
		}
	}
	setBool(&settings.MemoryEnabled, req.MemoryEnabled)
	setBool(&settings.DocumentsEnabled, req.DocumentsEnabled)
	setBool(&settings.InboundMessagesEnabled, req.InboundMessagesEnabled)

	if budget := req.MonthlyIndexingBudgetUSD; budget != nil &&
		!budget.Equal(settings.MonthlyIndexingBudgetUSD) {
		settings.MonthlyIndexingBudgetUSD = *budget
		changed = true
	}

	if req.Paused == nil {
		return changed
	}

	switch {
	case *req.Paused && (!settings.Paused || settings.PausedReason != airetrieval.PauseReasonManual):
		pausedAt := now
		settings.Paused = true
		settings.PausedReason = airetrieval.PauseReasonManual
		settings.PausedAt = &pausedAt
		changed = true
	case !*req.Paused && settings.Paused:
		settings.Paused = false
		settings.PausedReason = ""
		settings.PausedAt = nil
		changed = true
	}

	return changed
}

func (s *Service) Reindex(
	ctx context.Context,
	req *serviceports.ReindexAIRetrievalSourceRequest,
) (*serviceports.AIRetrievalStatus, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"sourceType",
			errortypes.ErrRequired,
			"Source type is required",
		)
	}
	if err := validateTenant(req.TenantInfo); err != nil {
		return nil, err
	}
	if err := validateSourceType(req.SourceType); err != nil {
		return nil, err
	}

	settings, err := s.repo.GetSettings(ctx, req.TenantInfo)
	if err != nil {
		return nil, fmt.Errorf("read retrieval settings: %w", err)
	}
	availability, err := s.vectorizer.Availability(ctx, req.TenantInfo)
	if err != nil {
		return nil, fmt.Errorf("check retrieval availability: %w", err)
	}
	if err = ReindexRefusal(settings, availability, req.SourceType); err != nil {
		return nil, err
	}

	if err = s.indexer.Reindex(ctx, req.TenantInfo, req.SourceType); err != nil {
		return nil, err
	}
	s.logReindex(settings, req)

	return s.Status(ctx, req.TenantInfo)
}

func ReindexRefusal(
	settings *airetrieval.Settings,
	availability airetrieval.Availability,
	sourceType airetrieval.SourceType,
) error {
	if !settings.SourceEnabled(sourceType) {
		return sourceOffRefusal(sourceType)
	}

	switch availability.Reason {
	case airetrieval.UnavailableReasonExtensionMissing,
		airetrieval.UnavailableReasonTooOld:
		return errortypes.NewBusinessError(
			"The database has no usable pgvector, so nothing can be indexed. Install pgvector 0.8 or newer, then run trenova db enable-vector.",
		)
	case airetrieval.UnavailableReasonSchemaMissing:
		return errortypes.NewBusinessError(
			"The retrieval tables are missing, so nothing can be indexed. Run trenova db enable-vector.",
		)
	case airetrieval.UnavailableReasonNoProvider:
		return errortypes.NewBusinessError(
			"No provider is routed the Embedding task, so nothing can be indexed. Route it on the Providers tab.",
		)
	case airetrieval.UnavailableReasonDisabled,
		airetrieval.UnavailableReasonBudgetPaused,
		airetrieval.UnavailableReasonNotIndexed,
		airetrieval.UnavailableReasonQueryTimeout,
		airetrieval.UnavailableReasonProviderFailed:
	}

	if !settings.HasActiveModel() {
		return errortypes.NewBusinessError(
			"Nothing has been indexed yet. Indexing starts on its own within the hour.",
		)
	}

	return nil
}

func sourceOffRefusal(sourceType airetrieval.SourceType) error {
	switch sourceType {
	case airetrieval.SourceTypeMemory:
		return errortypes.NewBusinessError("Turn memories on before re-indexing them.")
	case airetrieval.SourceTypeDocument:
		return errortypes.NewBusinessError("Turn documents on before re-indexing them.")
	case airetrieval.SourceTypeInboundMessage:
	}

	return errortypes.NewBusinessError("Turn inbound email on before re-indexing it.")
}

func SourceLabel(sourceType airetrieval.SourceType) string {
	switch sourceType {
	case airetrieval.SourceTypeMemory:
		return "memories"
	case airetrieval.SourceTypeDocument:
		return "documents"
	case airetrieval.SourceTypeInboundMessage:
		return "inbound email"
	default:
		return sourceType.String()
	}
}

func auditResourceID(settings *airetrieval.Settings) string {
	if settings.ID.IsNotNil() {
		return settings.ID.String()
	}

	return settings.OrganizationID.String()
}

func (s *Service) logSettings(
	saved, previous *airetrieval.Settings,
	actor *serviceports.RequestActor,
) {
	auditActor := actor.AuditActorOrSystem()
	if err := s.audit.LogAction(&serviceports.LogActionParams{
		Resource:       permission.ResourceAIProvider,
		ResourceID:     auditResourceID(saved),
		Operation:      permission.OpUpdate,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		CurrentState:   jsonutils.MustToJSON(saved),
		PreviousState:  jsonutils.MustToJSON(previous),
		OrganizationID: saved.OrganizationID,
		BusinessUnitID: saved.BusinessUnitID,
	}, auditservice.WithComment("Retrieval settings updated")); err != nil {
		s.l.Error("failed to log retrieval settings audit", zap.Error(err))
	}
}

func (s *Service) logReindex(
	settings *airetrieval.Settings,
	req *serviceports.ReindexAIRetrievalSourceRequest,
) {
	auditActor := req.Actor.AuditActorOrSystem()
	if err := s.audit.LogAction(&serviceports.LogActionParams{
		Resource:      permission.ResourceAIProvider,
		ResourceID:    auditResourceID(settings),
		Operation:     permission.OpUpdate,
		UserID:        auditActor.UserID,
		PrincipalType: auditActor.PrincipalType,
		PrincipalID:   auditActor.PrincipalID,
		APIKeyID:      auditActor.APIKeyID,
		CurrentState: map[string]any{
			"sourceType":      req.SourceType,
			"activeModelKey":  settings.ActiveModelKey,
			"pendingModelKey": settings.PendingModelKey,
		},
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
	}, auditservice.WithComment(
		"Retrieval re-index requested for "+SourceLabel(req.SourceType),
	)); err != nil {
		s.l.Error("failed to log retrieval re-index audit", zap.Error(err))
	}
}
