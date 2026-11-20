package tenantlimitsconfig_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config/tenantlimitsconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSwitch_AutoFollowsTheTenancy(t *testing.T) {
	t.Parallel()

	assert.True(t, tenantlimitsconfig.SwitchAuto.On(true))
	assert.False(t, tenantlimitsconfig.SwitchAuto.On(false))
	assert.True(t, tenantlimitsconfig.Switch("").On(true), "unset is auto")
	assert.True(t, tenantlimitsconfig.SwitchOn.On(false))
	assert.False(t, tenantlimitsconfig.SwitchOff.On(true))
}

func TestConfig_RefusesAnUnknownSwitch(t *testing.T) {
	t.Parallel()

	cfg := &tenantlimitsconfig.Config{
		HeavyJobs: tenantlimitsconfig.HeavyJobs{Organizations: "sometimes"},
	}
	require.ErrorIs(t, cfg.Validate(), tenantlimitsconfig.ErrInvalidSwitch)
	require.NoError(t, (&tenantlimitsconfig.Config{}).Validate())
}

func TestConfig_SlotForFillsWhatIsUnsetFromTheDefaults(t *testing.T) {
	t.Parallel()

	cfg := &tenantlimitsconfig.Config{HeavyJobs: tenantlimitsconfig.HeavyJobs{
		PDF: tenantlimitsconfig.Slot{Total: 8},
	}}

	pdf := cfg.SlotFor(tenantlimitsconfig.KindPDF)
	assert.Equal(t, 8, pdf.Total)
	assert.Equal(t, 2, pdf.PerOrganization)
	assert.Equal(t, 2*time.Minute, pdf.Hold)
	assert.Equal(t, 15*time.Second, pdf.Wait)

	extraction := cfg.SlotFor(tenantlimitsconfig.KindExtraction)
	assert.Equal(t, 1, extraction.Total)
	assert.Equal(t, 1, extraction.PerOrganization)
}

func TestConfig_OrganizationEmailPerHourIsZeroWhenTheBoundIsOff(t *testing.T) {
	t.Parallel()

	cfg := &tenantlimitsconfig.Config{}
	assert.Equal(t, 300, cfg.OrganizationEmailPerHour(true))
	assert.Zero(t, cfg.OrganizationEmailPerHour(false))

	cfg.Email = tenantlimitsconfig.Email{
		Organizations:          tenantlimitsconfig.SwitchOn,
		PerOrganizationPerHour: 50,
	}
	assert.Equal(t, 50, cfg.OrganizationEmailPerHour(false))

	var unset *tenantlimitsconfig.Config
	assert.Zero(t, unset.OrganizationEmailPerHour(true))
}
