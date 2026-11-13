//go:build integration

package exceptionsourcerepository_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/adapters/postgres/exceptionsourcerepository"
	"github.com/emoss08/trenova/internal/modules/agent/ports/podchaseports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/shared/pulid"
)

func (h *harness) worker(t *testing.T, first, last string, userID pulid.ID) pulid.ID {
	t.Helper()
	id := pulid.MustNew("wrk_")
	var user any
	if userID.IsNotNil() {
		user = userID.String()
	}
	h.exec(t, `INSERT INTO workers (
			id, organization_id, business_unit_id, state_id, first_name, last_name,
			address_line1, city, postal_code, gender, user_id
		) VALUES ($1, $2, $3, $4, $5, $6, '1 Main St', 'Columbus', '43004', 'Other', $7)`,
		id.String(), h.tenant.OrgID.String(), h.tenant.BuID.String(),
		h.data.State.ID.String(), first, last, user)
	return id
}

func (h *harness) drive(t *testing.T, moveID, workerID pulid.ID) {
	t.Helper()
	h.exec(t, `INSERT INTO assignments (
			id, organization_id, business_unit_id, shipment_move_id, primary_worker_id
		) VALUES ($1, $2, $3, $4, $5)`,
		pulid.MustNew("a_").String(), h.tenant.OrgID.String(), h.tenant.BuID.String(),
		moveID.String(), workerID.String())
}

func (h *harness) haul(t *testing.T, moveID pulid.ID, name, email string) pulid.ID {
	t.Helper()
	carrierID := pulid.MustNew("car_")
	h.exec(t, `INSERT INTO carriers (id, organization_id, business_unit_id, code, name, email)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		carrierID.String(), h.tenant.OrgID.String(), h.tenant.BuID.String(),
		carrierID.String()[:10], name, email)
	h.exec(t, `INSERT INTO carrier_assignments (
			id, organization_id, business_unit_id, shipment_move_id, carrier_id, status,
			base_rate, base_amount, total_cost
		) VALUES ($1, $2, $3, $4, $5, $6, 1000, 1000, 1000)`,
		pulid.MustNew("casn_").String(), h.tenant.OrgID.String(), h.tenant.BuID.String(),
		moveID.String(), carrierID.String(), string(shipmentvocab.CarrierAssignmentStatusConfirmed))
	return carrierID
}

func (h *harness) shipmentWithMoves(t *testing.T, code string, moves int) *testutil.ShipmentGraph {
	t.Helper()
	statuses := make([]shipmentvocab.MoveStatus, moves)
	for i := range statuses {
		statuses[i] = shipmentvocab.MoveStatusNew
	}
	graph := testutil.CreateShipmentGraph(t, h.ctx, h.db, h.fixture, h.tenant,
		testutil.ShipmentGraphParams{
			BOL:          "BOL-" + code,
			ProNumber:    "PRO-" + code,
			ShipmentID:   pulid.MustNew("shp_"),
			MoveStatuses: statuses,
		},
	)
	h.exec(t, `UPDATE shipments SET status = $1, owner_id = $2 WHERE id = $3`,
		string(shipmentvocab.StatusCompleted), h.data.User.ID.String(), graph.Shipment.ID.String())
	return graph
}

func (h *harness) chaseFacts() podchaseports.FactsReader {
	return exceptionsourcerepository.NewPODChaseFacts(
		exceptionsourcerepository.Params{DB: postgres.WrapDB(h.db)},
	)
}

// The chase reads, for the shipments asked about that still lack a proof of
// delivery, who hauled each one's last move: the driver and whether they are
// on Dash, or the carrier and its email.
func TestListPODChaseFactsReadsWhoHauledTheLastMove(t *testing.T) {
	h := newHarness(t)
	podType := h.podType(t)
	delivered := h.now - 30*3600

	driven := h.shipment(t, "CHASE-1", shipmentvocab.StatusCompleted)
	h.arrive(t, driven.Moves[0].Stops[1], delivered)
	h.drive(t, driven.Moves[0].ID, h.worker(t, "Laura", "Wright", h.data.User.ID))

	hauled := h.shipment(t, "CHASE-2", shipmentvocab.StatusCompleted)
	h.arrive(t, hauled.Moves[0].Stops[1], delivered)
	carrierID := h.haul(t, hauled.Moves[0].ID, "Acme Freight", "  ops@acme.test ")
	h.drive(t, hauled.Moves[0].ID, h.worker(t, "Theresa", "Aguilar", pulid.Nil))

	filed := h.shipment(t, "CHASE-3", shipmentvocab.StatusCompleted)
	h.arrive(t, filed.Moves[0].Stops[1], delivered)
	h.attach(t, filed.Shipment.ID, podType, document.StatusActive)

	unasked := h.shipment(t, "CHASE-4", shipmentvocab.StatusCompleted)
	h.arrive(t, unasked.Moves[0].Stops[1], delivered)

	facts, err := h.chaseFacts().ListPODChaseFacts(h.ctx, &podchaseports.ListFactsRequest{
		TenantInfo:      h.tenant,
		DeliveredBefore: h.now - 24*3600,
		ShipmentIDs:     []pulid.ID{driven.Shipment.ID, hauled.Shipment.ID, filed.Shipment.ID},
	})
	require.NoError(t, err)
	require.Len(t, facts, 2, "a shipment with its proof on file, or not asked about, is left out")

	byShipment := make(map[pulid.ID]*podchaseports.Facts, len(facts))
	for _, fact := range facts {
		byShipment[fact.ShipmentID] = fact
	}

	onDash := byShipment[driven.Shipment.ID]
	require.NotNil(t, onDash)
	assert.Equal(t, "PRO-CHASE-1", onDash.ProNumber)
	assert.Equal(t, "BOL-CHASE-1", onDash.BOL)
	assert.Equal(t, delivered, onDash.DeliveredAt)
	require.NotNil(t, onDash.Driver)
	assert.Equal(t, "Laura Wright", onDash.Driver.Name)
	assert.True(t, onDash.Driver.OnDash, "a driver with a user signs in to Dash")
	assert.Nil(t, onDash.Carrier)

	byCarrier := byShipment[hauled.Shipment.ID]
	require.NotNil(t, byCarrier)
	require.NotNil(t, byCarrier.Driver)
	assert.False(t, byCarrier.Driver.OnDash)
	require.NotNil(t, byCarrier.Carrier)
	assert.Equal(t, carrierID, byCarrier.Carrier.CarrierID)
	assert.Equal(t, "Acme Freight", byCarrier.Carrier.Name)
	assert.Equal(t, "ops@acme.test", byCarrier.Carrier.Email)
}

// With several moves, the chase asks whoever hauled the last one, the move
// that made the delivery.
func TestListPODChaseFactsAsksTheLastMovesDriver(t *testing.T) {
	h := newHarness(t)
	delivered := h.now - 30*3600

	graph := h.shipmentWithMoves(t, "CHASE-5", 2)
	h.arrive(t, graph.Moves[1].Stops[1], delivered)
	h.drive(t, graph.Moves[0].ID, h.worker(t, "First", "Leg", pulid.Nil))
	h.drive(t, graph.Moves[1].ID, h.worker(t, "Final", "Leg", h.data.User.ID))

	facts, err := h.chaseFacts().ListPODChaseFacts(h.ctx, &podchaseports.ListFactsRequest{
		TenantInfo:      h.tenant,
		DeliveredBefore: h.now - 24*3600,
		ShipmentIDs:     []pulid.ID{graph.Shipment.ID},
	})
	require.NoError(t, err)
	require.Len(t, facts, 1)
	require.NotNil(t, facts[0].Driver)
	assert.Equal(t, "Final Leg", facts[0].Driver.Name)
	assert.True(t, facts[0].Driver.OnDash)
}
