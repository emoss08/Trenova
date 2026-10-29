package equipmenttools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/masterdatakit"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/equipment/app/tractorservice"
	"github.com/emoss08/trenova/internal/modules/equipment/app/trailerservice"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentcontinuity"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/tractorports"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/trailerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeTractors struct {
	*agenttoolstest.FakeMaster[tractor.Tractor]
	plan    *tractorservice.LocatePlan
	located *tractorports.LocateTractorRequest
}

func (f *fakeTractors) Get(
	context.Context,
	tractorports.GetTractorByIDRequest,
) (*tractor.Tractor, error) {
	return f.MasterGet()
}

func (f *fakeTractors) PlanCreate(
	ctx context.Context,
	e *tractor.Tractor,
) (*tractor.Tractor, error) {
	return f.MasterPlanCreate(ctx, e)
}

func (f *fakeTractors) Create(
	ctx context.Context,
	e *tractor.Tractor,
	a *permissionports.RequestActor,
) (*tractor.Tractor, error) {
	return f.MasterCreate(ctx, e, a)
}

func (f *fakeTractors) PlanUpdate(
	ctx context.Context,
	e *tractor.Tractor,
) (*kernelports.RecordChange[tractor.Tractor], error) {
	return f.MasterPlanUpdate(ctx, e)
}

func (f *fakeTractors) Update(
	ctx context.Context,
	e *tractor.Tractor,
	a *permissionports.RequestActor,
) (*tractor.Tractor, error) {
	return f.MasterUpdate(ctx, e, a)
}

func (f *fakeTractors) PlanLocate(
	context.Context,
	*tractorports.LocateTractorRequest,
) (*tractorservice.LocatePlan, error) {
	if f.Refusal != nil {
		return nil, f.Refusal
	}

	return f.plan, nil
}

func (f *fakeTractors) Locate(
	_ context.Context,
	req *tractorports.LocateTractorRequest,
) (*equipmentcontinuity.EquipmentContinuity, error) {
	if err := f.Guard.Write(); err != nil {
		return nil, err
	}
	f.located = req

	return &equipmentcontinuity.EquipmentContinuity{}, nil
}

func TestCreateTractor_DefaultsWhatTheFormDefaults(t *testing.T) {
	t.Parallel()

	tractors := &fakeTractors{FakeMaster: &agenttoolstest.FakeMaster[tractor.Tractor]{
		Guard: &agenttoolstest.WriteGuard{},
		SetID: func(tr *tractor.Tractor) { tr.ID = pulid.MustNew("trac_") },
	}}
	tool := NewCreateTractorTool(tractors, agenttoolstest.TexasState())
	params := agenttoolstest.IdempotentParams(map[string]any{
		masterdatakit.MdCode:               "T-100",
		paramPrimaryWorkerID:               pulid.MustNew("wrk_").String(),
		masterdatakit.MdEquipmentTypeID:    pulid.MustNew("et_").String(),
		masterdatakit.MdManufacturerID:     pulid.MustNew("em_").String(),
		masterdatakit.MdState:              "TX",
		masterdatakit.MdRegistrationExpiry: "2027-03-31",
		masterdatakit.MdVIN:                "1fuj",
	})

	result, err := tool.(agenttoolports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	created := tractors.Created
	assert.Equal(t, domaintypes.EquipmentStatusAvailable, created.Status)
	assert.Equal(t, domaintypes.IFTAFuelTypeDiesel, created.FuelType)
	assert.True(t, created.IFTAQualified)
	assert.Equal(t, "1FUJ", created.Vin)
	require.NotNil(t, created.RegistrationExpiry)
	assert.Equal(t, int64(1_806_451_200), *created.RegistrationExpiry)
	assert.Equal(t, tractorRecordEntity, result.Record.EntityType)

	bad := agenttoolstest.IdempotentParams(map[string]any{
		masterdatakit.MdCode:               "T-100",
		paramPrimaryWorkerID:               pulid.MustNew("wrk_").String(),
		masterdatakit.MdEquipmentTypeID:    pulid.MustNew("et_").String(),
		masterdatakit.MdManufacturerID:     pulid.MustNew("em_").String(),
		masterdatakit.MdRegistrationExpiry: "03/31/2027",
	})
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), bad))
}

func TestLocateTractor_ShowsWhereItWasAndGoes(t *testing.T) {
	t.Parallel()

	from := pulid.MustNew("loc_")
	to := &location.Location{ID: pulid.MustNew("loc_"), Name: "Dallas yard"}
	unit := &tractor.Tractor{ID: pulid.MustNew("trac_"), Code: "T-100", Version: 5}
	tractors := &fakeTractors{
		FakeMaster: &agenttoolstest.FakeMaster[tractor.Tractor]{
			Guard: &agenttoolstest.WriteGuard{},
		},
		plan: &tractorservice.LocatePlan{
			Tractor:  unit,
			Location: to,
			Current:  &equipmentcontinuity.EquipmentContinuity{CurrentLocationID: from},
		},
	}
	tool := NewLocateTractorTool(tractors)
	params := agenttoolstest.ExecuteParams(map[string]any{
		toolparams.ParamTractorID: unit.ID.String(),
		paramNewLocationID:        to.ID.String(),
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		tractors.Guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Contains(t, preview.Summary, "Would record tractor T-100 at Dallas yard")
	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, to.ID, tractors.located.NewLocationID)

	tractors.Refusal = errortypes.NewBusinessError(
		"Tractor is currently in progress on another move",
	)
	refused, err := tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	agenttoolstest.RequireWarning(t, refused, agentpreview.PreviewWarningWouldFail)
}

type fakeTrailers struct {
	*agenttoolstest.FakeMaster[trailer.Trailer]
	plan    *trailerservice.LocatePlan
	located *trailerports.LocateTrailerRequest
}

func (f *fakeTrailers) Get(
	context.Context,
	trailerports.GetTrailerByIDRequest,
) (*trailer.Trailer, error) {
	return f.MasterGet()
}

func (f *fakeTrailers) PlanCreate(
	ctx context.Context,
	e *trailer.Trailer,
) (*trailer.Trailer, error) {
	return f.MasterPlanCreate(ctx, e)
}

func (f *fakeTrailers) Create(
	ctx context.Context,
	e *trailer.Trailer,
	a *permissionports.RequestActor,
) (*trailer.Trailer, error) {
	return f.MasterCreate(ctx, e, a)
}

func (f *fakeTrailers) PlanUpdate(
	ctx context.Context,
	e *trailer.Trailer,
) (*kernelports.RecordChange[trailer.Trailer], error) {
	return f.MasterPlanUpdate(ctx, e)
}

func (f *fakeTrailers) Update(
	ctx context.Context,
	e *trailer.Trailer,
	a *permissionports.RequestActor,
) (*trailer.Trailer, error) {
	return f.MasterUpdate(ctx, e, a)
}

func (f *fakeTrailers) PlanLocate(
	context.Context,
	*trailerports.LocateTrailerRequest,
	*permissionports.RequestActor,
) (*trailerservice.LocatePlan, error) {
	return f.plan, nil
}

func (f *fakeTrailers) Locate(
	_ context.Context,
	req *trailerports.LocateTrailerRequest,
	_ *permissionports.RequestActor,
) (*equipmentcontinuity.EquipmentContinuity, error) {
	if err := f.Guard.Write(); err != nil {
		return nil, err
	}
	f.located = req

	return &equipmentcontinuity.EquipmentContinuity{}, nil
}

func TestLocateTrailer_ShowsTheReRatedShipmentAndWaitsForAPerson(t *testing.T) {
	t.Parallel()

	unit := &trailer.Trailer{ID: pulid.MustNew("tr_"), Code: "TR-9", Version: 1}
	previous := &shipment.Shipment{
		ID:                pulid.MustNew("shp_"),
		ProNumber:         "PRO-77",
		TotalChargeAmount: decimal.NewNullDecimal(decimal.RequireFromString("1000")),
		Version:           4,
	}
	updated := *previous
	updated.TotalChargeAmount = decimal.NewNullDecimal(decimal.RequireFromString("1080"))
	trailers := &fakeTrailers{
		FakeMaster: &agenttoolstest.FakeMaster[trailer.Trailer]{
			Guard: &agenttoolstest.WriteGuard{},
		},
		plan: &trailerservice.LocatePlan{
			Trailer:  unit,
			Location: &location.Location{ID: pulid.MustNew("loc_"), Name: "Drop yard"},
			Current: &equipmentcontinuity.EquipmentContinuity{
				CurrentLocationID: pulid.MustNew("loc_"),
			},
			Previous: previous,
			Updated:  &updated,
		},
	}
	tool := NewLocateTrailerTool(trailers)
	args := map[string]any{
		paramTrailerID:     unit.ID.String(),
		paramNewLocationID: trailers.plan.Location.ID.String(),
	}

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		trailers.Guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(
				t.Context(),
				agenttoolstest.ExecuteParams(args),
			)
		},
	)
	assert.Contains(t, preview.Summary, "re-rating it")
	shipmentChange := agenttoolstest.PreviewChange(t, preview, 1)
	assert.Equal(t, permission.ResourceShipment, shipmentChange.Resource)
	require.NotNil(t, shipmentChange.Money)

	require.ErrorIs(
		t,
		tool.Execute(t.Context(), agenttoolstest.ExecuteParams(args)),
		toolspec.ErrNeedsAPersonsApproval,
	)
	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.ApprovedParams(args)))
	assert.Equal(t, unit.ID, trailers.located.TrailerID)
	assert.Equal(t, []agent.EgressClass{agent.EgressMoney}, tool.Policy().Egress)
}
