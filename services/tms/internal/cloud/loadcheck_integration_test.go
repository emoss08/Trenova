//go:build integration

package cloud_test

import (
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantkeys"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type danglingRow struct {
	Constraint string `db:"constraint_name"`
	Table      string `db:"table_name"`
	Columns    string `db:"column_names"`
	Rows       int64  `db:"row_count"`
}

func TestTheLoadCheckRefusesARowNamingAnotherTenantsRecordThroughEveryExceptionKey(t *testing.T) {
	in := newInstall(t, true, sourceKey)
	loaded := in.tenantNamed(t, "Trenova Logistics")
	other := seedtest.SeedAdditionalTenant(t, in.ctx, in.admin, "ZL")
	stranger := other.User.ID.String()
	provider := pulid.MustNew("idp_").String()
	org, bu := loaded.OrganizationID.String(), loaded.BusinessUnitID.String()

	in.exec(t, `INSERT INTO identity_providers (id, organization_id, business_unit_id, name, slug,
    protocol, oidc_issuer_url, oidc_client_id, oidc_redirect_url)
VALUES ($1, $2, $3, 'Other tenant', 'other-tenant', 'OIDC', 'https://idp.test', 'client', 'https://app.test/cb')`,
		provider, other.Organization.ID.String(), other.BusinessUnit.ID.String())
	in.exec(t, `INSERT INTO auth_events (id, user_id, identity_provider_id, organization_id, business_unit_id, provider, outcome)
VALUES ($1, $2, $3, $4, $5, 'password', 'success')`,
		pulid.MustNew("aue_").String(), stranger, provider, org, bu)
	in.exec(t, `INSERT INTO notifications (id, organization_id, business_unit_id, target_user_id, event_type, title, message, source)
VALUES ($1, $2, $3, $4, 'test.cross_tenant', 'A notice', 'For someone else', 'test')`,
		pulid.MustNew("notif_").String(), org, bu, stranger)
	in.exec(t, `INSERT INTO risk_decisions (id, user_id, organization_id, business_unit_id, outcome)
VALUES ($1, $2, $3, $4, 'allow')`,
		pulid.MustNew("rsk_").String(), stranger, org, bu)

	plan := in.plan(t)
	found := map[string]int64{}
	for _, query := range plan.DanglingQueries(loaded, 80) {
		rows, err := keel.QueryAll(in.ctx, in.admin, pgx.RowToStructByName[danglingRow], query)
		require.NoError(t, err)
		for _, row := range rows {
			if row.Rows > 0 {
				found[row.Constraint] += row.Rows
			}
		}
	}
	for constraint := range tenantkeys.Exceptions() {
		assert.Positive(t, found[constraint],
			"a row of the loaded tenant naming another tenant's record through %s is found", constraint)
	}

	otherTenant := dbscope.Tenant{
		OrganizationID: other.Organization.ID,
		BusinessUnitID: other.BusinessUnit.ID,
	}
	for _, query := range plan.DanglingQueries(otherTenant, 80) {
		rows, err := keel.QueryAll(in.ctx, in.admin, pgx.RowToStructByName[danglingRow], query)
		require.NoError(t, err)
		for _, row := range rows {
			assert.Zero(t, row.Rows, "%s: the other tenant's own rows refer only to its own records",
				row.Constraint)
		}
	}
}
