package replycheck

import (
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
)

const (
	GuardBoard = "board"
	GuardDraft = "draft"

	draftCorrectionLead = "Your reply writes out a message for the person to send"
	DraftRetryReason    = "The reply wrote out a message to send instead of drafting it as a " +
		"document, so it is being drafted again."

	quotedOpeningRunes = 40
)

var (
	salutation = regexp.MustCompile(`(?m)^[ \t]*(?:>[ \t]*)*(?:\*\*|__)?` +
		`((?:Hi|Hello|Hey|Dear|Good (?:morning|afternoon|evening))\b[^\n]{0,60}?,[^\n]*)`)
	subjectLine = regexp.MustCompile(`(?mi)^[ \t]*(?:>[ \t]*)*(?:\*\*|__)?(subject:[^\n]*)`)
	signOff     = regexp.MustCompile(`(?mi)^[ \t]*(?:>[ \t]*)*(?:\*\*|__)?(?:thanks|thank you|` +
		`many thanks|regards|best regards|kind regards|best|sincerely|cheers)\b[^\n]{0,40}$`)
	unquoted = strings.NewReplacer(">", " ", "*", " ", "_", " ", "\"", " ", "“", " ", "”", " ")
)

func AskAgain(
	t Turn,
	draft *llmports.ChatCompletionResult,
	state *BoardAskState,
) (SendBack, bool) {
	if back, ask := AskForDraft(t, draft, state); ask {
		return back, true
	}

	return AskForBoard(t, draft, state)
}

func AskForDraft(
	t Turn,
	draft *llmports.ChatCompletionResult,
	state *BoardAskState,
) (SendBack, bool) {
	if state.Drafted || draft == nil || !t.Offers(publishArtifactTool) || t.Request() == nil ||
		!t.Request().AskedByPerson() || published(t) {
		return SendBack{}, false
	}
	opening := draftedMessage(t, draft.Text)
	if opening == "" {
		return SendBack{}, false
	}
	state.Drafted = true

	back := SendBackDraft(
		draft,
		DraftRetryReason,
		PassEvent(assistantstreamports.RegroundPublishDraft, DraftRetryReason, nil),
		draftCorrection(opening, boardable(t) && readsBoardRecords(t) &&
			strings.Contains(t.Request().Input, "?")),
	)
	back.Guard = GuardDraft

	return back, true
}

func published(t Turn) bool {
	return slices.ContainsFunc(t.Artifacts(), func(shown agentruntimeports.ShownArtifact) bool {
		return shown.Kind == string(assistantartifact.KindDocument)
	})
}

func draftedMessage(t Turn, reply string) string {
	openings := make([]string, 0, 2)
	for _, match := range subjectLine.FindAllStringSubmatch(reply, -1) {
		openings = append(openings, strings.TrimSpace(unquoted.Replace(match[1])))
	}
	for _, at := range salutation.FindAllStringSubmatchIndex(reply, -1) {
		quoted := strings.Contains(reply[at[0]:at[2]], ">")
		if !quoted && !signOff.MatchString(reply[at[1]:]) {
			continue
		}
		openings = append(openings, strings.TrimSpace(unquoted.Replace(reply[at[2]:at[3]])))
	}
	if len(openings) == 0 {
		return ""
	}
	heard := heardTexts(t)
	for _, opening := range openings {
		if !slices.ContainsFunc(heard, func(text string) bool {
			return strings.Contains(text, start(flat(opening)))
		}) {
			return opening
		}
	}

	return ""
}

func heardTexts(t Turn) []string {
	heard := make([]string, 0, len(t.Messages())+1)
	if t.Request() != nil {
		heard = append(heard, flat(t.Request().Input))
	}
	for idx := range t.Messages() {
		message := &t.Messages()[idx]
		if message.Role == llmports.RoleTool || message.Role == llmports.RoleUser {
			heard = append(heard, flat(message.Content))
		}
	}

	return heard
}

func flat(text string) string {
	return strings.Join(strings.Fields(strings.ToLower(unquoted.Replace(text))), " ")
}

func start(text string) string {
	if utf8.RuneCountInString(text) <= quotedOpeningRunes {
		return text
	}

	return string([]rune(text)[:quotedOpeningRunes])
}

func readsBoardRecords(t Turn) bool {
	seen := make(map[string]bool, 4)
	for idx := range t.Messages() {
		message := &t.Messages()[idx]
		if message.Role != llmports.RoleTool || message.IsError {
			continue
		}
		for _, ref := range boardRecords(message.Content) {
			seen[ref.ID] = true
		}
	}

	return len(seen) >= minBoardRecords
}

func draftCorrection(opening string, records bool) string {
	question := ""
	if records {
		question = " When the person also asked about the records themselves (which, whose, how " +
			"many, show me), answer that once the draft's result is back, with show_answer, " +
			"linking the draft in its words as [its title](artifact:<id>) with the id that " +
			"result gave; the board is your reply."
	}

	return draftCorrectionLead + ` (it opens "` + opening + `"). A message the person will ` +
		"send is a write-up, never written out in your reply: call publish_artifact with the " +
		"message as its body (docType \"Draft\", a title saying who it is for; one document " +
		"holds every message when there are several), then reply in two or three sentences " +
		"that point to it. Nothing is sent: the person sends it themselves. If you were " +
		"quoting a message someone sent, reply again as you did." + question + UnseenCorrection
}
