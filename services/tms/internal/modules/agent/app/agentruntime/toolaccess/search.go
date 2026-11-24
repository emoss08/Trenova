package toolaccess

import (
	"context"
	"slices"
	"strings"
	"sync"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolcatalog"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentcapability"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	StatusLoaded         = "loaded"
	StatusAvailable      = "available"
	StatusNotPermitted   = "not_permitted"
	StatusNotInThisAgent = "not_in_this_agent"

	ConfidenceHigh = "high"
	ConfidenceLow  = "low"

	MaxSearchTokens   = 1500
	maxSearchChars    = MaxSearchTokens * 4
	searchSummaryMax  = 160
	searchHintMax     = 140
	searchLoadedLimit = 3
	searchQueryParam  = "query"
	searchMetricsMax  = 900
)

type SearchResult struct {
	Name       string `json:"name"`
	Entity     string `json:"entity,omitempty"`
	Area       string `json:"area"`
	Summary    string `json:"summary"`
	Status     string `json:"status"`
	Use        string `json:"use,omitempty"`
	Permission string `json:"permission,omitempty"`
	HandOff    string `json:"handOff,omitempty"`
}

type SearchPayload struct {
	Query      string         `json:"query"`
	Confidence string         `json:"confidence"`
	Results    []SearchResult `json:"results"`
	Rephrase   []string       `json:"rephrase,omitempty"`
	Next       string         `json:"next"`
	Metrics    string         `json:"metrics,omitempty"`
}

type SearchAnswer struct {
	Content string     `json:"content"`
	Found   []string   `json:"found,omitempty"`
	HandOff []pulid.ID `json:"handOff,omitempty"`
}

type capabilitySet struct {
	once  sync.Once
	index *agenttoolcatalog.CapabilityIndex
	err   error
}

func (s *Access) capabilities() (*agenttoolcatalog.CapabilityIndex, error) {
	s.capabilitySet.once.Do(func() {
		s.capabilitySet.index, s.capabilitySet.err =
			agenttoolcatalog.BuildCapabilityIndex(s.queryTools, s.actionTools, runtimetools.Policies())
	})

	return s.capabilitySet.index, s.capabilitySet.err
}

func (s *Access) Capabilities() (*agentcapability.Registry, error) {
	index, err := s.capabilities()
	if err != nil {
		return nil, err
	}

	return index.Registry, nil
}

func (s *Access) ToolFor(capability *agentcapability.Capability) string {
	return capability.Name
}

func checkedTool(capability *agentcapability.Capability) string {
	if !capability.IsEntityRead() {
		return capability.Name
	}
	entity, ok := agentcapability.EntityNamed(capability.Entity)
	if !ok {
		return ""
	}
	if capability.Verb == agentcapability.VerbGet {
		return entity.Get
	}

	return entity.List
}

func (s *Access) ResolveSearch(
	ctx context.Context,
	set *ToolSet,
	actor *permissionports.RequestActor,
	arguments map[string]any,
) SearchAnswer {
	query, _ := arguments[searchQueryParam].(string)
	query = strings.TrimSpace(query)
	if query == "" {
		return SearchAnswer{Content: "Say what you need in a few words: the record, what to do with it " +
			"and anything it must match. The capabilities are matched against it."}
	}
	index, err := s.capabilities()
	if err != nil {
		s.logger.Error("the capability registry could not be built", zap.Error(err))

		return SearchAnswer{Content: "The capability search is unavailable right now. Use the tools you hold."}
	}

	metrics := s.findMetrics(ctx, set, actor, query)
	answer := index.Search(query, s.capabilitySemantic(ctx, set, actor))
	payload := SearchPayload{Query: query, Confidence: ConfidenceHigh}
	if answer.LowConfidence {
		payload.Confidence = ConfidenceLow
		payload.Rephrase = answer.Rephrasings
	}

	checked := make([]string, 0, len(answer.Hits))
	for idx := range answer.Hits {
		if tool := checkedTool(&answer.Hits[idx].Capability); tool != "" {
			checked = append(checked, tool)
		}
	}
	permitted := s.PermittedWith(ctx, actor, outsideSet(set, checked), nil)

	found := make([]string, 0, len(answer.Hits))
	outOfScope := make([]string, 0, len(answer.Hits))
	loaded := 0
	for idx := range answer.Hits {
		capability := &answer.Hits[idx].Capability
		tool := s.ToolFor(capability)
		check := checkedTool(capability)
		if tool == "" || check == "" {
			continue
		}
		result := SearchResult{
			Name:    tool,
			Entity:  entityOf(capability),
			Area:    capability.Area,
			Summary: clip(capability.Summary, searchSummaryMax),
			Use:     clip(capability.UseWhen, searchHintMax),
		}
		held := slices.Contains(set.Allowed, check)
		switch {
		case s.offPlan(set, check):
			result.Status = StatusNotPermitted
			result.Permission = s.planPermission(set, check)
		case !held && !slices.Contains(permitted, check):
			result.Status = StatusNotPermitted
			result.Permission = s.permissionOf(check)
		case !held && !set.Scope.Allows(check) && answer.LowConfidence:
			continue
		case !held && !set.Scope.Allows(check):
			result.Status = StatusNotInThisAgent
			outOfScope = append(outOfScope, check)
		case set.Offers(tool):
			result.Status = StatusLoaded
			found = append(found, tool)
		default:
			result.Status = StatusAvailable
			found = append(found, tool)
			if loaded < searchLoadedLimit && s.Load(set, tool) {
				loaded++
			}
		}
		payload.Results = append(payload.Results, result)
	}

	var holders []pulid.ID
	if len(outOfScope) > 0 && !set.unattended {
		note, agents := s.handOffNote(ctx, actor, set.Self, outOfScope)
		holders = agents
		for idx := range payload.Results {
			if payload.Results[idx].Status == StatusNotInThisAgent {
				payload.Results[idx].HandOff = handOffHint(note)
			}
		}
	}
	payload.Next = nextStep(&payload, set.unattended)
	if metrics.Text != "" {
		payload.Metrics = clip(metrics.Text, searchMetricsMax)
	}

	return SearchAnswer{Content: encodeSearch(&payload), Found: metrics.Names(found), HandOff: holders}
}

func (s *Access) capabilitySemantic(
	ctx context.Context,
	set *ToolSet,
	actor *permissionports.RequestActor,
) map[string]float64 {
	semantic := s.ToolSemantic(ctx, actorTenant(actor), set.Query)
	if semantic == nil || len(semantic.Similarity) == 0 {
		return nil
	}
	out := make(map[string]float64, len(semantic.Similarity))
	for name, similarity := range semantic.Similarity {
		if similarity < semantic.Floor {
			continue
		}
		out[agentcapability.CapabilityIDForTool(name)] = max(out[agentcapability.CapabilityIDForTool(name)], similarity)
	}

	return out
}

func (s *Access) permissionOf(tool string) string {
	resource, operation, ok := s.ToolGate(tool)
	if !ok {
		return ""
	}

	return resource.String() + ":" + string(operation)
}

func (s *Access) offPlan(set *ToolSet, tool string) bool {
	_, off := s.planDenial(set, tool)
	return off
}

func (s *Access) planPermission(set *ToolSet, tool string) string {
	denial, off := s.planDenial(set, tool)
	if !off || denial == nil {
		return ""
	}
	if denial.AddOnName != "" {
		return "the " + denial.AddOnName + " add-on"
	}

	return string(denial.Feature)
}

func outsideSet(set *ToolSet, tools []string) []string {
	out := make([]string, 0, len(tools))
	for _, tool := range tools {
		if set.Offers(tool) || slices.Contains(set.Allowed, tool) || slices.Contains(out, tool) {
			continue
		}
		out = append(out, tool)
	}

	return out
}

func entityOf(capability *agentcapability.Capability) string {
	if capability.IsEntityRead() {
		return capability.Entity
	}

	return ""
}

func handOffHint(note string) string {
	if note == "" {
		return "Offer to hand the conversation to an agent whose scope includes it; do not do the work here."
	}

	return clip(note, 2*searchHintMax)
}

func nextStep(payload *SearchPayload, unattended bool) string {
	statuses := make(map[string]bool, 4)
	for idx := range payload.Results {
		statuses[payload.Results[idx].Status] = true
	}
	if payload.Confidence == ConfidenceLow && !statuses[StatusLoaded] && !statuses[StatusAvailable] {
		return "Low confidence: these only loosely match. Search once more with the record, the " +
			"action and the fields, then report the gap naming both searches."
	}
	switch {
	case statuses[StatusLoaded] || statuses[StatusAvailable]:
		return "Call the loaded or available capability by name; an available one loads when called."
	case statuses[StatusNotInThisAgent] && unattended:
		return "It exists but is outside this agent's scope. Raise it for a person rather than working around it."
	case statuses[StatusNotInThisAgent]:
		return "It exists but is outside this agent's scope. Offer a hand-off; do not improvise."
	case statuses[StatusNotPermitted]:
		return "It exists but the person lacks the permission named. Say so; do not claim it is missing."
	case payload.Confidence == ConfidenceLow:
		return "Low confidence. Search once more with the record, the action and the fields, then report the gap naming both searches."
	}

	return "Nothing matched. Search once more with the record, the action and the fields before saying it cannot be done."
}

func clip(text string, limit int) string {
	text = strings.TrimSpace(text)
	if len(text) <= limit {
		return text
	}
	cut := strings.LastIndex(text[:limit], " ")
	if cut < limit/2 {
		cut = limit
	}

	return strings.TrimRight(text[:cut], " ,;:") + "…"
}

func encodeSearch(payload *SearchPayload) string {
	for {
		encoded, err := sonic.MarshalString(payload)
		if err != nil {
			return payload.Next
		}
		if len(encoded) <= maxSearchChars || len(payload.Results) <= 1 {
			return encoded
		}
		trimmed := false
		for idx := range payload.Results {
			if payload.Results[idx].Use != "" {
				payload.Results[idx].Use = ""
				trimmed = true
			}
		}
		if !trimmed {
			payload.Results = payload.Results[:len(payload.Results)-1]
		}
	}
}
