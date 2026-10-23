package agentquerytoolservice

import (
	"testing"

	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCarrierIntelEvent_NamesTheListThatHandsOutItsID(t *testing.T) {
	t.Parallel()

	spec := carrierIntelEventGetSpec()
	tool, err := newGraphQLGetTool(&spec, &keyedReader{}, nil)
	require.NoError(t, err)
	assert.Contains(t, tool.Description(), "carrier_list_intelligence_events")
	assert.NotContains(t, tool.Description(), "No tool lists")

	properties, _ := tool.ParamSchema()[toolschema.KeyProperties].(map[string]any)
	eventID, _ := properties["eventId"].(map[string]any)
	assert.Contains(t, eventID[toolschema.KeyDescription], "carrier_list_intelligence_events")
}
