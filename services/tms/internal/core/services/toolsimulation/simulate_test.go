package toolsimulation

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type plainTool struct{ name string }

func (t plainTool) Name() string                { return t.name }
func (t plainTool) Description() string         { return "" }
func (t plainTool) ParamSchema() map[string]any { return map[string]any{} }
func (t plainTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          t.name,
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceShipment,
		Operation:     permission.OpUpdate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agent.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agent.ExternalReadNever,
		Rationale:     "A plain stub.",
	}
}

func (t plainTool) Execute(context.Context, serviceports.ToolExecuteParams) error {
	return errors.New("must not run in simulation")
}

type previewingTool struct {
	plainTool
	preview *agent.ToolSimulation
	err     error
}

func (t previewingTool) Simulate(
	context.Context,
	serviceports.ToolExecuteParams,
) (*agent.ToolSimulation, error) {
	return t.preview, t.err
}

func TestSimulate_UsesTheToolsOwnPreview(t *testing.T) {
	t.Parallel()

	tool := previewingTool{
		plainTool: plainTool{name: "update_tractor_status"},
		preview: &agent.ToolSimulation{
			Summary: "Tractor 101 would go from Available to OutOfService.",
			Changes: []agent.FieldChange{{Field: "status", From: "Available", To: "OutOfService"}},
		},
	}

	preview := Simulate(t.Context(), tool, serviceports.ToolExecuteParams{})

	assert.True(t, preview.Previewed)
	assert.Equal(t, "Tractor 101 would go from Available to OutOfService.", preview.Summary)
	assert.Contains(t, preview.Describe(), "- status: Available → OutOfService")
}

// A tool with no preview is still described: a person reading the run has to
// see what was asked for, and the summary must not pretend more was known.
func TestSimulate_DescribesAToolWithoutAPreview(t *testing.T) {
	t.Parallel()

	preview := Simulate(
		t.Context(),
		plainTool{name: "cancel_shipment"},
		serviceports.ToolExecuteParams{
			Params: map[string]any{
				"shipmentId":   "shp_1",
				"cancelReason": "Customer pulled the load",
				"count":        2,
			},
		},
	)

	assert.False(t, preview.Previewed)
	assert.Contains(t, preview.Summary, "Would run cancel_shipment")
	assert.Contains(t, preview.Summary, "no preview of its own")
	require.Len(t, preview.Changes, 3)
	assert.Equal(
		t,
		agent.FieldChange{Field: "cancelReason", To: "Customer pulled the load"},
		preview.Changes[0],
	)
	assert.Equal(t, agent.FieldChange{Field: "count", To: "2"}, preview.Changes[1])
}

func TestSimulate_KeepsTheRequestWhenAPreviewFails(t *testing.T) {
	t.Parallel()

	tool := previewingTool{
		plainTool: plainTool{name: "assign_move"},
		err:       errors.New("move not found"),
	}

	preview := Simulate(t.Context(), tool, serviceports.ToolExecuteParams{
		Params: map[string]any{"shipmentMoveId": "smv_1"},
	})

	assert.False(t, preview.Previewed)
	assert.Contains(t, preview.Summary, "move not found")
	assert.Equal(t, "smv_1", preview.Changes[0].To)
}
