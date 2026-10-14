package ratingtest

import (
	"context"
	"slices"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/equipmenttypeports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/usstateports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationports"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/engine"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/resolver"
	"github.com/emoss08/trenova/internal/modules/rating/app/ratinghydration"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratematrixports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype"
	"github.com/emoss08/trenova/internal/modules/reference/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/modules/reference/ports/commodityports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/hazardousmaterialports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/servicetypeports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/shipmenttypeports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type RelationStore struct {
	mu      sync.Mutex
	fetches map[string][][]pulid.ID
	tenants map[string][]pagination.TenantInfo

	Customers     map[pulid.ID]*customer.Customer
	ServiceTypes  map[pulid.ID]*servicetype.ServiceType
	ShipmentTypes map[pulid.ID]*shipmenttype.ShipmentType
	Equipment     map[pulid.ID]*equipmenttype.EquipmentType
	Locations     map[pulid.ID]*location.Location
	States        map[pulid.ID]*usstate.UsState
	Commodities   map[pulid.ID]*commodity.Commodity
	Hazmats       map[pulid.ID]*hazardousmaterial.HazardousMaterial
}

func NewRelationStore() *RelationStore {
	return &RelationStore{
		fetches:       make(map[string][][]pulid.ID),
		tenants:       make(map[string][]pagination.TenantInfo),
		Customers:     make(map[pulid.ID]*customer.Customer),
		ServiceTypes:  make(map[pulid.ID]*servicetype.ServiceType),
		ShipmentTypes: make(map[pulid.ID]*shipmenttype.ShipmentType),
		Equipment:     make(map[pulid.ID]*equipmenttype.EquipmentType),
		Locations:     make(map[pulid.ID]*location.Location),
		States:        make(map[pulid.ID]*usstate.UsState),
		Commodities:   make(map[pulid.ID]*commodity.Commodity),
		Hazmats:       make(map[pulid.ID]*hazardousmaterial.HazardousMaterial),
	}
}

func (s *RelationStore) Fetches(kind string) [][]pulid.ID {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.fetches[kind])
}

func (s *RelationStore) Tenants(kind string) []pagination.TenantInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.tenants[kind])
}

func fetchFrom[R any](
	s *RelationStore,
	kind string,
	tenantInfo pagination.TenantInfo,
	ids []pulid.ID,
	records map[pulid.ID]*R,
) []*R {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.fetches[kind] = append(s.fetches[kind], slices.Clone(ids))
	s.tenants[kind] = append(s.tenants[kind], tenantInfo)

	out := make([]*R, 0, len(ids))
	for _, id := range ids {
		if record, ok := records[id]; ok {
			copied := *record
			out = append(out, &copied)
		}
	}
	return out
}

type storeCustomers struct {
	customerports.CustomerRepository
	s *RelationStore
}

func (r storeCustomers) GetByIDs(
	_ context.Context,
	req customerports.GetCustomersByIDsRequest,
) ([]*customer.Customer, error) {
	return fetchFrom(r.s, "customer", req.TenantInfo, req.CustomerIDs, r.s.Customers), nil
}

type storeServiceTypes struct {
	servicetypeports.ServiceTypeRepository
	s *RelationStore
}

func (r storeServiceTypes) GetByIDs(
	_ context.Context,
	req servicetypeports.GetServiceTypesByIDsRequest,
) ([]*servicetype.ServiceType, error) {
	return fetchFrom(r.s, "serviceType", req.TenantInfo, req.ServiceTypeIDs, r.s.ServiceTypes), nil
}

type storeShipmentTypes struct {
	shipmenttypeports.ShipmentTypeRepository
	s *RelationStore
}

func (r storeShipmentTypes) GetByIDs(
	_ context.Context,
	req shipmenttypeports.GetShipmentTypesByIDsRequest,
) ([]*shipmenttype.ShipmentType, error) {
	return fetchFrom(
		r.s,
		"shipmentType",
		req.TenantInfo,
		req.ShipmentTypeIDs,
		r.s.ShipmentTypes,
	), nil
}

type storeEquipmentTypes struct {
	equipmenttypeports.EquipmentTypeRepository
	s *RelationStore
}

func (r storeEquipmentTypes) GetByIDs(
	_ context.Context,
	req equipmenttypeports.GetEquipmentTypesByIDsRequest,
) ([]*equipmenttype.EquipmentType, error) {
	return fetchFrom(r.s, "equipmentType", req.TenantInfo, req.EquipmentTypeIDs, r.s.Equipment), nil
}

type storeLocations struct {
	locationports.LocationRepository
	s *RelationStore
}

func (r storeLocations) GetByIDs(
	_ context.Context,
	req locationports.GetLocationsByIDsRequest,
) ([]*location.Location, error) {
	return fetchFrom(r.s, "location", req.TenantInfo, req.LocationIDs, r.s.Locations), nil
}

type storeStates struct {
	usstateports.UsStateRepository
	s *RelationStore
}

func (r storeStates) GetByIDs(_ context.Context, ids []pulid.ID) ([]*usstate.UsState, error) {
	return fetchFrom(r.s, "state", pagination.TenantInfo{}, ids, r.s.States), nil
}

type storeCommodities struct {
	commodityports.CommodityRepository
	s *RelationStore
}

func (r storeCommodities) GetByIDs(
	_ context.Context,
	req commodityports.GetCommoditiesByIDsRequest,
) ([]*commodity.Commodity, error) {
	return fetchFrom(r.s, "commodity", req.TenantInfo, req.CommodityIDs, r.s.Commodities), nil
}

type storeHazmats struct {
	hazardousmaterialports.HazardousMaterialRepository
	s *RelationStore
}

func (r storeHazmats) GetByIDs(
	_ context.Context,
	req hazardousmaterialports.GetHazardousMaterialsByIDsRequest,
) ([]*hazardousmaterial.HazardousMaterial, error) {
	return fetchFrom(r.s, "hazmat", req.TenantInfo, req.HazardousMaterialIDs, r.s.Hazmats), nil
}

func NewEnvironmentBuilder(t *testing.T) *engine.EnvironmentBuilder {
	t.Helper()

	registry, err := formula.NewSchemaRegistry()
	require.NoError(t, err)

	return engine.NewEnvironmentBuilder(engine.EnvironmentBuilderParams{
		Registry: registry,
		Resolver: resolver.NewResolver(),
	})
}

func HydratorParams(
	env *engine.EnvironmentBuilder,
	store *RelationStore,
) ratinghydration.Params {
	return ratinghydration.Params{
		Environment:       env,
		CustomerRepo:      storeCustomers{s: store},
		ServiceTypeRepo:   storeServiceTypes{s: store},
		ShipmentTypeRepo:  storeShipmentTypes{s: store},
		EquipmentTypeRepo: storeEquipmentTypes{s: store},
		LocationRepo:      storeLocations{s: store},
		StateRepo:         storeStates{s: store},
		CommodityRepo:     storeCommodities{s: store},
		HazmatRepo:        storeHazmats{s: store},
	}
}

func NewHydrator(t *testing.T, store *RelationStore) *ratinghydration.Hydrator {
	t.Helper()

	hydrator, err := ratinghydration.New(HydratorParams(NewEnvironmentBuilder(t), store))
	require.NoError(t, err)

	return hydrator
}

type templateStore struct {
	formulatemplateports.FormulaTemplateRepository
	templates map[pulid.ID]*formulatemplate.FormulaTemplate
}

func (r templateStore) GetByID(
	_ context.Context,
	req formulatemplateports.GetFormulaTemplateByIDRequest,
) (*formulatemplate.FormulaTemplate, error) {
	template, ok := r.templates[req.TemplateID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Formula template not found")
	}
	copied := *template
	return &copied, nil
}

type noScheduledVersions struct {
	formulatemplateports.FormulaTemplateVersionRepository
}

func (noScheduledVersions) ListScheduled(
	context.Context,
	*formulatemplateports.ListScheduledVersionsRequest,
) ([]*formulatemplate.FormulaTemplateVersion, error) {
	return nil, nil
}

func (noScheduledVersions) ListApproved(
	context.Context,
	*formulatemplateports.ListScheduledVersionsRequest,
) ([]*formulatemplate.FormulaTemplateVersion, error) {
	return nil, nil
}

type noLookupTables struct {
	ratematrixports.RateMatrixRepository
}

func (noLookupTables) GetLookupStamp(context.Context, pagination.TenantInfo) (string, error) {
	return "", nil
}

func (noLookupTables) GetLookupData(
	context.Context,
	*ratematrixports.GetRateMatrixLookupDataRequest,
) ([]*ratematrixports.RateMatrixLookupData, error) {
	return nil, nil
}

func NewFormulaService(
	t *testing.T,
	templates ...*formulatemplate.FormulaTemplate,
) *formula.Service {
	t.Helper()

	registry, err := formula.NewSchemaRegistry()
	require.NoError(t, err)

	res := resolver.NewResolver()
	env := engine.NewEnvironmentBuilder(engine.EnvironmentBuilderParams{
		Registry: registry,
		Resolver: res,
	})
	eng, err := engine.NewEngine(engine.Params{
		Registry:   registry,
		Resolver:   res,
		EnvBuilder: env,
	})
	require.NoError(t, err)

	byID := make(map[pulid.ID]*formulatemplate.FormulaTemplate, len(templates))
	for _, template := range templates {
		byID[template.ID] = template
	}

	return formula.NewService(formula.ServiceParams{
		Logger:         zap.NewNop(),
		Registry:       registry,
		Engine:         eng,
		Resolver:       res,
		Repo:           templateStore{templates: byID},
		VersionRepo:    noScheduledVersions{},
		RateMatrixRepo: noLookupTables{},
	})
}
