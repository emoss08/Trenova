package replycheck

import (
	"strings"
	"unicode"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/mdtable"
)

const (
	// minReprintRows is the fewest body rows a markdown table needs before
	// it is taken for a reprint of a kept table. Below it the reply is
	// answering with a few rows, which shownNote allows.
	minReprintRows = 6
	// minReprintOverlap is the share of the reprint's first column, or of
	// the kept table's title words, that has to match before one is taken
	// for the other.
	minReprintOverlap = 0.5
	minTitleWordRunes = 3

	ReplyPassReplacedReason = "The reply wrote out what is kept beside the conversation, so " +
		"it is shown as recorded."
	StripIDsReason = "The reply named internal record ids, which a person never reads; " +
		"they were taken out."
	PointToTableReason = "The reply reprinted a table kept beside the conversation; the " +
		"reprint was taken out and the reply's own sentence points to it."
)

// ReplyReplacedEvent puts the reply the turn recorded in place of the one
// that streamed, for a correction the model was not asked to make.
func ReplyReplacedEvent(text, reason string) kernelports.StreamEvent {
	return kernelports.StreamEvent{
		Event: assistantstreamports.AssistantEventReplyReplaced,
		Data:  assistantstreamports.AssistantReplyReplacedEvent{Text: text, Reason: reason},
	}
}

func PassEvent(
	action assistantstreamports.RegroundAction,
	reason string,
	table *agentruntimeports.ShownArtifact,
) kernelports.StreamEvent {
	data := assistantstreamports.AssistantReplyRegroundedEvent{Action: action, Reason: reason}
	if table != nil {
		data.ArtifactID = table.ID
	}

	return kernelports.StreamEvent{
		Event: assistantstreamports.AssistantEventReplyRegrounded,
		Data:  data,
	}
}

func Refold(t Turn, callID, stale, folded string) {
	if t.Result() == nil {
		return
	}
	for idx := range t.Result().Messages {
		kept := &t.Result().Messages[idx]
		if kept.ToolCallID == callID && strings.HasSuffix(kept.Content, stale) {
			kept.Content = strings.TrimSuffix(kept.Content, stale) + folded
		}
	}
}

func KeptTables(t Turn) []agentruntimeports.ShownArtifact {
	tables := make([]agentruntimeports.ShownArtifact, 0, len(t.Artifacts()))
	for idx := range t.Artifacts() {
		if t.Artifacts()[idx].Tabular() {
			tables = append(tables, t.Artifacts()[idx])
		}
	}

	return tables
}

// PointToTables takes out each markdown table in the reply that reprints a
// kept table the reply does not already point to, links the table from the
// reply's own sentence nearest the reprint, and reports the first table it
// pointed to. The server writes no words of its own: it cannot know the
// language the person and the model are using (DB-237). A reply that
// wrote out a list of late loads and then the same loads by customer used to
// keep the second copy, because only the first reprint was replaced; each
// kept table is now pointed to once, wherever its reprint stands.
func PointToTables(
	reply string,
	kept []agentruntimeports.ShownArtifact,
) (string, *agentruntimeports.ShownArtifact) {
	if len(kept) == 0 {
		return reply, nil
	}
	tables := mdtable.Find(reply)
	if len(tables) == 0 {
		return reply, nil
	}
	pointed := make(map[string]bool, len(kept))
	for _, id := range ArtifactRefIDs(reply) {
		pointed[id] = true
	}

	type replacement struct {
		table *mdtable.Table
		match *agentruntimeports.ShownArtifact
	}
	replacements := make([]replacement, 0, len(tables))
	for idx := range tables {
		table := &tables[idx]
		if len(table.Rows) < minReprintRows {
			continue
		}
		match := reprinted(table, kept, pointed)
		if match == nil {
			continue
		}
		pointed[match.ID.String()] = true
		replacements = append(replacements, replacement{table: table, match: match})
	}
	if len(replacements) == 0 {
		return reply, nil
	}

	gaps := make([]string, len(replacements)+1)
	last := 0
	for idx := range replacements {
		gaps[idx] = reply[last:replacements[idx].table.Start]
		last = replacements[idx].table.End
	}
	gaps[len(replacements)] = reply[last:]
	for idx := range replacements {
		refs := []string{ArtifactRef(replacements[idx].match)}
		if lines := proseLines(gaps[idx]); len(lines) > 0 {
			gaps[idx] = attachAt(gaps[idx], lines[len(lines)-1], refs)
			continue
		}
		if lines := proseLines(gaps[idx+1]); len(lines) > 0 {
			gaps[idx+1] = attachAt(gaps[idx+1], lines[0], refs)
			continue
		}
		gaps[idx] += "\n\n" + refs[0]
	}

	var b strings.Builder
	b.Grow(len(reply))
	for _, gap := range gaps {
		gap = strings.Trim(gap, "\n")
		if strings.TrimSpace(gap) == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(gap)
	}

	return b.String(), replacements[0].match
}

// reprinted is the kept table, not yet pointed to, that the markdown table
// writes out: the one whose first column the markdown's first column matches
// most, or whose title the line above the markdown and its first header name,
// at least half way. Only the first header counts: it says what each row is,
// while a later one is a figure about it, and a per-customer summary with a
// "Shipments" column is not a reprint of the shipments table. The latest kept table wins a tie, being the one the
// reply most likely describes.
func reprinted(
	table *mdtable.Table,
	kept []agentruntimeports.ShownArtifact,
	pointed map[string]bool,
) *agentruntimeports.ShownArtifact {
	firstColumn := make([]string, 0, len(table.Rows))
	for _, row := range table.Rows {
		if len(row) > 0 {
			firstColumn = append(firstColumn, mdtable.Plain(row[0]))
		}
	}
	named := table.Lead
	if len(table.Header) > 0 {
		named += " " + table.Header[0]
	}
	around := wordsOf(named)

	var best *agentruntimeports.ShownArtifact
	bestScore := 0.0
	for idx := len(kept) - 1; idx >= 0; idx-- {
		candidate := &kept[idx]
		if pointed[candidate.ID.String()] {
			continue
		}
		score := max(labelOverlap(firstColumn, candidate.Labels),
			titleOverlap(candidate.Title, around))
		if score >= minReprintOverlap && score > bestScore {
			best, bestScore = candidate, score
		}
	}

	return best
}

// labelOverlap is the share of the markdown's first-column values that are
// values of the kept table's first column.
func labelOverlap(firstColumn, labels []string) float64 {
	if len(firstColumn) == 0 || len(labels) == 0 {
		return 0
	}
	known := make(map[string]struct{}, len(labels))
	for _, label := range labels {
		known[mdtable.Plain(label)] = struct{}{}
	}
	matched := 0
	for _, value := range firstColumn {
		if _, ok := known[value]; ok && value != "" {
			matched++
		}
	}

	return float64(matched) / float64(len(firstColumn))
}

// titleOverlap is the share of the kept table's title words that the line
// above the markdown or its first header uses.
func titleOverlap(title string, around map[string]struct{}) float64 {
	words := wordsOf(title)
	if len(words) == 0 {
		return 0
	}
	matched := 0
	for word := range words {
		if _, ok := around[word]; ok {
			matched++
		}
	}

	return float64(matched) / float64(len(words))
}

// wordsOf is the text's words of three letters or more, in lower case and
// without a plural s, so "Shipments" in a title and "shipment" in a header
// are the same word.
func wordsOf(text string) map[string]struct{} {
	fields := strings.FieldsFunc(strings.ToLower(mdtable.Plain(text)), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	words := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		if len([]rune(field)) < minTitleWordRunes {
			continue
		}
		words[strings.TrimSuffix(field, "s")] = struct{}{}
	}

	return words
}
