package equipmenttools

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateTractorStatus_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	unit := &tractor.Tractor{
		ID:      pulid.MustNew("trc_"),
		Code:    "T-4471",
		Status:  domaintypes.EquipmentStatusAvailable,
		Version: 7,
	}
	before := *unit
	tractors := &fakeTractorStatusUpdater{units: map[pulid.ID]*tractor.Tractor{unit.ID: unit}}
	tool := NewUpdateTractorStatusTool(tractors).(*UpdateTractorStatusTool)
	params := agenttoolstest.ExecuteParams(map[string]any{
		"tractorIds": []any{unit.ID.String()},
		"status":     "OutOfService",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&tractors.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), params)
		},
	)

	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, permission.ResourceTractor, change.Resource)
	assert.Equal(t, agentpreview.PreviewOperationUpdate, change.Operation)
	assert.Equal(t, "T-4471", change.Label)
	require.NotNil(t, change.Version)
	assert.Equal(t, int64(7), *change.Version)
	status := agenttoolstest.FieldByPath(t, change, "status")
	assert.Equal(t, "Available", status.Before)
	assert.Equal(t, "OutOfService", status.After)

	require.NoError(t, tool.Execute(t.Context(), params))
	agenttoolstest.RequireUpdateParity(t, change, &before, unit, toolpreview.Only("status"))
}

func TestUpdateTrailerStatus_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	unit := &trailer.Trailer{
		ID:      pulid.MustNew("trl_"),
		Code:    "R-220",
		Status:  domaintypes.EquipmentStatusOOS,
		Version: 2,
	}
	before := *unit
	trailers := &fakeTrailerStatusUpdater{units: map[pulid.ID]*trailer.Trailer{unit.ID: unit}}
	tool := NewUpdateTrailerStatusTool(trailers).(*UpdateTrailerStatusTool)
	params := agenttoolstest.ExecuteParams(map[string]any{
		"trailerIds": []any{unit.ID.String()},
		"status":     "Available",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&trailers.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), params)
		},
	)
	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, "R-220", change.Label)

	require.NoError(t, tool.Execute(t.Context(), params))
	agenttoolstest.RequireUpdateParity(t, change, &before, unit, toolpreview.Only("status"))
}

func TestUpdateTractorStatus_PreviewNamesAUnitThatIsNotThere(t *testing.T) {
	t.Parallel()

	unit := &tractor.Tractor{ID: pulid.MustNew("trc_"), Code: "T-1", Status: "Available"}
	missing := pulid.MustNew("trc_")
	tractors := &fakeTractorStatusUpdater{units: map[pulid.ID]*tractor.Tractor{unit.ID: unit}}
	tool := NewUpdateTractorStatusTool(tractors).(*UpdateTractorStatusTool)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&tractors.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
				"tractorIds": []any{unit.ID.String(), missing.String()},
				"status":     "Sold",
			}))
		},
	)

	require.Len(t, preview.Changes, 1)
	agenttoolstest.RequireWarning(t, preview, agentpreview.PreviewWarningWouldFail)
	assert.Contains(t, preview.Warnings[0].Args, missing.String())
	assert.Contains(t, preview.Summary, "2 tractors")
}
