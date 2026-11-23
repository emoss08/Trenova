package replycheck

import (
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolresult"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
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
	Asked bool
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
	count := len(boardRecords(content))
	if labels, rows := tableRecords(shown); max(rows, len(labels)) > count {
		count = max(rows, len(labels))
	}
	if count < minBoardRecords {
		return content
	}

	return content + boardNoteLead + strconv.Itoa(count) + boardNoteTail
}

func ReplaceShownNote(content, stale, folded string) (string, bool) {
	body, note := splitBoardNote(content)
	if !strings.HasSuffix(body, stale) {
		return content, false
	}

	return strings.TrimSuffix(body, stale) + folded + note, true
}

func splitBoardNote(content string) (body, note string) {
	if !strings.HasSuffix(content, boardNoteTail) {
		return content, ""
	}
	start := strings.LastIndex(content, boardNoteLead)
	if start < 0 {
		return content, ""
	}
	count := content[start+len(boardNoteLead) : len(content)-len(boardNoteTail)]
	if _, err := strconv.Atoi(count); err != nil {
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
	if _, table := PointToTables(draft.Text, KeptTables(t)); table != nil {
		return SendBack{}, false
	}
	if len(ArtifactRefIDs(draft.Text)) > 0 && !writesOutTable(t, draft.Text) &&
		!namesLookedUp(t, draft.Text) {
		return SendBack{}, false
	}
	named := namedBoardRecords(t, draft.Text)
	if len(named) < minBoardRecords {
		return SendBack{}, false
	}

	state.Asked = true

	return SendBackDraft(
		draft,
		BoardRetryReason,
		PassEvent(assistantstreamports.RegroundAnswerWithBoard, BoardRetryReason, nil),
		boardCorrection(named),
	), true
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

func tableRecords(shown *agentruntimeports.ShownArtifact) ([]string, int) {
	if shown == nil || !shown.Tabular() || !board.Supports(permission.Resource(shown.RowKind)) {
		return nil, 0
	}

	return shown.Labels, shown.Rows
}

func boardCorrection(named []string) string {
	return boardCorrectionLead + " it names " + strings.Join(named, ", ") + ". An answer " +
		"that weighs records together, or gives a count, a total or a comparison across them, " +
		"belongs on a board in the conversation, not in prose and not as a pointer to a table: " +
		"call show_answer with the records it rests on and the answer in a sentence or two, " +
		"writing {count}, {total} or {largest} where a figure goes, never typed. It is drawn " +
		"as your reply and ends your turn, so make any other call first. If a board cannot show this answer, " +
		"or the person only asked to see a list or asked for one fact, reply again as you did." +
		UnseenCorrection
}

func isBoardCorrection(content string) bool {
	return strings.HasPrefix(content, boardCorrectionLead)
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
			return !board.Supports(permission.Resource(ref.Type))
		},
	)
}
