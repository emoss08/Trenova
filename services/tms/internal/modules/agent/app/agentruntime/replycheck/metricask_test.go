package replycheck

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAskForBoard_NamesEveryMetricTheProseStatesWithItsShareOfTheFigures(t *testing.T) {
	t.Parallel()

	perMile := `{"metric":"fuelCostPerMile","figures":[` +
		`{"name":"currentCostPerMile","value":"$0.543"},{"name":"pricePerGallon","value":"$3.61"}]}`
	byTractor := `{"metric":"fuelCostByTractor","figures":[` +
		`{"name":"tractorsAboveRange","value":"23"},{"name":"highestCostPerMile","value":"$2.66"},` +
		`{"name":"lowestCostPerMile","value":"$0.181"}]}`
	turn := &boardTurn{messages: []llmports.Message{
		{Role: llmports.RoleUser, Content: "how has fuel cost per mile moved this month, and " +
			"which tractors are off the fleet's line?"},
		{Role: llmports.RoleTool, ToolName: agenttoolports.MetricReadTool, Content: perMile},
		{Role: llmports.RoleTool, ToolName: agenttoolports.MetricReadTool, Content: byTractor},
	}}
	reply := "Fuel cost per mile is **$0.543** this month at $3.61 a gallon. Tractor 1211 is " +
		"furthest above at $2.66, and 1167 furthest below at $0.181."

	back, ask := AskForBoard(turn, &llmports.ChatCompletionResult{Text: reply}, &BoardAskState{})

	require.True(t, ask)
	correction := back.Messages[len(back.Messages)-1].Content
	assert.True(t, strings.HasPrefix(correction, metricCorrectionLead))
	assert.Contains(t, correction, `a group for each, {"metric": "fuelCostPerMile"}, {"metric": "fuelCostByTractor"} and`,
		"beyond-i-2: the correction named only the per-tractor metric and every board lost the trend")
	assert.Contains(t, correction, "at most 6 figures in all, so give each metric its share: "+
		"{currentCostPerMile:fuelCostPerMile}, {previousCostPerMile:fuelCostPerMile}, "+
		"{pricePerGallon:fuelCostPerMile}; and {tractorsAboveRange:fuelCostByTractor}, "+
		"{highestTractor:fuelCostByTractor}, {highestCostPerMile:fuelCostByTractor}.")
}
