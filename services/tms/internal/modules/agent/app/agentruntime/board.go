package agentruntime

import (
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	showBoardName      = "show_board"
	maxBoardBriefRunes = 6000
)

const showBoardDescription = "Lay out your answer to a question that spans several records as " +
	"a short page beside the conversation: a board. Use it when the answer compares, ranks or " +
	"explains a set of records, a close or a lane, and would otherwise be a long reply. You " +
	"write only the words: the title, a verdict of one or two sentences, prose and at most two " +
	"notes. Every number, status, row and count is read by Trenova from what each block names " +
	"when the person opens the board, so never write a figure in the words: point at the block " +
	"that shows it. Name only records your tools returned in this conversation, by their ids. " +
	"Cite a block in the verdict or in prose as [words][n], n being the block's position from " +
	"1; only blocks that read data can be cited. Blocks: figures (1 to 4 figures, each a " +
	"metric, records with a measure and how to add them up, or an analysis figure); records " +
	"(up to 25 records of one kind, each drawn with its status and an optional measure, " +
	"furthest first); compare (2 to 4 records side by side, rows taken from an analysis table " +
	"whose key column holds their ids); table (an analysis table or a table artifact); chart " +
	"(an analysis chart); timeline (one record's history); checks (up to 12, each with a tone " +
	"and the records it counts or a metric); meter (the checks of a checks block, or a " +
	"percentage metric); prose; note (bad, warn, info or ok). To revise a board you showed, " +
	"pass its boardId with the whole new board; the earlier one is kept as a version. Write " +
	"plain sentences: no markdown, no bold, no headings, no lists. A prose block is one " +
	"paragraph of at most 600 characters; to say more, use another block. A note needs a " +
	"title, what it says in a line, and at most 400 characters of text."

const unboardableRefusal = "There is nowhere to show a board from here. Answer in your reply " +
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

func boardFigureSchema() map[string]any {
	aggregate := schemaEnum(
		string(board.AggregateCount), string(board.AggregateSum), string(board.AggregateMax),
	)
	aggregate[toolschema.KeyDescription] = "count (the records, or those whose measure is " +
		"above zero), sum or max of the measure. Defaults to count. Lateness only an " +
		"estimate projects is shown on its row and left out of figures."

	return schemaObject(map[string]any{
		boardPropLabel: schemaWords(board.MaxLabelRunes,
			"What the figure is, in a few words, with no number in it."),
		boardPropMetric:  boardMetricSchema(),
		boardPropRecords: schemaRecordIDs("The records the figure adds up."),
		boardPropMeasure: boardMeasureSchema(),
		"aggregate":      aggregate,
		"analysis":       schemaText(boardAnalysisRef),
		"figure":         schemaPosition("The analysis figure's position, from 1."),
	}, boardPropLabel)
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
		boardPropRecords: schemaRecordIDs("records, compare and note: the record ids."),
		boardPropMeasure: boardMeasureSchema(),
		"order":          schemaEnum(string(board.OrderMeasure), string(board.OrderGiven)),
		"analysis":       schemaText(boardAnalysisRef),
		"artifact":       schemaText("table: the artifactId of a table this conversation kept."),
		"table":          schemaPosition("The analysis table's position, from 1."),
		"chart":          schemaPosition("The analysis chart's position, from 1."),
		"key": schemaText(
			"compare: the analysis table's column that holds the record ids.",
		),
		"rows":          schemaList(board.MaxCompareRows, boardCompareRowSchema()),
		"limit":         schemaPosition("table and timeline: how many rows or entries to show."),
		"record":        schemaText("timeline: the id of the record whose history to show."),
		"checks":        schemaList(board.MaxChecks, boardCheckSchema()),
		"of":            schemaPosition("meter: the position of the checks block it measures."),
		boardPropMetric: boardMetricSchema(),
		"text": schemaWords(board.MaxProseRunes, "prose: one paragraph of plain sentences, "+
			"at most 600 characters. note: at most 400 characters."),
		boardPropTone: boardToneSchema(),
	}, boardPropType)
}

func showBoardSpec() kernelports.ToolSpec {
	blocks := schemaList(board.MaxBlocks, boardBlockSchema())
	blocks[toolschema.KeyMinItems] = 1

	return kernelports.ToolSpec{
		Name:        showBoardName,
		Description: showBoardDescription,
		Parameters: schemaObject(map[string]any{
			boardPropTitle: schemaWords(board.MaxTitleRunes, "What the board answers, in a few "+
				"words and in sentence case: \"Six late shipments on the Dallas lanes\"."),
			"verdict": schemaWords(board.MaxVerdictRunes, "The answer in one or two short "+
				"sentences, citing the blocks it rests on as [words][n]. Leave the detail to "+
				"the blocks."),
			"blocks": blocks,
			boardPropID: schemaText("The artifactId of a board you showed earlier in this " +
				"conversation, to replace it with this one. Leave it out for a new board."),
		}, boardPropTitle, "verdict", "blocks"),
	}
}

func blockTypeNames() []string {
	names := make([]string, 0, len(board.AllBlockTypes()))
	for _, blockType := range board.AllBlockTypes() {
		names = append(names, string(blockType))
	}

	return names
}

func boardStepOutcome(arguments map[string]any) toolOutcome {
	fields := make(map[string]any, len(arguments))
	for key, value := range arguments {
		if key != boardPropID {
			fields[key] = value
		}
	}
	parsed, problems := board.Parse(fields)

	var revises pulid.ID
	if raw := strings.TrimSpace(stringArg(arguments, boardPropID)); raw != "" {
		id, err := pulid.Parse(raw)
		if err != nil {
			problems = append(problems, "boardId is not an id a show_board result gave you")
		}
		revises = id
	}
	if len(problems) > 0 {
		return failedOutcome(
			"Tool %q was not run: %s.", showBoardName, strings.Join(problems, "; "),
		)
	}

	return toolOutcome{
		publishes: true,
		data:      agentports.BoardRequest{Board: *parsed, ArtifactID: revises},
	}
}

func (t *Turn) showBoard(arguments map[string]any) toolOutcome {
	outcome := boardStepOutcome(arguments)
	if outcome.failed {
		return outcome
	}
	request, _ := outcome.data.(agentports.BoardRequest)

	if unread := t.unreadRecords(request.Board.Records()); len(unread) > 0 {
		return failedOutcome("Tool %q was not run: the board names %s, which no tool in this "+
			"conversation returned. Read them first, or leave them out; never write an id from "+
			"memory.", showBoardName, strings.Join(unread, ", "))
	}
	if figures := readFinding(t, boardWords(&request.Board)).figures; len(figures) > 0 {
		return failedOutcome("Tool %q was not run: its words state %s, which nothing this "+
			"conversation read. The board shows its figures from its blocks; write the words "+
			"without numbers and cite the block that shows them.",
			showBoardName, strings.Join(figures, ", "))
	}

	return outcome
}

func (t *Turn) unreadRecords(refs map[permission.Resource][]pulid.ID) []string {
	read := make([]string, 0, len(t.messages)+1)
	for idx := range t.messages {
		message := &t.messages[idx]
		if message.Role == agentports.RoleTool && !message.IsError {
			read = append(read, message.Content)
		}
	}
	if t.req != nil {
		read = append(read, t.req.Input)
	}

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

func boardWords(shown *board.Board) string {
	words := make([]string, 0, len(shown.Blocks)+2)
	words = append(words, shown.Title, board.PlainText(shown.Verdict))
	for idx := range shown.Blocks {
		block := &shown.Blocks[idx]
		words = append(words, block.Title, board.PlainText(block.Text))
		for figure := range block.Figures {
			words = append(words, block.Figures[figure].Label)
		}
		for check := range block.Checks {
			words = append(words, block.Checks[check].Title, block.Checks[check].Detail)
		}
	}

	return strings.Join(slices.DeleteFunc(words, func(word string) bool { return word == "" }),
		"\n")
}

func boardContent(shown *agentports.ShownArtifact) string {
	encoded := fmt.Sprintf(`{"artifactId":%q,"title":%q,"status":"shown"}`,
		shown.ID.String(), shown.Title)
	brief := stringutils.TruncateRunes(shown.Brief, maxBoardBriefRunes)

	return FenceToolResult(showBoardName, encoded+"\n\n"+brief) +
		"\n\n[The board is beside the conversation, read live whenever it is opened. Above is " +
		"what it shows now. Reply in one or two sentences that point to it and say what to " +
		"do; do not repeat its figures or rows." + artifactRefNote(shown) + "]"
}
