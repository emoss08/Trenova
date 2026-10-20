package agentreadtest

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

// Tenant is an organization written for an integration test, and the person
// who reads in it.
type Tenant struct {
	Actor *permissionports.RequestActor
	// StateID is a US state written for the tenant, for records that need an
	// address.
	StateID pulid.ID
}

// Scope is the tenant's database scope, which row-level security reads.
func (t *Tenant) Scope() dbscope.Tenant {
	return dbscope.Tenant{
		OrganizationID: t.Actor.OrganizationID,
		BusinessUnitID: t.Actor.BusinessUnitID,
		UserID:         t.Actor.UserID,
	}
}

// SeedTenant writes a business unit, an organization and the person who
// reads in it straight through the
// admin connection, which row-level security does not hold. The label keeps
// two tenants' unique columns apart.
func SeedTenant(t *testing.T, admin *bun.DB, label string) *Tenant {
	t.Helper()

	ctx := t.Context()
	org, bu := pulid.MustNew("org_"), pulid.MustNew("bu_")
	state := "us_" + strings.ToLower(label) + org.String()[4:12]
	exec := func(query string, args ...any) {
		t.Helper()
		_, err := admin.ExecContext(ctx, query, args...)
		require.NoError(t, err)
	}
	exec("INSERT INTO us_states (id, name, abbreviation) VALUES (?, ?, ?)",
		state, "State "+label, label+"X")
	exec("INSERT INTO business_units (id, name, code) VALUES (?, ?, ?)",
		bu, "BU "+label, "BU"+label+bu.String()[3:9])
	exec(`INSERT INTO organizations (id, state_id, business_unit_id, name, scac_code, dot_number,
		bucket_name, address_line1, city, postal_code)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'Line 1', 'City', '75001')`,
		org, state, bu, "Org "+label, "SC"+label+"X", "10"+label, "bucket-"+strings.ToLower(label))

	tx, err := admin.BeginTx(ctx, nil)
	require.NoError(t, err)
	person := seedtest.NewUser(org, bu).
		WithUsername("reader_"+strings.ToLower(label)+org.String()[4:12]).
		WithEmail("reader_"+strings.ToLower(label)+org.String()[4:12]+"@example.com").
		Build(t, ctx, tx)
	require.NoError(t, tx.Commit())
	user := person.ID

	return &Tenant{
		Actor: &permissionports.RequestActor{
			PrincipalType:  permissionports.PrincipalTypeUser,
			PrincipalID:    user,
			UserID:         user,
			OrganizationID: org,
			BusinessUnitID: bu,
		},
		StateID: pulid.ID(state),
	}
}
