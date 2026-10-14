package ratinghydration

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"

	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/equipmenttypeports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/usstateports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationports"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/engine"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
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
	"go.uber.org/fx"
)

const shipmentSchemaID = "shipment"

type Params struct {
	fx.In

	Environment       *engine.EnvironmentBuilder
	CustomerRepo      customerports.CustomerRepository
	ServiceTypeRepo   servicetypeports.ServiceTypeRepository
	ShipmentTypeRepo  shipmenttypeports.ShipmentTypeRepository
	EquipmentTypeRepo equipmenttypeports.EquipmentTypeRepository
	LocationRepo      locationports.LocationRepository
	StateRepo         usstateports.UsStateRepository
	CommodityRepo     commodityports.CommodityRepository
	HazmatRepo        hazardousmaterialports.HazardousMaterialRepository
}

type Hydrator struct {
	steps []relationStep
}

type relationStep struct {
	relation string
	load     func(ctx context.Context, h *hydration) error
}

var ErrMixedTenants = errors.New(
	"shipments of different organizations cannot be loaded for rating together",
)

type hydration struct {
	entities   []*shipment.Shipment
	tenantInfo pagination.TenantInfo
	refresh    bool
}

//nolint:gocritic // fx param structs are passed by value
func New(p Params) (*Hydrator, error) {
	loaders := relationLoaders(&p)

	relations := p.Environment.GetRequiredPreloads(shipmentSchemaID)
	if len(relations) == 0 {
		return nil, fmt.Errorf("formula schema %q declares no relations", shipmentSchemaID)
	}

	missing := make([]string, 0)
	steps := make([]relationStep, 0, len(relations))
	for _, relation := range relations {
		load, ok := loaders[relation]
		if !ok {
			missing = append(missing, relation)
			continue
		}
		if load != nil {
			steps = append(steps, relationStep{relation: relation, load: load})
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf(
			"formula schema %q reads relations rating cannot load: %s",
			shipmentSchemaID,
			strings.Join(missing, ", "),
		)
	}

	sort.SliceStable(steps, func(i, j int) bool {
		return strings.Count(steps[i].relation, ".") < strings.Count(steps[j].relation, ".")
	})

	return &Hydrator{steps: steps}, nil
}

func (h *Hydrator) Relations() []string {
	relations := make([]string, 0, len(h.steps))
	for _, step := range h.steps {
		relations = append(relations, step.relation)
	}
	return relations
}

func (h *Hydrator) HydrateShipment(
	ctx context.Context,
	req *ratequoteports.HydrateShipmentRequest,
) error {
	if req == nil || req.Shipment == nil {
		return nil
	}

	return h.HydrateShipments(ctx, &ratequoteports.HydrateShipmentsRequest{
		Shipments:  []*shipment.Shipment{req.Shipment},
		TenantInfo: req.TenantInfo,
		Refresh:    req.Refresh,
	})
}

func (h *Hydrator) HydrateShipments(
	ctx context.Context,
	req *ratequoteports.HydrateShipmentsRequest,
) error {
	if req == nil {
		return nil
	}

	entities := make([]*shipment.Shipment, 0, len(req.Shipments))
	for _, entity := range req.Shipments {
		if entity != nil {
			entities = append(entities, entity)
		}
	}
	if len(entities) == 0 {
		return nil
	}

	tenantInfo, err := batchTenant(req.TenantInfo, entities)
	if err != nil {
		return err
	}

	state := &hydration{
		entities:   entities,
		tenantInfo: tenantInfo,
		refresh:    req.Refresh,
	}

	for _, step := range h.steps {
		if err = step.load(ctx, state); err != nil {
			return err
		}
	}

	return nil
}

func batchTenant(
	requested pagination.TenantInfo,
	entities []*shipment.Shipment,
) (pagination.TenantInfo, error) {
	if requested.OrgID.IsNotNil() && requested.BuID.IsNotNil() {
		return requested, nil
	}

	first := entities[0]
	for _, entity := range entities[1:] {
		if entity.OrganizationID != first.OrganizationID ||
			entity.BusinessUnitID != first.BusinessUnitID {
			return pagination.TenantInfo{}, ErrMixedTenants
		}
	}

	return pagination.TenantInfo{
		OrgID:  first.OrganizationID,
		BuID:   first.BusinessUnitID,
		UserID: requested.UserID,
	}, nil
}

func relationLoaders(p *Params) map[string]func(context.Context, *hydration) error {
	return map[string]func(context.Context, *hydration) error{
		"Moves":                      nil,
		"Moves.Stops":                nil,
		"Commodities":                nil,
		"AdditionalCharges":          nil,
		"Customer":                   loadCustomers(p.CustomerRepo),
		"ServiceType":                loadServiceTypes(p.ServiceTypeRepo),
		"ShipmentType":               loadShipmentTypes(p.ShipmentTypeRepo),
		"TractorType":                loadTractorTypes(p.EquipmentTypeRepo),
		"TrailerType":                loadTrailerTypes(p.EquipmentTypeRepo),
		"Moves.Stops.Location":       loadStopLocations(p.LocationRepo),
		"Moves.Stops.Location.State": loadLocationStates(p.StateRepo),
		"Commodities.Commodity":      loadCommodities(p.CommodityRepo),
		"Commodities.Commodity.HazardousMaterial": loadHazardousMaterials(p.HazmatRepo),
	}
}

type belongsTo[H, R any] struct {
	field      string
	label      string
	foreignKey func(H) pulid.ID
	get        func(H) *R
	set        func(H, *R)
	key        func(*R) pulid.ID
	fetch      func(context.Context, pagination.TenantInfo, []pulid.ID) ([]*R, error)
}

func hydrateBelongsTo[H, R any](
	ctx context.Context,
	h *hydration,
	link belongsTo[H, R],
	holders []H,
) error {
	pending := make(map[pulid.ID][]H, len(holders))
	for _, holder := range holders {
		id := link.foreignKey(holder)
		if id.IsNil() {
			link.set(holder, nil)
			continue
		}
		if current := link.get(holder); current != nil && !h.refresh && link.key(current) == id {
			continue
		}
		pending[id] = append(pending[id], holder)
	}
	if len(pending) == 0 {
		return nil
	}

	ids := slices.Sorted(maps.Keys(pending))
	loaded, err := link.fetch(ctx, h.tenantInfo, ids)
	if err != nil {
		return err
	}

	for _, record := range loaded {
		if record == nil {
			continue
		}
		waiting, ok := pending[link.key(record)]
		if !ok {
			continue
		}
		for _, holder := range waiting {
			link.set(holder, record)
		}
		delete(pending, link.key(record))
	}

	if len(pending) > 0 {
		unknown := slices.Sorted(maps.Keys(pending))
		for _, id := range unknown {
			for _, holder := range pending[id] {
				link.set(holder, nil)
			}
		}
		return errortypes.NewValidationError(
			link.field,
			errortypes.ErrInvalidReference,
			"{0} {1} does not exist in this organization",
			link.label,
			unknown[0].String(),
		)
	}

	return nil
}

func shipmentHolder(h *hydration) []*shipment.Shipment {
	return h.entities
}

func loadCustomers(repo customerports.CustomerRepository) func(context.Context, *hydration) error {
	link := belongsTo[*shipment.Shipment, customer.Customer]{
		field:      "customerId",
		label:      "Customer",
		foreignKey: func(s *shipment.Shipment) pulid.ID { return s.CustomerID },
		get:        func(s *shipment.Shipment) *customer.Customer { return s.Customer },
		set:        func(s *shipment.Shipment, c *customer.Customer) { s.Customer = c },
		key:        func(c *customer.Customer) pulid.ID { return c.ID },
		fetch: func(
			ctx context.Context,
			tenantInfo pagination.TenantInfo,
			ids []pulid.ID,
		) ([]*customer.Customer, error) {
			return repo.GetByIDs(ctx, customerports.GetCustomersByIDsRequest{
				TenantInfo:  tenantInfo,
				CustomerIDs: ids,
			})
		},
	}
	return func(ctx context.Context, h *hydration) error {
		return hydrateBelongsTo(ctx, h, link, shipmentHolder(h))
	}
}

func loadServiceTypes(
	repo servicetypeports.ServiceTypeRepository,
) func(context.Context, *hydration) error {
	link := belongsTo[*shipment.Shipment, servicetype.ServiceType]{
		field:      "serviceTypeId",
		label:      "Service type",
		foreignKey: func(s *shipment.Shipment) pulid.ID { return s.ServiceTypeID },
		get:        func(s *shipment.Shipment) *servicetype.ServiceType { return s.ServiceType },
		set:        func(s *shipment.Shipment, v *servicetype.ServiceType) { s.ServiceType = v },
		key:        func(v *servicetype.ServiceType) pulid.ID { return v.ID },
		fetch: func(
			ctx context.Context,
			tenantInfo pagination.TenantInfo,
			ids []pulid.ID,
		) ([]*servicetype.ServiceType, error) {
			return repo.GetByIDs(ctx, servicetypeports.GetServiceTypesByIDsRequest{
				TenantInfo:     tenantInfo,
				ServiceTypeIDs: ids,
			})
		},
	}
	return func(ctx context.Context, h *hydration) error {
		return hydrateBelongsTo(ctx, h, link, shipmentHolder(h))
	}
}

func loadShipmentTypes(
	repo shipmenttypeports.ShipmentTypeRepository,
) func(context.Context, *hydration) error {
	link := belongsTo[*shipment.Shipment, shipmenttype.ShipmentType]{
		field:      "shipmentTypeId",
		label:      "Shipment type",
		foreignKey: func(s *shipment.Shipment) pulid.ID { return s.ShipmentTypeID },
		get:        func(s *shipment.Shipment) *shipmenttype.ShipmentType { return s.ShipmentType },
		set:        func(s *shipment.Shipment, v *shipmenttype.ShipmentType) { s.ShipmentType = v },
		key:        func(v *shipmenttype.ShipmentType) pulid.ID { return v.ID },
		fetch: func(
			ctx context.Context,
			tenantInfo pagination.TenantInfo,
			ids []pulid.ID,
		) ([]*shipmenttype.ShipmentType, error) {
			return repo.GetByIDs(ctx, shipmenttypeports.GetShipmentTypesByIDsRequest{
				TenantInfo:      tenantInfo,
				ShipmentTypeIDs: ids,
			})
		},
	}
	return func(ctx context.Context, h *hydration) error {
		return hydrateBelongsTo(ctx, h, link, shipmentHolder(h))
	}
}

func equipmentTypeFetcher(
	repo equipmenttypeports.EquipmentTypeRepository,
) func(context.Context, pagination.TenantInfo, []pulid.ID) ([]*equipmenttype.EquipmentType, error) {
	return func(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		ids []pulid.ID,
	) ([]*equipmenttype.EquipmentType, error) {
		return repo.GetByIDs(ctx, equipmenttypeports.GetEquipmentTypesByIDsRequest{
			TenantInfo:       tenantInfo,
			EquipmentTypeIDs: ids,
		})
	}
}

func equipmentTypeKey(v *equipmenttype.EquipmentType) pulid.ID {
	return v.ID
}

func loadTractorTypes(
	repo equipmenttypeports.EquipmentTypeRepository,
) func(context.Context, *hydration) error {
	link := belongsTo[*shipment.Shipment, equipmenttype.EquipmentType]{
		field:      "tractorTypeId",
		label:      "Tractor type",
		foreignKey: func(s *shipment.Shipment) pulid.ID { return s.TractorTypeID },
		get:        func(s *shipment.Shipment) *equipmenttype.EquipmentType { return s.TractorType },
		set:        func(s *shipment.Shipment, v *equipmenttype.EquipmentType) { s.TractorType = v },
		key:        equipmentTypeKey,
		fetch:      equipmentTypeFetcher(repo),
	}
	return func(ctx context.Context, h *hydration) error {
		return hydrateBelongsTo(ctx, h, link, shipmentHolder(h))
	}
}

func loadTrailerTypes(
	repo equipmenttypeports.EquipmentTypeRepository,
) func(context.Context, *hydration) error {
	link := belongsTo[*shipment.Shipment, equipmenttype.EquipmentType]{
		field:      "trailerTypeId",
		label:      "Trailer type",
		foreignKey: func(s *shipment.Shipment) pulid.ID { return s.TrailerTypeID },
		get:        func(s *shipment.Shipment) *equipmenttype.EquipmentType { return s.TrailerType },
		set:        func(s *shipment.Shipment, v *equipmenttype.EquipmentType) { s.TrailerType = v },
		key:        equipmentTypeKey,
		fetch:      equipmentTypeFetcher(repo),
	}
	return func(ctx context.Context, h *hydration) error {
		return hydrateBelongsTo(ctx, h, link, shipmentHolder(h))
	}
}

func stopsOf(entities []*shipment.Shipment) []*shipment.Stop {
	stops := make([]*shipment.Stop, 0, len(entities)*2)
	for _, entity := range entities {
		for _, move := range entity.Moves {
			if move == nil {
				continue
			}
			for _, stop := range move.Stops {
				if stop != nil {
					stops = append(stops, stop)
				}
			}
		}
	}
	return stops
}

func loadStopLocations(
	repo locationports.LocationRepository,
) func(context.Context, *hydration) error {
	link := belongsTo[*shipment.Stop, location.Location]{
		field:      "locationId",
		label:      "Location",
		foreignKey: func(s *shipment.Stop) pulid.ID { return s.LocationID },
		get:        func(s *shipment.Stop) *location.Location { return s.Location },
		set:        func(s *shipment.Stop, v *location.Location) { s.Location = v },
		key:        func(v *location.Location) pulid.ID { return v.ID },
		fetch: func(
			ctx context.Context,
			tenantInfo pagination.TenantInfo,
			ids []pulid.ID,
		) ([]*location.Location, error) {
			return repo.GetByIDs(ctx, locationports.GetLocationsByIDsRequest{
				TenantInfo:  tenantInfo,
				LocationIDs: ids,
			})
		},
	}
	return func(ctx context.Context, h *hydration) error {
		return hydrateBelongsTo(ctx, h, link, stopsOf(h.entities))
	}
}

func stopLocationsOf(entities []*shipment.Shipment) []*location.Location {
	stops := stopsOf(entities)
	locations := make([]*location.Location, 0, len(stops))
	for _, stop := range stops {
		if stop.Location != nil {
			locations = append(locations, stop.Location)
		}
	}
	return locations
}

func loadLocationStates(
	repo usstateports.UsStateRepository,
) func(context.Context, *hydration) error {
	link := belongsTo[*location.Location, usstate.UsState]{
		field:      "stateId",
		label:      "State",
		foreignKey: func(l *location.Location) pulid.ID { return l.StateID },
		get:        func(l *location.Location) *usstate.UsState { return l.State },
		set:        func(l *location.Location, v *usstate.UsState) { l.State = v },
		key:        func(v *usstate.UsState) pulid.ID { return v.ID },
		fetch: func(
			ctx context.Context,
			_ pagination.TenantInfo,
			ids []pulid.ID,
		) ([]*usstate.UsState, error) {
			return repo.GetByIDs(ctx, ids)
		},
	}
	return func(ctx context.Context, h *hydration) error {
		return hydrateBelongsTo(ctx, h, link, stopLocationsOf(h.entities))
	}
}

func shipmentCommoditiesOf(entities []*shipment.Shipment) []*shipment.ShipmentCommodity {
	lines := make([]*shipment.ShipmentCommodity, 0, len(entities))
	for _, entity := range entities {
		for _, line := range entity.Commodities {
			if line != nil {
				lines = append(lines, line)
			}
		}
	}
	return lines
}

func loadCommodities(
	repo commodityports.CommodityRepository,
) func(context.Context, *hydration) error {
	link := belongsTo[*shipment.ShipmentCommodity, commodity.Commodity]{
		field:      "commodityId",
		label:      "Commodity",
		foreignKey: func(c *shipment.ShipmentCommodity) pulid.ID { return c.CommodityID },
		get:        func(c *shipment.ShipmentCommodity) *commodity.Commodity { return c.Commodity },
		set:        func(c *shipment.ShipmentCommodity, v *commodity.Commodity) { c.Commodity = v },
		key:        func(v *commodity.Commodity) pulid.ID { return v.ID },
		fetch: func(
			ctx context.Context,
			tenantInfo pagination.TenantInfo,
			ids []pulid.ID,
		) ([]*commodity.Commodity, error) {
			return repo.GetByIDs(ctx, commodityports.GetCommoditiesByIDsRequest{
				TenantInfo:   tenantInfo,
				CommodityIDs: ids,
			})
		},
	}
	return func(ctx context.Context, h *hydration) error {
		return hydrateBelongsTo(ctx, h, link, shipmentCommoditiesOf(h.entities))
	}
}

func commodityRecordsOf(entities []*shipment.Shipment) []*commodity.Commodity {
	records := make([]*commodity.Commodity, 0, len(entities))
	for _, entity := range entities {
		for _, line := range entity.Commodities {
			if line != nil && line.Commodity != nil {
				records = append(records, line.Commodity)
			}
		}
	}
	return records
}

func loadHazardousMaterials(
	repo hazardousmaterialports.HazardousMaterialRepository,
) func(context.Context, *hydration) error {
	link := belongsTo[*commodity.Commodity, hazardousmaterial.HazardousMaterial]{
		field:      "hazardousMaterialId",
		label:      "Hazardous material",
		foreignKey: func(c *commodity.Commodity) pulid.ID { return c.HazardousMaterialID },
		get: func(c *commodity.Commodity) *hazardousmaterial.HazardousMaterial {
			return c.HazardousMaterial
		},
		set: func(c *commodity.Commodity, v *hazardousmaterial.HazardousMaterial) {
			c.HazardousMaterial = v
		},
		key: func(v *hazardousmaterial.HazardousMaterial) pulid.ID { return v.ID },
		fetch: func(
			ctx context.Context,
			tenantInfo pagination.TenantInfo,
			ids []pulid.ID,
		) ([]*hazardousmaterial.HazardousMaterial, error) {
			return repo.GetByIDs(ctx, hazardousmaterialports.GetHazardousMaterialsByIDsRequest{
				TenantInfo:           tenantInfo,
				HazardousMaterialIDs: ids,
			})
		},
	}
	return func(ctx context.Context, h *hydration) error {
		return hydrateBelongsTo(ctx, h, link, commodityRecordsOf(h.entities))
	}
}
