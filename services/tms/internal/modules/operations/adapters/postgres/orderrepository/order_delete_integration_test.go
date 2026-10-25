//go:build integration

package orderrepository_test

import (
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/bootstrap/crossref"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/orderrepository"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestDeleteIfEmpty_KeepsAnOrderBillingStillReferences(t *testing.T) {
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

	newOrder := func(number string) pulid.ID {
		id := pulid.MustNew("ord_")
		_, insertErr := keel.Exec(sys, db, `
			INSERT INTO orders (id, organization_id, business_unit_id, customer_id, order_number)
			VALUES ($1, $2, $3, $4, $5)`,
			id.String(), seeded.OrganizationID.String(), seeded.BusinessUnitID.String(),
			seeded.CustomerID.String(), number,
		)
		require.NoError(t, insertErr)
		return id
	}
	empty := newOrder("ORD-EMPTY-1")
	queued := newOrder("ORD-QUEUED-1")
	_, err = keel.Exec(sys, db, `
		INSERT INTO billing_queue_items (id, organization_id, business_unit_id, shipment_id, order_id)
		VALUES ($1, $2, $3, $4, $5)`,
		pulid.MustNew("bqi_").String(), seeded.OrganizationID.String(),
		seeded.BusinessUnitID.String(), seeded.ShipmentID.String(), queued.String(),
	)
	require.NoError(t, err)

	repo := orderrepository.New(orderrepository.Params{
		DB:         postgres.NewTestConnection(db),
		Logger:     zap.NewNop(),
		References: crossref.OrderReferences(),
	})
	tenant := pagination.TenantInfo{OrgID: seeded.OrganizationID, BuID: seeded.BusinessUnitID}
	tenantCtx := dbscope.WithTenant(ctx, tenant.DBTenant())

	other := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	deleted, err := repo.DeleteIfEmpty(dbscope.WithTenant(ctx, other.DBTenant()), other, empty)
	require.NoError(t, err)
	assert.Zero(t, deleted, "another tenant never deletes the order")

	deleted, err = repo.DeleteIfEmpty(tenantCtx, tenant, queued)
	require.NoError(t, err)
	assert.Zero(t, deleted, "a billing queue item still references the order")

	deleted, err = repo.DeleteIfEmpty(tenantCtx, tenant, empty)
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)
}
