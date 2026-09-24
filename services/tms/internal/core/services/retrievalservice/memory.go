package retrievalservice

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentmemoryservice"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/rankfusion"
	"go.uber.org/zap"
)

const (
	DefaultSimilarMemories = 50
	MaxSimilarMemories     = 200
)

var _ serviceports.MemoryVectorSearcher = (*Searcher)(nil)

func AsMemoryVectorSearcher(s *Searcher) serviceports.MemoryVectorSearcher { return s }

func (s *Searcher) SimilarMemories(
	ctx context.Context,
	req serviceports.SimilarMemoriesRequest,
) (serviceports.SimilarMemories, error) {
	result := serviceports.SimilarMemories{
		Memories: []serviceports.MemorySimilarity{},
		Floor:    s.tuning.SimilarityFloor,
	}
	if req.TenantInfo.OrgID.IsNil() || req.TenantInfo.BuID.IsNil() {
		return result, errors.New("similar memories need an organization and a business unit")
	}

	settings, err := s.repo.GetSettings(ctx, req.TenantInfo)
	if err != nil {
		s.l.Warn("retrieval settings could not be read; recalling memories by words only",
			zap.String("organizationId", req.TenantInfo.OrgID.String()),
			zap.Error(err))
		result.Semantics.Reason = airetrieval.UnavailableReasonProviderFailed
		return result, nil
	}
	if !settings.SourceEnabled(airetrieval.SourceTypeMemory) {
		result.Semantics.Reason = airetrieval.UnavailableReasonDisabled
		return result, nil
	}

	query, reason := s.memoryQuery(ctx, req)
	if reason != "" {
		result.Semantics.Reason = reason
		return result, nil
	}

	limit := req.Limit
	if limit <= 0 {
		limit = DefaultSimilarMemories
	}

	hits, err := s.repo.Search(ctx, repositories.VectorSearchRequest{
		TenantInfo:  req.TenantInfo,
		SourceTypes: []airetrieval.SourceType{airetrieval.SourceTypeMemory},
		ModelKey:    query.ModelKey,
		Dimensions:  query.Dimensions,
		Query:       query.Vector,
		Limit:       min(limit, MaxSimilarMemories),
	})
	if err != nil {
		var unavailable *airetrieval.UnavailableError
		if errors.As(err, &unavailable) {
			result.Semantics.Reason = unavailable.Reason
			return result, nil
		}
		s.l.Warn("memory vector search failed; recalling by words only",
			zap.String("organizationId", req.TenantInfo.OrgID.String()),
			zap.Error(err))
		result.Semantics.Reason = airetrieval.UnavailableReasonProviderFailed
		return result, nil
	}

	for _, hit := range hits {
		result.Memories = append(result.Memories, serviceports.MemorySimilarity{
			MemoryID:   hit.SourceID,
			Similarity: hit.Similarity,
		})
	}
	result.Semantics.Used = true

	return result, nil
}

func (s *Searcher) memoryQuery(
	ctx context.Context,
	req serviceports.SimilarMemoriesRequest,
) (serviceports.QueryVector, airetrieval.UnavailableReason) {
	if req.Query.Usable() {
		return req.Query, ""
	}
	if s.vectorizer == nil {
		return serviceports.QueryVector{}, airetrieval.UnavailableReasonNoProvider
	}
	if strings.TrimSpace(req.Text) == "" {
		reason := req.Query.Reason
		if reason == "" {
			reason = airetrieval.UnavailableReasonNotIndexed
		}
		return serviceports.QueryVector{}, reason
	}

	query, err := s.vectorizer.Vectorize(ctx, serviceports.QueryVectorRequest{
		TenantInfo:  req.TenantInfo,
		Text:        req.Text,
		Attribution: req.Attribution,
	})
	if err != nil {
		s.l.Warn("memory query could not be embedded; recalling by words only",
			zap.String("organizationId", req.TenantInfo.OrgID.String()),
			zap.Error(err))
		return serviceports.QueryVector{}, airetrieval.UnavailableReasonProviderFailed
	}
	if !query.Usable() {
		reason := query.Reason
		if reason == "" {
			reason = airetrieval.UnavailableReasonNotIndexed
		}
		return serviceports.QueryVector{}, reason
	}

	return query, ""
}

type MemoryRanker struct {
	searcher serviceports.MemoryVectorSearcher
	tuning   SearchTuning
	l        *zap.Logger
}

func NewMemoryRanker(searcher *Searcher) serviceports.MemoryRanker {
	return &MemoryRanker{searcher: searcher, tuning: searcher.tuning, l: searcher.l}
}

func NewMemoryRankerFrom(
	searcher serviceports.MemoryVectorSearcher,
	tuning SearchTuning,
	logger *zap.Logger,
) *MemoryRanker {
	return &MemoryRanker{searcher: searcher, tuning: tuning.normalized(), l: logger}
}

func (r *MemoryRanker) RankMemories(
	ctx context.Context,
	req serviceports.RankMemoriesRequest,
) ([]*agent.Memory, error) {
	recency := agentmemoryservice.RankByRecencyAndUse(req.Memories, req.Now)
	if !req.Query.Usable() || len(recency) < 2 {
		return recency, nil
	}

	similar, err := r.searcher.SimilarMemories(ctx, serviceports.SimilarMemoriesRequest{
		TenantInfo: req.TenantInfo,
		Query:      req.Query,
		Limit:      MaxSimilarMemories,
	})
	if err != nil {
		return recency, err
	}
	if !similar.Semantics.Used {
		return recency, nil
	}

	return FuseMemoryRanking(recency, similar.Memories, r.tuning), nil
}

func FuseMemoryRanking(
	recency []*agent.Memory,
	similar []serviceports.MemorySimilarity,
	tuning SearchTuning,
) []*agent.Memory {
	tuning = tuning.normalized()

	candidates := make(map[pulid.ID]struct{}, len(recency))
	recencyOrder := make([]pulid.ID, 0, len(recency))
	for _, memory := range recency {
		candidates[memory.ID] = struct{}{}
		recencyOrder = append(recencyOrder, memory.ID)
	}

	similarOrder := make([]pulid.ID, 0, len(similar))
	for _, hit := range similar {
		if _, ok := candidates[hit.MemoryID]; ok && hit.Similarity >= tuning.SimilarityFloor {
			similarOrder = append(similarOrder, hit.MemoryID)
		}
	}
	if len(similarOrder) == 0 {
		return recency
	}

	scores := rankfusion.Reciprocal(tuning.RRFK, similarOrder, recencyOrder)
	ranked := slices.Clone(recency)
	slices.SortStableFunc(ranked, func(a, b *agent.Memory) int {
		switch left, right := scores[a.ID], scores[b.ID]; {
		case left > right:
			return -1
		case left < right:
			return 1
		default:
			return 0
		}
	})

	return ranked
}
