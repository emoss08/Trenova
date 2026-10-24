//go:build integration

package orderrepository_test

import (
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/orderrepository"
	"github.com/emoss08/trenova/internal/modules/operations/ports/orderports"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type seededShipment struct {
	ShipmentID     pulid.ID
	CustomerID     pulid.ID
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}

func TestGetByID_LoadsTheCustomerAndChargesWithoutTheShipments(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	sys := dbscope.WithSystem(ctx, "test")
	rows, err := keel.QueryAll(sys, db, pgx.RowToStructByPos[seededShipment], `
		SELECT sp.id, sp.customer_id, sp.organization_id, sp.business_unit_id
		FROM shipments AS sp
		ORDER BY sp.id
		LIMIT 1`)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	seeded := rows[0]
	orderID := pulid.MustNew("ord_")

	_, err = keel.Exec(sys, db, `
		INSERT INTO orders (id, organization_id, business_unit_id, customer_id, order_number)
		VALUES ($1, $2, $3, $4, 'ORD-INCLUDES-1')`,
		orderID.String(), seeded.OrganizationID.String(), seeded.BusinessUnitID.String(),
		seeded.CustomerID.String(),
	)
	require.NoError(t, err)
	_, err = keel.Exec(sys, db,
		"UPDATE shipments SET order_id = $1 WHERE id = $2 AND organization_id = $3",
		orderID.String(), seeded.ShipmentID.String(), seeded.OrganizationID.String(),
	)
	require.NoError(t, err)
	_, err = keel.Exec(sys, db, `
		INSERT INTO order_charges (id, organization_id, business_unit_id, order_id, description, amount)
		VALUES ($1, $2, $3, $4, 'Lumper', 85)`,
		pulid.MustNew("ochg_").String(), seeded.OrganizationID.String(),
		seeded.BusinessUnitID.String(), orderID.String(),
	)
	require.NoError(t, err)

	repo := orderrepository.New(orderrepository.Params{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	})
	tenant := pagination.TenantInfo{OrgID: seeded.OrganizationID, BuID: seeded.BusinessUnitID}
	tenantCtx := dbscope.WithTenant(ctx, tenant.DBTenant())

	card, err := repo.GetByID(tenantCtx, orderports.GetOrderByIDRequest{
		ID:              orderID,
		TenantInfo:      tenant,
		IncludeCustomer: true,
		IncludeCharges:  true,
	})
	require.NoError(t, err)
	require.NotNil(t, card.Customer)
	assert.NotEmpty(t, card.Charges)
	assert.Empty(t, card.Shipments)

	full, err := repo.GetByID(tenantCtx, orderports.GetOrderByIDRequest{
		ID:              orderID,
		TenantInfo:      tenant,
		IncludeShipment: true,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, full.Shipments)
	assert.NotNil(t, full.Customer)
	assert.NotEmpty(t, full.Charges)

	bare, err := repo.GetByID(tenantCtx, orderports.GetOrderByIDRequest{
		ID:         orderID,
		TenantInfo: tenant,
	})
	require.NoError(t, err)
	assert.Nil(t, bare.Customer)
	assert.Empty(t, bare.Charges)

	other := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	_, err = repo.GetByID(dbscope.WithTenant(ctx, other.DBTenant()), orderports.GetOrderByIDRequest{
		ID:         orderID,
		TenantInfo: other,
	})
	require.Error(t, err, "another tenant never reads the order")
}
