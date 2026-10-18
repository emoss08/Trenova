package modeladapter

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
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

type liveBindingArm struct {
	name    string
	err     error
	status  int
	dropped int
}

func (a liveBindingArm) String() string {
	switch {
	case a.err == nil:
		return fmt.Sprintf("%s: 200, %d thinking block(s) dropped", a.name, a.dropped)
	case a.status != 0:
		return fmt.Sprintf("%s: refused with %d: %s", a.name, a.status, firstLine(a.err.Error()))
	default:
		return fmt.Sprintf("%s: error %s", a.name, firstLine(a.err.Error()))
	}
}

func firstLine(text string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(text), "\n")

	return line
}

func liveBindingCall(t *testing.T, model string, request *Request) *Call {
	t.Helper()

	return &Call{
		Provider: &aiprovider.Provider{
			Kind:            aiprovider.KindAnthropicMessages,
			Model:           model,
			Platform:        aiprovider.PlatformDirect,
			ReasoningEffort: aiprovider.ReasoningMedium,
			BaseURL:         strings.TrimSpace(os.Getenv(liveBindingURLEnv)),
		},
		APIKey:  os.Getenv(liveBindingKeyEnv),
		Client:  &http.Client{Timeout: 2 * time.Minute},
		Request: request,
	}
}

func liveBindingTools(grown bool) []ToolSpec {
	tools := []ToolSpec{{
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
		tools = append(tools, ToolSpec{
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

func findToolsLiveSpec() ToolSpec {
	return ToolSpec{
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

func liveBindingMint(t *testing.T, model string) []Message {
	t.Helper()

	question := Message{
		Role: RoleUser,
		Content: "Shipment S-1042 was due in Memphis this morning and the receiver says " +
			"it has not arrived. Check it and tell me whether we need to warn the customer.",
	}
	for attempt := 1; attempt <= liveBindingMints; attempt++ {
		resp, err := NewAnthropicAdapter().Complete(t.Context(), liveBindingCall(t, model, &Request{
			System:    liveBindingSystem,
			Messages:  []Message{question},
			Tools:     liveBindingTools(false),
			MaxTokens: 4096,
		}))
		require.NoError(t, err, "the first request of the turn must succeed")
		if resp.Reasoning == nil || resp.Reasoning.Signature == "" || len(resp.ToolCalls) == 0 {
			t.Logf("mint %d returned no signed thinking with a tool call; asking again", attempt)
			continue
		}

		calls := resp.ToolCalls
		results := make([]Message, 0, len(calls))
		for _, call := range calls {
			results = append(results, Message{
				Role:       RoleTool,
				ToolCallID: call.ID,
				ToolName:   call.Name,
				Content: `{"proNumber":"S-1042","status":"InTransit","customer":"Acme Foods",` +
					`"nextStop":{"city":"Memphis","scheduledArrival":"08:00","eta":"14:30"}}`,
			})
		}

		return append([]Message{
			question,
			{Role: RoleAssistant, Content: resp.Text, ToolCalls: calls, Reasoning: resp.Reasoning},
		}, results...)
	}

	t.Fatalf("inconclusive: %s answered %d times without a signed thinking block and a "+
		"tool call, so there was nothing to replay", model, liveBindingMints)

	return nil
}

func liveBindingReplay(
	t *testing.T,
	name, model string,
	history []Message,
	grown bool,
	behavior anthropic.BetaThinkingPrefixMismatchBehavior,
) liveBindingArm {
	t.Helper()

	call := liveBindingCall(t, model, &Request{
		System:    liveBindingSystem,
		Messages:  history,
		Tools:     liveBindingTools(grown),
		MaxTokens: 4096,
	})
	arm := liveBindingArm{name: name}
	if behavior == "" {
		resp, err := NewAnthropicAdapter().Complete(t.Context(), call)
		arm.err = err
		if resp != nil {
			arm.dropped = resp.ThinkingDropped
		}
		arm.status = statusOf(err)

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

	service, err := anthropicMessages(t.Context(), call, false, newPlatformCredentials())
	require.NoError(t, err)
	message, err := service.New(t.Context(), params, anthropicBlockingOptions(call)...)
	arm.err = err
	arm.status = statusOf(err)
	if message != nil {
		arm.dropped = thinkingDropped(message.InputTransformations)
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
	var transport *TransportError
	if errors.As(err, &transport) {
		return transport.StatusCode
	}
	var apiErr *anthropic.Error
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode
	}

	return 0
}

/*
Does Haiku 5.5 refuse the history edit Desk makes within a turn?

Haiku 5.5 runs the same preserved-thinking check as Opus and Sonnet 5.5: a
thinking block replayed after a change to system, tools or earlier messages is a
400, enforced by default for accounts created on or after 2026-08-31 and, on an
older account, for any request that sets prefix_mismatch_behavior. The adapter
sets drop_block only for Opus >= 5.5, Sonnet >= 5.5 and Fable >= 5.1
(anthropicTraits), and Desk still changes tools mid-turn when find_tools loads
one. This replays a minted thinking block four ways and reports each:

  honest      the same tools, as every call of a turn that loads nothing
  as sent     a tool added, exactly as the adapter sends it for this model today
  forced      the same edit with prefix_mismatch_behavior "error" (opts any account in)
  drop_block  the same edit with prefix_mismatch_behavior "drop_block" (what bindsPrefix sends)

It fails only when the honest replay or the request the adapter sends today is
refused: that is the evidence for setting bindsPrefix on Haiku 5.5, and the
guard once it is set. Opt in with TRENOVA_LIVE_ANTHROPIC_BINDING=1 and
ANTHROPIC_API_KEY; TRENOVA_LIVE_ANTHROPIC_MODEL picks another model and
TRENOVA_LIVE_ANTHROPIC_BASE_URL another endpoint (a gateway production uses). It sends
five small requests at the model's normal rates.
*/
func TestLiveAnthropic_HaikuThinkingBindingOnAMidTurnToolChange(t *testing.T) {
	if os.Getenv(liveBindingEnv) == "" || os.Getenv(liveBindingKeyEnv) == "" {
		t.Skipf("live test: set %s=1 and %s to run it", liveBindingEnv, liveBindingKeyEnv)
	}
	model := liveBindingModel
	if override := strings.TrimSpace(os.Getenv(liveBindingModelEnv)); override != "" {
		model = override
	}

	history := liveBindingMint(t, model)
	arms := []liveBindingArm{
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
	require.NoError(t, honest.err, "an unedited replay must be accepted")
	if forced.err == nil && forced.dropped == 0 {
		t.Logf("the forced arm was accepted with nothing dropped: the check does not " +
			"see a tool added mid-turn as an edit on %s", model)
	}
	require.NoError(t, asSent.err,
		"%s refuses the request Desk sends after find_tools loads a tool: set "+
			"bindsPrefix for it in anthropicTraits", model)
}
