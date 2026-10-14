package rateagreement

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func conditionedAccessorial() *RateAgreementAccessorial {
	return &RateAgreementAccessorial{
		RateAgreementID:     pulid.MustNew("ra_"),
		AccessorialChargeID: pulid.MustNew("acc_"),
		Method:              accessorialcharge.MethodFlat,
		Amount:              decimal.NewFromInt(25),
		AutoApply:           true,
		ApplyCondition:      "totalStops > 2",
	}
}

func TestAccessorialCondition_IsInTheCurrentLanguageUnlessItSaysOtherwise(t *testing.T) {
	t.Parallel()

	accessorial := conditionedAccessorial()
	multiErr := errortypes.NewMultiError()
	accessorial.Validate(multiErr)

	assert.False(t, multiErr.HasErrors(), multiErr.Error())
	assert.Equal(t, formulatypes.CurrentLanguageVersion, accessorial.ConditionLanguageVersion)
}

func TestAccessorialCondition_RefusesALanguageTheServerCannotRead(t *testing.T) {
	t.Parallel()

	accessorial := conditionedAccessorial()
	accessorial.ConditionLanguageVersion = formulatypes.LanguageVersion(2)
	multiErr := errortypes.NewMultiError()
	accessorial.Validate(multiErr)

	assert.True(t, multiErr.HasErrors())
	assert.Contains(t, multiErr.Error(), "conditionLanguageVersion")
}
