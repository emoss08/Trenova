package agentevalgate_test

import (
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentevalgate"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentcapability"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreset"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttemplate"
	"github.com/stretchr/testify/require"
)

var baselineDesks = map[string]agenttemplate.Template{
	agentevalgate.AgentUniversal: agenttemplate.TemplateDispatchAssistant,
	"dispatch":                   agenttemplate.TemplateDispatchAssistant,
	"billing":                    agenttemplate.TemplateBillingAssistant,
	"safety":                     agenttemplate.TemplateComplianceAssistant,
}

func baselineReach() []string {
	var reach []string
	for _, template := range agenttemplate.AllTemplates() {
		for _, name := range agentpreset.StarterTools(template) {
			if !slices.Contains(reach, name) {
				reach = append(reach, name)
			}
		}
	}

	return reach
}

func TestCapabilityHarnessBaseline(t *testing.T) {
	capabilityHarnessEnabled(t)

	kit := newKit(t)
	suite, err := agentevalgate.LoadCapabilitySuite(agentevalgate.CapabilitySuitePath)
	require.NoError(t, err)
	reach := baselineReach()
	turns := make(map[*agentevalgate.CapabilityCase]string, len(suite.Cases))

	permitted := func(c *agentevalgate.CapabilityCase, names []string) []string {
		out := make([]string, 0, len(names))
		for _, name := range names {
			policy := ""
			if tool, ok := kit.Queries.Get(name); ok {
				policy = string(tool.Policy().Resource) + ":read"
			}
			if tool, ok := kit.Actions.Get(name); ok {
				policy = string(tool.Policy().Resource) + ":" + string(tool.Policy().Operation)
			}
			if slices.Contains(c.Denied, policy) {
				continue
			}
			out = append(out, name)
		}

		return out
	}

	system := &agentevalgate.CapabilitySystem{
		Name: "Before (find_tools, 64-tool desks)",
		Turn: func(c *agentevalgate.CapabilityCase) (agentevalgate.TurnObservation, error) {
			tools := agentpreset.StarterTools(baselineDesks[c.AgentName()])
			observation, answer := runCapabilityTurn(t, kit, deniedPermissions(c), tools,
				"find_tools", "need", c.Request)
			turns[c] = answer

			return observation, nil
		},
		Search: func(c *agentevalgate.CapabilityCase) (agentevalgate.SearchAnswer, error) {
			allowed := permitted(c, reach)
			found := kit.Catalog.Find(allowed, c.Request, 6)
			names := make([]string, 0, len(found))
			for idx := range found {
				names = append(names, found[idx].Name)
			}

			return agentevalgate.SearchAnswer{
				Candidates: capabilityCandidates(names),
				Tokens:     agentevalgate.TextTokens(turns[c]),
			}, nil
		},
		Read: baselineReadPlan,
	}

	report, err := agentevalgate.EvaluateCapabilities(suite, system)
	require.NoError(t, err)
	results := readCapabilityResults(t)
	reportCapabilities(t, report, nil)
	if *update {
		results.Baseline = &report.Metrics
		writeCapabilityResults(t, results)
	}
}

func baselineReadPlan(c *agentevalgate.CapabilityCase) agentevalgate.ReadPlan {
	if len(c.Chain) > 0 {
		return agentevalgate.ReadPlan{Calls: len(c.Chain)}
	}
	entity, verb, ok := agentcapability.ParseCapabilityID(c.Expect)
	if !ok {
		return agentevalgate.ReadPlan{Calls: 1}
	}
	described, _ := agentcapability.EntityNamed(entity)
	calls := 1
	summary := false
	for _, name := range c.Includes {
		include, found := described.Include(name)
		switch {
		case !found:
		case include.Reader != "":
			calls++
		case include.FromSummary:
			summary = true
		}
	}
	if summary && verb == agentcapability.VerbList {
		calls++
	}

	return agentevalgate.ReadPlan{Calls: calls}
}
