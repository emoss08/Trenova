package carrierintelcontrol_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelcontrol"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelrule"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateRuleSettings(t *testing.T) {
	t.Parallel()

	control := carrierintelcontrol.NewDefaultControl(pulid.MustNew("org_"), pulid.MustNew("bu_"))
	control.Rules = carrierintelrule.RuleSettings{
		carrierintelrule.RuleSafetyISSHigh: {
			Action: carrierintel.RuleActionWarn,
			Params: map[string]string{"minimum": "150"},
		},
		carrierintelrule.RuleCode("nope.rule"): {Action: carrierintel.RuleActionWarn},
		carrierintelrule.RuleBasicsAlert: {
			Action: carrierintel.RuleAction("Maybe"),
			Params: map[string]string{"basics": "UnsafeDriving,Bogus"},
		},
	}

	multiErr := errortypes.NewMultiError()
	control.Validate(multiErr)
	require.True(t, multiErr.HasErrors())
	fields := make([]string, 0, len(multiErr.Errors))
	for _, e := range multiErr.Errors {
		fields = append(fields, e.Field)
	}
	assert.Contains(t, fields, "rules.safety.iss_high.params.minimum")
	assert.Contains(t, fields, "rules.nope.rule")
	assert.Contains(t, fields, "rules.basics.alert.action")
	assert.Contains(t, fields, "rules.basics.alert.params.basics")
}

func TestControlValidate_DefaultsAreValid(t *testing.T) {
	t.Parallel()

	control := carrierintelcontrol.NewDefaultControl(pulid.MustNew("org_"), pulid.MustNew("bu_"))
	multiErr := errortypes.NewMultiError()
	control.Validate(multiErr)
	assert.False(t, multiErr.HasErrors(), multiErr.Error())
}
