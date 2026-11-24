package boardanswer

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardregistry"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardtyped"
	"github.com/emoss08/trenova/shared/inflect"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

func (t *translator) namesWhole(named string) bool {
	_, part := boardtyped.WholeOf(t.kind)
	_, measure := boardregistry.Catalog().MeasureOf(t.kind, named)
	own := boardtyped.Whole{Kind: t.kind, Nouns: boardtyped.NounsOf(t.kind)}

	return part && named != "" && !measure && !own.Names(named)
}

func (t *translator) typedWholes(answer *string, groups []group) {
	whole, ok := boardtyped.WholeOf(t.kind)
	texts, sets := []*string{answer}, make([][]pulid.ID, len(groups))
	for idx := range groups {
		texts = append(texts, &groups[idx].note)
		if groups[idx].set != nil {
			sets[idx] = groups[idx].set.IDs
		}
	}
	if problem := whole.Counted(texts, t.kind, sets, t.wholes); ok && problem != "" {
		t.addf("%s", problem)
	}
}

func (t *translator) wholesKnown(used []figure, groups []group) {
	whole, _ := boardtyped.WholeOf(t.kind)
	for _, found := range used {
		if _, known := whole.Of(scopeOf(&found, groups).IDs, t.wholes); found.whole && !known {
			t.addf("{%s} counts the %s the records belong to, and this conversation did not "+
				"read whose each one is; write {%s} for the records themselves", found.raw,
				inflect.PluralNoun(whole.Kind.Noun()), NameCount)
			return
		}
	}
}

func (t *translator) wholeReading(found *figure, key, scope string, groups []group) board.Reading {
	whole, _ := boardtyped.WholeOf(t.kind)
	label := stringutils.UpperFirst(inflect.PluralNoun(whole.Kind.Noun()))
	if scope != "" {
		label += ", " + scope
	}

	return whole.Reading(key, label, scopeOf(found, groups).IDs, t.wholes)
}
