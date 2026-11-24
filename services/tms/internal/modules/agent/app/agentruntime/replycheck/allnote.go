package replycheck

import (
	"strconv"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolresult"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardtyped"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/shared/inflect"
)

const AllFigure = "{all}"

func partNote(content string, shown *agentruntimeports.ShownArtifact) string {
	if shown == nil || !shown.Tabular() {
		return ""
	}
	payload, ok := toolresult.Payload(content)
	if !ok {
		return ""
	}
	fields, _ := payload.(map[string]any)
	total, _ := fields["total"].(float64)
	count, _ := fields["count"].(float64)
	if total <= count {
		return ""
	}

	return " They are a page of a list of " + strconv.Itoa(int(total)) + ": where the words " +
		"mean the whole list, write " + AllFigure + ", its own number, read again whenever " +
		"the board is read; {count} is only the records you show."
}

func wholesNote(content string) string {
	for _, ref := range boardRecords(content) {
		whole, part := boardtyped.WholeOf(recordkind.Kind(ref.Type))
		if !part {
			continue
		}
		wholes := inflect.PluralNoun(whole.Kind.Noun())

		return " Each belongs to one of the " + wholes + ": where the words count the " +
			wholes + " rather than the " + inflect.PluralNoun(recordkind.Kind(ref.Type).Noun()) +
			", write {count:" + wholes + "}, or {count_1:" + wholes + "} for one group's, " +
			"never a typed number."
	}

	return ""
}
