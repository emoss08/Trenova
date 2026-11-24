package runtimetools

import (
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardregistry"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
)

const besideMark = "lists beside it"

func missingRecords(t Turn, layout *board.Board) string {
	if askedForRecords(t) {
		return ""
	}
	for idx := range layout.Blocks {
		block := &layout.Blocks[idx]
		if block.Type != board.BlockMetric {
			continue
		}
		metric, known := boardregistry.Catalog().MetricNamed(block.Metric)
		if !known || metric.Records == nil {
			continue
		}
		if slices.ContainsFunc(layout.Blocks, func(other board.Block) bool {
			return other.Records != nil && other.Records.Kind == metric.Records.Kind
		}) {
			continue
		}
		return fmt.Sprintf("Tool %q was not run: the %s board %s %s, and no group holds "+
			"them. Read them with %s if this turn has not, then send it again with their ids "+
			"as a records group.", ShowAnswerTool, metric.Name, besideMark,
			metric.Records.Words, metric.Records.From)
	}

	return ""
}

func askedForRecords(t Turn) bool {
	return slices.ContainsFunc(t.Messages(), func(message llmports.Message) bool {
		return message.Role == llmports.RoleTool && message.ToolName == ShowAnswerTool &&
			strings.Contains(message.Content, besideMark)
	})
}
