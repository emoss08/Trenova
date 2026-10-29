package locationtools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/masterdatakit"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeLocations struct {
	*agenttoolstest.FakeMaster[location.Location]
	bulk *locationports.BulkUpdateLocationStatusRequest
}

func (f *fakeLocations) Get(
	context.Context,
	locationports.GetLocationByIDRequest,
) (*location.Location, error) {
	return f.MasterGet()
}

func (f *fakeLocations) PlanUpdate(
	ctx context.Context,
	e *location.Location,
) (*kernelports.RecordChange[location.Location], error) {
	return f.MasterPlanUpdate(ctx, e)
}

func (f *fakeLocations) Update(
	ctx context.Context,
	e *location.Location,
	a *permissionports.RequestActor,
) (*location.Location, error) {
	return f.MasterUpdate(ctx, e, a)
}

func (f *fakeLocations) PlanBulkUpdateStatus(
	_ context.Context,
	req *locationports.BulkUpdateLocationStatusRequest,
) ([]kernelports.RecordChange[location.Location], error) {
	after := *f.Stored
	after.Status = req.Status

	return []kernelports.RecordChange[location.Location]{{Before: f.Stored, After: &after}}, nil
}

func (f *fakeLocations) BulkUpdateStatus(
	_ context.Context,
	req *locationports.BulkUpdateLocationStatusRequest,
) ([]*location.Location, error) {
	if err := f.Guard.Write(); err != nil {
		return nil, err
	}
	f.bulk = req

	return []*location.Location{f.Stored}, nil
}

func TestUpdateLocation_ChangesTheAddressAndKeepsTheCode(t *testing.T) {
	t.Parallel()

	states := agenttoolstest.TexasState()
	stored := &location.Location{
		ID:                 pulid.MustNew("loc_"),
		Code:               "DALWH01",
		Name:               "Dallas warehouse",
		AddressLine1:       "1 Dock St",
		City:               "Dallas",
		StateID:            states.State.ID,
		PostalCode:         "75201",
		LocationCategoryID: pulid.MustNew("lc_"),
		Status:             domaintypes.StatusActive,
		Version:            3,
	}
	locations := &fakeLocations{FakeMaster: &agenttoolstest.FakeMaster[location.Location]{
		Guard: &agenttoolstest.WriteGuard{}, Stored: stored,
	}}
	tool := NewUpdateLocationTool(locations, states)
	params := agenttoolstest.ExecuteParams(map[string]any{
		paramLocationID:              stored.ID.String(),
		masterdatakit.MdAddressLine1: "9 Dock St",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		locations.Guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Equal(
		t,
		"9 Dock St",
		agenttoolstest.FieldByPath(
			t,
			agenttoolstest.PreviewChange(t, preview, 0),
			masterdatakit.MdAddressLine1,
		).After,
	)
	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, "DALWH01", locations.Updated.Code)
	assert.Equal(t, "9 Dock St", locations.Updated.AddressLine1)

	status := NewUpdateLocationStatusTool(locations, states)
	require.NoError(t, status.Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"locationIds":             []any{stored.ID.String()},
		changepreview.FieldStatus: "Inactive",
	})))
	assert.Equal(t, domaintypes.StatusInactive, locations.bulk.Status)
}
