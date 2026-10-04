package onboardingservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/locationcategory"
	"github.com/emoss08/trenova/internal/core/domain/servicetype"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/formulatemplateservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingCreator[T any] struct {
	created []T
	setID   func(T)
	failAt  int
	err     error
}

func (r *recordingCreator[T]) Create(_ context.Context, entity T, _ *services.RequestActor) (T, error) {
	if r.err != nil && len(r.created) == r.failAt {
		var zero T
		return zero, r.err
	}
	if r.setID != nil {
		r.setID(entity)
	}
	r.created = append(r.created, entity)
	return entity, nil
}

type fakeTemplates struct {
	installed []*formulatemplate.FormulaTemplate
}

func (f *fakeTemplates) InstallStandards(
	context.Context,
	pagination.TenantInfo,
) (*formulatemplateservice.InstallStandardsResponse, error) {
	return &formulatemplateservice.InstallStandardsResponse{Installed: f.installed}, nil
}

type fakeTemplateFinder struct {
	found []*formulatemplate.FormulaTemplate
}

func (f *fakeTemplateFinder) FindByNames(
	context.Context,
	repositories.GetFormulaTemplatesByNamesRequest,
) ([]*formulatemplate.FormulaTemplate, error) {
	return f.found, nil
}

type fakeStates struct {
	state *usstate.UsState
	err   error
}

func (f *fakeStates) GetByAbbreviation(context.Context, string) (*usstate.UsState, error) {
	return f.state, f.err
}

type sampleHarness struct {
	customers      *recordingCreator[*customer.Customer]
	locations      *recordingCreator[*location.Location]
	locationTypes  *recordingCreator[*locationcategory.LocationCategory]
	workers        *recordingCreator[*worker.Worker]
	tractors       *recordingCreator[*tractor.Tractor]
	trailers       *recordingCreator[*trailer.Trailer]
	equipmentTypes *recordingCreator[*equipmenttype.EquipmentType]
	manufacturers  *recordingCreator[*equipmentmanufacturer.EquipmentManufacturer]
	serviceTypes   *recordingCreator[*servicetype.ServiceType]
	shipmentTypes  *recordingCreator[*shipmenttype.ShipmentType]
	shipments      *recordingCreator[*shipment.Shipment]
	templates      *fakeTemplates
	finder         *fakeTemplateFinder
	loader         *SampleData
}

func newSampleHarness() *sampleHarness {
	h := &sampleHarness{
		customers:      &recordingCreator[*customer.Customer]{setID: func(e *customer.Customer) { e.ID = pulid.MustNew("cus_") }},
		locations:      &recordingCreator[*location.Location]{setID: func(e *location.Location) { e.ID = pulid.MustNew("loc_") }},
		locationTypes:  &recordingCreator[*locationcategory.LocationCategory]{setID: func(e *locationcategory.LocationCategory) { e.ID = pulid.MustNew("lc_") }},
		workers:        &recordingCreator[*worker.Worker]{setID: func(e *worker.Worker) { e.ID = pulid.MustNew("wrk_") }},
		tractors:       &recordingCreator[*tractor.Tractor]{},
		trailers:       &recordingCreator[*trailer.Trailer]{},
		equipmentTypes: &recordingCreator[*equipmenttype.EquipmentType]{setID: func(e *equipmenttype.EquipmentType) { e.ID = pulid.MustNew("et_") }},
		manufacturers:  &recordingCreator[*equipmentmanufacturer.EquipmentManufacturer]{setID: func(e *equipmentmanufacturer.EquipmentManufacturer) { e.ID = pulid.MustNew("em_") }},
		serviceTypes:   &recordingCreator[*servicetype.ServiceType]{setID: func(e *servicetype.ServiceType) { e.ID = pulid.MustNew("st_") }},
		shipmentTypes:  &recordingCreator[*shipmenttype.ShipmentType]{setID: func(e *shipmenttype.ShipmentType) { e.ID = pulid.MustNew("sht_") }},
		shipments:      &recordingCreator[*shipment.Shipment]{},
		templates:      &fakeTemplates{},
		finder:         &fakeTemplateFinder{},
	}
	h.loader = &SampleData{
		customers:      h.customers,
		locations:      h.locations,
		locationTypes:  h.locationTypes,
		workers:        h.workers,
		tractors:       h.tractors,
		trailers:       h.trailers,
		equipmentTypes: h.equipmentTypes,
		manufacturers:  h.manufacturers,
		serviceTypes:   h.serviceTypes,
		shipmentTypes:  h.shipmentTypes,
		shipments:      h.shipments,
		templates:      h.templates,
		templateFinder: h.finder,
		states:         &fakeStates{state: &usstate.UsState{ID: pulid.MustNew("us_"), Abbreviation: "TX"}},
		now: func() time.Time {
			return time.Date(2026, time.October, 4, 15, 0, 0, 0, time.UTC)
		},
	}
	return h
}

func sampleRequest() *SampleDataRequest {
	info := pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
	return &SampleDataRequest{
		TenantInfo: info,
		Actor:      &services.RequestActor{UserID: info.UserID},
		Organization: &tenant.Organization{
			ID:           info.OrgID,
			AddressLine1: "1 Main St",
			City:         "Dallas",
			StateID:      pulid.MustNew("us_"),
			PostalCode:   "75201",
			Timezone:     "America/Chicago",
		},
	}
}

func TestSampleDataCreatesTheDocumentedSet(t *testing.T) {
	t.Parallel()

	h := newSampleHarness()
	templateID := pulid.MustNew("ft_")
	h.templates.installed = []*formulatemplate.FormulaTemplate{
		{ID: pulid.MustNew("ft_"), Name: "Per Mile"},
		{ID: templateID, Name: flatRateTemplateName},
	}
	req := sampleRequest()

	require.NoError(t, h.loader.Load(t.Context(), req))

	assert.Len(t, h.customers.created, 2)
	assert.Len(t, h.locations.created, 4)
	assert.Len(t, h.workers.created, 1)
	assert.Len(t, h.tractors.created, 1)
	assert.Len(t, h.trailers.created, 1)
	assert.Len(t, h.shipments.created, 2)
	assert.Len(t, h.equipmentTypes.created, 2)

	for _, c := range h.customers.created {
		assert.Equal(t, req.TenantInfo.OrgID, c.OrganizationID)
		assert.False(t, c.StateID.IsNil())
		multiErr := errortypes.NewMultiError()
		c.Validate(multiErr)
		assert.False(t, multiErr.HasErrors(), multiErr.Error())
	}
	for _, l := range h.locations.created {
		assert.Equal(t, h.locationTypes.created[0].ID, l.LocationCategoryID)
		l.NormalizeGeofence()
		multiErr := errortypes.NewMultiError()
		l.Validate(multiErr)
		assert.False(t, multiErr.HasErrors(), multiErr.Error())
	}

	driver := h.workers.created[0]
	assert.Equal(t, req.Organization.StateID, driver.StateID)
	multiErr := errortypes.NewMultiError()
	driver.Validate(multiErr)
	assert.False(t, multiErr.HasErrors(), multiErr.Error())

	truck := h.tractors.created[0]
	assert.Equal(t, driver.ID, truck.PrimaryWorkerID)
	multiErr = errortypes.NewMultiError()
	truck.Validate(multiErr)
	assert.False(t, multiErr.HasErrors(), multiErr.Error())

	multiErr = errortypes.NewMultiError()
	h.trailers.created[0].Validate(multiErr)
	assert.False(t, multiErr.HasErrors(), multiErr.Error())

	for _, equipment := range h.equipmentTypes.created {
		multiErr = errortypes.NewMultiError()
		equipment.Validate(multiErr)
		assert.False(t, multiErr.HasErrors(), multiErr.Error())
	}

	for _, s := range h.shipments.created {
		assert.Equal(t, templateID, s.FormulaTemplateID)
		assert.Equal(t, req.TenantInfo.UserID, s.EnteredByID)
		multiErr = errortypes.NewMultiError()
		s.Validate(multiErr)
		assert.False(t, multiErr.HasErrors(), multiErr.Error())
		require.Len(t, s.Moves, 1)
		require.Len(t, s.Moves[0].Stops, 2)
		pickup, delivery := s.Moves[0].Stops[0], s.Moves[0].Stops[1]
		assert.Equal(t, shipment.StopTypePickup, pickup.Type)
		assert.Equal(t, shipment.StopTypeDelivery, delivery.Type)
		assert.Less(t, pickup.ScheduledWindowStart, delivery.ScheduledWindowStart)
		assert.Greater(t, pickup.ScheduledWindowStart, h.loader.now().Unix())
		for _, stop := range s.Moves[0].Stops {
			multiErr = errortypes.NewMultiError()
			stop.Validate(multiErr)
			assert.False(t, multiErr.HasErrors(), multiErr.Error())
		}
	}
}

func TestSampleDataFindsAnAlreadyInstalledFlatRate(t *testing.T) {
	t.Parallel()

	h := newSampleHarness()
	templateID := pulid.MustNew("ft_")
	h.finder.found = []*formulatemplate.FormulaTemplate{{ID: templateID, Name: flatRateTemplateName}}

	require.NoError(t, h.loader.Load(t.Context(), sampleRequest()))
	assert.Equal(t, templateID, h.shipments.created[0].FormulaTemplateID)
}

func TestSampleDataWithoutAFlatRateFails(t *testing.T) {
	t.Parallel()

	h := newSampleHarness()
	err := h.loader.Load(t.Context(), sampleRequest())
	require.Error(t, err)
	assert.Empty(t, h.customers.created)
}

func TestSampleDataStopsAtTheFirstQuotaRefusal(t *testing.T) {
	t.Parallel()

	h := newSampleHarness()
	h.finder.found = []*formulatemplate.FormulaTemplate{{ID: pulid.MustNew("ft_"), Name: flatRateTemplateName}}
	h.shipments.err = errortypes.NewQuotaExceededError("shipments.total", 1, 1, "free_demo")
	h.shipments.failAt = 1

	err := h.loader.Load(t.Context(), sampleRequest())
	require.True(t, errortypes.IsQuotaExceededError(err))
	assert.Len(t, h.shipments.created, 1)
}

func TestSampleDataNeedsTheOrganization(t *testing.T) {
	t.Parallel()

	h := newSampleHarness()
	req := sampleRequest()
	req.Organization = nil
	require.True(t, errortypes.IsBusinessError(h.loader.Load(t.Context(), req)))
}

func TestSampleDataNeedsTheSampleState(t *testing.T) {
	t.Parallel()

	h := newSampleHarness()
	h.loader.states = &fakeStates{err: errortypes.NewNotFoundError("State not found")}
	require.Error(t, h.loader.Load(t.Context(), sampleRequest()))
	assert.Empty(t, h.locationTypes.created)
}
