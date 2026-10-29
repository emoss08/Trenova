package agentmatch

import (
	"cmp"
	"context"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolcatalog"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentsuggestion"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/retrievalports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	recommendationToolLimit = 8
	matchToolSummaryRunes   = 140
	matchContextBoost       = 0.25
)

type recommendationCandidate struct {
	definition *agentdefinition.Definition
	score      float64
	matched    []string
	usable     int
	current    bool
	holdsTop   bool
}

func (s *Matcher) MatchAgent(
	ctx context.Context,
	req *agentports.AgentMatchRequest,
) (*agentports.AgentMatch, error) {
	if req == nil || req.Actor == nil ||
		req.Actor.PrincipalType != permissionports.PrincipalTypeUser || s.catalog == nil {
		return noMatch(), nil
	}

	text := strings.TrimSpace(req.Text)
	if text == "" {
		return noMatch(), nil
	}
	if utf8.RuneCountInString(text) > agentports.MaxAgentRecommendationTextRunes {
		return nil, errortypes.NewValidationError(
			"text",
			errortypes.ErrInvalidLength,
			"A question can be matched to an agent when it is at most {0} characters.",
			agentports.MaxAgentRecommendationTextRunes,
		)
	}
	resources, err := matchContextResources(req.ContextKinds)
	if err != nil {
		return nil, err
	}

	pool, err := s.matchPool(ctx, req.Actor)
	if err != nil {
		return nil, err
	}
	matchable := s.matchableTools()
	if len(pool.agents) == 0 || !s.catalog.MentionsAny(matchable, text) {
		return noMatch(), nil
	}

	semantic := s.access.ToolSemantic(
		ctx,
		req.Actor.TenantInfo(),
		s.matchVector(ctx, req.Actor, text),
	)
	wide := s.boostForContext(s.catalog.Match(agenttoolcatalog.Query{
		Allowed:  matchable,
		Text:     text,
		Limit:    len(matchable),
		Semantic: semantic,
	}), resources)
	if len(wide) == 0 {
		return noMatch(), nil
	}
	if !s.catalog.NamesAny(matchable, text) && !semantic.Clears(matchable) &&
		!s.anyAbout(wide, resources) {
		return noMatch(), nil
	}
	if best := topTier(wide); !pool.permitsBest(best) {
		return &agentports.AgentMatch{
			Outcome: agentsuggestion.MatchNoAgent,
			Tools:   s.matchTools(best),
		}, nil
	}

	ranked := s.boostForContext(s.catalog.Match(agenttoolcatalog.Query{
		Allowed:  pool.permitted,
		Text:     text,
		Limit:    recommendationToolLimit,
		Semantic: semantic,
	}), resources)

	best := bestCandidate(pool.agents, pool.held, pool.permitted, ranked, req.CurrentAgentID)
	if best == nil {
		return noMatch(), nil
	}
	if best.current {
		return s.matchFor(agentsuggestion.MatchCurrentAgent, best.definition, best.matched), nil
	}
	if current, delegate, matched := pool.delegateHolding(
		req.CurrentAgentID,
		ranked,
		s.access.LooksUp,
	); current != nil {
		found := s.matchFor(agentsuggestion.MatchCurrentAgent, current, matched)
		found.Delegate = delegate

		return found, nil
	}

	return s.matchFor(agentsuggestion.MatchBetterAgent, best.definition, best.matched), nil
}

func noMatch() *agentports.AgentMatch {
	return &agentports.AgentMatch{Outcome: agentsuggestion.MatchNone}
}

func (s *Matcher) matchFor(
	outcome agentsuggestion.MatchOutcome,
	definition *agentdefinition.Definition,
	matched []string,
) *agentports.AgentMatch {
	return &agentports.AgentMatch{
		Outcome: outcome,
		Agent:   definition,
		Tools:   s.matchTools(matched),
	}
}

func (s *Matcher) matchTools(names []string) []agentports.AgentMatchTool {
	if len(names) > agentports.MaxAgentRecommendationTools {
		names = names[:agentports.MaxAgentRecommendationTools]
	}
	tools := make([]agentports.AgentMatchTool, 0, len(names))
	for _, name := range names {
		tool := agentports.AgentMatchTool{Name: name}
		if descriptor, ok := s.catalog.Descriptor(name); ok {
			tool.Summary = stringutils.Ellipsize(
				stringutils.CollapseWhitespace(stringutils.FirstSentence(descriptor.Description)),
				matchToolSummaryRunes,
			)
		}
		tools = append(tools, tool)
	}

	return tools
}

func (s *Matcher) matchVector(
	ctx context.Context,
	actor *permissionports.RequestActor,
	text string,
) retrievalports.QueryVector {
	if s.vectorizer == nil || !s.mayEmbedMatch(ctx, actor) {
		return retrievalports.QueryVector{}
	}

	return s.access.Vectorize(ctx, &retrievalports.QueryVectorRequest{
		TenantInfo: actor.TenantInfo(),
		Text:       text,
		Focus:      text,
		Attribution: aiusageports.AIUsageAttribution{
			UserID:  actor.UserID,
			Feature: aiusage.FeatureAgentRecommendation,
		},
	})
}

func (s *Matcher) matchableTools() []string {
	if s.matchable == nil {
		return agentdefinition.WithoutCoreTools(s.catalog.Names())
	}

	return s.matchable()
}

func (s *Matcher) boostForContext(
	matches []agenttoolcatalog.Match,
	resources map[permission.Resource]struct{},
) []agenttoolcatalog.Match {
	if len(resources) == 0 || len(matches) == 0 {
		return matches
	}
	boosted := false
	for idx := range matches {
		if s.isAbout(matches[idx].Name, resources) {
			matches[idx].Score *= 1 + matchContextBoost
			boosted = true
		}
	}
	if boosted {
		slices.SortStableFunc(matches, func(a, b agenttoolcatalog.Match) int {
			return cmp.Compare(b.Score, a.Score)
		})
	}

	return matches
}

func (s *Matcher) anyAbout(
	matches []agenttoolcatalog.Match,
	resources map[permission.Resource]struct{},
) bool {
	for idx := range matches {
		if s.isAbout(matches[idx].Name, resources) {
			return true
		}
	}

	return false
}

func (s *Matcher) isAbout(name string, resources map[permission.Resource]struct{}) bool {
	if len(resources) == 0 {
		return false
	}
	resource, _, ok := s.access.ToolGate(name)
	if !ok {
		return false
	}
	_, about := resources[resource]

	return about
}

func matchContextResources(kinds []string) (map[permission.Resource]struct{}, error) {
	if len(kinds) > agentports.MaxAgentMatchContextKinds {
		return nil, errortypes.NewValidationError(
			"contextKinds",
			errortypes.ErrInvalid,
			"A question can name at most {0} kinds of record.",
			agentports.MaxAgentMatchContextKinds,
		)
	}

	resources := make(map[permission.Resource]struct{}, len(kinds))
	for _, kind := range kinds {
		if agentpagecontext.IsKnownPageEntityType(kind) {
			resources[permission.RecordKind(kind).Resource()] = struct{}{}
			continue
		}
		if agentpagecontext.IsKnownPageResource(kind) {
			resources[permission.Resource(kind)] = struct{}{}
		}
	}

	return resources, nil
}

func topTier(ranked []agenttoolcatalog.Match) []string {
	if len(ranked) == 0 {
		return nil
	}
	top := ranked[0].Score
	names := make([]string, 0, len(ranked))
	for _, match := range ranked {
		if match.Score != top {
			break
		}
		names = append(names, match.Name)
	}

	return names
}

func bestCandidate(
	agents []*agentdefinition.Definition,
	held []map[string]struct{},
	permitted []string,
	ranked []agenttoolcatalog.Match,
	currentID pulid.ID,
) *recommendationCandidate {
	if len(ranked) == 0 {
		return nil
	}

	weights := rankWeights(ranked)
	top := ranked[0].Score
	candidates := make([]recommendationCandidate, 0, len(agents))
	for idx, definition := range agents {
		candidate := recommendationCandidate{
			definition: definition,
			matched:    make([]string, 0, len(ranked)),
			current:    !currentID.IsNil() && definition.ID == currentID,
		}
		for position, match := range ranked {
			if _, ok := held[idx][match.Name]; !ok {
				continue
			}
			candidate.score += weights[position]
			candidate.matched = append(candidate.matched, match.Name)
			if match.Score == top {
				candidate.holdsTop = true
			}
		}
		if !candidate.holdsTop {
			continue
		}
		if candidate.current {
			return &candidate
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

	slices.SortStableFunc(candidates, compareCandidates)

	return &candidates[0]
}

func rankWeights(ranked []agenttoolcatalog.Match) []float64 {
	weights := make([]float64, len(ranked))
	rank := 0
	for idx := range ranked {
		if idx > 0 && ranked[idx].Score != ranked[idx-1].Score {
			rank++
		}
		weights[idx] = 1 / float64(rank+1)
	}

	return weights
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
