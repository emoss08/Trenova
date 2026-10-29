package anthropicadapter

import (
	"errors"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladaptertest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiplatform"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aireasoning"
	"github.com/stretchr/testify/require"
)

const (
	liveBindingEnv      = "TRENOVA_LIVE_ANTHROPIC_BINDING"
	liveBindingModelEnv = "TRENOVA_LIVE_ANTHROPIC_MODEL"
	liveBindingKeyEnv   = "ANTHROPIC_API_KEY"
	liveBindingURLEnv   = "TRENOVA_LIVE_ANTHROPIC_BASE_URL"
	liveBindingModel    = "claude-haiku-5-5"
	liveBindingMints    = 3
)

func liveBindingCall(t *testing.T, model string, request *modeladapter.Request) *modeladapter.Call {
	t.Helper()

	return &modeladapter.Call{
		Provider: &aiprovider.Provider{
			Kind:            aiprovider.KindAnthropicMessages,
			Model:           model,
			Platform:        aiplatform.PlatformDirect,
			ReasoningEffort: aireasoning.ReasoningMedium,
			BaseURL:         strings.TrimSpace(os.Getenv(liveBindingURLEnv)),
		},
		APIKey:  os.Getenv(liveBindingKeyEnv),
		Client:  &http.Client{Timeout: 2 * time.Minute},
		Request: request,
	}
}

func liveBindingTools(grown bool) []modeladapter.ToolSpec {
	tools := []modeladapter.ToolSpec{{
		Name:        "get_shipment",
		Description: "Look up a shipment by its pro number and return its status and stops.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"number": map[string]any{"type": "string", "description": "The pro number"},
			},
			"required":             []any{"number"},
			"additionalProperties": false,
		},
	}, findToolsLiveSpec()}
	if grown {
		tools = append(tools, modeladapter.ToolSpec{
			Name:        "list_workers",
			Description: "List drivers and other workers, optionally by status.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"status": map[string]any{"type": "string"},
				},
				"additionalProperties": false,
			},
		})
	}

	return tools
}

func findToolsLiveSpec() modeladapter.ToolSpec {
	return modeladapter.ToolSpec{
		Name:        "find_tools",
		Description: "Load more tools when nothing you hold does the job.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"need": map[string]any{"type": "string"},
			},
			"required":             []any{"need"},
			"additionalProperties": false,
		},
	}
}

const liveBindingSystem = "You are a dispatcher's assistant for a trucking company. Think " +
	"carefully before you act. Always look a shipment up with get_shipment before saying " +
	"anything about it."

func liveBindingMint(t *testing.T, model string) []modeladapter.Message {
	t.Helper()

	question := modeladapter.Message{
		Role: modeladapter.RoleUser,
		Content: "Shipment S-1042 was due in Memphis this morning and the receiver says " +
			"it has not arrived. Check it and tell me whether we need to warn the customer.",
	}
	for attempt := 1; attempt <= liveBindingMints; attempt++ {
		resp, err := NewAnthropicAdapter().Complete(t.Context(), liveBindingCall(t, model, &modeladapter.Request{
			System:    liveBindingSystem,
			Messages:  []modeladapter.Message{question},
			Tools:     liveBindingTools(false),
			MaxTokens: 4096,
		}))
		require.NoError(t, err, "the first request of the turn must succeed")
		if resp.Reasoning == nil || resp.Reasoning.Signature == "" || len(resp.ToolCalls) == 0 {
			t.Logf("mint %d returned no signed thinking with a tool call; asking again", attempt)
			continue
		}

		calls := resp.ToolCalls
		results := make([]modeladapter.Message, 0, len(calls))
		for _, call := range calls {
			results = append(results, modeladapter.Message{
				Role:       modeladapter.RoleTool,
				ToolCallID: call.ID,
				ToolName:   call.Name,
				Content: `{"proNumber":"S-1042","status":"InTransit","customer":"Acme Foods",` +
					`"nextStop":{"city":"Memphis","scheduledArrival":"08:00","eta":"14:30"}}`,
			})
		}

		return append([]modeladapter.Message{
			question,
			{
				Role:      modeladapter.RoleAssistant,
				Content:   resp.Text,
				ToolCalls: calls,
				Reasoning: resp.Reasoning,
			},
		}, results...)
	}

	t.Fatalf("inconclusive: %s answered %d times without a signed thinking block and a "+
		"tool call, so there was nothing to replay", model, liveBindingMints)

	return nil
}

func liveBindingReplay(
	t *testing.T,
	name, model string,
	history []modeladapter.Message,
	grown bool,
	behavior anthropic.BetaThinkingPrefixMismatchBehavior,
) modeladaptertest.LiveBindingArm {
	t.Helper()

	call := liveBindingCall(t, model, &modeladapter.Request{
		System:    liveBindingSystem,
		Messages:  history,
		Tools:     liveBindingTools(grown),
		MaxTokens: 4096,
	})
	arm := modeladaptertest.LiveBindingArm{Name: name}
	if behavior == "" {
		resp, err := NewAnthropicAdapter().Complete(t.Context(), call)
		arm.Err = err
		if resp != nil {
			arm.Dropped = resp.ThinkingDropped
		}
		arm.Status = statusOf(err)

		return arm
	}

	attached := attachFiles(t.Context(), call)
	params := anthropicParams(call, &attached)
	binding := anthropic.BetaThinkingBlockBindingParam{PrefixMismatchBehavior: behavior}
	if params.Thinking.OfAdaptive == nil {
		params.Thinking = adaptiveThinking("")
	}
	params.Thinking.OfAdaptive.BlockBinding = binding
	if !hasBeta(params.Betas, anthropicBindingBeta) {
		params.Betas = append(params.Betas, anthropicBindingBeta)
	}

	service, err := anthropicMessages(t.Context(), call, false, NewPlatformCredentials())
	require.NoError(t, err)
	message, err := service.New(t.Context(), params, anthropicBlockingOptions(call)...)
	arm.Err = err
	arm.Status = statusOf(err)
	if message != nil {
		arm.Dropped = thinkingDropped(message.InputTransformations)
	}

	return arm
}

func hasBeta(betas []anthropic.AnthropicBeta, want anthropic.AnthropicBeta) bool {
	for _, beta := range betas {
		if beta == want {
			return true
		}
	}

	return false
}

func statusOf(err error) int {
	var transport *modeladapter.TransportError
	if errors.As(err, &transport) {
		return transport.StatusCode
	}
	var apiErr *anthropic.Error
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode
	}

	return 0
}

func TestLiveAnthropic_HaikuThinkingBindingOnAMidTurnToolChange(t *testing.T) {
	if os.Getenv(liveBindingEnv) == "" || os.Getenv(liveBindingKeyEnv) == "" {
		t.Skipf("live test: set %s=1 and %s to run it", liveBindingEnv, liveBindingKeyEnv)
	}
	model := liveBindingModel
	if override := strings.TrimSpace(os.Getenv(liveBindingModelEnv)); override != "" {
		model = override
	}

	history := liveBindingMint(t, model)
	arms := []modeladaptertest.LiveBindingArm{
		liveBindingReplay(t, "honest", model, history, false, ""),
		liveBindingReplay(t, "as sent", model, history, true, ""),
		liveBindingReplay(t, "forced", model, history, true,
			anthropic.BetaThinkingPrefixMismatchBehaviorError),
		liveBindingReplay(t, "drop_block", model, history, true,
			anthropic.BetaThinkingPrefixMismatchBehaviorDropBlock),
	}

	lines := make([]string, 0, len(arms))
	for _, arm := range arms {
		lines = append(lines, arm.String())
	}
	t.Logf("thinking binding on %s after a mid-turn tool change:\n  %s",
		model, strings.Join(lines, "\n  "))

	honest, asSent, forced := arms[0], arms[1], arms[2]
	require.NoError(t, honest.Err, "an unedited replay must be accepted")
	if forced.Err == nil && forced.Dropped == 0 {
		t.Logf("the forced arm was accepted with nothing dropped: the check does not "+
			"see a tool added mid-turn as an edit on %s", model)
	}
	require.NoError(t, asSent.Err,
		"%s refuses the request Desk sends after find_tools loads a tool: set "+
			"bindsPrefix for it in anthropicTraits", model)
}
