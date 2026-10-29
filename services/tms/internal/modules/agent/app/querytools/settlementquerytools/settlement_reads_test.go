package settlementquerytools

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/gettool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/graphqlread"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/readspec"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
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

	current := map[string]any{querytool.FieldEffectiveFrom: float64(now - day)}
	ended := map[string]any{
		querytool.FieldEffectiveFrom: float64(now - 30*day),
		querytool.FieldEffectiveTo:   float64(now - day),
	}
	future := map[string]any{querytool.FieldEffectiveFrom: float64(now + day)}
	for _, row := range []map[string]any{current, ended, future} {
		markInForce(row, nil, clk)
	}

	assert.Equal(t, true, current[fieldInForce])
	assert.Equal(t, false, ended[fieldInForce])
	assert.Equal(t, false, future[fieldInForce])

	hidden := map[string]any{querytool.FieldEffectiveFrom: float64(now - day)}
	markInForce(hidden, map[string]struct{}{querytool.FieldEffectiveTo: {}}, clk)
	assert.NotContains(t, hidden, fieldInForce, "with an end date withheld nothing is said")
}

func TestSettlementReads_AreBoundAndDeclareTheirTaint(t *testing.T) {
	t.Parallel()

	for _, spec := range ReadSpecs() {
		read, ok := graphqlread.Reads[spec.Name]
		require.Truef(t, ok, "%s is not bound in reads.yml", spec.Name)
		assert.Equal(t, graphqlread.VerbList, read.Verb, spec.Name)
		assert.Equal(t, spec.Resource.String(), read.Resource, spec.Name)
	}
	for _, spec := range GetSpecs() {
		read, ok := graphqlread.Reads[spec.Name]
		require.Truef(t, ok, "%s is not bound in reads.yml", spec.Name)
		assert.Equal(t, graphqlread.VerbGet, read.Verb, spec.Name)
		assert.Equal(t, spec.Resource.String(), read.Resource, spec.Name)
		if len(spec.Outside) > 0 {
			assert.Equal(t, agenttooltrust.ExternalReadMarked, spec.Reads, spec.Name)
		}
	}
}

func TestDriverEarnings_TakeAWorkerID(t *testing.T) {
	t.Parallel()

	var spec readspec.Get
	for _, candidate := range GetSpecs() {
		if candidate.Name == ToolGetDriverEarnings {
			spec = candidate
		}
	}
	tool := &gettool.Tool{Spec: spec}
	property := tool.ParamSchema()["properties"].(map[string]any)[querytool.FieldWorkerID].(map[string]any)

	assert.Equal(t, permission.ResourceWorker.String(), property[toolschema.KeyRecordOf])
}
