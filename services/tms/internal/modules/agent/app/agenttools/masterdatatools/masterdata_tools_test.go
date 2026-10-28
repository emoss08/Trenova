package masterdatatools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
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
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/modules/reference/ports/commodityports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/hazardousmaterialports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeMaster[T any] struct {
	guard   *agenttoolstest.WriteGuard
	stored  *T
	refusal error
	created *T
	updated *T
	status  any
	setID   func(*T)
}

func (f *fakeMaster[T]) get() (*T, error) {
	copied := *f.stored

	return &copied, nil
}

func (f *fakeMaster[T]) planCreate(_ context.Context, entity *T) (*T, error) {
	if f.refusal != nil {
		return nil, f.refusal
	}

	return entity, nil
}

func (f *fakeMaster[T]) create(
	ctx context.Context,
	entity *T,
	_ *permissionports.RequestActor,
) (*T, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	planned, err := f.planCreate(ctx, entity)
	if err != nil {
		return nil, err
	}
	f.setID(planned)
	f.created = planned

	return planned, nil
}

func (f *fakeMaster[T]) planUpdate(
	_ context.Context,
	entity *T,
) (*kernelports.RecordChange[T], error) {
	if f.refusal != nil {
		return nil, f.refusal
	}

	return &kernelports.RecordChange[T]{Before: f.stored, After: entity}, nil
}

func (f *fakeMaster[T]) update(
	_ context.Context,
	entity *T,
	_ *permissionports.RequestActor,
) (*T, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.updated = entity

	return entity, nil
}

type fakeCustomers struct {
	*fakeMaster[customer.Customer]
}

func (f fakeCustomers) Get(
	context.Context,
	customerports.GetCustomerByIDRequest,
) (*customer.Customer, error) {
	return f.get()
}

func (f fakeCustomers) PlanCreate(
	ctx context.Context,
	e *customer.Customer,
) (*customer.Customer, error) {
	return f.planCreate(ctx, e)
}

func (f fakeCustomers) Create(
	ctx context.Context,
	e *customer.Customer,
	a *permissionports.RequestActor,
) (*customer.Customer, error) {
	return f.create(ctx, e, a)
}

func (f fakeCustomers) PlanUpdate(
	ctx context.Context,
	e *customer.Customer,
) (*kernelports.RecordChange[customer.Customer], error) {
	return f.planUpdate(ctx, e)
}

func (f fakeCustomers) Update(
	ctx context.Context,
	e *customer.Customer,
	a *permissionports.RequestActor,
) (*customer.Customer, error) {
	return f.update(ctx, e, a)
}

func (f fakeCustomers) PlanBulkUpdateStatus(
	_ context.Context,
	req *customerports.BulkUpdateCustomerStatusRequest,
) ([]kernelports.RecordChange[customer.Customer], error) {
	after := *f.stored
	after.Status = req.Status

	return []kernelports.RecordChange[customer.Customer]{{Before: f.stored, After: &after}}, nil
}

func (f fakeCustomers) BulkUpdateStatus(
	_ context.Context,
	req *customerports.BulkUpdateCustomerStatusRequest,
) ([]*customer.Customer, error) {
	f.status = req

	return []*customer.Customer{f.stored}, f.guard.Write()
}

func TestCreateCustomer_StatusEmailRecipientsAreHeldAsLeavingTheOrganization(t *testing.T) {
	t.Parallel()

	states := texasState()
	customers := fakeCustomers{&fakeMaster[customer.Customer]{
		guard: &agenttoolstest.WriteGuard{},
		setID: func(c *customer.Customer) { c.ID = pulid.MustNew("cus_") },
	}}
	tool := NewCreateCustomerTool(customers, states)
	args := map[string]any{
		mdCode:         "ACME",
		MdName:         "Acme Foods",
		mdAddressLine1: "1 Main St",
		mdCity:         "Dallas",
		mdState:        "TX",
		mdPostalCode:   "75201",
	}

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		customers.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), idempotentParams(args))
		},
	)
	assert.Equal(
		t,
		"TX",
		agenttoolstest.FieldByPath(t, agenttoolstest.PreviewChange(t, preview, 0), mdState).After,
	)
	assert.Equal(t, agent.EgressInternal, tool.Policy().Classify(idempotentParams(args)).Egress)

	args[customerUpdateRecipients] = "ops@acme.example"
	call := tool.Policy().Classify(idempotentParams(args))
	assert.Equal(t, agent.EgressExternalRecipient, call.Egress)
	assert.Equal(t, agent.TierPropose, call.MaxTier)

	result, err := tool.(agenttoolports.ToolResultReporter).ExecuteWithResult(t.Context(),
		idempotentParams(args))
	require.NoError(t, err)
	assert.Equal(t, states.State.ID, customers.created.StateID)
	assert.Equal(t, customer.StatusUpdateNone, customers.created.StatusUpdatePreference)
	assert.Equal(t, changepreview.CustomerRecordEntity, result.Record.EntityType)

	delete(args, mdState)
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(),
		idempotentParams(args)))
}

func TestUpdateCustomer_LeavesTheBillingProfileAlone(t *testing.T) {
	t.Parallel()

	states := texasState()
	stored := &customer.Customer{
		ID:             pulid.MustNew("cus_"),
		Code:           "ACME",
		Name:           "Acme Foods",
		StateID:        states.State.ID,
		Status:         domaintypes.StatusActive,
		BillingProfile: &customer.CustomerBillingProfile{},
		Version:        7,
	}
	customers := fakeCustomers{
		&fakeMaster[customer.Customer]{guard: &agenttoolstest.WriteGuard{}, stored: stored},
	}
	tool := NewUpdateCustomerTool(customers, states)
	params := agenttoolstest.ExecuteParams(
		map[string]any{toolparams.ParamCustomerID: stored.ID.String(), mdCity: "Plano"},
	)

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, "Plano", customers.updated.City)
	assert.Nil(t, customers.updated.BillingProfile)
	assert.Equal(t, int64(7), customers.updated.Version)

	status := NewUpdateCustomerStatusTool(customers, states)
	require.NoError(t, status.Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"customerIds":             []any{stored.ID.String()},
		changepreview.FieldStatus: "Inactive",
	})))
	require.IsType(t, &customerports.BulkUpdateCustomerStatusRequest{}, customers.status)
}

type fakeCommodities struct {
	*fakeMaster[commodity.Commodity]
}

func (f fakeCommodities) Get(
	context.Context,
	commodityports.GetCommodityByIDRequest,
) (*commodity.Commodity, error) {
	return f.get()
}

func (f fakeCommodities) PlanCreate(
	ctx context.Context,
	e *commodity.Commodity,
) (*commodity.Commodity, error) {
	return f.planCreate(ctx, e)
}

func (f fakeCommodities) Create(
	ctx context.Context,
	e *commodity.Commodity,
	a *permissionports.RequestActor,
) (*commodity.Commodity, error) {
	return f.create(ctx, e, a)
}

func (f fakeCommodities) PlanUpdate(
	ctx context.Context,
	e *commodity.Commodity,
) (*kernelports.RecordChange[commodity.Commodity], error) {
	return f.planUpdate(ctx, e)
}

func (f fakeCommodities) Update(
	ctx context.Context,
	e *commodity.Commodity,
	a *permissionports.RequestActor,
) (*commodity.Commodity, error) {
	return f.update(ctx, e, a)
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
	return nil, f.guard.Write()
}

func TestCreateCommodity_ReadsDecimalsAndTheFreightClass(t *testing.T) {
	t.Parallel()

	commodities := fakeCommodities{&fakeMaster[commodity.Commodity]{
		guard: &agenttoolstest.WriteGuard{},
		setID: func(c *commodity.Commodity) { c.ID = pulid.MustNew("com_") },
	}}
	tool := NewCreateCommodityTool(commodities)
	params := idempotentParams(map[string]any{
		MdName:           "Frozen peas",
		mdDescription:    "Frozen vegetables on pallets",
		"freightClass":   "Class70",
		"weightPerUnit":  "42.5",
		"minTemperature": -10,
		"maxTemperature": 0,
		"stackable":      true,
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		commodities.guard,
		func() (*agent.ToolPreview, error) {
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
	assert.Equal(t, commodity.FreightClass70, commodities.created.FreightClass)
	require.NotNil(t, commodities.created.WeightPerUnit)
	assert.InDelta(t, 42.5, *commodities.created.WeightPerUnit, 0.001)
	assert.Equal(t, -10, *commodities.created.MinTemperature)

	bad := idempotentParams(map[string]any{
		MdName: "Peas", mdDescription: "Peas", "freightClass": "Class71",
	})
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), bad))

	status := NewUpdateCommodityStatusTool(commodities)
	statusParams := agenttoolstest.ExecuteParams(map[string]any{
		"commodityIds":            []any{pulid.MustNew("com_").String()},
		changepreview.FieldStatus: "Inactive",
	})
	statusPreview, err := status.(agenttoolports.ToolPreviewer).Preview(t.Context(), statusParams)
	require.NoError(t, err)
	agenttoolstest.RequireWarning(t, statusPreview, agent.PreviewWarningWouldFail)
}

type fakeHazmats struct {
	*fakeMaster[hazardousmaterial.HazardousMaterial]
}

func (f fakeHazmats) Get(
	context.Context,
	hazardousmaterialports.GetHazardousMaterialByIDRequest,
) (*hazardousmaterial.HazardousMaterial, error) {
	return f.get()
}

func (f fakeHazmats) PlanCreate(
	ctx context.Context,
	e *hazardousmaterial.HazardousMaterial,
) (*hazardousmaterial.HazardousMaterial, error) {
	return f.planCreate(ctx, e)
}

func (f fakeHazmats) Create(
	ctx context.Context,
	e *hazardousmaterial.HazardousMaterial,
	a *permissionports.RequestActor,
) (*hazardousmaterial.HazardousMaterial, error) {
	return f.create(ctx, e, a)
}

func (f fakeHazmats) PlanUpdate(
	ctx context.Context,
	e *hazardousmaterial.HazardousMaterial,
) (*kernelports.RecordChange[hazardousmaterial.HazardousMaterial], error) {
	return f.planUpdate(ctx, e)
}

func (f fakeHazmats) Update(
	ctx context.Context,
	e *hazardousmaterial.HazardousMaterial,
	a *permissionports.RequestActor,
) (*hazardousmaterial.HazardousMaterial, error) {
	return f.update(ctx, e, a)
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
	return nil, f.guard.Write()
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
	materials := fakeHazmats{&fakeMaster[hazardousmaterial.HazardousMaterial]{
		guard:  &agenttoolstest.WriteGuard{},
		stored: stored,
		setID:  func(h *hazardousmaterial.HazardousMaterial) { h.ID = pulid.MustNew("hm_") },
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
		materials.guard,
		func() (*agent.ToolPreview, error) {
			return update.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Equal(
		t,
		"HazardClass8",
		agenttoolstest.FieldByPath(t, agenttoolstest.PreviewChange(t, preview, 0), "class").After,
	)
	require.NoError(t, update.Execute(t.Context(), params))
	assert.True(t, materials.updated.PlacardRequired)
	assert.Equal(t, "1203", materials.updated.UNNumber)
}

type fakeLocations struct {
	*fakeMaster[location.Location]
	bulk *locationports.BulkUpdateLocationStatusRequest
}

func (f *fakeLocations) Get(
	context.Context,
	locationports.GetLocationByIDRequest,
) (*location.Location, error) {
	return f.get()
}

func (f *fakeLocations) PlanUpdate(
	ctx context.Context,
	e *location.Location,
) (*kernelports.RecordChange[location.Location], error) {
	return f.planUpdate(ctx, e)
}

func (f *fakeLocations) Update(
	ctx context.Context,
	e *location.Location,
	a *permissionports.RequestActor,
) (*location.Location, error) {
	return f.update(ctx, e, a)
}

func (f *fakeLocations) PlanBulkUpdateStatus(
	_ context.Context,
	req *locationports.BulkUpdateLocationStatusRequest,
) ([]kernelports.RecordChange[location.Location], error) {
	after := *f.stored
	after.Status = req.Status

	return []kernelports.RecordChange[location.Location]{{Before: f.stored, After: &after}}, nil
}

func (f *fakeLocations) BulkUpdateStatus(
	_ context.Context,
	req *locationports.BulkUpdateLocationStatusRequest,
) ([]*location.Location, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.bulk = req

	return []*location.Location{f.stored}, nil
}

func TestUpdateLocation_ChangesTheAddressAndKeepsTheCode(t *testing.T) {
	t.Parallel()

	states := texasState()
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
	locations := &fakeLocations{fakeMaster: &fakeMaster[location.Location]{
		guard: &agenttoolstest.WriteGuard{}, stored: stored,
	}}
	tool := NewUpdateLocationTool(locations, states)
	params := agenttoolstest.ExecuteParams(map[string]any{
		paramLocationID: stored.ID.String(),
		mdAddressLine1:  "9 Dock St",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		locations.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Equal(
		t,
		"9 Dock St",
		agenttoolstest.FieldByPath(
			t,
			agenttoolstest.PreviewChange(t, preview, 0),
			mdAddressLine1,
		).After,
	)
	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, "DALWH01", locations.updated.Code)
	assert.Equal(t, "9 Dock St", locations.updated.AddressLine1)

	status := NewUpdateLocationStatusTool(locations, states)
	require.NoError(t, status.Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"locationIds":             []any{stored.ID.String()},
		changepreview.FieldStatus: "Inactive",
	})))
	assert.Equal(t, domaintypes.StatusInactive, locations.bulk.Status)
}

type fakeTractors struct {
	*fakeMaster[tractor.Tractor]
	plan    *tractorservice.LocatePlan
	located *tractorports.LocateTractorRequest
}

func (f *fakeTractors) Get(
	context.Context,
	tractorports.GetTractorByIDRequest,
) (*tractor.Tractor, error) {
	return f.get()
}

func (f *fakeTractors) PlanCreate(
	ctx context.Context,
	e *tractor.Tractor,
) (*tractor.Tractor, error) {
	return f.planCreate(ctx, e)
}

func (f *fakeTractors) Create(
	ctx context.Context,
	e *tractor.Tractor,
	a *permissionports.RequestActor,
) (*tractor.Tractor, error) {
	return f.create(ctx, e, a)
}

func (f *fakeTractors) PlanUpdate(
	ctx context.Context,
	e *tractor.Tractor,
) (*kernelports.RecordChange[tractor.Tractor], error) {
	return f.planUpdate(ctx, e)
}

func (f *fakeTractors) Update(
	ctx context.Context,
	e *tractor.Tractor,
	a *permissionports.RequestActor,
) (*tractor.Tractor, error) {
	return f.update(ctx, e, a)
}

func (f *fakeTractors) PlanLocate(
	context.Context,
	*tractorports.LocateTractorRequest,
) (*tractorservice.LocatePlan, error) {
	if f.refusal != nil {
		return nil, f.refusal
	}

	return f.plan, nil
}

func (f *fakeTractors) Locate(
	_ context.Context,
	req *tractorports.LocateTractorRequest,
) (*equipmentcontinuity.EquipmentContinuity, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.located = req

	return &equipmentcontinuity.EquipmentContinuity{}, nil
}

func TestCreateTractor_DefaultsWhatTheFormDefaults(t *testing.T) {
	t.Parallel()

	tractors := &fakeTractors{fakeMaster: &fakeMaster[tractor.Tractor]{
		guard: &agenttoolstest.WriteGuard{},
		setID: func(tr *tractor.Tractor) { tr.ID = pulid.MustNew("trac_") },
	}}
	tool := NewCreateTractorTool(tractors, texasState())
	params := idempotentParams(map[string]any{
		mdCode:               "T-100",
		paramPrimaryWorkerID: pulid.MustNew("wrk_").String(),
		mdEquipmentTypeID:    pulid.MustNew("et_").String(),
		mdManufacturerID:     pulid.MustNew("em_").String(),
		mdState:              "TX",
		mdRegistrationExpiry: "2027-03-31",
		mdVIN:                "1fuj",
	})

	result, err := tool.(agenttoolports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	created := tractors.created
	assert.Equal(t, domaintypes.EquipmentStatusAvailable, created.Status)
	assert.Equal(t, domaintypes.IFTAFuelTypeDiesel, created.FuelType)
	assert.True(t, created.IFTAQualified)
	assert.Equal(t, "1FUJ", created.Vin)
	require.NotNil(t, created.RegistrationExpiry)
	assert.Equal(t, int64(1_806_451_200), *created.RegistrationExpiry)
	assert.Equal(t, tractorRecordEntity, result.Record.EntityType)

	bad := idempotentParams(map[string]any{
		mdCode:               "T-100",
		paramPrimaryWorkerID: pulid.MustNew("wrk_").String(),
		mdEquipmentTypeID:    pulid.MustNew("et_").String(),
		mdManufacturerID:     pulid.MustNew("em_").String(),
		mdRegistrationExpiry: "03/31/2027",
	})
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), bad))
}

func TestLocateTractor_ShowsWhereItWasAndGoes(t *testing.T) {
	t.Parallel()

	from := pulid.MustNew("loc_")
	to := &location.Location{ID: pulid.MustNew("loc_"), Name: "Dallas yard"}
	unit := &tractor.Tractor{ID: pulid.MustNew("trac_"), Code: "T-100", Version: 5}
	tractors := &fakeTractors{
		fakeMaster: &fakeMaster[tractor.Tractor]{guard: &agenttoolstest.WriteGuard{}},
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
		tractors.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Contains(t, preview.Summary, "Would record tractor T-100 at Dallas yard")
	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, to.ID, tractors.located.NewLocationID)

	tractors.refusal = errortypes.NewBusinessError(
		"Tractor is currently in progress on another move",
	)
	refused, err := tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	agenttoolstest.RequireWarning(t, refused, agent.PreviewWarningWouldFail)
}

type fakeTrailers struct {
	*fakeMaster[trailer.Trailer]
	plan    *trailerservice.LocatePlan
	located *trailerports.LocateTrailerRequest
}

func (f *fakeTrailers) Get(
	context.Context,
	trailerports.GetTrailerByIDRequest,
) (*trailer.Trailer, error) {
	return f.get()
}

func (f *fakeTrailers) PlanCreate(
	ctx context.Context,
	e *trailer.Trailer,
) (*trailer.Trailer, error) {
	return f.planCreate(ctx, e)
}

func (f *fakeTrailers) Create(
	ctx context.Context,
	e *trailer.Trailer,
	a *permissionports.RequestActor,
) (*trailer.Trailer, error) {
	return f.create(ctx, e, a)
}

func (f *fakeTrailers) PlanUpdate(
	ctx context.Context,
	e *trailer.Trailer,
) (*kernelports.RecordChange[trailer.Trailer], error) {
	return f.planUpdate(ctx, e)
}

func (f *fakeTrailers) Update(
	ctx context.Context,
	e *trailer.Trailer,
	a *permissionports.RequestActor,
) (*trailer.Trailer, error) {
	return f.update(ctx, e, a)
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
	if err := f.guard.Write(); err != nil {
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
		fakeMaster: &fakeMaster[trailer.Trailer]{guard: &agenttoolstest.WriteGuard{}},
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
		trailers.guard,
		func() (*agent.ToolPreview, error) {
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
