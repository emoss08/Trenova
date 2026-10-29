package inboundfixtures

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider/aiproviderdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate/usstatedb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier/carrierdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/holdreason"
	"github.com/emoss08/trenova/internal/modules/reference/domain/holdreason/holdreasondb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/seeds/fixtures/agentfixtures"
	"github.com/emoss08/trenova/internal/seeds/fixtures/tenantfixtures"
	"github.com/emoss08/trenova/internal/seeds/fixtures/triagefixtures"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	seedExceptionHoldTag = "OSD"
)

func loadSeedCarriers(
	ctx context.Context,
	db *keel.DB,
	org *tenant.Organization,
) (map[string]*carrier.Carrier, error) {
	rows, err := carrierdb.Carriers.AllTenants().Where(
		carrierdb.Carriers.OrganizationID.Eq(org.ID),
		carrierdb.Carriers.BusinessUnitID.Eq(org.BusinessUnitID),
		carrierdb.Carriers.Code.In(triagefixtures.CarrierCodes...),
	).All(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("load carriers: %w", err)
	}

	byCode := make(map[string]*carrier.Carrier, len(rows))
	for _, row := range rows {
		byCode[row.Code] = row
	}
	for _, code := range triagefixtures.CarrierCodes {
		if _, ok := byCode[code]; !ok {
			return nil, fmt.Errorf(
				"need seeded carrier %s: %w",
				code,
				tenantfixtures.ErrFixtureMissing,
			)
		}
	}

	return byCode, nil
}

func loadSeedDriver(
	ctx context.Context,
	db *keel.DB,
	org *tenant.Organization,
) (*worker.Worker, error) {
	driver, err := workerdb.Workers.AllTenants().Where(
		workerdb.Workers.OrganizationID.Eq(org.ID),
		workerdb.Workers.BusinessUnitID.Eq(org.BusinessUnitID),
		workerdb.Workers.Email.Eq(triagefixtures.DriverEmail),
	).First(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("need the seeded driver %s: %w", triagefixtures.DriverEmail, err)
	}

	return driver, nil
}

func ensureSeedBroker(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	org *tenant.Organization,
) (*customer.Customer, error) {
	existing, err := customerdb.Customers.AllTenants().Where(
		customerdb.Customers.OrganizationID.Eq(org.ID),
		customerdb.Customers.BusinessUnitID.Eq(org.BusinessUnitID),
		customerdb.Customers.Code.Eq(triagefixtures.BrokerCode),
	).First(ctx, db)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, keel.ErrNotFound) {
		return nil, fmt.Errorf("look up the seeded broker: %w", err)
	}

	state, err := tenantfixtures.StateByAbbreviation(ctx, db, "VA")
	if err != nil {
		return nil, fmt.Errorf("get VA state: %w", err)
	}

	broker := &customer.Customer{
		ID:                     pulid.MustNew("cus_"),
		OrganizationID:         org.ID,
		BusinessUnitID:         org.BusinessUnitID,
		StateID:                state.ID,
		Status:                 domaintypes.StatusActive,
		AllowConsolidation:     true,
		Code:                   triagefixtures.BrokerCode,
		Name:                   "Blue Ridge Brokerage",
		AddressLine1:           "10 Franklin Rd SW",
		City:                   "Roanoke",
		PostalCode:             "24011",
		BrokerVettingEnabled:   true,
		DOTNumber:              "3141592",
		MCNumber:               "1123581",
		StatusUpdatePreference: customer.StatusUpdateNone,
		ConsolidationPriority:  1,
	}
	if err = customerdb.Customers.Create(ctx, db, broker); err != nil {
		return nil, fmt.Errorf("insert the seeded broker: %w", err)
	}

	return broker, nil
}

func loadOpenLane(
	ctx context.Context,
	db *keel.DB,
	ship *shipment.Shipment,
) (*triagefixtures.Lane, error) {
	type laneStop struct {
		Type        shipment.StopType
		City        string
		State       string
		WindowStart int64
	}

	move := shipmentdb.ShipmentMoves
	stop := shipmentdb.Stops
	loc := locationdb.Locations
	state := usstatedb.UsStates
	rows, err := keel.Into[laneStop](ctx, db, move.AllTenants().
		Join(stop,
			stop.ShipmentMoveID.EqCol(move.ID),
			stop.OrganizationID.EqCol(move.OrganizationID),
			stop.BusinessUnitID.EqCol(move.BusinessUnitID),
		).
		Join(loc,
			loc.ID.EqCol(stop.LocationID),
			loc.OrganizationID.EqCol(stop.OrganizationID),
			loc.BusinessUnitID.EqCol(stop.BusinessUnitID),
		).
		Join(state, state.ID.EqCol(loc.StateID)).
		Pick(stop.Type, loc.City, state.Abbreviation, stop.ScheduledWindowStart).
		Where(
			move.ShipmentID.Eq(ship.ID),
			move.OrganizationID.Eq(ship.OrganizationID),
			move.BusinessUnitID.Eq(ship.BusinessUnitID),
		).
		Order(move.Sequence.Asc(), stop.Sequence.Asc()))
	if err != nil {
		return nil, fmt.Errorf("load the lane of %s: %w", ship.ProNumber, err)
	}

	lane := &triagefixtures.Lane{Shipment: ship}
	for _, row := range rows {
		place := row.City + ", " + row.State
		switch row.Type {
		case shipment.StopTypePickup, shipment.StopTypeSplitPickup:
			if lane.Origin == "" {
				lane.Origin = place
				lane.OriginCity = row.City
				lane.OriginState = row.State
				lane.PickupAt = row.WindowStart
			}
		case shipment.StopTypeDelivery, shipment.StopTypeSplitDelivery:
			lane.Destination = place
			lane.DestinationCity = row.City
			lane.DestinationState = row.State
		}
	}
	if lane.Origin == "" || lane.Destination == "" || lane.PickupAt == 0 {
		return nil, fmt.Errorf("%s has no pickup and delivery: %w",
			ship.ProNumber, tenantfixtures.ErrFixtureMissing)
	}

	return lane, nil
}

func ensureExceptionHoldReason(ctx context.Context, db *keel.DB, org *tenant.Organization) error {
	reason := &holdreason.HoldReason{
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		Type:           holdreason.HoldTypeOperational,
		Code:           seedExceptionHoldTag,
		Label:          "Overage, shortage or damage",
		Description: "Freight arrived short, over or damaged; " +
			"billing waits until it is settled.",
		DefaultSeverity:      holdreason.HoldSeverityBlocking,
		DefaultBlocksBilling: true,
		Active:               true,
		SortOrder:            80,
	}
	if err := holdreasondb.HoldReasons.Create(ctx, db, reason); err != nil {
		return fmt.Errorf("insert the OS&D hold reason: %w", err)
	}

	return nil
}

func routeInboundClassification(ctx context.Context, db *keel.DB, org *tenant.Organization) error {
	p := aiproviderdb.Providers
	task := string(aitask.TaskInboundClassification)
	if _, err := p.AllTenants().Where(
		p.OrganizationID.Eq(org.ID),
		p.BusinessUnitID.Eq(org.BusinessUnitID),
		p.Name.Eq(agentfixtures.SeedAIProviderLocalName),
		keel.Not(p.Tasks.Has(task)),
	).Update(ctx, db, p.Tasks.Append(task)); err != nil {
		return fmt.Errorf("route inbound classification to the local model: %w", err)
	}

	return nil
}
