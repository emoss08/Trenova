package toolsimulation

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type plainTool struct{ name string }

func (t plainTool) Name() string                { return t.name }
func (t plainTool) Description() string         { return "" }
func (t plainTool) ParamSchema() map[string]any { return map[string]any{} }
func (t plainTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
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

func (t plainTool) Execute(context.Context, agenttoolports.ToolExecuteParams) error {
	return errors.New("must not run in simulation")
}

type previewingTool struct {
	plainTool
	record   *agent.ToolPreview
	failed   error
	readOnly *bool
}

func (t previewingTool) Preview(
	ctx context.Context,
	_ agenttoolports.ToolExecuteParams,
) (*agent.ToolPreview, error) {
	if t.readOnly != nil {
		*t.readOnly = kernelports.IsReadOnly(ctx)
	}

	return t.record, t.failed
}

func cancelPreview() *agent.ToolPreview {
	return &agent.ToolPreview{
		Summary: "Would cancel PRO-100.",
		Changes: []agent.RecordChange{{
			Operation: agent.PreviewOperationUpdate,
			Label:     "PRO-100",
			Fields: []agent.PreviewFieldChange{
				{Path: "status", Label: "Status", Before: "New", After: "Canceled"},
			},
		}},
	}
}

// A tool with no preview is still described: a person reading the run has to
// see what was asked for, and the summary must not pretend more was known.
func TestSimulate_DescribesAToolWithoutAPreview(t *testing.T) {
	t.Parallel()

	preview := Simulate(
		t.Context(),
		plainTool{name: "shipment_cancel"},
		&agenttoolports.ToolExecuteParams{
			Params: map[string]any{
				"shipmentId":   "shp_1",
				"cancelReason": "Customer pulled the load",
				"count":        2,
			},
		},
	)

	assert.False(t, preview.Previewed)
	assert.Contains(t, preview.Summary, "Would run shipment_cancel")
	assert.Contains(t, preview.Summary, "no preview of its own")
	require.Len(t, preview.Changes, 3)
	assert.Equal(
		t,
		agent.FieldChange{Field: "Cancel reason", To: "Customer pulled the load"},
		preview.Changes[0],
	)
	assert.Equal(t, agent.FieldChange{Field: "Count", To: "2"}, preview.Changes[1])
}

func TestSimulate_ReadsTheRecordPreview(t *testing.T) {
	t.Parallel()

	tool := previewingTool{plainTool: plainTool{name: "shipment_cancel"}, record: cancelPreview()}

	preview := Simulate(t.Context(), tool, &agenttoolports.ToolExecuteParams{})

	assert.True(t, preview.Previewed)
	assert.Equal(t, "Would cancel PRO-100.", preview.Summary)
	assert.Contains(t, preview.Describe(), "- Status: New → Canceled")

	tool.failed = errors.New("shipment not found")
	tool.record = nil
	failed := Simulate(t.Context(), tool, &agenttoolports.ToolExecuteParams{
		Params: map[string]any{"shipmentId": "shp_1", "_owner": "usr_1"},
	})
	assert.False(t, failed.Previewed)
	assert.Contains(t, failed.Summary, "shipment not found")
	require.Len(t, failed.Changes, 1, "the owner the runtime writes is not shown")
	assert.Equal(t, "shp_1", failed.Changes[0].To)
}

// Work a preview hands off outside its transaction, such as a counter bumped
// in the background, reads the mark and stays undone.
func TestSimulate_PreviewsInAReadOnlyContext(t *testing.T) {
	t.Parallel()

	readOnly := false
	tool := previewingTool{
		plainTool: plainTool{name: "shipment_cancel"},
		record:    cancelPreview(),
		readOnly:  &readOnly,
	}

	Simulate(t.Context(), tool, &agenttoolports.ToolExecuteParams{})

	assert.True(t, readOnly)
}

func TestFromBaseline_ReadsTheSnapshotsPreview(t *testing.T) {
	t.Parallel()

	simulation := FromBaseline("shipment_cancel", nil, &agentruntimeports.ProposalBaselineResult{
		Preview: cancelPreview(),
	})

	assert.True(t, simulation.Previewed)
	assert.Contains(t, simulation.Describe(), "- Status: New → Canceled")
}

// A preview the snapshot refused, because it tried to write, is reported as
// failed rather than asked again where its write would land.
func TestFromBaseline_ReportsAFailedPreviewWithoutRunningItAgain(t *testing.T) {
	t.Parallel()

	simulation := FromBaseline(
		"shipment_cancel",
		map[string]any{"shipmentId": "shp_1"},
		&agentruntimeports.ProposalBaselineResult{
			PreviewErr: errors.New("cannot execute UPDATE in a read-only transaction"),
		},
	)

	assert.False(t, simulation.Previewed)
	assert.Contains(t, simulation.Summary, "read-only transaction")
	require.Len(t, simulation.Changes, 1)
	assert.Equal(t, "shp_1", simulation.Changes[0].To)
}

func TestFromBaseline_DescribesWhenNothingWasPreviewed(t *testing.T) {
	t.Parallel()

	for _, baseline := range []*agentruntimeports.ProposalBaselineResult{nil, {}} {
		simulation := FromBaseline(
			"shipment_cancel",
			map[string]any{"shipmentId": "shp_1"},
			baseline,
		)

		assert.False(t, simulation.Previewed)
		assert.Contains(t, simulation.Summary, "Would run shipment_cancel")
	}
}
