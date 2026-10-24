package dbpool

import (
	"context"
	"database/sql/driver"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type nopGuard struct{}

func (nopGuard) BeginCommand(_ context.Context, command string) (string, error) {
	return command, nil
}

func (nopGuard) GuardStatement(context.Context) error { return nil }

func (nopGuard) OnBegin(context.Context, keel.TxInfo) ([]keel.Statement, error) {
	return nil, nil
}

func TestKeelScopeFunctionsReadTheDatabaseScope(t *testing.T) {
	t.Parallel()

	tenant := dbscope.Tenant{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		UserID:         pulid.MustNew("usr_"),
	}
	values, ok := tenantOf(dbscope.WithTenant(t.Context(), tenant))
	require.True(t, ok)
	assert.Equal(t, []any{tenant.OrganizationID.String(), tenant.BusinessUnitID.String()}, values)

	_, ok = tenantOf(dbscope.WithSystem(t.Context(), "sweep"))
	assert.False(t, ok)
	_, ok = tenantOf(t.Context())
	assert.False(t, ok)

	reason, ok := systemOf(dbscope.WithSystem(t.Context(), "sweep"))
	require.True(t, ok)
	assert.Equal(t, "sweep", reason)
	_, ok = systemOf(dbscope.WithTenant(t.Context(), tenant))
	assert.False(t, ok)

	assert.Regexp(t, `^shp_[0-9A-Z]{26}$`, newID("shp_"))
}

func TestKeelConfigSetsOnBeginOnlyWithAGuard(t *testing.T) {
	t.Parallel()

	assert.Nil(t, KeelConfig(nil, nil).OnBegin)

	cfg := KeelConfig(nopGuard{}, nil)
	assert.NotNil(t, cfg.OnBegin)
	assert.Equal(t, []string{"organization_id", "business_unit_id"}, cfg.TenantColumns)
	assert.False(t, cfg.SkipEnums)
	assert.NotNil(t, cfg.JSON)
	assert.NotNil(t, cfg.Now)
	assert.NotNil(t, cfg.NewID)
}

func TestBeginCommand(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		isolation int
		readOnly  bool
		want      string
	}{
		"default":             {want: "BEGIN"},
		"read only":           {readOnly: true, want: "BEGIN READ ONLY"},
		"serializable":        {isolation: 6, want: "BEGIN ISOLATION LEVEL SERIALIZABLE"},
		"repeatable readonly": {isolation: 4, readOnly: true, want: "BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY"},
	}
	for name, tc := range cases {
		got, err := beginCommand(driverTxOptions(tc.isolation, tc.readOnly))
		require.NoError(t, err, name)
		assert.Equal(t, tc.want, got, name)
	}

	_, err := beginCommand(driverTxOptions(7, false))
	require.Error(t, err)
}

func driverTxOptions(isolation int, readOnly bool) driver.TxOptions {
	return driver.TxOptions{Isolation: driver.IsolationLevel(isolation), ReadOnly: readOnly}
}
