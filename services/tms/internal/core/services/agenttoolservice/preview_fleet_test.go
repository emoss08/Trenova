package agenttoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
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
	tool := newUpdateTractorStatusTool(tractors).(*updateTractorStatusTool)
	params := executeParams(map[string]any{
		"tractorIds": []any{unit.ID.String()},
		"status":     "OutOfService",
	})

	preview := previewWithoutWrites(t, &tractors.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	change := previewChange(t, preview, 0)
	assert.Equal(t, permission.ResourceTractor, change.Resource)
	assert.Equal(t, agent.PreviewOperationUpdate, change.Operation)
	assert.Equal(t, "T-4471", change.Label)
	require.NotNil(t, change.Version)
	assert.Equal(t, int64(7), *change.Version)
	status := fieldByPath(t, change, "status")
	assert.Equal(t, "Available", status.Before)
	assert.Equal(t, "OutOfService", status.After)

	require.NoError(t, tool.Execute(t.Context(), params))
	requireUpdateParity(t, change, &before, unit, toolpreview.Only("status"))
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
	tool := newUpdateTrailerStatusTool(trailers).(*updateTrailerStatusTool)
	params := executeParams(map[string]any{
		"trailerIds": []any{unit.ID.String()},
		"status":     "Available",
	})

	preview := previewWithoutWrites(t, &trailers.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})
	change := previewChange(t, preview, 0)
	assert.Equal(t, "R-220", change.Label)

	require.NoError(t, tool.Execute(t.Context(), params))
	requireUpdateParity(t, change, &before, unit, toolpreview.Only("status"))
}

func TestUpdateTractorStatus_PreviewNamesAUnitThatIsNotThere(t *testing.T) {
	t.Parallel()

	unit := &tractor.Tractor{ID: pulid.MustNew("trc_"), Code: "T-1", Status: "Available"}
	missing := pulid.MustNew("trc_")
	tractors := &fakeTractorStatusUpdater{units: map[pulid.ID]*tractor.Tractor{unit.ID: unit}}
	tool := newUpdateTractorStatusTool(tractors).(*updateTractorStatusTool)

	preview := previewWithoutWrites(t, &tractors.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), executeParams(map[string]any{
			"tractorIds": []any{unit.ID.String(), missing.String()},
			"status":     "Sold",
		}))
	})

	require.Len(t, preview.Changes, 1)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
	assert.Contains(t, preview.Warnings[0].Args, missing.String())
	assert.Contains(t, preview.Summary, "2 tractors")
}
