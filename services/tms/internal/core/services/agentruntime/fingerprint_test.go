package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime/agentruntimetest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fingerprintWorld struct {
	lookup *agentruntimetest.StubQueryTool
	write  *agentruntimetest.StubActionTool
}

func newFingerprintWorld() *fingerprintWorld {
	return &fingerprintWorld{
		lookup: &agentruntimetest.StubQueryTool{
			ToolName: "find_shipments",
			Desc:     "Finds shipments by status.",
		},
		write: &agentruntimetest.StubActionTool{
			ToolName: "assign_driver",
			Tier:     agent.TierPropose,
			Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"driverId": map[string]any{"type": "string"},
				},
			},
		},
	}
}

func (w *fingerprintWorld) service() *Service {
	return newRuntime(
		&scriptedCompletion{},
		&stubQueryRegistry{Tools: []serviceports.AgentQueryTool{w.lookup}},
		&stubActionRegistry{Tools: []serviceports.AgentTool{w.write}},
		nil,
	)
}

func TestFingerprint_TheSameAgentGivesTheSameHash(t *testing.T) {
	t.Parallel()

	world := newFingerprintWorld()
	provider := pulid.MustNew("aiprv_")

	first := world.service().Fingerprint(
		testDefinition("find_shipments", "assign_driver"), provider, "model-a",
	)
	second := world.service().Fingerprint(
		testDefinition("find_shipments", "assign_driver"), provider, "model-a",
	)

	require.NotNil(t, first)
	assert.Equal(t, first.Hash(), second.Hash())
	assert.Equal(t, first.PromptHash, second.PromptHash)
	assert.Equal(t, first.ToolSpecHash, second.ToolSpecHash)
	assert.Len(t, first.Hash(), 64)
}

func TestFingerprint_ToolOrderDoesNotChangeTheToolHash(t *testing.T) {
	t.Parallel()

	world := newFingerprintWorld()
	forward := world.service().Fingerprint(
		testDefinition("find_shipments", "assign_driver"), pulid.Nil, "",
	)
	backward := world.service().Fingerprint(
		testDefinition("assign_driver", "find_shipments"), pulid.Nil, "",
	)

	assert.Equal(t, forward.ToolSpecHash, backward.ToolSpecHash)
}

func TestFingerprint_AnyChangeToTheAgentChangesTheHash(t *testing.T) {
	t.Parallel()

	baseline := newFingerprintWorld()
	base := baseline.service().Fingerprint(
		testDefinition("find_shipments", "assign_driver"), pulid.Nil, "model-a",
	)

	tests := []struct {
		name   string
		change func(w *fingerprintWorld) *agent.Fingerprint
		prompt bool
		tools  bool
	}{
		{
			name: "a tool's description",
			change: func(w *fingerprintWorld) *agent.Fingerprint {
				w.lookup.Desc = "Finds shipments by status and customer."
				return w.service().Fingerprint(
					testDefinition("find_shipments", "assign_driver"), pulid.Nil, "model-a",
				)
			},
			tools: true,
		},
		{
			name: "a tool's schema",
			change: func(w *fingerprintWorld) *agent.Fingerprint {
				w.write.Schema = map[string]any{
					"type": "object",
					"properties": map[string]any{
						"driverId": map[string]any{"type": "string"},
						"note":     map[string]any{"type": "string"},
					},
				}
				return w.service().Fingerprint(
					testDefinition("find_shipments", "assign_driver"), pulid.Nil, "model-a",
				)
			},
			tools: true,
		},
		{
			name: "a tool's default tier",
			change: func(w *fingerprintWorld) *agent.Fingerprint {
				w.write.Tier = agent.TierAutoExecute
				return w.service().Fingerprint(
					testDefinition("find_shipments", "assign_driver"), pulid.Nil, "model-a",
				)
			},
			tools: true,
		},
		{
			name: "the tier the agent sets for a tool",
			change: func(w *fingerprintWorld) *agent.Fingerprint {
				definition := testDefinition("find_shipments", "assign_driver")
				definition.ToolTiers = map[string]agent.AutonomyTier{
					"assign_driver": agent.TierActWithApproval,
				}
				return w.service().Fingerprint(definition, pulid.Nil, "model-a")
			},
			tools: true,
		},
		{
			name: "the instructions",
			change: func(w *fingerprintWorld) *agent.Fingerprint {
				definition := testDefinition("find_shipments", "assign_driver")
				definition.Instructions = "Help dispatch, and be brief."
				return w.service().Fingerprint(definition, pulid.Nil, "model-a")
			},
			prompt: true,
		},
		{
			name: "the tools it holds",
			change: func(w *fingerprintWorld) *agent.Fingerprint {
				return w.service().Fingerprint(
					testDefinition("find_shipments"), pulid.Nil, "model-a",
				)
			},
			tools: true,
		},
		{
			name: "the model",
			change: func(w *fingerprintWorld) *agent.Fingerprint {
				return w.service().Fingerprint(
					testDefinition("find_shipments", "assign_driver"), pulid.Nil, "model-b",
				)
			},
		},
		{
			name: "the saved version",
			change: func(w *fingerprintWorld) *agent.Fingerprint {
				definition := testDefinition("find_shipments", "assign_driver")
				definition.Version = 2
				return w.service().Fingerprint(definition, pulid.Nil, "model-a")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			changed := tt.change(newFingerprintWorld())
			assert.NotEqual(t, base.Hash(), changed.Hash(), "the fingerprint hash must move")
			if tt.prompt {
				assert.NotEqual(t, base.PromptHash, changed.PromptHash)
			}
			if tt.tools {
				assert.NotEqual(t, base.ToolSpecHash, changed.ToolSpecHash)
			}
		})
	}
}

func TestFingerprint_ChangesNameWhatMoved(t *testing.T) {
	t.Parallel()

	world := newFingerprintWorld()
	before := world.service().Fingerprint(
		testDefinition("find_shipments", "assign_driver"), pulid.Nil, "model-a",
	)
	definition := testDefinition("find_shipments", "assign_driver")
	definition.Version = 4
	after := world.service().Fingerprint(definition, pulid.Nil, "model-b")

	changes := after.Changes(before)
	fields := make([]agent.FingerprintField, 0, len(changes))
	for _, change := range changes {
		fields = append(fields, change.Field)
	}
	assert.Equal(t, []agent.FingerprintField{
		agent.FingerprintFieldDefinition,
		agent.FingerprintFieldModel,
	}, fields)
	assert.Empty(t, after.Changes(after))
}
