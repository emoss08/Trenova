package businesscentral_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/shared/businesscentral"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompanyRefRoundTrip(t *testing.T) {
	t.Parallel()

	envs := []string{"Production", "uat_sandbox-2", "a", "S-1_x", strings.Repeat("e", 29)}
	for _, env := range envs {
		ref, err := businesscentral.NewCompanyRef(strings.ToUpper(testTenant), env, testCompany)
		require.NoError(t, err, env)
		encoded := ref.String()
		assert.Equal(t, strings.ReplaceAll(testTenant, "-", "")+"_"+
			strings.ReplaceAll(testCompany, "-", "")+"_"+env, encoded)

		parsed, err := businesscentral.ParseCompanyRef(encoded)
		require.NoError(t, err, env)
		assert.Equal(t, ref, parsed)
		assert.Equal(t, testTenant, parsed.TenantID)
		assert.Equal(t, testCompany, parsed.CompanyID)
		assert.Equal(t, env, parsed.Environment)
		require.NoError(t, parsed.Validate())
	}
}

func TestParseCompanyRefRejectsMalformedInput(t *testing.T) {
	t.Parallel()

	tenant := strings.ReplaceAll(testTenant, "-", "")
	company := strings.ReplaceAll(testCompany, "-", "")
	for _, raw := range []string{
		"",
		tenant,
		tenant + "_" + company,
		tenant + "_" + company + "_",
		tenant + "-" + company + "_Production",
		tenant + "_" + company + "-Production",
		strings.Replace(tenant, "6", "g", 1) + "_" + company + "_Production",
		tenant[:31] + "-_" + company + "_Production",
	} {
		_, err := businesscentral.ParseCompanyRef(raw)
		require.ErrorIs(t, err, businesscentral.ErrInvalidCompanyRef, raw)
	}
	bad := []string{"1prod", "_prod", "prod env", "prod/../x", "prod'", strings.Repeat("e", 30)}
	for _, env := range bad {
		_, err := businesscentral.ParseCompanyRef(tenant + "_" + company + "_" + env)
		require.ErrorIs(t, err, businesscentral.ErrInvalidEnvironment, env)
	}
}

func TestCompanyRefStringIsEmptyWhenInvalid(t *testing.T) {
	t.Parallel()

	assert.Empty(t, businesscentral.CompanyRef{}.String())
	require.Error(t, businesscentral.CompanyRef{TenantID: testTenant}.Validate())
	assert.True(t, businesscentral.ValidEnvironmentName("Sandbox-01_a"))
	assert.False(t, businesscentral.ValidEnvironmentName(""))
}
