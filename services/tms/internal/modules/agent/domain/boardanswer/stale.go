package boardanswer

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardtyped"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
)

func (t *translator) staleDays(answer *string, groups []group, measure board.Measure) *figure {
	spec, ok := measure.Spec(t.kind)
	if !ok || spec.Shape != boardcatalog.ShapeCountdown {
		return nil
	}
	texts := []string{*answer}
	for idx := range groups {
		texts = append(texts, groups[idx].note)
	}
	typed := boardtyped.Days(strings.Join(texts, "\n"))
	if len(typed) == 0 {
		return nil
	}
	shown := union(groups)
	if len(shown.IDs) != 1 {
		t.addf("the words type %s, a count of days that the rows' %s reads live and that is "+
			"wrong by tomorrow. Write the day it falls on instead (\"on Dec 1\"), {%s} for the "+
			"soonest, or one record's own as {%s:<its id>}", boardtyped.Quoted(typed), measure,
			NameLargest, measure)
		return nil
	}
	mark := fmt.Sprintf("{%s:%s}", measure, shown.IDs[0])
	*answer = boardtyped.MarkDays(*answer, mark)
	for idx := range groups {
		groups[idx].note = boardtyped.MarkDays(groups[idx].note, mark)
	}

	return &figure{
		raw:     mark[1 : len(mark)-1],
		measure: measure,
		record:  shown.IDs[0],
		metric:  noMetric,
	}
}
