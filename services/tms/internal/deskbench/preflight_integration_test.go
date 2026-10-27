//go:build integration

package deskbench

import (
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfileData_RefusesAnOrganizationTheFixtureSeedDidNotFill(t *testing.T) {
	ctx, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)
	data := seedtest.SeedFullTestData(t, ctx, admin)

	bench := &Bench{deps: deps{DB: dbtest.NewRLSConnection(t, admin, dsn)}}
	session := &Session{Tenant: pagination.TenantInfo{
		OrgID:  data.Organization.ID,
		BuID:   data.BusinessUnit.ID,
		UserID: data.User.ID,
	}}

	_, err := bench.ProfileData(t.Context(), session)
	require.ErrorIs(t, err, ErrNoFixtureData)
}

func TestUnmetRequirements_SkipsUntilTheEnvironmentHoldsWhatTheScenarioNeeds(t *testing.T) {
	ctx, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)
	data := seedtest.SeedFullTestData(t, ctx, admin)

	bench := &Bench{deps: deps{DB: dbtest.NewRLSConnection(t, admin, dsn)}}
	session := &Session{Tenant: pagination.TenantInfo{
		OrgID:  data.Organization.ID,
		BuID:   data.BusinessUnit.ID,
		UserID: data.User.ID,
	}}
	browser := Requirement{Extension: "BrowserUse"}
	requires := []Requirement{
		browser,
		{EmailProfile: "Notifications"},
		{EDILading: "ACMEMFG"},
	}

	unmet, err := bench.unmetRequirements(t.Context(), session, requires)
	require.NoError(t, err)
	for idx := range requires {
		assert.Contains(t, unmet, requires[idx].Label())
	}

	_, err = keel.Exec(ctx, admin, `INSERT INTO agent_extensions
		(id, business_unit_id, organization_id, type, enabled, configuration)
		VALUES ($1, $2, $3, 'BrowserUse', false, '{"apiKey": "sealed"}')`,
		pulid.MustNew("aext_").String(), data.BusinessUnit.ID.String(),
		data.Organization.ID.String())
	require.NoError(t, err)

	unmet, err = bench.unmetRequirements(t.Context(), session, []Requirement{browser})
	require.NoError(t, err)
	assert.Contains(t, unmet, browser.Label(), "a disabled extension does not count")

	_, err = keel.Exec(ctx, admin, `UPDATE agent_extensions SET enabled = true
		WHERE organization_id = $1 AND type = 'BrowserUse'`, data.Organization.ID.String())
	require.NoError(t, err)

	unmet, err = bench.unmetRequirements(t.Context(), session, []Requirement{browser})
	require.NoError(t, err)
	assert.Empty(t, unmet)
}
