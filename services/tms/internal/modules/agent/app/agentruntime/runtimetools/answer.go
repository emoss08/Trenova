package runtimetools

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/boardrecords"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/replycheck"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolcall"
	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardanswer"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardregistry"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardtyped"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/inflect"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const ShowAnswerTool = "show_answer"

const showAnswerDescription = "Answer in the conversation with the records your answer rests " +
	"on. Use it when the answer weighs two or more records together, or gives a count, a " +
	"total or a comparison across them: which loads are late and why, what one customer's " +
	"open invoices add up to, how two groups of loads differ. It is drawn in the conversation " +
	"as your reply and ends your turn: you are not asked again and write nothing after it, so " +
	"make every other call first (a proposal, another read). Give a title, the answer in a " +
	"sentence or two, and one group of records, two to set side by side, or three to stack, " +
	"each with a short title and an optional note. Records are ids a tool returned in this " +
	"conversation, all of one kind: %s. Trenova shows each record with its " +
	"status, and with its measure when you name one. Where the answer or a note says a " +
	"figure, write {count} (the records in all groups), {total} or {largest} (the sum and the " +
	"largest of the measure, so set measure). One record's own figure is its measure and its " +
	"number: {revenue:A-1001}, {lateness:A-1002}. That is how records are " +
	"compared, and one answer may read several measures that way: \"A-1001 is " +
	"{lateness:A-1001} behind and brings in {revenue:A-1001}, against " +
	"{lateness:A-1002} and {revenue:A-1002}\". A group of several records has " +
	"its own {count_1}, {total_1} and {largest_1}, {total_2} the second group's, and so on, " +
	"and any figure may name its measure after a colon, as {total:revenue}. Time is never " +
	"added up across records, and {count} counts only the records shown; when they are a " +
	"page of a longer list, {all} is the whole list's number.%s Trenova writes the value in and keeps it live. An amount or a percentage you type must be one a tool result in this " +
	"conversation holds, or one your own words work out from such figures; for a sum or the " +
	"largest across the records, write {total} or {largest}. Write plain sentences: no " +
	"markdown, no lists; the one link they may hold is to an artifact this turn made (a draft, a " +
	"write-up), as [words](artifact:<id>) with the id its result gave. A group may instead be " +
	"a metric Trenova keeps: give metric (one of " +
	"the names in its list), and for or period when the metric takes them; Trenova chooses how " +
	"it is drawn (a trend, bars, a grid, a log, checks or a table). Write a metric's figure as " +
	"{figure:metric}, as {lowest:onTimeByWeek}; each metric lists its figures. For columns " +
	"compared side by side, an analysis chart or a page beside the conversation, use show_board."

func showAnswerWords() string {
	kinds := boardregistry.Catalog().Kinds()
	names := make([]string, 0, len(kinds))
	wholes := ""
	for idx := range kinds {
		names = append(names, kinds[idx].Plural())
		if whole, part := boardtyped.WholeOf(kinds[idx].Kind); part && wholes == "" {
			plural := inflect.PluralNoun(whole.Kind.Noun())
			wholes = fmt.Sprintf(" Where the records are parts of others, {count:%s} is how "+
				"many %s the %s shown belong to: never type that count.", plural, plural,
				kinds[idx].Plural())
		}
	}

	return fmt.Sprintf(showAnswerDescription, stringutils.JoinAlternatives(names), wholes)
}

func ShowAnswerSpec() kernelports.ToolSpec {
	groups := schemaList(boardanswer.MaxGroups, schemaObject(map[string]any{
		boardPropTitle: schemaWords(board.MaxBlockTitleRunes, "What the records in this group "+
			"have in common, in a few words: \"Waiting on one driver's hours\"."),
		boardPropRecords: answerRecordsSchema(),
		"note": schemaWords(board.MaxCaptionRunes, "One plain line under the group: what to "+
			"know about these records. A figure goes in as the answer's do: one record's own "+
			"as {revenue:A-1001}, a group's as {total_1}."),
		boardPropMetric: metricNameSchema(),
		"for": schemaText("With metric: the id or number of the record the metric reads, " +
			"when it reads one."),
		"period": periodSchema(),
	}, boardPropTitle))
	groups[toolschema.KeyMinItems] = 1
	groups[toolschema.KeyDescription] = "One group, two to set side by side, or three to " +
		"stack: each records of one kind, or a metric."

	return kernelports.ToolSpec{
		Name:        ShowAnswerTool,
		Description: showAnswerWords(),
		Parameters: schemaObject(map[string]any{
			boardPropTitle: schemaWords(board.MaxTitleRunes, "What the answer is about, in a "+
				"few words and in sentence case: \"Late loads on the Dallas lanes\"."),
			"answer": schemaWords(boardanswer.MaxAnswerRunes, "The answer in a sentence or two, "+
				"as the person reads it, with {count}, {all}, {total} or {largest} where a figure goes."),
			boardPropMeasure: boardMeasureSchema(),
			"groups":         groups,
			"actions":        schemaList(boardanswer.MaxActions, boardActionSchema()),
		}, boardPropTitle, "answer", "groups"),
	}
}

func answerRecordsSchema() map[string]any {
	records := schemaRecordIDs("1 to " + strconv.Itoa(board.MaxListRecords) + " record ids " +
		"of one kind, each one a tool returned in this conversation.")
	records[toolschema.KeyMinItems] = 1
	records[toolschema.KeyMaxItems] = board.MaxListRecords

	return records
}

func AnswerStepOutcome(arguments map[string]any) toolcall.Result {
	answer, problems := boardanswer.Translate(arguments)
	if len(problems) > 0 {
		return toolcall.Failed("Tool %q was not run: %s.", ShowAnswerTool,
			strings.Join(problems, "; "))
	}

	return answerOutcome(answer)
}

func answerOutcome(answer *boardanswer.Answer) toolcall.Result {
	return toolcall.Result{
		Publishes: true,
		Data: agentruntimeports.BoardRequest{
			Board: answer.Board,
			Asked: board.PlaceConversation,
		},
	}
}

func showAnswer(t Turn, arguments map[string]any) toolcall.Result {
	read := boardrecords.Of(toolTexts(t)).ThisTurn(turnTexts(t))
	arguments, decoded := boardrecords.DecodeLists(arguments, "groups", "actions")
	arguments, numbered := read.Numbers(arguments)
	arguments, figured := read.FigureNumbers(arguments)
	repaired := decoded || numbered || figured
	shown, words := boardrecords.AnswerShown(arguments)
	if unkept := boardrecords.UnkeptFigures(turnTexts(t), words); len(unkept) > 0 {
		return toolcall.Failed("Tool %q was not run: %s %s nothing now: this turn's metric_read "+
			"returned no value for it (withheld or not kept), so it would be drawn as a dash. "+
			"Leave it out, and say in words what is not available if the question needs it.",
			ShowAnswerTool, strings.Join(unkept, ", "), inflect.Pluralize("reads", "read",
				len(unkept)))
	}
	if refusal := read.PartsRefusal(ShowAnswerTool, shown, words); refusal != "" {
		return toolcall.Failed("%s", refusal)
	}
	if unshown := read.Unshown(shown, words); len(unshown) > 0 &&
		!boardrecords.MetricsListRecords(boardrecords.GroupMetrics(arguments)) {
		swap, swaps := read.SwapFor(shown, words, unshown)
		if !swaps {
			return toolcall.Failed("%s", unshownRefusal(ShowAnswerTool, "group", unshown, &swap))
		}
		arguments, repaired = swap.Apply(arguments), true
	}
	if subjects, grouped := read.GroupFigures(arguments); grouped {
		arguments, repaired = subjects, true
	}
	if withMetrics, added := boardrecords.MetricGroups(arguments); len(added) > 0 {
		arguments, repaired = withMetrics, true
	}
	if withWholes, parted := read.WholesOf(arguments); parted {
		arguments, repaired = withWholes, true
	}
	answer, problems := boardanswer.Translate(arguments)
	if len(problems) > 0 {
		return answerRefusal(read, problems, shown)
	}
	if unread := unreadRecords(t, answer.Board.Records()); len(unread) > 0 {
		repairs, repairable := misprefixRepairs(readTexts(t), unread)
		if !repairable {
			return toolcall.Failed(
				"Tool %q was not run: the answer names %s, which no tool in this "+
					"conversation returned. Read them first, or leave them out; never write an id "+
					"from memory.%s",
				ShowAnswerTool,
				strings.Join(unread, ", "),
				misprefixed(readTexts(t), unread),
			)
		}
		arguments, repaired = boardrecords.ReplaceIDs(arguments, repairs), true
		if answer, problems = boardanswer.Translate(arguments); len(problems) > 0 {
			return answerRefusal(read, problems, shown)
		}
	}
	if unmade := unmadeLinks(t, answer.Board.ArtifactLinks()); len(unmade) > 0 {
		return linkRefusal(ShowAnswerTool, unmade)
	}
	count := countedWhole(read, answer, arguments)
	if count.refusal != "" {
		return toolcall.Failed("%s", count.refusal)
	}
	answer, arguments, repaired = count.answer, count.arguments, repaired || count.repaired
	if refusal := missingRecords(t, &answer.Board); refusal != "" {
		return toolcall.Failed("%s", refusal)
	}
	if typed := replycheck.UnreadAmounts(t, answer.Words); len(typed) > 0 {
		return toolcall.Failed(
			"Tool %q was not run: it states %s, which no tool result in this conversation "+
				"holds. Write {count}, {total} or {largest} where the figure goes (set measure "+
				"for a total or the largest), or take it out.",
			ShowAnswerTool,
			figureList(typed),
		)
	}

	outcome := answerOutcome(answer)
	if repaired {
		outcome.RanWith = arguments
	}

	return outcome
}

func answerRefusal(
	read boardrecords.Read,
	problems []string,
	shown map[recordkind.Kind][]pulid.ID,
) toolcall.Result {
	return toolcall.Failed("Tool %q was not run: %s.", ShowAnswerTool,
		read.WithExample(strings.Join(problems, "; "), shown))
}

func figureList(figures []string) string {
	quoted := make([]string, 0, len(figures))
	for _, figure := range figures {
		quoted = append(quoted, strconv.Quote(figure))
	}

	return strings.Join(quoted, ", ")
}
