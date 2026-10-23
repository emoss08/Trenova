package agentdefinition

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/pkg/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTariffReference_TheFullExampleChecksAndPassesItsTests(t *testing.T) {
	t.Parallel()

	start := strings.Index(tariffReference, "tariff \"Per Mile\"\n")
	require.Positive(t, start)
	source := tariffReference[start:]

	check, err := rules.CheckTariff(t.Context(), &rules.TariffCheckRequest{
		Source:  source,
		Kind:    rules.TariffKindTariff,
		Context: &rules.TariffContext{},
	})
	require.NoError(t, err)
	require.True(t, check.OK, "%+v", check.Diagnostics)

	run, err := rules.TestTariff(t.Context(), &rules.TariffTestRequest{
		Source:  source,
		Context: &rules.TariffContext{},
	})
	require.NoError(t, err)
	require.Len(t, run.Tests, 2)
	assert.True(t, run.Passed(), "%+v", run.Tests)
	assert.True(t, run.Tests[1].IsProperty())
}
