package agentquerytoolservice

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarkInForce_SaysWhichAssignmentGovernsPayToday(t *testing.T) {
	t.Parallel()

	clk := filtercatalog.NewClock("UTC")
	now := clk.Instant()
	day := int64(time.Hour/time.Second) * 24

	current := map[string]any{fieldEffectiveFrom: float64(now - day)}
	ended := map[string]any{
		fieldEffectiveFrom: float64(now - 30*day),
		fieldEffectiveTo:   float64(now - day),
	}
	future := map[string]any{fieldEffectiveFrom: float64(now + day)}
	for _, row := range []map[string]any{current, ended, future} {
		markInForce(row, nil, clk)
	}

	assert.Equal(t, true, current[fieldInForce])
	assert.Equal(t, false, ended[fieldInForce])
	assert.Equal(t, false, future[fieldInForce])

	hidden := map[string]any{fieldEffectiveFrom: float64(now - day)}
	markInForce(hidden, map[string]struct{}{fieldEffectiveTo: {}}, clk)
	assert.NotContains(t, hidden, fieldInForce, "with an end date withheld nothing is said")
}

func TestSettlementReads_AreBoundAndDeclareTheirTaint(t *testing.T) {
	t.Parallel()

	for _, spec := range settlementReadSpecs() {
		read, ok := graphQLReads[spec.name]
		require.Truef(t, ok, "%s is not bound in reads.yml", spec.name)
		assert.Equal(t, graphQLVerbList, read.Verb, spec.name)
		assert.Equal(t, spec.resource.String(), read.Resource, spec.name)
	}
	for _, spec := range settlementGetSpecs() {
		read, ok := graphQLReads[spec.name]
		require.Truef(t, ok, "%s is not bound in reads.yml", spec.name)
		assert.Equal(t, graphQLVerbGet, read.Verb, spec.name)
		assert.Equal(t, spec.resource.String(), read.Resource, spec.name)
		if len(spec.outside) > 0 {
			assert.Equal(t, agent.ExternalReadMarked, spec.reads, spec.name)
		}
	}
}

func TestDriverEarnings_TakeAWorkerID(t *testing.T) {
	t.Parallel()

	var spec getSpec
	for _, candidate := range settlementGetSpecs() {
		if candidate.name == toolGetDriverEarnings {
			spec = candidate
		}
	}
	tool := &getTool{spec: spec}
	property := tool.ParamSchema()["properties"].(map[string]any)[fieldWorkerID].(map[string]any)

	assert.Equal(t, permission.ResourceWorker.String(), property[toolschema.KeyRecordOf])
}
