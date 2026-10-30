//go:build integration

package billingtest

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount/glaccountdb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

type Org struct {
	ID             pulid.ID
	BusinessUnitID pulid.ID
}

type User struct {
	ID pulid.ID
}

type Shipment struct {
	ID         pulid.ID
	CustomerID pulid.ID
	ProNumber  string
	BOL        string
}

type Tenant struct {
	Ctx      context.Context
	Info     pagination.TenantInfo
	Org      Org
	User     User
	Shipment Shipment
}

func SeededTenant(t *testing.T, ctx context.Context, db *keel.DB) *Tenant {
	t.Helper()

	sys := dbscope.WithSystem(ctx, "billing test fixture")
	orgs := tenantdb.Organizations
	org, err := keel.IntoOne[Org](sys, db, orgs.AllTenants().
		Pick(orgs.ID, orgs.BusinessUnitID).
		Order(orgs.ID.Asc()).
		Limit(1))
	require.NoError(t, err)

	users := tenantdb.Users
	user, err := keel.IntoOne[User](sys, db, users.AllTenants().
		Pick(users.ID).
		Where(users.CurrentOrganizationID.Eq(org.ID), users.BusinessUnitID.Eq(org.BusinessUnitID)).
		Order(users.ID.Asc()).
		Limit(1))
	require.NoError(t, err)

	info := pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID, UserID: user.ID}
	ctx = dbscope.WithTenant(ctx, info.DBTenant())

	cus := customerdb.Customers
	customerID, err := keel.IntoOne[pulid.ID](ctx, db, cus.
		Pick(cus.ID.Min()).
		Where(seedtest.CustomersWithShipmentsWithoutReceivables()))
	require.NoError(t, err)
	require.True(t, customerID.IsNotNil(), "a seeded customer should have shipments and no receivables")

	sp := shipmentdb.Shipments
	shipment, err := keel.IntoOne[Shipment](ctx, db, sp.
		Pick(sp.ID, sp.CustomerID, sp.ProNumber, sp.BOL.Coalesce("")).
		Where(sp.CustomerID.Eq(customerID)).
		Order(sp.CreatedAt.Desc(), sp.ID.Desc()).
		Limit(1))
	require.NoError(t, err)

	return &Tenant{Ctx: ctx, Info: info, Org: org, User: user, Shipment: shipment}
}

func GLAccountID(t *testing.T, ctx context.Context, db *keel.DB, accountCode string) pulid.ID {
	t.Helper()

	gl := glaccountdb.GLAccounts
	id, err := keel.IntoOne[pulid.ID](ctx, db, gl.
		Pick(gl.ID).
		Where(gl.AccountCode.Eq(accountCode)).
		Order(gl.ID.Asc()).
		Limit(1))
	require.NoError(t, err)

	return id
}
