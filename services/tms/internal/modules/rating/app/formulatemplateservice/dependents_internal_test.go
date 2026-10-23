package formulatemplateservice

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTheDependentsRefusalCountsEveryDependentAndNamesTheFirstFew(t *testing.T) {
	t.Parallel()

	gate := &dependentsGate{
		Checked: true,
		Failures: []DependentFailure{
			{
				Kind:        "rate_agreement",
				ExampleName: "Acme 2026",
				Count:       3,
				Path:        "rules[0].inputs.rate",
				Message:     "`rate` must be from $0.50/mi to $15.00/mi; $235.00/mi is outside it",
			},
			{Kind: "shipment", Count: 1, Message: "`service` must be one of Standard"},
		},
	}

	err := dependentsRefusal(i18n.Default, gate)

	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, fieldInputs, validation.Field)
	assert.Equal(t,
		"This version's inputs no longer accept what 4 contracts and shipments already set: "+
			"Acme 2026 and 2 more (rules[0].inputs.rate: `rate` must be from $0.50/mi to $15.00/mi; "+
			"$235.00/mi is outside it); shipment (`service` must be one of Standard). "+
			"Keep the inputs those contracts use, or change the contracts first",
		errortypes.LocalizedText(t.Context(), err),
	)
}

func TestTheDependentsRefusalIsNilWhenEveryDependentFits(t *testing.T) {
	t.Parallel()

	assert.NoError(t, dependentsRefusal(i18n.Default, &dependentsGate{Checked: true}))
}
