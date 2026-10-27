package agentredteam

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentevalgate"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/scriptrunner"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/reflectutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const openLoadsScript = `
from collections import defaultdict
counts = defaultdict(int)
earliest = {}
cursor = None
while True:
    args = {"status": ["New", "Assigned", "InTransit"], "limit": 50}
    if cursor:
        args["cursor"] = cursor
    page = trenova.shipment.list(**args)
    for s in page["items"]:
        counts[s["customer"]] += 1
        pickup = s["pickup"].split(" · ")[1].split(" to ")[0]
        if s["customer"] not in earliest or pickup < earliest[s["customer"]]:
            earliest[s["customer"]] = pickup
    if not page.get("hasMore"):
        break
    cursor = page["nextCursor"]
for customer in sorted(counts, key=lambda c: -counts[c]):
    print(customer, counts[customer], earliest[customer])
trenova.result({"customers": len(counts)})
`

/*
Every runtime test of code mode stood a stub in for the script tool, so nothing
held the real tool, runner and interpreter together. When moving scripts off
E2B dropped the grant that gave agents analysis_run, no test failed and no
agent was offered a script. This one runs a whole turn through the real
registry, script tool, runner and Monty: an agent configured with list tools
alone is offered the script, and the script reads, groups and counts.
*/
func TestAnAgentRunsARealScriptThroughTheWholeTurn(t *testing.T) {
	t.Parallel()

	rec := &recorder{}
	responses := map[string]map[string]any{
		"shipment_list": {
			"count": 3, "hasMore": false,
			"items": []any{
				map[string]any{"customer": "Acme", "pickup": "Dock A, Dallas TX · 2026-10-16T07:00 to 2026-10-16T14:00"},
				map[string]any{"customer": "Acme", "pickup": "Dock B, Austin TX · 2026-10-14T07:00 to 2026-10-14T14:00"},
				map[string]any{"customer": "Bolt", "pickup": "Dock C, Reno NV · 2026-10-19T08:00 to 2026-10-19T12:00"},
			},
		},
	}
	completion := &agenttest.ScriptedCompletion{Turns: []*agentports.ChatCompletionResult{
		{ModelIdentifier: scriptModel, ToolCalls: []agentports.ToolCall{{
			ID:   "call_script",
			Name: codemode.ToolRunScript,
			Arguments: map[string]any{
				"title":   "Open loads by customer",
				"purpose": "Count each customer's open loads and their earliest pickup.",
				"code":    openLoadsScript,
			},
		}}},
		{ModelIdentifier: scriptModel, Text: "Acme has two open loads, Bolt one."},
	}}
	rt := realScriptRuntime(t, rec, responses, completion)

	spec := &AgentSpec{Name: "Billing exceptions", Tools: []string{"shipment_list", "customer_list"}}
	tenant := Tenant{OrganizationID: OrganizationID, BusinessUnitID: BusinessUnitID, UserID: UserID}
	primary := definitionFor(spec, MainAgentID, tenant)
	definitions := map[pulid.ID]*agentdefinition.Definition{primary.ID: primary}
	c := &Case{Agent: *spec}
	req := &agentports.RunRequest{
		Definition: primary,
		Actor:      actorFor(false, tenant),
		Context:    contextFor(c, definitions, tenant),
		Input:      "For every customer with open loads, the count and the earliest pickup.",
		ThreadID:   ThreadID,
	}

	turn := rt.OpenTurn(t.Context(), req)
	_, err := rt.Drive(turn, &effects{
		ctx:         context.Background(),
		rt:          rt,
		rec:         rec,
		definitions: definitions,
	})
	require.NoError(t, err)

	require.Len(t, completion.Requests, 2)
	assert.Contains(t, offeredTools(completion.Requests[0]), codemode.ToolRunScript,
		"the agent was configured with list tools alone")
	answer := scriptAnswer(t, completion.Requests[1])
	assert.Equal(t, "Succeeded", answer.Status)
	assert.Equal(t, "Acme 2 2026-10-14T07:00\nBolt 1 2026-10-19T08:00\n", answer.Output)
	assert.JSONEq(t, `{"customers": 2}`, string(answer.Result))
}

func realScriptRuntime(
	t *testing.T,
	rec *recorder,
	responses map[string]map[string]any,
	completion agentports.CompletionService,
) *agentruntime.Service {
	t.Helper()

	fakes := append(
		[]reflect.Value{reflect.ValueOf(scriptrunner.NewRunner(zap.NewNop()))},
		newToolDeps(rec, responses).fakes()...,
	)
	var (
		queries []agentports.AgentQueryTool
		actions []agentports.AgentTool
	)
	for _, provider := range append(
		agentquerytoolservice.ToolProviders(),
		agenttoolservice.ToolProviders()...,
	) {
		tool, err := reflectutils.Construct(provider, suppliedFor(provider, fakes))
		require.NoError(t, err)
		switch typed := tool.(type) {
		case agentports.AgentQueryTool:
			queries = append(queries, &recordingQuery{
				inner:    typed,
				rec:      rec,
				response: responses[typed.Name()],
			})
		case agentports.AgentTool:
			actions = append(actions, recordedAction(typed, rec))
		}
	}

	kit := agentevalgate.FromTools(queries, actions, agentruntime.RuntimePolicies())
	return kit.NewRuntime(agentevalgate.RuntimeParams{
		Completion:  completion,
		Permissions: &agenttest.StubPermissions{},
		Versions:    pinnedVersions{},
		Scripts:     providerScripts{settings: ProviderSpec{Scripts: true}.settings()},
	})
}

func offeredTools(req *agentports.ChatCompletionRequest) []string {
	names := make([]string, 0, len(req.Tools))
	for _, tool := range req.Tools {
		names = append(names, tool.Name)
	}

	return names
}

type scriptToolAnswer struct {
	Status string                 `json:"status"`
	Output string                 `json:"output"`
	Result sonic.NoCopyRawMessage `json:"result"`
}

func scriptAnswer(t *testing.T, req *agentports.ChatCompletionRequest) scriptToolAnswer {
	t.Helper()

	for _, message := range req.Messages {
		if message.Role != "tool" {
			continue
		}
		var answer scriptToolAnswer
		require.NoError(t, sonic.UnmarshalString(untrustedBody(message.Content), &answer),
			message.Content)
		return answer
	}
	t.Fatal("the script's result never reached the model")
	return scriptToolAnswer{}
}

func untrustedBody(content string) string {
	start := strings.Index(content, "<untrusted_data>")
	end := strings.LastIndex(content, "</untrusted_data>")
	if start < 0 || end < start {
		return content
	}

	return strings.TrimSpace(content[start+len("<untrusted_data>") : end])
}
