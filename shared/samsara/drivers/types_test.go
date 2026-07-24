package drivers

import (
	"testing"

	samsaraspec "github.com/emoss08/trenova/shared/samsara/internal/samsaraspec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestELDSettingsAliases(t *testing.T) {
	t.Parallel()

	cycle := ELDRulesetCycle(samsaraspec.DriverEldRulesetCycleUSA70Hour8Day)
	shift := ELDRulesetShift("US Interstate Property")
	restart := ELDRulesetRestart("34-hour Restart")
	restBreak := ELDRulesetRestBreak("Property (off-duty/sleeper)")
	jurisdiction := ELDRulesetJurisdiction("CA")

	rulesets := ELDRulesets{
		ELDRuleset{
			Cycle:        &cycle,
			Shift:        &shift,
			Restart:      &restart,
			Break:        &restBreak,
			Jurisdiction: &jurisdiction,
		},
	}
	settings := ELDSettings{Rulesets: &rulesets}

	driver := Driver{EldSettings: &settings}
	require.NotNil(t, driver.EldSettings)
	require.NotNil(t, driver.EldSettings.Rulesets)
	require.Len(t, *driver.EldSettings.Rulesets, 1)

	got := (*driver.EldSettings.Rulesets)[0]
	require.NotNil(t, got.Cycle)
	assert.Equal(t, samsaraspec.DriverEldRulesetCycleUSA70Hour8Day, *got.Cycle)
	require.NotNil(t, got.Jurisdiction)
	assert.Equal(t, "CA", *got.Jurisdiction)
}
