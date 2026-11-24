package agentevalgate_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentevalgate"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentcapability"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/stretchr/testify/require"
)

type capabilityResults struct {
	Baseline *agentevalgate.CapabilityMetrics `json:"baseline,omitempty"`
	Current  *agentevalgate.CapabilityMetrics `json:"current,omitempty"`
}

func readCapabilityResults(t *testing.T) capabilityResults {
	t.Helper()

	var results capabilityResults
	raw, err := os.ReadFile(agentevalgate.CapabilityResultsPath)
	if os.IsNotExist(err) {
		return results
	}
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &results))

	return results
}

func writeCapabilityResults(t *testing.T, results capabilityResults) {
	t.Helper()

	encoded, err := json.MarshalIndent(results, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(agentevalgate.CapabilityResultsPath, append(encoded, '\n'), 0o600))
}

func runCapabilityTurn(
	t *testing.T,
	kit *agentevalgate.Kit,
	permissions *agenttest.StubPermissions,
	definition *agentdefinition.Definition,
	searchTool string,
	searchArg string,
	request string,
) (agentevalgate.TurnObservation, string) {
	t.Helper()

	completion := &agenttest.ScriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.ToolTurn(searchTool, map[string]any{searchArg: request}),
		runtimetest.TextTurn("Done."),
	}}
	rt := kit.NewRuntime(agentevalgate.RuntimeParams{Completion: completion, Permissions: permissions})
	_, err := rt.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: definition,
		Actor:      runtimetest.Actor(),
		Input:      request,
	})
	require.NoError(t, err)

	observation := agentevalgate.TurnObservation{}
	searchAnswer := ""
	for idx, req := range completion.Requests {
		tokens := 0
		for _, spec := range req.Tools {
			if spec.Deferred {
				continue
			}
			tokens += agentevalgate.SchemaTokens(spec.Name, spec.Description, spec.Parameters)
			if idx == 0 {
				observation.Opened = append(observation.Opened, agentcapability.CapabilityIDForTool(spec.Name))
			}
		}
		observation.SchemaTokens = append(observation.SchemaTokens, tokens)
		for _, message := range req.Messages {
			if message.Role == llmports.RoleTool && message.ToolName == searchTool {
				searchAnswer = message.Content
			}
		}
	}

	return observation, searchAnswer
}

func deniedPermissions(c *agentevalgate.CapabilityCase) *agenttest.StubPermissions {
	denied := make(map[string]bool, len(c.Denied))
	for _, permission := range c.Denied {
		denied[permission] = true
	}

	return &agenttest.StubPermissions{Denied: denied}
}

func capabilityCandidates(names []string) []agentevalgate.Candidate {
	out := make([]agentevalgate.Candidate, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		id := agentcapability.CapabilityIDForTool(name)
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, agentevalgate.Candidate{ID: id, Status: agentevalgate.StatusAvailable})
	}

	return out
}

func reportCapabilities(
	t *testing.T,
	report *agentevalgate.CapabilityReport,
	baseline *agentevalgate.CapabilityMetrics,
) {
	t.Helper()

	rows := []agentevalgate.CapabilityMetrics{report.Metrics}
	if baseline != nil && baseline.System != report.Metrics.System {
		rows = append([]agentevalgate.CapabilityMetrics{*baseline}, rows...)
	}
	t.Log("\n" + agentevalgate.MetricsTable(rows...))
	misses := report.Misses()
	if len(misses) > 0 {
		t.Logf("%d misses:\n%s", len(misses), strings.Join(misses, "\n"))
	}
}

func capabilityHarnessEnabled(t *testing.T) {
	t.Helper()

	if os.Getenv("TRENOVA_CAPABILITY_HARNESS") == "" && !*update {
		t.Skip("set TRENOVA_CAPABILITY_HARNESS=1 to run the capability harness")
	}
}
