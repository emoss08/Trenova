package commoditytools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/masterdatakit"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/modules/reference/ports/commodityports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/hazardousmaterialports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeCommodities struct {
	*agenttoolstest.FakeMaster[commodity.Commodity]
}

func (f fakeCommodities) Get(
	context.Context,
	commodityports.GetCommodityByIDRequest,
) (*commodity.Commodity, error) {
	return f.MasterGet()
}

func (f fakeCommodities) PlanCreate(
	ctx context.Context,
	e *commodity.Commodity,
) (*commodity.Commodity, error) {
	return f.MasterPlanCreate(ctx, e)
}

func (f fakeCommodities) Create(
	ctx context.Context,
	e *commodity.Commodity,
	a *permissionports.RequestActor,
) (*commodity.Commodity, error) {
	return f.MasterCreate(ctx, e, a)
}

func (f fakeCommodities) PlanUpdate(
	ctx context.Context,
	e *commodity.Commodity,
) (*kernelports.RecordChange[commodity.Commodity], error) {
	return f.MasterPlanUpdate(ctx, e)
}

func (f fakeCommodities) Update(
	ctx context.Context,
	e *commodity.Commodity,
	a *permissionports.RequestActor,
) (*commodity.Commodity, error) {
	return f.MasterUpdate(ctx, e, a)
}

func (f fakeCommodities) PlanBulkUpdateStatus(
	context.Context,
	*commodityports.BulkUpdateCommodityStatusRequest,
) ([]kernelports.RecordChange[commodity.Commodity], error) {
	return nil, errortypes.NewValidationError("commodityIds", errortypes.ErrInvalid,
		"These are not commodity records of this organization")
}

func (f fakeCommodities) BulkUpdateStatus(
	context.Context,
	*commodityports.BulkUpdateCommodityStatusRequest,
) ([]*commodity.Commodity, error) {
	return nil, f.Guard.Write()
}

func TestCreateCommodity_ReadsDecimalsAndTheFreightClass(t *testing.T) {
	t.Parallel()

	commodities := fakeCommodities{&agenttoolstest.FakeMaster[commodity.Commodity]{
		Guard: &agenttoolstest.WriteGuard{},
		SetID: func(c *commodity.Commodity) { c.ID = pulid.MustNew("com_") },
	}}
	tool := NewCreateCommodityTool(commodities)
	params := agenttoolstest.IdempotentParams(map[string]any{
		masterdatakit.MdName:        "Frozen peas",
		masterdatakit.MdDescription: "Frozen vegetables on pallets",
		"freightClass":              "Class70",
		"weightPerUnit":             "42.5",
		"minTemperature":            -10,
		"maxTemperature":            0,
		"stackable":                 true,
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		commodities.Guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Equal(
		t,
		"42.5",
		agenttoolstest.FieldByPath(
			t,
			agenttoolstest.PreviewChange(t, preview, 0),
			"weightPerUnit",
		).After,
	)

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, commodity.FreightClass70, commodities.Created.FreightClass)
	require.NotNil(t, commodities.Created.WeightPerUnit)
	assert.InDelta(t, 42.5, *commodities.Created.WeightPerUnit, 0.001)
	assert.Equal(t, -10, *commodities.Created.MinTemperature)

	bad := agenttoolstest.IdempotentParams(map[string]any{
		masterdatakit.MdName: "Peas", masterdatakit.MdDescription: "Peas", "freightClass": "Class71",
	})
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), bad))

	status := NewUpdateCommodityStatusTool(commodities)
	statusParams := agenttoolstest.ExecuteParams(map[string]any{
		"commodityIds":            []any{pulid.MustNew("com_").String()},
		changepreview.FieldStatus: "Inactive",
	})
	statusPreview, err := status.(agenttoolports.ToolPreviewer).Preview(t.Context(), statusParams)
	require.NoError(t, err)
	agenttoolstest.RequireWarning(t, statusPreview, agentpreview.PreviewWarningWouldFail)
}

type fakeHazmats struct {
	*agenttoolstest.FakeMaster[hazardousmaterial.HazardousMaterial]
}

func (f fakeHazmats) Get(
	context.Context,
	hazardousmaterialports.GetHazardousMaterialByIDRequest,
) (*hazardousmaterial.HazardousMaterial, error) {
	return f.MasterGet()
}

func (f fakeHazmats) PlanCreate(
	ctx context.Context,
	e *hazardousmaterial.HazardousMaterial,
) (*hazardousmaterial.HazardousMaterial, error) {
	return f.MasterPlanCreate(ctx, e)
}

func (f fakeHazmats) Create(
	ctx context.Context,
	e *hazardousmaterial.HazardousMaterial,
	a *permissionports.RequestActor,
) (*hazardousmaterial.HazardousMaterial, error) {
	return f.MasterCreate(ctx, e, a)
}

func (f fakeHazmats) PlanUpdate(
	ctx context.Context,
	e *hazardousmaterial.HazardousMaterial,
) (*kernelports.RecordChange[hazardousmaterial.HazardousMaterial], error) {
	return f.MasterPlanUpdate(ctx, e)
}

func (f fakeHazmats) Update(
	ctx context.Context,
	e *hazardousmaterial.HazardousMaterial,
	a *permissionports.RequestActor,
) (*hazardousmaterial.HazardousMaterial, error) {
	return f.MasterUpdate(ctx, e, a)
}

func (f fakeHazmats) PlanBulkUpdateStatus(
	context.Context,
	*hazardousmaterialports.BulkUpdateHazardousMaterialStatusRequest,
) ([]kernelports.RecordChange[hazardousmaterial.HazardousMaterial], error) {
	return nil, nil
}

func (f fakeHazmats) BulkUpdateStatus(
	context.Context,
	*hazardousmaterialports.BulkUpdateHazardousMaterialStatusRequest,
) ([]*hazardousmaterial.HazardousMaterial, error) {
	return nil, f.Guard.Write()
}

func TestHazardousMaterialTools_AlwaysWaitForAPerson(t *testing.T) {
	t.Parallel()

	stored := &hazardousmaterial.HazardousMaterial{
		ID:           pulid.MustNew("hm_"),
		Name:         "Gasoline",
		Description:  "Motor fuel",
		Class:        hazardousmaterial.HazardousClass3,
		PackingGroup: hazardousmaterial.PackingGroupII,
		UNNumber:     "1203",
		Version:      2,
	}
	materials := fakeHazmats{&agenttoolstest.FakeMaster[hazardousmaterial.HazardousMaterial]{
		Guard:  &agenttoolstest.WriteGuard{},
		Stored: stored,
		SetID:  func(h *hazardousmaterial.HazardousMaterial) { h.ID = pulid.MustNew("hm_") },
	}}
	for _, tool := range []agenttoolports.AgentTool{
		NewCreateHazardousMaterialTool(materials),
		NewUpdateHazardousMaterialTool(materials),
	} {
		assert.Equal(t, agent.TierPropose, tool.Policy().MaxTier, tool.Name())
		require.NotNil(t, tool.Policy().TaintHold, tool.Name())
	}

	update := NewUpdateHazardousMaterialTool(materials)
	params := agenttoolstest.ExecuteParams(map[string]any{
		paramHazmatID:     stored.ID.String(),
		"placardRequired": true,
		"class":           "HazardClass8",
	})
	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		materials.Guard,
		func() (*agentpreview.ToolPreview, error) {
			return update.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Equal(
		t,
		"HazardClass8",
		agenttoolstest.FieldByPath(t, agenttoolstest.PreviewChange(t, preview, 0), "class").After,
	)
	require.NoError(t, update.Execute(t.Context(), params))
	assert.True(t, materials.Updated.PlacardRequired)
	assert.Equal(t, "1203", materials.Updated.UNNumber)
}
