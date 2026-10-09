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
	req *serviceports.SimilarMemoriesRequest,
) (serviceports.SimilarMemories, error) {
	result := serviceports.SimilarMemories{
		Memories: []serviceports.MemorySimilarity{},
		Floor:    s.tuning.SimilarityFloor,
	}
	if req == nil || req.TenantInfo.OrgID.IsNil() || req.TenantInfo.BuID.IsNil() {
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

	hits, err := s.repo.Search(ctx, &repositories.VectorSearchRequest{
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
	req *serviceports.SimilarMemoriesRequest,
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

	query, err := s.vectorizer.Vectorize(ctx, &serviceports.QueryVectorRequest{
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
	req *serviceports.RankMemoriesRequest,
) (serviceports.RankedMemories, error) {
	if req == nil {
		return serviceports.RankedMemories{Memories: []*agent.Memory{}}, nil
	}

	recency := agentmemoryservice.RankByRecencyAndUse(req.Memories, req.Now)
	ranked := serviceports.RankedMemories{Memories: recency}
	if !req.Query.Usable() || len(recency) == 0 {
		return ranked, nil
	}

	similar, err := r.searcher.SimilarMemories(ctx, &serviceports.SimilarMemoriesRequest{
		TenantInfo: req.TenantInfo,
		Query:      req.Query,
		Limit:      MaxSimilarMemories,
	})
	if err != nil {
		return ranked, err
	}
	if !similar.Semantics.Used {
		return ranked, nil
	}

	ranked.Semantic = true
	ranked.Similar = similarCandidates(recency, similar.Memories, r.tuning)
	ranked.Memories = fuseBySimilarity(recency, ranked.Similar, r.tuning.normalized().RRFK)

	return ranked, nil
}

func similarCandidates(
	candidates []*agent.Memory,
	similar []serviceports.MemorySimilarity,
	tuning SearchTuning,
) []pulid.ID {
	tuning = tuning.normalized()

	known := make(map[pulid.ID]struct{}, len(candidates))
	for _, memory := range candidates {
		known[memory.ID] = struct{}{}
	}

	ids := make([]pulid.ID, 0, min(len(similar), len(candidates)))
	for _, hit := range similar {
		if _, ok := known[hit.MemoryID]; ok && hit.Similarity >= tuning.SimilarityFloor {
			ids = append(ids, hit.MemoryID)
		}
	}

	return ids
}

func FuseMemoryRanking(
	recency []*agent.Memory,
	similar []serviceports.MemorySimilarity,
	tuning SearchTuning,
) []*agent.Memory {
	tuning = tuning.normalized()

	return fuseBySimilarity(recency, similarCandidates(recency, similar, tuning), tuning.RRFK)
}

func fuseBySimilarity(recency []*agent.Memory, similarOrder []pulid.ID, rrfk int) []*agent.Memory {
	if len(similarOrder) == 0 {
		return recency
	}

	recencyOrder := make([]pulid.ID, 0, len(recency))
	for _, memory := range recency {
		recencyOrder = append(recencyOrder, memory.ID)
	}

	scores := rankfusion.Reciprocal(rrfk, similarOrder, recencyOrder)
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
