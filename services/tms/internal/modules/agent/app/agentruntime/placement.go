package agentruntime

import (
	"strings"
	"unicode"

	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

const changeArtifactPlacement = "agent-loop-artifact-placement"

const (
	placementCorrectionLead = "Your reply does not name its artifacts inside sentences."
	placementRetryReason    = "The reply named an artifact outside a sentence, so it is " +
		"being written again."
	placementPlacedReason = "The rewritten reply still named an artifact outside a sentence, " +
		"so it was moved into one."
	detailsPointerLead = "The details are in "
	cutMarkup          = "*_"
	cutTrailers        = ".,;:!?"
)

type placementState struct {
	retried bool
}

type strayLink struct {
	start int
	end   int
	id    string
	words string
}

type placementProblems struct {
	stray    []strayLink
	missing  []agentports.ShownArtifact
	invented []strayLink
}

func (p *placementProblems) empty() bool {
	return len(p.stray) == 0 && len(p.missing) == 0 && len(p.invented) == 0
}

func (s *Service) placeArtifacts(
	t *Turn,
	fx TurnEffects,
	completion *agentports.ChatCompletionResult,
	state *placementState,
) bool {
	if completion == nil || t.req.Delegation != nil || strings.TrimSpace(completion.Text) == "" {
		return false
	}

	problems := t.placementProblems(completion.Text)
	if problems.empty() || !fx.Supports(changeArtifactPlacement) {
		return false
	}

	if !state.retried {
		state.retried = true
		fx.Emit(kernelports.StreamEvent{
			Event: agentports.AssistantEventRetrying,
			Data: agentports.AssistantRetryingEvent{
				Attempt: 1,
				Reason:  placementRetryReason,
				Kind:    agentports.RetryKindRestart,
			},
		})
		fx.Emit(passEvent(agentports.RegroundRelinkArtifacts, placementRetryReason, nil))
		t.messages = append(
			t.messages,
			agentports.Message{
				Role:          agentports.RoleAssistant,
				Content:       completion.Text,
				ReplyID:       completion.ProviderMessageID,
				ReplyProvider: completion.ProviderID,
			},
			agentports.Message{
				Role:    agentports.RoleUser,
				Content: t.placementCorrection(&problems),
			},
		)

		return true
	}

	placed := t.placed(completion.Text, &problems)
	fx.Emit(passEvent(agentports.RegroundPlaceArtifacts, placementPlacedReason, nil))
	fx.Emit(replyReplacedEvent(placed, placementPlacedReason))
	completion.Text = placed

	return false
}

func (t *Turn) placementProblems(reply string) placementProblems {
	var problems placementProblems
	linked := make(map[string]bool, 4)
	for _, link := range artifactLinkSpans(reply) {
		linked[link.id] = true
		switch {
		case !t.knowsArtifact(link.id):
			problems.invented = append(problems.invented, link)
		case !inSentence(reply, link):
			problems.stray = append(problems.stray, link)
		}
	}
	for idx := range t.artifacts {
		shown := &t.artifacts[idx]
		if shown.MustBeLinked() && !linked[shown.ID.String()] {
			problems.missing = append(problems.missing, *shown)
		}
	}

	return problems
}

func (t *Turn) knowsArtifact(id string) bool {
	if _, ok := t.shownArtifact(id); ok {
		return true
	}
	needle := ArtifactHrefScheme + id
	for idx := range t.messages {
		if t.discardedDraft(idx) {
			continue
		}
		if strings.Contains(t.messages[idx].Content, needle) {
			return true
		}
	}

	return strings.Contains(t.system, needle)
}

func (t *Turn) discardedDraft(idx int) bool {
	if isCorrection(&t.messages[idx]) {
		return true
	}

	return t.messages[idx].Role == agentports.RoleAssistant && idx+1 < len(t.messages) &&
		isCorrection(&t.messages[idx+1])
}

func isCorrection(message *agentports.Message) bool {
	return message.Role == agentports.RoleUser &&
		(strings.HasPrefix(message.Content, placementCorrectionLead) ||
			isGroundingCorrection(message.Content))
}

func (t *Turn) shownArtifact(id string) (*agentports.ShownArtifact, bool) {
	for idx := range t.artifacts {
		if t.artifacts[idx].ID.String() == id {
			return &t.artifacts[idx], true
		}
	}

	return nil, false
}

func (t *Turn) placementCorrection(problems *placementProblems) string {
	var b strings.Builder
	b.WriteString(placementCorrectionLead)
	for _, link := range problems.stray {
		b.WriteString(" ")
		b.WriteString(t.refFor(link))
		b.WriteString(" stands outside a sentence.")
	}
	for idx := range problems.missing {
		b.WriteString(" ")
		b.WriteString(ArtifactRef(&problems.missing[idx]))
		b.WriteString(" is not named, and the person needs it to read your answer.")
	}
	for _, link := range problems.invented {
		b.WriteString(" artifact:")
		b.WriteString(link.id)
		b.WriteString(" is not an artifact this conversation holds.")
	}
	b.WriteString(" Write the reply again with each artifact named once, as written above, " +
		"inside the sentence that says what it shows. Keep everything else you said.")

	return b.String()
}

func (t *Turn) refFor(link strayLink) string {
	if shown, ok := t.shownArtifact(link.id); ok {
		return ArtifactRef(shown)
	}

	return artifactRefOf(link.id, link.words)
}

func (t *Turn) placed(reply string, problems *placementProblems) string {
	cuts := make(map[int]strayLink, len(problems.stray)+len(problems.invented))
	for _, link := range problems.stray {
		cuts[link.start] = widenCut(reply, link)
	}
	for _, link := range problems.invented {
		cuts[link.start] = link
	}

	invented := make(map[string]bool, len(problems.invented))
	for _, link := range problems.invented {
		invented[link.id] = true
	}

	var b strings.Builder
	b.Grow(len(reply) + 128)
	lineStart := 0
	for lineStart <= len(reply) {
		lineEnd := strings.IndexByte(reply[lineStart:], '\n')
		if lineEnd < 0 {
			lineEnd = len(reply)
		} else {
			lineEnd += lineStart
		}
		line, cut := cutLinks(reply, lineStart, lineEnd, cuts, invented)
		if !cut || hasWord(line) {
			b.WriteString(strings.TrimRight(line, " \t"))
			if lineEnd < len(reply) {
				b.WriteByte('\n')
			}
		}
		lineStart = lineEnd + 1
	}

	refs := make([]string, 0, len(problems.stray)+len(problems.missing))
	kinds := make([]string, 0, cap(refs))
	seen := make(map[string]bool, cap(refs))
	for _, link := range problems.stray {
		if seen[link.id] {
			continue
		}
		seen[link.id] = true
		refs = append(refs, t.refFor(link))
		kinds = append(kinds, t.kindOf(link.id))
	}
	for idx := range problems.missing {
		shown := &problems.missing[idx]
		if seen[shown.ID.String()] {
			continue
		}
		seen[shown.ID.String()] = true
		refs = append(refs, ArtifactRef(shown))
		kinds = append(kinds, shown.Kind)
	}

	out := strings.TrimRight(collapseBlankLines(b.String()), " \t\n")
	if len(refs) == 0 {
		return out
	}
	sentence := pointerLead(kinds) + joinRefs(refs) + "."
	if out == "" {
		return sentence
	}

	return out + "\n\n" + sentence
}

func cutLinks(
	reply string,
	start, end int,
	cuts map[int]strayLink,
	invented map[string]bool,
) (string, bool) {
	var b strings.Builder
	cut := false
	last := start
	for _, link := range artifactLinkSpansIn(reply, start, end) {
		chosen, ok := cuts[link.start]
		if !ok {
			continue
		}
		b.WriteString(reply[last:max(chosen.start, last)])
		if invented[chosen.id] {
			b.WriteString(chosen.words)
		} else {
			cut = true
		}
		last = chosen.end
	}
	if last == start {
		return reply[start:end], false
	}
	b.WriteString(reply[last:end])

	return strings.ReplaceAll(b.String(), "  ", " "), cut
}

func widenCut(reply string, link strayLink) strayLink {
	for link.start > 0 && strings.IndexByte(cutMarkup, reply[link.start-1]) >= 0 {
		link.start--
	}
	for link.end < len(reply) && strings.IndexByte(cutMarkup, reply[link.end]) >= 0 {
		link.end++
	}
	if link.end < len(reply) && strings.IndexByte(cutTrailers, reply[link.end]) >= 0 {
		link.end++
	}

	return link
}

func (t *Turn) kindOf(id string) string {
	if shown, ok := t.shownArtifact(id); ok {
		return shown.Kind
	}

	return ""
}

var pointerLeads = map[assistantartifact.Kind]string{
	assistantartifact.KindTableView:       tablePointerLead,
	assistantartifact.KindReportPreview:   tablePointerLead,
	assistantartifact.KindEntityCard:      "The full record is in ",
	assistantartifact.KindDocument:        "The full write-up is in ",
	assistantartifact.KindReportRun:       "The report is in ",
	assistantartifact.KindRateExplanation: "How the rate was reached is in ",
}

func pointerLead(kinds []string) string {
	if len(kinds) != 1 {
		return detailsPointerLead
	}
	if lead, ok := pointerLeads[assistantartifact.Kind(kinds[0])]; ok {
		return lead
	}

	return detailsPointerLead
}

func joinRefs(refs []string) string {
	switch len(refs) {
	case 0:
		return ""
	case 1:
		return refs[0]
	default:
		return strings.Join(refs[:len(refs)-1], ", ") + " and " + refs[len(refs)-1]
	}
}

func collapseBlankLines(text string) string {
	for strings.Contains(text, "\n\n\n") {
		text = strings.ReplaceAll(text, "\n\n\n", "\n\n")
	}

	return strings.TrimLeft(text, "\n")
}

func StrayArtifactLinks(text string) []string {
	var ids []string
	for _, link := range artifactLinkSpans(text) {
		if !inSentence(text, link) {
			ids = append(ids, link.id)
		}
	}

	return ids
}

func artifactLinkSpans(text string) []strayLink {
	return artifactLinkSpansIn(text, 0, len(text))
}

func artifactLinkSpansIn(text string, start, end int) []strayLink {
	segment := text[start:end]
	if !strings.Contains(segment, ArtifactHrefScheme) {
		return nil
	}
	matches := artifactLink.FindAllStringSubmatchIndex(segment, -1)
	links := make([]strayLink, 0, len(matches))
	for _, match := range matches {
		links = append(links, strayLink{
			start: start + match[0],
			end:   start + match[1],
			words: segment[match[2]:match[3]],
			id:    segment[match[4]:match[5]],
		})
	}

	return links
}

func inSentence(text string, link strayLink) bool {
	lineStart := strings.LastIndexByte(text[:link.start], '\n') + 1
	lineEnd := strings.IndexByte(text[link.end:], '\n')
	if lineEnd < 0 {
		lineEnd = len(text)
	} else {
		lineEnd += link.end
	}
	line := text[lineStart:lineEnd]
	if insideFence(text[:lineStart]) || !proseLine(line) {
		return false
	}

	from := sentenceStart(text[lineStart:link.start]) + lineStart
	to := link.end + sentenceEnd(text[link.end:lineEnd])

	return hasWord(artifactLink.ReplaceAllString(text[from:to], ""))
}

func insideFence(before string) bool {
	open := false
	for line := range strings.SplitSeq(before, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			open = !open
		}
	}

	return open
}

func proseLine(line string) bool {
	trimmed := strings.TrimSpace(line)

	return !strings.HasPrefix(trimmed, "|") && !strings.HasPrefix(trimmed, "#")
}

func sentenceStart(before string) int {
	for idx := len(before) - 1; idx > 0; idx-- {
		if isSpace(before[idx]) && endsSentence(before[idx-1]) {
			return idx
		}
	}

	return 0
}

func sentenceEnd(after string) int {
	for idx := 0; idx < len(after); idx++ {
		if !endsSentence(after[idx]) {
			continue
		}
		if idx+1 == len(after) || isSpace(after[idx+1]) {
			return idx + 1
		}
	}

	return len(after)
}

func endsSentence(c byte) bool {
	return c == '.' || c == '!' || c == '?'
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t'
}

func hasWord(text string) bool {
	run := 0
	for _, r := range text {
		if unicode.IsLetter(r) {
			run++
			if run >= 2 {
				return true
			}
			continue
		}
		run = 0
	}

	return false
}
