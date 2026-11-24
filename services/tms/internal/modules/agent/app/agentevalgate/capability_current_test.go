package agentevalgate_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentevalgate"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolaccess"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentcapability"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreset"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttemplate"
	"github.com/stretchr/testify/require"
)

func harnessAgent(c *agentevalgate.CapabilityCase) *agentdefinition.Definition {
	template := harnessDesks[c.AgentName()]
	definition := runtimetest.Definition()
	definition.Template = template
	definition.Scope = agentpreset.ScopeFor(template)
	definition.PinnedTools = agentpreset.PinnedFor(template)

	return definition
}

var harnessDesks = map[string]agenttemplate.Template{
	agentevalgate.AgentUniversal: agenttemplate.TemplateUniversal,
	"dispatch":                   agenttemplate.TemplateDispatchAssistant,
	"billing":                    agenttemplate.TemplateBillingAssistant,
	"safety":                     agenttemplate.TemplateComplianceAssistant,
}

func TestCapabilityHarness(t *testing.T) {
	capabilityHarnessEnabled(t)

	kit := newKit(t)
	suite, err := agentevalgate.LoadCapabilitySuite(agentevalgate.CapabilitySuitePath)
	require.NoError(t, err)
	answers := make(map[*agentevalgate.CapabilityCase]string, len(suite.Cases))

	system := &agentevalgate.CapabilitySystem{
		Name: "After (search_tools, scoped agents)",
		Turn: func(c *agentevalgate.CapabilityCase) (agentevalgate.TurnObservation, error) {
			observation, answer := runCapabilityTurn(t, kit, deniedPermissions(c), harnessAgent(c),
				"search_tools", "query", c.Request)
			answers[c] = answer

			return observation, nil
		},
		Search: func(c *agentevalgate.CapabilityCase) (agentevalgate.SearchAnswer, error) {
			return parseSearch(answers[c]), nil
		},
		Read: currentReadPlan(kit),
	}

	report, err := agentevalgate.EvaluateCapabilities(suite, system)
	require.NoError(t, err)
	results := readCapabilityResults(t)
	reportCapabilities(t, report, results.Baseline)
	if *update {
		results.Current = &report.Metrics
		writeCapabilityResults(t, results)
	}
}

func parseSearch(content string) agentevalgate.SearchAnswer {
	answer := agentevalgate.SearchAnswer{Tokens: agentevalgate.TextTokens(content)}
	var payload toolaccess.SearchPayload
	if err := sonic.UnmarshalString(content, &payload); err != nil {
		return answer
	}
	answer.LowConfidence = payload.Confidence == toolaccess.ConfidenceLow
	for _, result := range payload.Results {
		id := agentcapability.CapabilityIDForTool(result.Name)
		switch {
		case result.Entity != "" && result.Name == agentcapability.RecordGet:
			id = agentcapability.EntityCapabilityID(result.Entity, agentcapability.VerbGet)
		case result.Entity != "":
			id = agentcapability.EntityCapabilityID(result.Entity, agentcapability.VerbList)
		}
		answer.Candidates = append(answer.Candidates, agentevalgate.Candidate{ID: id, Status: result.Status})
	}

	return answer
}

func currentReadPlan(kit *agentevalgate.Kit) func(c *agentevalgate.CapabilityCase) agentevalgate.ReadPlan {
	_, consolidated := kit.Queries.Get(agentcapability.RecordsQuery)

	return func(c *agentevalgate.CapabilityCase) agentevalgate.ReadPlan {
		if len(c.Chain) > 0 {
			return agentevalgate.ReadPlan{Calls: len(c.Chain)}
		}
		entity, verb, ok := agentcapability.ParseCapabilityID(c.Expect)
		if !ok || consolidated {
			return agentevalgate.ReadPlan{Calls: 1}
		}
		described, _ := agentcapability.EntityNamed(entity)
		calls, summary := 1, false
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
}
