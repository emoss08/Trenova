package retrievalquery

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/airetrieval"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	queryCacheSize = 1024
	queryCacheTTL  = 15 * time.Minute
)

var _ agentports.QueryVectorizer = (*Service)(nil)

type Params struct {
	fx.In

	Logger     *zap.Logger
	Repository agentports.AIRetrievalRepository
	Embeddings aiusageports.EmbeddingService `optional:"true"`
}

type Service struct {
	l          *zap.Logger
	repo       agentports.AIRetrievalRepository
	embeddings aiusageports.EmbeddingService
	vectors    *expirable.LRU[queryKey, agentports.QueryVector]
}

type queryKey struct {
	organizationID pulid.ID
	businessUnitID pulid.ID
	modelKey       string
	textHash       string
}

func New(p Params) *Service {
	return &Service{
		l:          p.Logger.Named("service.retrievalquery"),
		repo:       p.Repository,
		embeddings: p.Embeddings,
		vectors: expirable.NewLRU[queryKey, agentports.QueryVector](
			queryCacheSize,
			nil,
			queryCacheTTL,
		),
	}
}

func (s *Service) Vectorize(
	ctx context.Context,
	req *agentports.QueryVectorRequest,
) (agentports.QueryVector, error) {
	if req == nil {
		return agentports.QueryVector{}, agentports.ErrQueryTextRequired
	}

	text := QueryText(req.Text)
	if text == "" {
		return agentports.QueryVector{}, agentports.ErrQueryTextRequired
	}
	if err := validateTenant(req.TenantInfo); err != nil {
		return agentports.QueryVector{}, err
	}

	settings, reason, err := s.searchable(ctx, req.TenantInfo)
	if err != nil {
		return agentports.QueryVector{}, err
	}
	if reason != "" {
		return agentports.UnavailableQueryVector(reason), nil
	}

	key := queryKey{
		organizationID: req.TenantInfo.OrgID,
		businessUnitID: req.TenantInfo.BuID,
		modelKey:       settings.ActiveModelKey,
		textHash:       hashutils.SHA256Hex(text),
	}
	if cached, ok := s.vectors.Get(key); ok {
		return cached, nil
	}

	result, err := s.embeddings.Embed(ctx, &aiusageports.EmbedRequest{
		TenantInfo:  req.TenantInfo,
		Purpose:     aiusageports.EmbeddingPurposeQuery,
		Inputs:      []string{text},
		ModelKey:    settings.ActiveModelKey,
		Attribution: req.Attribution,
	})
	if err != nil {
		return s.embedFailed(ctx, req.TenantInfo, settings.ActiveModelKey, err)
	}

	vector, err := queryVectorFrom(&result, settings)
	if err != nil {
		s.l.Error("the embedding provider answered a search query with an unusable vector; "+
			"searching by keyword",
			zap.String("organization", req.TenantInfo.OrgID.String()),
			zap.String("modelKey", settings.ActiveModelKey),
			zap.Error(err),
		)

		return agentports.UnavailableQueryVector(airetrieval.UnavailableReasonProviderFailed), nil
	}

	s.vectors.Add(key, vector)

	return vector, nil
}

func (s *Service) Availability(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (airetrieval.Availability, error) {
	if err := validateTenant(tenant); err != nil {
		return airetrieval.Availability{}, err
	}

	storage, err := s.repo.VectorAvailability(ctx)
	if err != nil {
		return airetrieval.Availability{}, fmt.Errorf("read vector storage availability: %w", err)
	}
	if !storage.Available {
		return storage, nil
	}

	unavailable := func(reason airetrieval.UnavailableReason) airetrieval.Availability {
		return airetrieval.Availability{
			Reason:             reason,
			ExtensionInstalled: storage.ExtensionInstalled,
			ExtensionVersion:   storage.ExtensionVersion,
		}
	}

	settings, err := s.repo.GetSettings(ctx, tenant)
	if err != nil {
		return airetrieval.Availability{}, fmt.Errorf("read retrieval settings: %w", err)
	}
	if reason := pausedReason(settings); reason != "" {
		return unavailable(reason), nil
	}

	provided, err := s.hasProvider(ctx, tenant)
	if err != nil {
		return airetrieval.Availability{}, err
	}
	if !provided {
		return unavailable(airetrieval.UnavailableReasonNoProvider), nil
	}
	if !settings.HasActiveModel() {
		return unavailable(airetrieval.UnavailableReasonNotIndexed), nil
	}

	return storage, nil
}

func (s *Service) searchable(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (*airetrieval.Settings, airetrieval.UnavailableReason, error) {
	storage, err := s.repo.VectorAvailability(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("read vector storage availability: %w", err)
	}
	if !storage.Available {
		return nil, storage.Reason, nil
	}
	if s.embeddings == nil {
		return nil, airetrieval.UnavailableReasonNoProvider, nil
	}

	settings, err := s.repo.GetSettings(ctx, tenant)
	if err != nil {
		return nil, "", fmt.Errorf("read retrieval settings: %w", err)
	}
	if reason := pausedReason(settings); reason != "" {
		return nil, reason, nil
	}
	if settings.HasActiveModel() {
		return settings, "", nil
	}

	provided, err := s.hasProvider(ctx, tenant)
	if err != nil {
		return nil, "", err
	}
	if !provided {
		return nil, airetrieval.UnavailableReasonNoProvider, nil
	}

	return nil, airetrieval.UnavailableReasonNotIndexed, nil
}

func (s *Service) hasProvider(ctx context.Context, tenant pagination.TenantInfo) (bool, error) {
	if s.embeddings == nil {
		return false, nil
	}

	_, err := s.embeddings.ConfiguredModelKey(ctx, tenant)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, agentports.ErrNoProviderConfigured):
		return false, nil
	default:
		return false, fmt.Errorf("read the embedding provider: %w", err)
	}
}

func (s *Service) embedFailed(
	ctx context.Context,
	tenant pagination.TenantInfo,
	modelKey string,
	err error,
) (agentports.QueryVector, error) {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return agentports.QueryVector{}, ctxErr
	}

	reason := EmbedFailureReason(err)
	fields := []zap.Field{
		zap.String("organization", tenant.OrgID.String()),
		zap.String("modelKey", modelKey),
		zap.String("reason", reason.String()),
		zap.Error(err),
	}
	switch reason {
	case airetrieval.UnavailableReasonNoProvider:
		s.l.Info("no enabled provider serves the organization's embedding model; "+
			"searching by keyword", fields...)
	case airetrieval.UnavailableReasonQueryTimeout:
		s.l.Warn("embedding a search query took too long; searching by keyword", fields...)
	case airetrieval.UnavailableReasonProviderFailed,
		airetrieval.UnavailableReasonExtensionMissing,
		airetrieval.UnavailableReasonSchemaMissing,
		airetrieval.UnavailableReasonTooOld,
		airetrieval.UnavailableReasonDisabled,
		airetrieval.UnavailableReasonBudgetPaused,
		airetrieval.UnavailableReasonNotIndexed:
		s.l.Warn("embedding a search query failed; searching by keyword", fields...)
	}

	return agentports.UnavailableQueryVector(reason), nil
}

func EmbedFailureReason(err error) airetrieval.UnavailableReason {
	switch {
	case errors.Is(err, agentports.ErrNoProviderConfigured):
		return airetrieval.UnavailableReasonNoProvider
	case errors.Is(err, context.DeadlineExceeded):
		return airetrieval.UnavailableReasonQueryTimeout
	default:
		return airetrieval.UnavailableReasonProviderFailed
	}
}

func queryVectorFrom(
	result *aiusageports.EmbedResult,
	settings *airetrieval.Settings,
) (agentports.QueryVector, error) {
	if len(result.Vectors) != 1 {
		return agentports.QueryVector{}, fmt.Errorf(
			"expected one vector, got %d", len(result.Vectors),
		)
	}
	if result.ModelKey != "" && result.ModelKey != settings.ActiveModelKey {
		return agentports.QueryVector{}, fmt.Errorf(
			"embedded with %q, the index is built with %q",
			result.ModelKey, settings.ActiveModelKey,
		)
	}

	vector := result.Vectors[0]
	if len(vector) != settings.Dimensions {
		return agentports.QueryVector{}, fmt.Errorf(
			"expected %d dimensions, got %d", settings.Dimensions, len(vector),
		)
	}

	return agentports.QueryVector{
		Available:  true,
		Vector:     vector,
		ModelKey:   settings.ActiveModelKey,
		Dimensions: settings.Dimensions,
	}, nil
}

func pausedReason(settings *airetrieval.Settings) airetrieval.UnavailableReason {
	switch {
	case !settings.Paused:
		return ""
	case settings.PausedByBudget():
		return airetrieval.UnavailableReasonBudgetPaused
	default:
		return airetrieval.UnavailableReasonDisabled
	}
}

func QueryText(text string) string {
	return strings.TrimSpace(
		stringutils.TruncateRunes(strings.TrimSpace(text), agentports.MaxQueryTextRunes),
	)
}

func validateTenant(tenant pagination.TenantInfo) error {
	if tenant.OrgID.IsNil() || tenant.BuID.IsNil() {
		return agentports.ErrQueryTenantRequired
	}

	return nil
}
