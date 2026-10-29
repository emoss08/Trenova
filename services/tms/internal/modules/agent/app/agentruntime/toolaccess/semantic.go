package toolaccess

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolcatalog"
	"github.com/emoss08/trenova/internal/modules/agent/domain/airetrieval"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/retrievalports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/vectorutils"
	"go.uber.org/zap"
)

type FindAnswer struct {
	Content string     `json:"content"`
	Found   []string   `json:"found,omitempty"`
	HandOff []pulid.ID `json:"handOff,omitempty"`
}

type FoundTools struct {
	Content string
	Loaded  []string
	Found   []string
	HandOff []pulid.ID
}

type QueryVectorState struct {
	ModelKey   string `json:"modelKey"`
	Dimensions int    `json:"dimensions"`
	Vector     []byte `json:"vector"`
}

func packQueryVector(query retrievalports.QueryVector) *QueryVectorState {
	if !query.Usable() {
		return nil
	}

	return &QueryVectorState{
		ModelKey:   query.ModelKey,
		Dimensions: query.Dimensions,
		Vector:     vectorutils.Pack(query.Vector),
	}
}

func (q *QueryVectorState) unpack() retrievalports.QueryVector {
	if q == nil || q.ModelKey == "" || !airetrieval.IsAllowedDimension(q.Dimensions) {
		return retrievalports.QueryVector{}
	}

	vector, err := vectorutils.Unpack(q.Vector)
	if err != nil || len(vector) != q.Dimensions {
		return retrievalports.QueryVector{}
	}

	return retrievalports.QueryVector{
		Available:  true,
		Vector:     vector,
		ModelKey:   q.ModelKey,
		Dimensions: q.Dimensions,
	}
}

func (s *Access) TurnQueryVector(
	ctx context.Context,
	req *agentruntimeports.RunRequest,
) retrievalports.QueryVector {
	query := agentruntimeports.TurnQueryRequest(req)

	return s.Vectorize(ctx, &query)
}

func (s *Access) Vectorize(
	ctx context.Context,
	req *retrievalports.QueryVectorRequest,
) retrievalports.QueryVector {
	if s.vectorizer == nil || req == nil || strings.TrimSpace(req.Text) == "" ||
		req.TenantInfo.OrgID.IsNil() || req.TenantInfo.BuID.IsNil() {
		return retrievalports.QueryVector{}
	}

	query, err := s.vectorizer.Vectorize(ctx, req)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return retrievalports.QueryVector{}
		}
		s.logger.Warn("could not embed the turn's request; ranking tools by keyword",
			zap.String("organization", req.TenantInfo.OrgID.String()),
			zap.Error(err),
		)

		return retrievalports.QueryVector{}
	}

	return query
}

func (s *Access) ToolSemantic(
	ctx context.Context,
	tenant pagination.TenantInfo,
	query retrievalports.QueryVector,
) *agenttoolcatalog.Semantic {
	if s.vectors == nil || s.catalog == nil || !query.Usable() ||
		tenant.OrgID.IsNil() || tenant.BuID.IsNil() {
		return nil
	}

	similarities, err := s.vectors.Similarities(ctx, &retrievalports.CatalogSimilarityRequest{
		TenantInfo: tenant,
		Corpus:     airetrieval.CatalogCorpusTools,
		Items:      s.catalog.Items(),
		Query:      query,
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil
		}
		s.logger.Warn("could not compare the tool catalog with the request; ranking by keyword",
			zap.String("organization", tenant.OrgID.String()),
			zap.Error(err),
		)

		return nil
	}
	if !similarities.Available {
		return nil
	}

	return agenttoolcatalog.SemanticFrom(similarities)
}

func foundByCall(history []conversation.Message) map[string][]string {
	found := make(map[string][]string, 4)
	for idx := range history {
		message := &history[idx]
		if message.Role != conversation.RoleTool ||
			message.ToolName != runtimetools.FindToolsName ||
			message.ToolCallID == "" ||
			len(message.FoundTools) == 0 {
			continue
		}
		found[message.ToolCallID] = message.FoundTools
	}

	return found
}

func actorTenant(actor *permissionports.RequestActor) pagination.TenantInfo {
	if actor == nil {
		return pagination.TenantInfo{}
	}

	return actor.TenantInfo()
}
