package benchrun

import (
	"testing"

	"github.com/emoss08/trenova/internal/deskbench/benchscenario"
	"github.com/stretchr/testify/assert"
)

func TestRunUnits_RunsAnExclusiveGroupInOrderForEachProviderAndRepeat(t *testing.T) {
	t.Parallel()

	guide := &benchscenario.Scenario{Name: "edi-guide-new-partner", Exclusive: "acme-retail-edi"}
	revised := &benchscenario.Scenario{Name: "edi-guide-revised", Exclusive: "Acme-Retail-EDI "}
	alone := &benchscenario.Scenario{Name: "edi-pasted-x12"}
	planned := make([]plannedCase, 0, 12)
	for scenarioOrder, scenario := range []*benchscenario.Scenario{guide, revised, alone} {
		for providerOrder, provider := range []string{"Anthropic", "OpenAI"} {
			for rep := 1; rep <= 2; rep++ {
				planned = append(planned, plannedCase{
					scenario: scenario, provider: provider, repeat: rep,
					scenarioOrder: scenarioOrder, providerOrder: providerOrder,
				})
			}
		}
	}

	units := runUnits(planned)

	assert.Len(t, units, 5, "one unit for the group, one for each case of the other scenario")
	labels := make([]string, 0, len(units[0]))
	for _, idx := range units[0] {
		labels = append(labels, planned[idx].label())
	}
	assert.Equal(t, []string{
		"edi-guide-new-partner @ Anthropic #1", "edi-guide-revised @ Anthropic #1",
		"edi-guide-new-partner @ Anthropic #2", "edi-guide-revised @ Anthropic #2",
		"edi-guide-new-partner @ OpenAI #1", "edi-guide-revised @ OpenAI #1",
		"edi-guide-new-partner @ OpenAI #2", "edi-guide-revised @ OpenAI #2",
	}, labels, "each revised case follows the new-partner case that saved the partner")
	for _, unit := range units[1:] {
		assert.Len(t, unit, 1)
	}
}
