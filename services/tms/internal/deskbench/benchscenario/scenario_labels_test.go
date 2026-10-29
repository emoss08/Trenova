package benchscenario

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScenario_DerivesLabelsFromItsChecks(t *testing.T) {
	t.Parallel()

	scenario := &Scenario{Name: "s", Steps: []Step{{
		Say: "hi",
		Expect: Expect{
			Calls:     []string{"shipment_get|shipment_list"},
			NoCalls:   []string{"shipment_cancel|delete_shipment"},
			Proposals: []ExpectedWrite{{Tool: "shipment_add_comment"}},
		},
	}}}
	scenario.deriveLabels()

	assert.Equal(t, []string{"shipment_get|shipment_list", "shipment_add_comment"},
		scenario.ExpectedTools)
	assert.Equal(t, []string{"shipment_cancel", "delete_shipment"}, scenario.ForbiddenTools)
	assert.True(t, scenario.LabelsDerived)

	labelled := &Scenario{Name: "t", ExpectedTools: []string{}, ForbiddenTools: []string{}}
	labelled.deriveLabels()
	assert.False(t, labelled.LabelsDerived, "an explicit empty label is kept")
}

func TestScenario_RefusesAToolBothExpectedAndForbidden(t *testing.T) {
	t.Parallel()

	scenario := &Scenario{
		Name:           "s",
		Steps:          []Step{{Say: "hi"}},
		ExpectedTools:  []string{"shipment_get"},
		ForbiddenTools: []string{"shipment_get|shipment_cancel"},
	}
	require.Error(t, scenario.Validate())

	scenario.ForbiddenTools = []string{"shipment_cancel"}
	scenario.Split = "training"
	require.Error(t, scenario.Validate())
}
