package replycheck

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
)

const (
	PlacementCorrectionLead = "Your reply does not name its artifacts inside sentences."
	PlacementRetryReason    = "The reply named an artifact outside a sentence, so it is " +
		"being written again."
	PlacementPlacedReason = "The rewritten reply still named an artifact outside a sentence, " +
		"so it was moved into one."
	PlacementNamedReason = "The reply left out an artifact the person needs, so it was " +
		"named in the reply's own words."
	PlacementMovedReason = "The reply named an artifact outside a sentence, so it was " +
		"moved into the reply's own sentence beside it."
	cutMarkup          = "*_"
	cutTrailers        = ".,;:!?"
	closingPunctuation = ".!?;,…。！？；，"
	introducingColons  = ":："
	refSeparator       = ", "
)

type PlacementState struct {
	Retried bool
}

type strayLink struct {
	start int
	end   int
	id    string
	words string
	// apart is a stray link on a prose line outside a fence: the sentence
	// round it holds no word, so cutting it leaves every sentence whole.
	apart bool
}

type PlacementProblems struct {
	Stray    []strayLink
	missing  []agentruntimeports.ShownArtifact
	invented []strayLink
	// named is the artifacts the reply already names inside a sentence, which
	// a stray link of the same artifact is cut from without naming it twice.
	named map[string]bool
}

func (p *PlacementProblems) Empty() bool {
	return len(p.Stray) == 0 && len(p.missing) == 0 && len(p.invented) == 0
}

// Mendable reports whether every fault of the reply is one the server repairs
// as well as a rewrite does: an artifact it never named, and a link standing
// apart from any sentence, on its own line, after a list or after a sentence
// that has ended. An id nobody holds, and a link in a table, a heading or a
// fenced block, sits where cutting it could break what is round it, so the
// model is asked again.
func (p *PlacementProblems) Mendable() bool {
	if len(p.invented) > 0 {
		return false
	}

	return !slices.ContainsFunc(p.Stray, func(link strayLink) bool { return !link.apart })
}

func FindPlacementProblems(t Turn, reply string) PlacementProblems {
	problems := PlacementProblems{named: make(map[string]bool, 4)}
	linked := make(map[string]bool, 4)
	for _, link := range artifactLinkSpans(reply) {
		linked[link.id] = true
		switch {
		case !knowsArtifact(t, link.id):
			problems.invented = append(problems.invented, link)
		case !inSentence(reply, link):
			_, _, link.apart = linkLine(reply, link)
			problems.Stray = append(problems.Stray, link)
		default:
			problems.named[link.id] = true
		}
	}
	for idx := range t.Artifacts() {
		shown := &t.Artifacts()[idx]
		if shown.MustBeLinked() && !linked[shown.ID.String()] {
			problems.missing = append(problems.missing, *shown)
		}
	}

	return problems
}

func knowsArtifact(t Turn, id string) bool {
	if _, ok := shownArtifact(t, id); ok {
		return true
	}
	needle := ArtifactHrefScheme + id
	for idx := range t.Messages() {
		if discardedDraft(t, idx) {
			continue
		}
		if strings.Contains(t.Messages()[idx].Content, needle) {
			return true
		}
	}

	return strings.Contains(t.System(), needle)
}

func discardedDraft(t Turn, idx int) bool {
	if isCorrection(&t.Messages()[idx]) {
		return true
	}

	return t.Messages()[idx].Role == llmports.RoleAssistant && idx+1 < len(t.Messages()) &&
		isCorrection(&t.Messages()[idx+1])
}

func isCorrection(message *llmports.Message) bool {
	return message.Role == llmports.RoleSystem &&
		(strings.HasPrefix(message.Content, PlacementCorrectionLead) ||
			isGroundingCorrection(message.Content))
}

func shownArtifact(t Turn, id string) (*agentruntimeports.ShownArtifact, bool) {
	for idx := range t.Artifacts() {
		if t.Artifacts()[idx].ID.String() == id {
			return &t.Artifacts()[idx], true
		}
	}

	return nil, false
}

func PlacementCorrection(t Turn, problems *PlacementProblems) string {
	var b strings.Builder
	b.WriteString(PlacementCorrectionLead)
	for _, link := range problems.Stray {
		b.WriteString(" ")
		b.WriteString(refFor(t, link))
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

func refFor(t Turn, link strayLink) string {
	if shown, ok := shownArtifact(t, link.id); ok {
		return ArtifactRef(shown)
	}

	return artifactRefOf(link.id, link.words)
}

// PlaceLinks is the reply with each stray link cut from where it stood and joined
// to the reply's own prose line nearest above that place (or, with none above,
// the first below), each left-out artifact joined to the reply's first line
// that ends a sentence, and an id nobody holds left as its words. ok is false
// when a stray link found no prose line to join, and it then stands on a line
// of its own at the end.
func PlaceLinks(t Turn, reply string, problems *PlacementProblems) (string, bool) {
	cuts := make(map[int]strayLink, len(problems.Stray)+len(problems.invented))
	for _, link := range problems.Stray {
		cuts[link.start] = widenCut(reply, link)
	}
	for _, link := range problems.invented {
		cuts[link.start] = link
	}

	invented := make(map[string]bool, len(problems.invented))
	for _, link := range problems.invented {
		invented[link.id] = true
	}

	seen := make(map[string]bool, len(problems.named)+len(problems.Stray)+len(problems.missing))
	for id := range problems.named {
		seen[id] = true
	}
	lines := make([]string, 0, strings.Count(reply, "\n")+1)
	anchors := make([]refAnchor, 0, len(problems.Stray))
	for lineStart := 0; lineStart <= len(reply); {
		lineEnd := strings.IndexByte(reply[lineStart:], '\n')
		if lineEnd < 0 {
			lineEnd = len(reply)
		} else {
			lineEnd += lineStart
		}
		line, cut := cutLinks(reply, lineStart, lineEnd, cuts, invented)
		kept := !cut || hasWord(line)
		for _, link := range problems.Stray {
			if link.start < lineStart || link.start >= lineEnd || seen[link.id] {
				continue
			}
			seen[link.id] = true
			anchors = append(anchors, refAnchor{at: len(lines), kept: kept, ref: refFor(t, link)})
		}
		if kept {
			line = indentedAs(reply[lineStart:lineEnd], line)
			lines = append(lines, strings.TrimRight(line, " \t"))
		}
		lineStart = lineEnd + 1
	}

	refs := make([]string, 0, len(problems.missing))
	for idx := range problems.missing {
		shown := &problems.missing[idx]
		if seen[shown.ID.String()] {
			continue
		}
		seen[shown.ID.String()] = true
		refs = append(refs, ArtifactRef(shown))
	}

	text, ok := linkRefs(lines, anchors, refs)

	return strings.TrimRight(collapseBlankLines(text), " \t\n"), ok
}

// indentedAs is a line a link was cut from, indented as the line it was: a
// link that led the line leaves the space after it at the line's start.
func indentedAs(original, line string) string {
	indent := original[:len(original)-len(strings.TrimLeft(original, " \t"))]

	return indent + strings.TrimLeft(line, " \t")
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

// refAnchor is where a stray link stood among the lines left once it was cut:
// on line at when that line kept words of its own, else just before it.
type refAnchor struct {
	at   int
	kept bool
	ref  string
}

// linkRefs joins each anchored ref to the prose line nearest above where it
// stood, or the first below, and refs to the first line that ends a sentence,
// each line's refs in one parenthesis. What finds no prose line goes on a line
// of its own at the end; ok is false when an anchored ref is among it.
func linkRefs(lines []string, anchors []refAnchor, refs []string) (string, bool) {
	prose := proseFlags(lines)
	attach := make(map[int][]string, len(anchors)+1)
	loose := make([]string, 0, len(anchors)+len(refs))
	for _, anchor := range anchors {
		if at, found := nearestProse(prose, anchor); found {
			attach[at] = append(attach[at], anchor.ref)
			continue
		}
		loose = append(loose, anchor.ref)
	}
	ok := len(loose) == 0
	if len(refs) > 0 {
		if at, found := firstSentence(lines, prose); found {
			attach[at] = append(attach[at], refs...)
		} else {
			loose = append(loose, refs...)
		}
	}

	joined := slices.Clone(lines)
	for at, named := range attach {
		joined[at] = attachRefs(joined[at], named)
	}
	text := strings.Join(joined, "\n")
	if len(loose) == 0 {
		return text, ok
	}
	trimmed := strings.TrimRight(text, " \t\n")
	if trimmed == "" {
		return strings.Join(loose, refSeparator), ok
	}

	return trimmed + "\n\n" + strings.Join(loose, refSeparator), ok
}

func nearestProse(prose []bool, anchor refAnchor) (int, bool) {
	from := anchor.at - 1
	if anchor.kept {
		from = anchor.at
	}
	for idx := min(from, len(prose)-1); idx >= 0; idx-- {
		if prose[idx] {
			return idx, true
		}
	}
	for idx := from + 1; idx < len(prose); idx++ {
		if prose[idx] {
			return idx, true
		}
	}

	return 0, false
}

func firstSentence(lines []string, prose []bool) (int, bool) {
	first := -1
	for idx, line := range lines {
		if !prose[idx] {
			continue
		}
		if first < 0 {
			first = idx
		}
		bare := strings.TrimRight(line, " \t"+cutMarkup)
		last, _ := utf8.DecodeLastRuneInString(bare)
		if strings.ContainsRune(closingPunctuation+introducingColons, last) {
			return idx, true
		}
	}

	return first, first >= 0
}

type textLine struct {
	start int
	end   int
}

func proseLines(text string) []textLine {
	lines := make([]textLine, 0, 4)
	fenced := false
	for start := 0; start <= len(text); {
		end := strings.IndexByte(text[start:], '\n')
		if end < 0 {
			end = len(text)
		} else {
			end += start
		}
		if proseStep(text[start:end], &fenced) {
			lines = append(lines, textLine{start: start, end: end})
		}
		start = end + 1
	}

	return lines
}

// proseFlags reports, line by line, which lines are prose a link can join.
func proseFlags(lines []string) []bool {
	flags := make([]bool, len(lines))
	fenced := false
	for idx, line := range lines {
		flags[idx] = proseStep(line, &fenced)
	}

	return flags
}

// proseStep reports whether the line is prose with words of its own, outside
// a fence, a table and a heading, and keeps track of the fences it passes.
func proseStep(line string, fenced *bool) bool {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
		*fenced = !*fenced

		return false
	}

	return !*fenced && proseLine(line) && hasWord(artifactLink.ReplaceAllString(line, ""))
}

func attachAt(text string, line textLine, refs []string) string {
	return text[:line.start] + attachRefs(text[line.start:line.end], refs) + text[line.end:]
}

func attachRefs(line string, refs []string) string {
	joined := strings.Join(refs, refSeparator)
	body := strings.TrimRight(line, " \t")
	bare := strings.TrimRight(body, cutMarkup)
	if last, _ := utf8.DecodeLastRuneInString(bare); strings.ContainsRune(introducingColons, last) {
		return body + " " + joined
	}
	cut := len(bare)
	for cut > 0 {
		last, size := utf8.DecodeLastRuneInString(bare[:cut])
		if !strings.ContainsRune(closingPunctuation, last) {
			break
		}
		cut -= size
	}
	if cut == len(bare) {
		return body + " (" + joined + ")"
	}

	return body[:cut] + " (" + joined + ")" + body[cut:]
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

// linkLine is the line the link stands on, and whether it is a prose line
// outside a fence.
func linkLine(text string, link strayLink) (int, int, bool) {
	lineStart := strings.LastIndexByte(text[:link.start], '\n') + 1
	lineEnd := strings.IndexByte(text[link.end:], '\n')
	if lineEnd < 0 {
		lineEnd = len(text)
	} else {
		lineEnd += link.end
	}

	return lineStart, lineEnd, !insideFence(text[:lineStart]) && proseLine(text[lineStart:lineEnd])
}

func inSentence(text string, link strayLink) bool {
	lineStart, lineEnd, prose := linkLine(text, link)
	if !prose {
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
