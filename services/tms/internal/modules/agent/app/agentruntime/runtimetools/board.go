package runtimetools

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/argcheck"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/boardrecords"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/replycheck"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolcall"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolresult"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardparse"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	maxBoardBriefRunes = 6000
)

const showBoardDescription = "Lay out a larger answer as a board: a page beside the " +
	"conversation (place beside, the default) with a verdict of one or two sentences over up " +
	"to 8 blocks. For an answer in the conversation with the records it rests on, use " +
	"show_answer instead: it is smaller and it is the reply. Use show_board for what " +
	"show_answer does not have: records compared side by side in columns, a chart, a table, " +
	"checks, a meter, a record's history, or more than two groups. A board of at most 4 " +
	"blocks with place conversation is drawn in the conversation as your reply and ends your " +
	"turn, so call every other tool first and put what to do next on the board, as a prose " +
	"block or an actions block. After a board beside the conversation, reply in a sentence or " +
	"two that point to it. You write only the words: the title, the verdict, prose, captions " +
	"and at most two notes. Every number, status, row and count is read by Trenova from what " +
	"each block names when the person opens the board, so never type a figure into the words " +
	"(a duration, an amount, a percentage or a count is refused; record numbers, times of day, " +
	"dates and a span such as the next 24 hours are fine). When the words must say a figure, " +
	"declare it as a reading (up to 6, each a key, a label and one source, as a figure has) " +
	"and write {key} where its value goes, as in \"There are {late} late loads\": Trenova " +
	"writes the value in. Otherwise point at the block that shows it. Name only records your " +
	"tools returned in this conversation, by their ids. Cite a block in the verdict or in " +
	"prose as [words][n], n being the block's position from 1, with any {key} outside the " +
	"cited words; only blocks that read data can be cited. Blocks: figures (1 to 8 figures, " +
	"each a metric, records with a measure and how to add them up, or an analysis figure); " +
	"records (up to 25 records of one kind, each drawn with its status and an optional " +
	"measure, furthest first); compare (2 to 4 records side by side, rows taken from an " +
	"analysis table whose key column holds their ids); table (an analysis table or a table " +
	"artifact); chart (an analysis chart); timeline (one record's history); checks (up to 12, " +
	"each with a tone and the records it counts or a metric); meter (the checks of the checks " +
	"block checksBlock names, or a percentage metric, never both); prose; note (bad, warn, " +
	"info or ok); work (up to 12 records of one kind, each with what is open about it: its " +
	"Watchtower conditions, the messages about it and the changes waiting on a person, read " +
	"when the board opens); actions (1 to 3 buttons, each a short label and the request it " +
	"sends as the person's own message, with the records it is about: a button never changes " +
	"anything itself, it only asks you, and what you then propose waits for approval as " +
	"usual). A block that reads data takes an optional caption: one plain line under its " +
	"heading, where a reading's {key} may go. Actions and prose cannot be cited. A field a " +
	"block does not take is ignored, and a figure Trenova cannot read is left off. To revise " +
	"a board you showed, pass its boardId with the whole new board; the earlier one is kept " +
	"as a version. Write plain sentences: no markdown, no bold, no headings, no lists. A " +
	"prose block is one paragraph of at most 600 characters; to say more, use another block. " +
	"A note needs a title, what it says in a line, and at most 400 characters of text."

const ShowBoardTool = "show_board"

const UnboardableRefusal = "There is nowhere to show a board from here. Answer in your reply " +
	"instead."

const (
	boardPropType    = "type"
	boardPropTitle   = "title"
	boardPropLabel   = "label"
	boardPropRecords = "records"
	boardPropMetric  = "metric"
	boardPropMeasure = "measure"
	boardPropTone    = "tone"
	boardPropDetail  = "detail"
	boardPropID      = "boardId"
	boardPropPlace   = "place"
	boardPropKey     = "key"
	boardAnalysisRef = "The artifactId of an analysis this conversation kept."
)

func schemaText(description string) map[string]any {
	return map[string]any{
		toolschema.KeyType:        toolschema.TypeString,
		toolschema.KeyDescription: description,
	}
}

func schemaWords(most int, description string) map[string]any {
	words := schemaText(description)
	words[toolschema.KeyMaxLength] = most

	return words
}

func schemaPosition(description string) map[string]any {
	return map[string]any{
		toolschema.KeyType:        toolschema.TypeInteger,
		toolschema.KeyMinimum:     1,
		toolschema.KeyDescription: description,
	}
}

func schemaRecordIDs(description string) map[string]any {
	return map[string]any{
		toolschema.KeyType:        toolschema.TypeArray,
		toolschema.KeyItems:       map[string]any{toolschema.KeyType: toolschema.TypeString},
		toolschema.KeyDescription: description,
	}
}

func schemaList(most int, items map[string]any) map[string]any {
	return map[string]any{
		toolschema.KeyType:     toolschema.TypeArray,
		toolschema.KeyMaxItems: most,
		toolschema.KeyItems:    items,
	}
}

func schemaEnum(values ...string) map[string]any {
	return map[string]any{toolschema.KeyType: toolschema.TypeString, toolschema.KeyEnum: values}
}

func schemaObject(properties map[string]any, required ...string) map[string]any {
	return map[string]any{
		toolschema.KeyType:                 toolschema.TypeObject,
		toolschema.KeyProperties:           properties,
		toolschema.KeyRequired:             required,
		toolschema.KeyAdditionalProperties: false,
	}
}

func boardToneSchema() map[string]any {
	return schemaEnum(
		string(board.ToneBad), string(board.ToneWarn), string(board.ToneInfo), string(board.ToneOK),
	)
}

func boardMeasureSchema() map[string]any {
	return schemaText("What to measure each record by. A shipment: lateness (past its next " +
		"window, or estimated to miss one still open) or revenue. An invoice: amountDue, total " +
		"or age. A journal entry: amount or pendingFor.")
}

func boardMetricSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType:        toolschema.TypeString,
		toolschema.KeyEnum:        board.MetricKeys(),
		toolschema.KeyDescription: "A figure Trenova keeps for the whole organization.",
	}
}

func boardFigureProperties(label string) map[string]any {
	aggregate := schemaEnum(
		string(board.AggregateCount), string(board.AggregateSum), string(board.AggregateMax),
	)
	aggregate[toolschema.KeyDescription] = "count (the records, or those whose measure is " +
		"above zero), sum or max of the measure. Defaults to count. Lateness only an " +
		"estimate projects is shown on its row and left out of figures."

	return map[string]any{
		boardPropLabel:   schemaWords(board.MaxLabelRunes, label),
		boardPropMetric:  boardMetricSchema(),
		boardPropRecords: schemaRecordIDs("The records the figure adds up."),
		boardPropMeasure: boardMeasureSchema(),
		"aggregate":      aggregate,
		"analysis":       schemaText(boardAnalysisRef),
		"figure":         schemaPosition("The analysis figure's position, from 1."),
	}
}

func boardFigureSchema() map[string]any {
	return schemaObject(
		boardFigureProperties("What the figure is, in a few words, with no number in it."),
		boardPropLabel,
	)
}

func boardReadingSchema() map[string]any {
	properties := boardFigureProperties("What the value is, in a few words, with no number " +
		"in it: shown where the person points at the value.")
	properties[boardPropKey] = schemaWords(board.MaxReadingKeyRunes, "The name the words "+
		"place it by as {key}: letters, digits and underscores, starting with a letter, "+
		"such as late, lateSix or furthest_behind. It is case-sensitive: {lateSix} places the "+
		"reading declared as lateSix.")

	return schemaObject(properties, boardPropKey, boardPropLabel)
}

func boardCheckSchema() map[string]any {
	return schemaObject(map[string]any{
		boardPropTitle: schemaWords(board.MaxCheckTitleRunes, "What the check is, in a line."),
		boardPropTone:  boardToneSchema(),
		boardPropDetail: schemaWords(board.MaxCheckDetailRunes,
			"One more line, with no number in it."),
		boardPropRecords: schemaRecordIDs("The records the check counts."),
		boardPropMetric:  boardMetricSchema(),
	}, boardPropTitle, boardPropTone)
}

func boardCompareRowSchema() map[string]any {
	return schemaObject(map[string]any{
		"column": schemaText("The analysis table's column."),
		boardPropLabel: schemaWords(board.MaxLabelRunes,
			"What the row reads as, when the column's name is not plain."),
		"better": schemaEnum(string(board.BetterHigher), string(board.BetterLower)),
	}, "column")
}

func boardActionSchema() map[string]any {
	return schemaObject(map[string]any{
		boardPropLabel: schemaWords(board.MaxActionLabelRunes,
			"What the button says, a few words in sentence case: \"Draft recoveries\"."),
		"ask": schemaWords(board.MaxActionAskRunes, "The request the button sends as the "+
			"person's own message, in their voice: \"Draft a recovery for each of these.\""),
		boardPropRecords: schemaRecordIDs("The records the request is about."),
	}, boardPropLabel, "ask")
}

func boardBlockSchema() map[string]any {
	half := map[string]any{
		toolschema.KeyType:        toolschema.TypeBoolean,
		toolschema.KeyDescription: "Half the width: two half blocks in a row share a line.",
	}

	return schemaObject(map[string]any{
		boardPropType: schemaEnum(blockTypeNames()...),
		boardPropTitle: schemaWords(board.MaxNoteTitleRunes, "The block's heading, in a few "+
			"words: at most 60 characters. A note must have one: what it says, in a line."),
		"half":           half,
		"figures":        schemaList(board.MaxFigures, boardFigureSchema()),
		boardPropRecords: schemaRecordIDs("records, compare, note and work: the record ids."),
		"actions":        schemaList(board.MaxActions, boardActionSchema()),
		boardPropMeasure: boardMeasureSchema(),
		"order":          schemaEnum(string(board.OrderMeasure), string(board.OrderGiven)),
		"analysis":       schemaText(boardAnalysisRef),
		"artifact":       schemaText("table: the artifactId of a table this conversation kept."),
		"table":          schemaPosition("The analysis table's position, from 1."),
		"chart":          schemaPosition("The analysis chart's position, from 1."),
		"key": schemaText(
			"compare: the analysis table's column that holds the record ids.",
		),
		"caption": schemaWords(board.MaxCaptionRunes, "A block that reads data: one plain "+
			"line under its heading. A figure goes in as a reading's {key}."),
		"rows":   schemaList(board.MaxCompareRows, boardCompareRowSchema()),
		"limit":  schemaPosition("table and timeline: how many rows or entries to show."),
		"record": schemaText("timeline: the id of the record whose history to show."),
		"checks": schemaList(board.MaxChecks, boardCheckSchema()),
		"checksBlock": schemaPosition("meter, instead of metric: the position of the " +
			"checks block whose checks it fills with."),
		boardPropMetric: boardMetricSchema(),
		"text": schemaWords(board.MaxProseRunes, "prose: one paragraph of plain sentences, "+
			"at most 600 characters. note: at most 400 characters. A figure goes in as a "+
			"reading's {key}."),
		boardPropTone: boardToneSchema(),
	}, boardPropType)
}

func ShowBoardSpec() kernelports.ToolSpec {
	blocks := schemaList(board.MaxBlocks, boardBlockSchema())
	blocks[toolschema.KeyMinItems] = 1

	return kernelports.ToolSpec{
		Name:        ShowBoardTool,
		Description: showBoardDescription,
		Parameters: schemaObject(map[string]any{
			boardPropTitle: schemaWords(board.MaxTitleRunes, "What the board answers, in a few "+
				"words and in sentence case: \"Six late shipments on the Dallas lanes\"."),
			"verdict": schemaWords(board.MaxVerdictRunes, "The answer in one or two short "+
				"sentences, citing the blocks it rests on as [words][n], with each figure it "+
				"states as a reading's {key}. In the conversation it is the reply the person "+
				"reads. Leave the detail to the blocks."),
			boardPropPlace: boardPlaceSchema(),
			"readings":     schemaList(board.MaxReadings, boardReadingSchema()),
			"blocks":       blocks,
			boardPropID: schemaText("The artifactId of a board you showed earlier in this " +
				"conversation, to replace it with this one. Leave it out for a new board."),
		}, boardPropTitle, "verdict", "blocks"),
	}
}

func boardPlaceSchema() map[string]any {
	place := schemaEnum(string(board.PlaceConversation), string(board.PlaceBeside))
	place[toolschema.KeyDescription] = "beside (the default): a page beside the conversation, " +
		"after which you reply. conversation: drawn in the conversation as your reply, which " +
		"ends your turn; a board of more than 4 blocks goes beside instead. For an answer in " +
		"the conversation, show_answer is the smaller tool."

	return place
}

func blockTypeNames() []string {
	names := make([]string, 0, len(board.AllBlockTypes()))
	for _, blockType := range board.AllBlockTypes() {
		names = append(names, string(blockType))
	}

	return names
}

func BoardStepOutcome(arguments map[string]any) toolcall.Result {
	fields := make(map[string]any, len(arguments))
	for key, value := range arguments {
		if key != boardPropID {
			fields[key] = value
		}
	}
	parsed, problems := boardparse.Parse(fields)

	var revises pulid.ID
	if raw := strings.TrimSpace(argcheck.String(arguments, boardPropID)); raw != "" {
		id, err := pulid.Parse(raw)
		if err != nil {
			problems = append(problems, "boardId is not an id a show_board result gave you")
		}
		revises = id
	}
	if len(problems) > 0 {
		return toolcall.Failed(
			"Tool %q was not run: %s.", ShowBoardTool, strings.Join(problems, "; "),
		)
	}

	return toolcall.Result{
		Publishes: true,
		Data: agentruntimeports.BoardRequest{
			Board:      *parsed,
			ArtifactID: revises,
			Asked:      board.Place(strings.TrimSpace(argcheck.String(arguments, boardPropPlace))),
		},
	}
}

func showBoard(t Turn, arguments map[string]any) toolcall.Result {
	read := boardrecords.Of(toolTexts(t))
	arguments, decoded := boardrecords.DecodeLists(arguments, "blocks", "readings")
	arguments, numbered := read.Numbers(arguments)
	repaired := decoded || numbered
	outcome := BoardStepOutcome(arguments)
	if outcome.Failed {
		return outcome
	}
	request, _ := outcome.Data.(agentruntimeports.BoardRequest)

	if unread := unreadRecords(t, request.Board.Records()); len(unread) > 0 {
		return toolcall.Failed(
			"Tool %q was not run: the board names %s, which no tool in this "+
				"conversation returned. Read them first, or leave them out; never write an id from "+
				"memory.%s",
			ShowBoardTool,
			strings.Join(unread, ", "),
			misprefixed(readTexts(t), unread),
		)
	}
	words := boardWords(&request.Board)
	if unshown := read.Unshown(request.Board.Records(), words); len(unshown) > 0 {
		swap, swaps := read.SwapFor(request.Board.Records(), words, unshown)
		if !swaps {
			return toolcall.Failed("%s", boardrecords.Refusal(ShowBoardTool, "block", unshown))
		}
		arguments, repaired = swap.Apply(arguments), true
		if outcome = BoardStepOutcome(arguments); outcome.Failed {
			return outcome
		}
	}
	if refusal := partialCount(read, &request.Board); refusal != "" {
		return toolcall.Failed("%s", refusal)
	}
	if repaired {
		outcome.RanWith = arguments
	}

	return outcome
}

func partialCount(read boardrecords.Read, layout *board.Board) string {
	counted := func(source *board.FigureSource, name string) string {
		if source.Aggregate != board.AggregateCount || source.Measure != "" ||
			source.Records == nil {
			return ""
		}
		if total, longer := read.Longer(source.Records.IDs); longer {
			return boardrecords.CountRefusal(ShowBoardTool, name, len(source.Records.IDs), total)
		}
		return ""
	}
	for idx := range layout.Readings {
		reading := &layout.Readings[idx]
		if refusal := counted(&reading.FigureSource, "{"+reading.Key+"}"); refusal != "" {
			return refusal
		}
	}
	for idx := range layout.Blocks {
		for figure := range layout.Blocks[idx].Figures {
			source := &layout.Blocks[idx].Figures[figure]
			if refusal := counted(source, strconv.Quote(source.Label)); refusal != "" {
				return refusal
			}
		}
	}

	return ""
}

func boardWords(layout *board.Board) string {
	texts := []string{layout.Title, layout.Verdict}
	for idx := range layout.Blocks {
		block := &layout.Blocks[idx]
		texts = append(texts, block.Title, block.Text, block.Caption)
		for _, check := range block.Checks {
			texts = append(texts, check.Title, check.Detail)
		}
	}

	return strings.Join(slices.DeleteFunc(texts, func(text string) bool {
		return text == ""
	}), "\n\n")
}

func toolTexts(t Turn) []string {
	read := make([]string, 0, len(t.Messages()))
	for idx := range t.Messages() {
		message := &t.Messages()[idx]
		if message.Role == llmports.RoleTool && !message.IsError {
			read = append(read, message.Content)
		}
	}

	return read
}

func readTexts(t Turn) []string {
	read := toolTexts(t)
	if t.Request() != nil {
		read = append(read, t.Request().Input)
	}

	return read
}

func unreadRecords(t Turn, refs map[permission.Resource][]pulid.ID) []string {
	read := readTexts(t)
	unread := make([]string, 0, 2)
	for _, ids := range refs {
		for _, id := range ids {
			text := id.String()
			if !slices.ContainsFunc(read, func(content string) bool {
				return strings.Contains(content, text)
			}) {
				unread = append(unread, text)
			}
		}
	}
	slices.Sort(unread)

	return unread
}

func BoardContent(
	tool string,
	shown *agentruntimeports.ShownArtifact,
	asked board.Place,
) string {
	place := board.Place(shown.Place)
	if place != board.PlaceConversation {
		place = board.PlaceBeside
	}
	encoded := fmt.Sprintf(`{"artifactId":%q,"title":%q,"status":"shown","place":%q}`,
		shown.ID.String(), shown.Title, string(place))
	brief := stringutils.TruncateRunes(shown.Brief, maxBoardBriefRunes)
	fenced := toolresult.Fence(tool, encoded+"\n\n"+brief)

	if place == board.PlaceConversation {
		return fenced + "\n\n[The board was drawn in the conversation as your reply, and the " +
			"turn ended with it: the person read its verdict with each reading written in, and " +
			"its blocks under it. It is read live whenever it is opened; above is what it " +
			"showed then. When you come back to it, do not repeat its figures or rows." +
			replycheck.ArtifactRefNote(shown) + "]"
	}
	moved := ""
	if asked == board.PlaceConversation {
		moved = fmt.Sprintf("You asked for it in the conversation, but it holds more than %d "+
			"blocks, so it was put beside the conversation instead. ", board.MaxConversationBlocks)
	}

	return fenced + "\n\n[" + moved + "The board is beside the conversation, read live " +
		"whenever it is opened. Above is what it shows now. Reply in one or two sentences that " +
		"point to it and say what to do; do not repeat its figures or rows." +
		replycheck.ArtifactRefNote(shown) + "]"
}

func ConversationBoard(outcome *toolcall.Result) *agentruntimeports.ShownArtifact {
	shown := outcome.Shown
	if outcome.Failed || shown == nil || shown.Kind != string(assistantartifact.KindBoard) ||
		board.Place(shown.Place) != board.PlaceConversation {
		return nil
	}

	return shown
}

func BoardReply(shown *agentruntimeports.ShownArtifact) string {
	if answer := strings.TrimSpace(shown.Answer); answer != "" {
		return replycheck.AttachRef(answer, shown)
	}

	return replycheck.ArtifactRef(shown)
}
