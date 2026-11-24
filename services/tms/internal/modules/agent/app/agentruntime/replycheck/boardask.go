package replycheck

import (
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/agent/domain/boardhint"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolresult"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardregistry"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	showBoardTool       = "show_board"
	showAnswerTool      = "show_answer"
	publishArtifactTool = "publish_artifact"
	minBoardRecords     = 2
	minBoardLabelRunes  = 4

	boardNoteLead = "\n\n[These are "
	boardNoteTail = " records. If your answer weighs them together, or gives a count, a total " +
		"or a comparison across them, do not point to a table and do not answer in prose: " +
		"answer with show_answer, naming the records it rests on and writing {count}, {total} " +
		"or {largest} where a figure goes, never typed. It is drawn in the conversation as " +
		"your reply and ends your turn, so make every other call first. Only a list the " +
		"person just asked to see is pointed to, and one record or one fact is a sentence.]"

	boardCorrectionLead = "Your reply answers across several records in prose:"
	BoardRetryReason    = "The reply answered across several records in prose, so it is " +
		"being asked for a board."
)

type BoardAskState struct {
	Asked   bool
	Drafted bool
}

func WithBoardNote(
	t Turn,
	tool, content string,
	failed bool,
	shown *agentruntimeports.ShownArtifact,
) string {
	if failed || tool == showBoardTool || tool == showAnswerTool || tool == publishArtifactTool ||
		!boardable(t) {
		return content
	}
	if hint := resultHint(content); hint != "" {
		return content + showNote(hint)
	}
	count := len(boardRecords(content))
	if labels, rows := tableRecords(shown); max(rows, len(labels)) > count {
		count = max(rows, len(labels))
	}
	if count < minBoardRecords {
		return content
	}
	if hint := turnHint(t); hint != "" {
		return content + showNote(hint)
	}
	note := boardNoteLead + strconv.Itoa(count) + strings.TrimSuffix(boardNoteTail, "]") +
		partNote(content, shown) + wholesNote(content) + "]"
	if follow := followUpOf(content, shown); follow != nil {
		note = strings.TrimSuffix(note, "]") + " " + boardhint.FollowUpLine(follow) + "]"
	}

	return content + note
}

func followUpOf(
	content string,
	shown *agentruntimeports.ShownArtifact,
) *boardcatalog.FollowUp {
	kinds := make([]recordkind.Kind, 0, 2)
	if shown != nil && shown.RowKind != "" {
		kinds = append(kinds, recordkind.Kind(shown.RowKind))
	}
	for _, ref := range boardRecords(content) {
		if kind := recordkind.Kind(ref.Type); !slices.Contains(kinds, kind) {
			kinds = append(kinds, kind)
		}
	}
	for _, kind := range kinds {
		if measure, ok := boardregistry.Catalog().DefaultMeasure(kind); ok &&
			measure.FollowUp != nil {
			return measure.FollowUp
		}
	}

	return nil
}

func showNote(hint string) string {
	return "\n\n[" + hint + "]"
}

func ReplaceShownNote(content, stale, folded string) (string, bool) {
	body, note := splitBoardNote(content)
	if !strings.HasSuffix(body, stale) {
		return content, false
	}

	return strings.TrimSuffix(body, stale) + folded + note, true
}

func splitBoardNote(content string) (body, note string) {
	if hint := toolresult.HowToShow(content); hint != "" &&
		strings.HasSuffix(content, showNote(hint)) {
		return strings.TrimSuffix(content, showNote(hint)), showNote(hint)
	}
	if start := strings.LastIndex(content, "\n\n["+boardhint.Lead); start >= 0 &&
		strings.HasSuffix(content, "]") {
		return content[:start], content[start:]
	}
	start := strings.LastIndex(content, boardNoteLead)
	if start < 0 || !strings.HasSuffix(content, "]") {
		return content, ""
	}
	count, rest, cut := strings.Cut(content[start+len(boardNoteLead):], " ")
	if _, err := strconv.Atoi(count); err != nil || !cut ||
		!strings.HasPrefix(" "+rest, strings.TrimSuffix(boardNoteTail, "]")) {
		return content, ""
	}

	return content[:start], content[start:]
}

func AskForBoard(
	t Turn,
	draft *llmports.ChatCompletionResult,
	state *BoardAskState,
) (SendBack, bool) {
	if state.Asked || draft == nil || !boardable(t) || len(t.Result().Actions) > 0 {
		return SendBack{}, false
	}
	var correction string
	stated := statedMetricFigures(t, draft.Text)
	subject := turnSubject(t)
	switch {
	case stated.metric != "":
		correction = metricCorrection(&stated, unreadRecordsOf(t, stated.metric))
	case subject.metric != "" && setOutItems(draft.Text) >= minSetOutItems:
		correction = subjectCorrection(&subject)
	default:
		if _, table := PointToTables(draft.Text, KeptTables(t)); table != nil {
			return SendBack{}, false
		}
		linked := pointsToList(t, draft.Text)
		if linked && !weighsLinkedRecords(t, draft.Text) && !setsOutHinted(t, draft.Text) {
			return SendBack{}, false
		}
		named := namedBoardRecords(t, draft.Text)
		if len(named) < minBoardRecords || (!linked && !weighsInProse(t, draft.Text)) {
			return SendBack{}, false
		}
		correction = boardCorrection(named, turnHint(t))
	}

	state.Asked = true
	back := SendBackDraft(
		draft,
		BoardRetryReason,
		PassEvent(assistantstreamports.RegroundAnswerWithBoard, BoardRetryReason, nil),
		correction,
	)
	back.Guard = GuardBoard

	return back, true
}

func turnHint(t Turn) string {
	messages := t.Messages()
	for idx := len(messages) - 1; idx >= 0; idx-- {
		message := &messages[idx]
		if message.Role == llmports.RoleUser {
			return ""
		}
		if message.Role != llmports.RoleTool || message.IsError {
			continue
		}
		if hint := resultHint(message.Content); hint != "" {
			return hint
		}
	}

	return ""
}

func pointsToList(t Turn, reply string) bool {
	return slices.ContainsFunc(ArtifactRefIDs(reply), func(id string) bool {
		return !slices.ContainsFunc(
			t.Artifacts(),
			func(shown agentruntimeports.ShownArtifact) bool {
				return shown.ID.String() == id &&
					shown.Kind == string(assistantartifact.KindAnalysis)
			},
		)
	})
}

func weighsLinkedRecords(t Turn, reply string) bool {
	if !writesOutTable(t, reply) && !namesLookedUp(t, reply) {
		return false
	}

	return slices.ContainsFunc(strings.Split(reply, "\n\n"), func(paragraph string) bool {
		return writesOutTable(t, paragraph) || namesLookedUp(t, paragraph)
	})
}

func UnreadAmounts(t Turn, words string) []string {
	return readFinding(t, words).figures
}

func namedBoardRecords(t Turn, reply string) []string {
	named := make([]string, 0, 4)
	seen := make(map[string]bool, 16)
	for idx := range t.Messages() {
		message := &t.Messages()[idx]
		if message.Role != llmports.RoleTool || message.IsError ||
			message.ToolName == showBoardTool || message.ToolName == showAnswerTool ||
			message.ToolName == publishArtifactTool {
			continue
		}
		for _, ref := range boardRecords(message.Content) {
			label := strings.TrimSpace(ref.Label)
			if seen[label] || utf8.RuneCountInString(label) < minBoardLabelRunes {
				continue
			}
			seen[label] = true
			if stringutils.ContainsWords(reply, label) {
				named = append(named, label)
			}
		}
	}
	for idx := range t.Artifacts() {
		labels, _ := tableRecords(&t.Artifacts()[idx])
		for _, label := range labels {
			label = strings.TrimSpace(label)
			if seen[label] || utf8.RuneCountInString(label) < minBoardLabelRunes {
				continue
			}
			seen[label] = true
			if stringutils.ContainsWords(reply, label) {
				named = append(named, label)
			}
		}
	}

	return named
}

func writesOutTable(t Turn, reply string) bool {
	return slices.ContainsFunc(t.Artifacts(), func(shown agentruntimeports.ShownArtifact) bool {
		labels, rows := tableRecords(&shown)
		if shown.Paged || rows < minBoardRecords || rows > board.MaxListRecords ||
			len(labels) != rows {
			return false
		}

		return !slices.ContainsFunc(labels, func(label string) bool {
			label = strings.TrimSpace(label)
			return utf8.RuneCountInString(label) < minBoardLabelRunes ||
				!stringutils.ContainsWords(reply, label)
		})
	})
}

func namesLookedUp(t Turn, reply string) bool {
	asked := askedTexts(t.Messages())
	if len(asked) == 0 {
		return false
	}

	seen := make(map[string]bool, 8)
	named := 0
	for idx := range t.Messages() {
		message := &t.Messages()[idx]
		if message.Role != llmports.RoleTool || message.IsError {
			continue
		}
		for _, ref := range boardRecords(message.Content) {
			label := strings.TrimSpace(ref.Label)
			if seen[ref.ID] || utf8.RuneCountInString(label) < minBoardLabelRunes ||
				(!asked[ref.ID] && !asked[strings.ToLower(label)]) {
				continue
			}
			seen[ref.ID] = true
			if stringutils.ContainsWords(reply, label) {
				named++
			}
		}
	}

	return named >= minBoardRecords
}

func askedTexts(messages []llmports.Message) map[string]bool {
	asked := make(map[string]bool, 8)
	var walk func(value any)
	walk = func(value any) {
		switch typed := value.(type) {
		case string:
			if text := strings.TrimSpace(typed); text != "" {
				asked[text] = true
				asked[strings.ToLower(text)] = true
			}
		case []any:
			for _, element := range typed {
				walk(element)
			}
		case map[string]any:
			for _, element := range typed {
				walk(element)
			}
		}
	}
	for idx := range messages {
		if messages[idx].Role != llmports.RoleAssistant {
			continue
		}
		for _, call := range messages[idx].ToolCalls {
			if call.Name != showAnswerTool && call.Name != showBoardTool {
				walk(call.Arguments)
			}
		}
	}

	return asked
}

func tableRecords(shown *agentruntimeports.ShownArtifact) (labels []string, rows int) {
	if shown == nil || !shown.Tabular() ||
		!boardregistry.Catalog().Supports(recordkind.Kind(shown.RowKind)) {
		return nil, 0
	}

	return shown.Labels, shown.Rows
}

func boardCorrection(named []string, hint string) string {
	if hint != "" {
		hint = " As this turn's result said: " + hint
	}

	return boardCorrectionLead + " it names " + strings.Join(named, ", ") + ". An answer " +
		"that weighs records together, or gives a count, a total or a comparison across them, " +
		"belongs on a board in the conversation, not in prose and not as a pointer to a table: " +
		"call show_answer with the records it rests on and the answer in a sentence or two, " +
		"writing {count}, {total} or {largest} where a figure goes, never typed. It is drawn " +
		"as your reply and ends your turn, so make any other call first. If a board cannot show this answer, " +
		"or the person only asked to see a list, or asked one fact about one record and the " +
		"others are named only to set them aside, reply again as you did." +
		hint + UnseenCorrection
}

func isBoardCorrection(content string) bool {
	return strings.HasPrefix(content, boardCorrectionLead) ||
		strings.HasPrefix(content, metricCorrectionLead) ||
		strings.HasPrefix(content, subjectCorrectionLead)
}

func boardable(t Turn) bool {
	if !t.Offers(showAnswerTool) || t.Request() == nil || !t.Request().AskedByPerson() {
		return false
	}

	return !slices.ContainsFunc(
		t.Artifacts(),
		func(shown agentruntimeports.ShownArtifact) bool {
			return shown.Kind == string(assistantartifact.KindBoard)
		},
	)
}

func boardRecords(content string) []agentpagecontext.EntityRef {
	return slices.DeleteFunc(
		toolresult.ReturnedRecords(content),
		func(ref agentpagecontext.EntityRef) bool {
			return !boardregistry.Catalog().Supports(recordkind.Kind(ref.Type))
		},
	)
}
