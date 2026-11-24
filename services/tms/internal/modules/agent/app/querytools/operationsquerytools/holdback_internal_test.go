package operationsquerytools

import (
	"testing"

	"github.com/emoss08/trenova/pkg/filtercatalog"

	"github.com/stretchr/testify/assert"
)

func TestAStaysHoldBackReachesTheAgentInWordsNotItsCode(t *testing.T) {
	t.Parallel()

	row := map[string]any{"holdBack": "NoticeMissed"}
	holdBackInWords(row, nil, filtercatalog.Clock{})
	assert.Equal(t, "no notice reached the customer inside its window", row["holdBack"])

	ready := map[string]any{}
	holdBackInWords(ready, nil, filtercatalog.Clock{})
	assert.NotContains(t, ready, "holdBack")
}
