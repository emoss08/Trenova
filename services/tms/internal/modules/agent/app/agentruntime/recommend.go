package agentruntime

import (
	"cmp"
	"context"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolcatalog"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

const recommendationToolLimit = 8

var _ agentports.AgentRecommender = (*Service)(nil)

type recommendationCandidate struct {
	definition *agentdefinition.Definition
	score      float64
	matched    []string
	usable     int
	current    bool
}

func (s *Service) RecommendAgent(
	ctx context.Context,
	req *agentports.AgentRecommendationRequest,
) (*agentports.AgentRecommendation, error) {
	if req == nil || req.Actor == nil ||
		req.Actor.PrincipalType != permissionports.PrincipalTypeUser || s.catalog == nil {
		return nil, nil
	}

	text := strings.TrimSpace(req.Text)
	if text == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(text) > agentports.MaxAgentRecommendationTextRunes {
		return nil, errortypes.NewValidationError(
			"text",
			errortypes.ErrInvalidLength,
			"A question can be matched to an agent when it is at most {0} characters.",
			agentports.MaxAgentRecommendationTextRunes,
		)
	}

	agents, err := OtherUsableAgents(ctx, &OtherUsableAgentsRequest{
		Permissions: s.permissions,
		Definitions: s.definitions,
		Actor:       req.Actor,
	})
	if err != nil {
		return nil, err
	}
	if len(agents) == 0 {
		return nil, nil
	}

	held, union := s.recommendableTools(agents)
	permitted := s.permittedWith(ctx, req.Actor, union, nil)
	if len(permitted) == 0 {
		return nil, nil
	}

	ranked := s.rankForRecommendation(ctx, req, text, permitted)
	if len(ranked) == 0 {
		return nil, nil
	}

	best := bestCandidate(agents, held, permitted, ranked, req.CurrentAgentID)
	if best == nil {
		return nil, nil
	}

	tools := best.matched
	if len(tools) > agentports.MaxAgentRecommendationTools {
		tools = tools[:agentports.MaxAgentRecommendationTools]
	}

	return &agentports.AgentRecommendation{
		Agent:   best.definition,
		Current: best.current,
		Tools:   tools,
	}, nil
}

func (s *Service) recommendableTools(
	agents []*agentdefinition.Definition,
) (held []map[string]struct{}, union []string) {
	held = make([]map[string]struct{}, len(agents))
	seen := make(map[string]struct{}, agentdefinition.MaxTools)
	union = make([]string, 0, agentdefinition.MaxTools)
	for idx, definition := range agents {
		names := agentdefinition.WithoutCoreTools(s.heldTools(definition))
		held[idx] = make(map[string]struct{}, len(names))
		for _, name := range names {
			held[idx][name] = struct{}{}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			union = append(union, name)
		}
	}

	return held, union
}

func (s *Service) rankForRecommendation(
	ctx context.Context,
	req *agentports.AgentRecommendationRequest,
	text string,
	permitted []string,
) []string {
	tenant := req.Actor.TenantInfo()
	query := s.vectorize(ctx, &agentports.QueryVectorRequest{
		TenantInfo: tenant,
		Text:       text,
		Focus:      text,
		Attribution: aiusageports.AIUsageAttribution{
			UserID:  req.Actor.UserID,
			Feature: aiusage.FeatureAgentTurn,
		},
	})

	return s.catalog.Match(agenttoolcatalog.Query{
		Allowed:  permitted,
		Text:     text,
		Limit:    recommendationToolLimit,
		Semantic: s.toolSemantic(ctx, tenant, query),
	})
}

func bestCandidate(
	agents []*agentdefinition.Definition,
	held []map[string]struct{},
	permitted []string,
	ranked []string,
	currentID pulid.ID,
) *recommendationCandidate {
	candidates := make([]recommendationCandidate, 0, len(agents))
	for idx, definition := range agents {
		candidate := recommendationCandidate{
			definition: definition,
			matched:    make([]string, 0, len(ranked)),
			current:    !currentID.IsNil() && definition.ID == currentID,
		}
		for position, name := range ranked {
			if _, ok := held[idx][name]; !ok {
				continue
			}
			candidate.score += 1 / float64(position+1)
			candidate.matched = append(candidate.matched, name)
		}
		if candidate.score == 0 {
			continue
		}
		for _, name := range permitted {
			if _, ok := held[idx][name]; ok {
				candidate.usable++
			}
		}
		candidates = append(candidates, candidate)
	}
	if len(candidates) == 0 {
		return nil
	}

	best := ranked[0]
	holders := make([]recommendationCandidate, 0, len(candidates))
	for idx := range candidates {
		if !slices.Contains(candidates[idx].matched, best) {
			continue
		}
		if candidates[idx].current {
			return &candidates[idx]
		}
		holders = append(holders, candidates[idx])
	}
	if len(holders) == 0 {
		return nil
	}

	slices.SortStableFunc(holders, compareCandidates)

	return &holders[0]
}

func compareCandidates(a, b recommendationCandidate) int {
	if a.score != b.score {
		return cmp.Compare(b.score, a.score)
	}
	if a.usable != b.usable {
		return cmp.Compare(a.usable, b.usable)
	}
	if byName := strings.Compare(
		strings.ToLower(a.definition.Name),
		strings.ToLower(b.definition.Name),
	); byName != 0 {
		return byName
	}

	return strings.Compare(a.definition.ID.String(), b.definition.ID.String())
}
