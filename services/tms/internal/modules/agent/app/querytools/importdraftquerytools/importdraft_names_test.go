package importdraftquerytools

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/pagedraft"
	"github.com/stretchr/testify/assert"
)

// Each import-page tool makes the page edit of the same name: the edit a
// person sees in the page's history is named for the tool that proposed it.
func TestImportDraftTools_AreNamedForTheEditTheyMake(t *testing.T) {
	t.Parallel()

	cases := []struct {
		tool   agenttoolports.AgentQueryTool
		action pagedraft.Action
	}{
		{&acceptFieldTool{}, pagedraft.ActionAcceptField},
		{&acceptAllConfidentTool{}, pagedraft.ActionAcceptAllConfident},
		{&setFieldValueTool{}, pagedraft.ActionSetFieldValue},
		{&setRequiredFieldTool{}, pagedraft.ActionSetRequiredField},
		{&setStopLocationTool{}, pagedraft.ActionSetStopLocation},
		{&setStopScheduleTool{}, pagedraft.ActionSetStopSchedule},
	}
	for _, c := range cases {
		assert.Equal(t, string(c.action), c.tool.Name())
	}
}
