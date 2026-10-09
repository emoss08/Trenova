package agentruntime

import (
	"strings"
	"unicode"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/mdtable"
	"github.com/emoss08/trenova/shared/recordids"
)

// changeReplyPasses is the loop passing over a final reply before it is
// recorded: internal ids taken out, a reprinted table pointed to.
const changeReplyPasses = "agent-loop-reply-passes"

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

	tablePointerLead = "The full table is in "

	replyPassReplacedReason = "The reply wrote out what is kept beside the conversation, so " +
		"it is shown as recorded."
	stripIDsReason = "The reply named internal record ids, which a person never reads; " +
		"they were taken out."
	pointToTableReason = "The reply reprinted a table kept beside the conversation; the " +
		"reprint was replaced by a sentence pointing to it."
)

// passReply holds a final reply to two rules the prompt already gives, in
// code beneath them: a person never reads an internal record id, and a table
// kept beside the conversation is pointed to, not written out again. Models
// break both, small ones most, and the prompt alone left a reply full of
// shp_01J… and a twenty-five row markdown copy of the table next to it.
//
// The reply has already streamed. When the pass changes it, a reply_replaced
// event puts the corrected reply in place of the streamed one, so the person
// reads what is recorded without the turn appearing to fail and start over.
// Each pass that changed something is a reply_regrounded event in the
// trajectory.
func (s *Service) passReply(
	t *Turn,
	fx TurnEffects,
	completion *serviceports.ChatCompletionResult,
) {
	if completion == nil || strings.TrimSpace(completion.Text) == "" {
		return
	}

	stripped := recordids.Strip(completion.Text)
	pointed, table := pointToTables(stripped, t.tables)
	if pointed == completion.Text || !fx.Supports(changeReplyPasses) {
		return
	}

	if stripped != completion.Text {
		fx.Emit(passEvent(serviceports.RegroundStripIDs, stripIDsReason, nil))
	}
	if table != nil {
		fx.Emit(passEvent(serviceports.RegroundPointToTable, pointToTableReason, table))
	}
	fx.Emit(replyReplacedEvent(pointed, replyPassReplacedReason))
	completion.Text = pointed
}

// replyReplacedEvent puts the reply the turn recorded in place of the one
// that streamed, for a correction the model was not asked to make.
func replyReplacedEvent(text, reason string) serviceports.StreamEvent {
	return serviceports.StreamEvent{
		Event: serviceports.AssistantEventReplyReplaced,
		Data:  serviceports.AssistantReplyReplacedEvent{Text: text, Reason: reason},
	}
}

func passEvent(
	action serviceports.RegroundAction,
	reason string,
	table *serviceports.ShownArtifact,
) serviceports.StreamEvent {
	data := serviceports.AssistantReplyRegroundedEvent{Action: action, Reason: reason}
	if table != nil {
		data.ArtifactID = table.ID
	}

	return serviceports.StreamEvent{Event: serviceports.AssistantEventReplyRegrounded, Data: data}
}

// keepTable remembers a table the call put beside the conversation, so the
// reply pass can point a reprint of it to it.
func (t *Turn) keepTable(outcome *toolOutcome) {
	if outcome.failed || outcome.shown == nil || !tabular(outcome.shown.Kind) {
		return
	}
	for idx := range t.tables {
		if t.tables[idx].ID == outcome.shown.ID {
			t.tables[idx] = *outcome.shown

			return
		}
	}
	t.tables = append(t.tables, *outcome.shown)
}

// pointToTables replaces each markdown table in the reply that reprints a
// kept table the reply does not already point to with one sentence that
// points to it, and reports the first table it pointed to. A reply that
// wrote out a list of late loads and then the same loads by customer used to
// keep the second copy, because only the first reprint was replaced; each
// kept table is now pointed to once, wherever its reprint stands.
func pointToTables(
	reply string,
	kept []serviceports.ShownArtifact,
) (string, *serviceports.ShownArtifact) {
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
		match *serviceports.ShownArtifact
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

	out := reply
	for idx := len(replacements) - 1; idx >= 0; idx-- {
		table, match := replacements[idx].table, replacements[idx].match
		sentence := tablePointerLead + ArtifactRef(match) + "."
		if table.End < len(out) {
			sentence += "\n"
		}
		out = out[:table.Start] + sentence + out[table.End:]
	}

	return out, replacements[0].match
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
	kept []serviceports.ShownArtifact,
	pointed map[string]bool,
) *serviceports.ShownArtifact {
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

	var best *serviceports.ShownArtifact
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
