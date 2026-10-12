package development

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/pkg/buncolgen/aiprovidercol"
	"github.com/emoss08/trenova/pkg/buncolgen/carriercol"
	"github.com/emoss08/trenova/pkg/buncolgen/customercol"
	"github.com/emoss08/trenova/pkg/buncolgen/holdreasoncol"
	"github.com/emoss08/trenova/pkg/buncolgen/locationcol"
	"github.com/emoss08/trenova/pkg/buncolgen/shipmentcol"
	"github.com/emoss08/trenova/pkg/buncolgen/usstatecol"
	"github.com/emoss08/trenova/pkg/buncolgen/workercol"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/reference/domain/holdreason"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

const (
	seedBrokerCode       = "BLRG"
	seedDriverEmail      = "carlos.rivera@example.com"
	seedOpenLanePro      = "SEED-SHP-001"
	seedDetentionPro     = "SEED-DET-008"
	seedExceptionHoldTag = "OSD"
)

var seedInboundCarrierCodes = []string{"IHF", "HPT", "LSH"}

type seedLane struct {
	shipment         *shipment.Shipment
	origin           string
	originCity       string
	originState      string
	destination      string
	destinationCity  string
	destinationState string
	pickupAt         int64
}

func loadSeedCarriers(
	ctx context.Context,
	tx bun.Tx,
	org *tenant.Organization,
) (map[string]*carrier.Carrier, error) {
	cols := carriercol.CarrierColumns
	rows := make([]*carrier.Carrier, 0, len(seedInboundCarrierCodes))
	if err := tx.NewSelect().
		Model(&rows).
		Where(cols.OrganizationID.Eq(), org.ID).
		Where(cols.BusinessUnitID.Eq(), org.BusinessUnitID).
		Where(cols.Code.In(), bun.List(seedInboundCarrierCodes)).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("load carriers: %w", err)
	}

	byCode := make(map[string]*carrier.Carrier, len(rows))
	for _, row := range rows {
		byCode[row.Code] = row
	}
	for _, code := range seedInboundCarrierCodes {
		if _, ok := byCode[code]; !ok {
			return nil, fmt.Errorf("need seeded carrier %s: %w", code, seedhelpers.ErrEntityNotFound)
		}
	}

	return byCode, nil
}

func loadSeedDriver(
	ctx context.Context,
	tx bun.Tx,
	org *tenant.Organization,
) (*worker.Worker, error) {
	cols := workercol.WorkerColumns
	driver := new(worker.Worker)
	if err := tx.NewSelect().
		Model(driver).
		Where(cols.OrganizationID.Eq(), org.ID).
		Where(cols.BusinessUnitID.Eq(), org.BusinessUnitID).
		Where(cols.Email.Eq(), seedDriverEmail).
		Limit(1).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("need the seeded driver %s: %w", seedDriverEmail, err)
	}

	return driver, nil
}

func ensureSeedBroker(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	org *tenant.Organization,
) (*customer.Customer, error) {
	cols := customercol.CustomerColumns
	existing := new(customer.Customer)
	err := tx.NewSelect().
		Model(existing).
		Where(cols.OrganizationID.Eq(), org.ID).
		Where(cols.BusinessUnitID.Eq(), org.BusinessUnitID).
		Where(cols.Code.Eq(), seedBrokerCode).
		Limit(1).
		Scan(ctx)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("look up the seeded broker: %w", err)
	}

	stateID, err := sc.GetStateByAbbreviation(ctx, "VA")
	if err != nil {
		return nil, fmt.Errorf("get VA state: %w", err)
	}

	broker := &customer.Customer{
		ID:                   pulid.MustNew("cus_"),
		OrganizationID:       org.ID,
		BusinessUnitID:       org.BusinessUnitID,
		StateID:              stateID,
		Status:               domaintypes.StatusActive,
		AllowConsolidation:   true,
		Code:                 seedBrokerCode,
		Name:                 "Blue Ridge Brokerage",
		AddressLine1:         "10 Franklin Rd SW",
		City:                 "Roanoke",
		PostalCode:           "24011",
		BrokerVettingEnabled: true,
		DOTNumber:            "3141592",
		MCNumber:             "1123581",
	}
	if _, err = tx.NewInsert().Model(broker).Exec(ctx); err != nil {
		return nil, fmt.Errorf("insert the seeded broker: %w", err)
	}

	return broker, nil
}

func loadOpenLane(
	ctx context.Context,
	tx bun.Tx,
	ship *shipment.Shipment,
) (*seedLane, error) {
	move := shipmentcol.ShipmentMoveColumns
	stop := shipmentcol.StopColumns
	loc := locationcol.LocationColumns
	state := usstatecol.UsStateColumns

	rows := make([]struct {
		Type        shipment.StopType `bun:"type"`
		City        string            `bun:"city"`
		State       string            `bun:"abbreviation"`
		WindowStart int64             `bun:"scheduled_window_start"`
	}, 0, 2)
	if err := tx.NewSelect().
		Model((*shipment.ShipmentMove)(nil)).
		ColumnExpr(stop.Type.Qualified()).
		ColumnExpr(loc.City.Qualified()).
		ColumnExpr(state.Abbreviation.Qualified()).
		ColumnExpr(stop.ScheduledWindowStart.Qualified()).
		Join("JOIN "+shipmentcol.StopTable.Name+" AS "+shipmentcol.StopTable.Alias).
		JoinOn(stop.ShipmentMoveID.EqColumn(move.ID)).
		JoinOn(stop.OrganizationID.EqColumn(move.OrganizationID)).
		JoinOn(stop.BusinessUnitID.EqColumn(move.BusinessUnitID)).
		Join("JOIN "+locationcol.LocationTable.Name+" AS "+locationcol.LocationTable.Alias).
		JoinOn(loc.ID.EqColumn(stop.LocationID)).
		JoinOn(loc.OrganizationID.EqColumn(stop.OrganizationID)).
		JoinOn(loc.BusinessUnitID.EqColumn(stop.BusinessUnitID)).
		Join("JOIN "+usstatecol.UsStateTable.Name+" AS "+usstatecol.UsStateTable.Alias).
		JoinOn(state.ID.EqColumn(loc.StateID)).
		Where(move.ShipmentID.Eq(), ship.ID).
		Where(move.OrganizationID.Eq(), ship.OrganizationID).
		Where(move.BusinessUnitID.Eq(), ship.BusinessUnitID).
		Order(move.Sequence.OrderAsc(), stop.Sequence.OrderAsc()).
		Scan(ctx, &rows); err != nil {
		return nil, fmt.Errorf("load the lane of %s: %w", ship.ProNumber, err)
	}

	lane := &seedLane{shipment: ship}
	for _, row := range rows {
		place := row.City + ", " + row.State
		switch row.Type {
		case shipment.StopTypePickup, shipment.StopTypeSplitPickup:
			if lane.origin == "" {
				lane.origin = place
				lane.originCity = row.City
				lane.originState = row.State
				lane.pickupAt = row.WindowStart
			}
		case shipment.StopTypeDelivery, shipment.StopTypeSplitDelivery:
			lane.destination = place
			lane.destinationCity = row.City
			lane.destinationState = row.State
		}
	}
	if lane.origin == "" || lane.destination == "" || lane.pickupAt == 0 {
		return nil, fmt.Errorf("%s has no pickup and delivery: %w",
			ship.ProNumber, seedhelpers.ErrEntityNotFound)
	}

	return lane, nil
}

func (l *seedLane) pickupDay(loc *time.Location) string {
	return time.Unix(l.pickupAt, 0).In(loc).Format(time.DateOnly)
}

func ensureExceptionHoldReason(ctx context.Context, tx bun.Tx, org *tenant.Organization) error {
	cols := holdreasoncol.HoldReasonColumns
	exists, err := tx.NewSelect().
		Model((*holdreason.HoldReason)(nil)).
		Where(cols.OrganizationID.Eq(), org.ID).
		Where(cols.BusinessUnitID.Eq(), org.BusinessUnitID).
		Where(cols.Code.Eq(), seedExceptionHoldTag).
		Exists(ctx)
	if err != nil {
		return fmt.Errorf("look up the OS&D hold reason: %w", err)
	}
	if exists {
		return nil
	}

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
	if _, err = tx.NewInsert().Model(reason).Exec(ctx); err != nil {
		return fmt.Errorf("insert the OS&D hold reason: %w", err)
	}

	return nil
}

func routeInboundClassification(ctx context.Context, tx bun.Tx, org *tenant.Organization) error {
	cols := aiprovidercol.ProviderColumns
	if _, err := tx.NewUpdate().
		Model((*aiprovider.Provider)(nil)).
		Set(cols.Tasks.SetExpr("array_append({}, ?)"), string(aiprovider.TaskInboundClassification)).
		Where(cols.OrganizationID.Eq(), org.ID).
		Where(cols.BusinessUnitID.Eq(), org.BusinessUnitID).
		Where(cols.Name.Eq(), SeedAIProviderLocalName).
		Where(cols.Tasks.Expr("NOT (? = ANY({}))"), string(aiprovider.TaskInboundClassification)).
		Exec(ctx); err != nil {
		return fmt.Errorf("route inbound classification to the local model: %w", err)
	}

	return nil
}
