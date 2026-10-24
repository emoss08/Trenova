//go:build integration

package orderrepository_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/testutil/seededtest"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/orderrepository"
	"github.com/emoss08/trenova/internal/modules/operations/ports/orderports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGetByID_LoadsTheCustomerAndChargesWithoutTheShipments(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	var seeded struct {
		OrderID        pulid.ID `bun:"-"`
		ShipmentID     pulid.ID `bun:"shipment_id"`
		CustomerID     pulid.ID `bun:"customer_id"`
		OrganizationID pulid.ID `bun:"organization_id"`
		BusinessUnitID pulid.ID `bun:"business_unit_id"`
	}
	require.NoError(t, db.NewRaw(`
		SELECT sp.id AS shipment_id, sp.customer_id, sp.organization_id, sp.business_unit_id
		FROM shipments AS sp
		ORDER BY sp.id
		LIMIT 1`).Scan(ctx, &seeded))
	seeded.OrderID = pulid.MustNew("ord_")
	_, err := db.NewRaw(`
		INSERT INTO orders (id, organization_id, business_unit_id, customer_id, order_number)
		VALUES (?, ?, ?, ?, 'ORD-INCLUDES-1')`,
		seeded.OrderID, seeded.OrganizationID, seeded.BusinessUnitID, seeded.CustomerID,
	).Exec(ctx)
	require.NoError(t, err)
	_, err = db.NewRaw(
		"UPDATE shipments SET order_id = ? WHERE id = ? AND organization_id = ?",
		seeded.OrderID, seeded.ShipmentID, seeded.OrganizationID,
	).Exec(ctx)
	require.NoError(t, err)

	_, err = db.NewRaw(`
		INSERT INTO order_charges (id, organization_id, business_unit_id, order_id, description, amount)
		VALUES (?, ?, ?, ?, 'Lumper', 85)`,
		pulid.MustNew("ochg_"), seeded.OrganizationID, seeded.BusinessUnitID, seeded.OrderID,
	).Exec(ctx)
	require.NoError(t, err)

	repo := orderrepository.New(orderrepository.Params{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	})
	tenant := pagination.TenantInfo{OrgID: seeded.OrganizationID, BuID: seeded.BusinessUnitID}

	card, err := repo.GetByID(ctx, orderports.GetOrderByIDRequest{
		ID:              seeded.OrderID,
		TenantInfo:      tenant,
		IncludeCustomer: true,
		IncludeCharges:  true,
	})
	require.NoError(t, err)
	require.NotNil(t, card.Customer)
	assert.NotEmpty(t, card.Charges)
	assert.Empty(t, card.Shipments)

	full, err := repo.GetByID(ctx, orderports.GetOrderByIDRequest{
		ID:              seeded.OrderID,
		TenantInfo:      tenant,
		IncludeShipment: true,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, full.Shipments)
	assert.NotNil(t, full.Customer)
	assert.NotEmpty(t, full.Charges)

	bare, err := repo.GetByID(ctx, orderports.GetOrderByIDRequest{
		ID:         seeded.OrderID,
		TenantInfo: tenant,
	})
	require.NoError(t, err)
	assert.Nil(t, bare.Customer)
	assert.Empty(t, bare.Charges)
}
