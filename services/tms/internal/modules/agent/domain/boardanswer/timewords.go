package boardanswer

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardtyped"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
)

func (t *translator) timeWords(used []figure, groups []group, measure board.Measure,
	texts ...string,
) {
	for _, found := range used {
		read, timed := t.timeOf(&found, groups, measure)
		for _, text := range texts {
			if timed && !found.counts() {
				t.timeWordsIn(text, "{"+found.raw+"}", &found, read)
			}
		}
	}
}

func (t *translator) timeOf(found *figure, groups []group, measure board.Measure) (string, bool) {
	if found.metric != noMetric {
		spec, _ := groups[found.metric].metric.Figure(found.named)
		return spec.Name, isTime(spec.Value)
	}
	read := found.measureOr(measure)
	spec, known := read.Spec(t.kind)

	return string(read), known && isTime(spec.Value)
}

func (t *translator) timeWordsIn(text, mark string, found *figure, read string) {
	for offset := 0; ; {
		idx := strings.Index(text[offset:], mark)
		if idx < 0 {
			return
		}
		at := offset + idx
		if unit := boardtyped.Unit(text[at+len(mark):]); unit != "" {
			t.units = append(t.units, mark+unit)
		}
		if word := boardtyped.MoneyWord(boardtyped.Clause(text, at, at+len(mark))); word != "" {
			t.addf("{%s} reads %s, which is time, yet the words around it call it %q; say what "+
				"the time is (waited, past free time, behind), never an amount", found.raw, read, word)
			return
		}
		if word := boardtyped.DayWord(text[:at]); word != "" {
			t.addf("{%s} reads %s, which is how long, yet %q before it makes it a day; say how "+
				"long (\"has waited {%s}\", \"{%s} old\")", found.raw, read, word, found.raw, found.raw)
			return
		}
		offset = at + len(mark)
	}
}

func isTime(kind boardcatalog.ValueKind) bool {
	return kind == boardcatalog.ValueDuration || kind == boardcatalog.ValueHours ||
		kind == boardcatalog.ValueDays
}
