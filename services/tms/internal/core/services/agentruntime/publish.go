package agentruntime

import (
	"fmt"
	"strings"
	"unicode/utf8"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	publishArtifactName = "publish_artifact"

	maxDocumentTitleRunes = 120
	maxDocumentBodyBytes  = 60000
	maxDocumentTypeRunes  = 40
	maxDocumentBasisRunes = 120
	maxDocumentSources    = 30
	maxSourceTextRunes    = 160
)

// publishArtifactDescription is a constant for the same reason
// askUserDescription is: it is addressed to a model, and the i18n extractor
// would otherwise harvest it.
const publishArtifactDescription = "Publish a written document beside the conversation, " +
	"where the person can read, copy and keep it. Use it when they ask for a write-up, a " +
	"summary, a brief, a handover or an artifact, and whenever your answer would run past " +
	"a screen: put the full text here in markdown and reply with two or three sentences " +
	"pointing to it. Not for a list or a single record you looked up (those are shown to " +
	"the person already, and the tool result says so). To revise a document you published, " +
	"pass its artifactId with the whole new text; the earlier text is kept as a version. " +
	"Cite what a sentence rests on with a footnote mark like [^1] and list each mark " +
	"under sources."

// publishArtifactSpec is the third tool the runtime answers itself. It is
// offered only where there is somewhere to publish to: a conversation with a
// pane beside it.
func publishArtifactSpec() serviceports.ToolSpec {
	return serviceports.ToolSpec{
		Name:        publishArtifactName,
		Description: publishArtifactDescription,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"title": map[string]any{
					"type": "string",
					"description": "What the document is, in a few words. " +
						"\"Shipment SEED-SHP-007 brief\".",
				},
				"body": map[string]any{
					"type": "string",
					"description": "The whole document in markdown: headings, lists and " +
						"tables. Only facts your tools returned.",
				},
				"docType": map[string]any{
					"type": "string",
					"description": "What kind of write-up it is, in a word or two: " +
						"\"Brief\", \"Summary\", \"Handover\".",
				},
				"basis": map[string]any{
					"type": "string",
					"description": "What it was written from, in a few words: " +
						"\"from 42 loads and 3 weather alerts\".",
				},
				"sources": map[string]any{
					"type": "array",
					"description": "What the body's [^N] marks point to, one entry per " +
						"number. Only tools you called in this conversation.",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"n":     map[string]any{"type": "integer"},
							"tool":  map[string]any{"type": "string", "description": "The tool that found it."},
							"label": map[string]any{"type": "string", "description": "What it is, in a few words."},
							"detail": map[string]any{
								"type":        "string",
								"description": "One more line: a time, a scope.",
							},
							"artifactId": map[string]any{
								"type": "string",
								"description": "The artifact that shows it, when its result " +
									"named one.",
							},
						},
						"required":             []string{"n", "tool", "label"},
						"additionalProperties": false,
					},
				},
				"artifactId": map[string]any{
					"type": "string",
					"description": "The id of a document you published earlier in this " +
						"conversation, from its publish_artifact result, to replace its text. " +
						"Leave it out to publish a new document.",
				},
			},
			"required":             []string{"title", "body"},
			"additionalProperties": false,
		},
	}
}

// publishOutcome checks a publish call and hands the document to the observer
// that keeps it. The result the model reads is written once the observer has
// said what it kept.
func publishOutcome(arguments map[string]any) toolOutcome {
	title := strings.TrimSpace(stringArg(arguments, "title"))
	body := strings.TrimSpace(stringArg(arguments, "body"))

	var problems []string
	switch {
	case title == "":
		problems = append(problems, "title is required")
	case utf8.RuneCountInString(title) > maxDocumentTitleRunes:
		problems = append(problems,
			fmt.Sprintf("title is longer than %d characters", maxDocumentTitleRunes))
	}
	switch {
	case body == "":
		problems = append(problems, "body is required")
	case len(body) > maxDocumentBodyBytes:
		problems = append(problems, fmt.Sprintf(
			"body is longer than %d characters; publish the part that matters",
			maxDocumentBodyBytes,
		))
	}

	var revises pulid.ID
	if raw := strings.TrimSpace(stringArg(arguments, "artifactId")); raw != "" {
		parsed, err := pulid.Parse(raw)
		if err != nil {
			problems = append(
				problems,
				"artifactId is not an id a publish_artifact result gave you",
			)
		} else {
			revises = parsed
		}
	}

	sources, sourceProblems := publishedSources(arguments["sources"])
	problems = append(problems, sourceProblems...)

	if len(problems) > 0 {
		return failedOutcome(
			"Tool %q was not run: %s.",
			publishArtifactName,
			strings.Join(problems, "; "),
		)
	}

	return toolOutcome{
		publishes: true,
		data: serviceports.PublishedDocument{
			Title:      title,
			Body:       body,
			ArtifactID: revises,
			DocType:    clipRunes(stringArg(arguments, "docType"), maxDocumentTypeRunes),
			Basis:      clipRunes(stringArg(arguments, "basis"), maxDocumentBasisRunes),
			Sources:    sources,
		},
	}
}

// publishedSources reads the sources a document cites. A source without a
// number or a label cannot be shown, and is a problem the model can fix.
func publishedSources(raw any) ([]serviceports.PublishedSource, []string) {
	entries, ok := raw.([]any)
	if !ok || len(entries) == 0 {
		return nil, nil
	}
	if len(entries) > maxDocumentSources {
		return nil, []string{fmt.Sprintf("at most %d sources", maxDocumentSources)}
	}

	sources := make([]serviceports.PublishedSource, 0, len(entries))
	seen := map[int]bool{}
	for _, entry := range entries {
		fields, isObject := entry.(map[string]any)
		if !isObject {
			return nil, []string{"each source is an object with n, tool and label"}
		}
		n, isNumber := fields["n"].(float64)
		if !isNumber {
			if whole, isInt := fields["n"].(int); isInt {
				n, isNumber = float64(whole), true
			}
		}
		label := clipRunes(stringArg(fields, "label"), maxSourceTextRunes)
		if !isNumber || n < 1 || n != float64(int(n)) || label == "" {
			return nil, []string{"each source needs a whole number n from 1 and a label"}
		}
		if seen[int(n)] {
			return nil, []string{fmt.Sprintf("source %d is listed twice", int(n))}
		}
		seen[int(n)] = true
		source := serviceports.PublishedSource{
			N:      int(n),
			Tool:   clipRunes(stringArg(fields, "tool"), maxSourceTextRunes),
			Label:  label,
			Detail: clipRunes(stringArg(fields, "detail"), maxSourceTextRunes),
		}
		if id, err := pulid.Parse(strings.TrimSpace(stringArg(fields, "artifactId"))); err == nil {
			source.ArtifactID = id
		}
		sources = append(sources, source)
	}

	return sources, nil
}

func clipRunes(text string, most int) string {
	text = strings.TrimSpace(text)
	if utf8.RuneCountInString(text) <= most {
		return text
	}

	return string([]rune(text)[:most])
}

// publishedContent is what the model reads after its document was kept.
func publishedContent(shown *serviceports.ShownArtifact) string {
	encoded := fmt.Sprintf(`{"artifactId":%q,"title":%q,"status":"published"}`,
		shown.ID.String(), shown.Title)

	return FenceToolResult(publishArtifactName, encoded) +
		"\n\n[The person can open this document beside the conversation. Reply in two or " +
		"three sentences that point to it; do not repeat its text." + artifactRefNote(shown) + "]"
}

// shownNote tells the model that a result it is reading can be put in front
// of the person, and that it is only when the reply points to it: a lookup
// made to find something out is not worth a place beside the conversation
// unless the answer rests on it. A small model otherwise reprints a
// twenty-five row list as a markdown table, or fetches every row of a list
// again one by one.
func shownNote(shown *serviceports.ShownArtifact) string {
	if tabular(shown.Kind) && shown.Opens {
		return fmt.Sprintf("\n\n[This view is kept beside the conversation as %q, with how many "+
			"rows it holds and the first few; the person opens it as the live table. Point to it "+
			"in the sentence that describes it, and do not write its rows out or paste a link. Say "+
			"what it was narrowed to, and anything the description asked for that it could not "+
			"express.%s]", shown.Title, artifactRefNote(shown))
	}
	if tabular(shown.Kind) && shown.Actionable {
		return fmt.Sprintf("\n\n[This result is kept as a table titled %q that the person works "+
			"from beside the conversation: they select rows there, review each one and act on "+
			"them. Point to it in the sentence that mentions it, however few rows it has, and do "+
			"not write its rows out as a markdown table. Answer with the count and the rows that "+
			"need attention. Work from the fields it already has instead of looking up each row "+
			"again.%s]", shown.Title, artifactRefNote(shown))
	}
	if tabular(shown.Kind) && shown.Rows > 0 && shown.Rows <= InlineRows {
		return fmt.Sprintf("\n\n[This result has %d rows, few enough to answer in your reply: "+
			"show the rows that answer the question as a markdown table with only the columns "+
			"it needs, and do not point to the table titled %q; a table you do not point to is "+
			"not kept. Work from the fields it already has instead of looking up each row "+
			"again.]", shown.Rows, shown.Title)
	}
	if tabular(shown.Kind) {
		return fmt.Sprintf("\n\n[This result is kept as %s titled %q, which the person can "+
			"open beside the conversation. It is too long to repeat: do not write its rows "+
			"out as a markdown table. Answer with the count and the rows that need attention, "+
			"and point to the table in the sentence that mentions it. Work from the fields it "+
			"already has instead of looking up each row again.%s]",
			artifactNoun(shown.Kind), shown.Title, artifactRefNote(shown))
	}

	return fmt.Sprintf("\n\n[This result can be shown to the person as %s titled %q, which "+
		"they can open beside the conversation. Point to it only if your answer rests on it; "+
		"a result you do not point to is not shown. Answer with what matters rather than "+
		"repeating its fields; the person reads the rest on it.%s]",
		artifactNoun(shown.Kind), shown.Title, artifactRefNote(shown))
}

// InlineRows is the most rows a reply repeats as a markdown table. A table
// that short is answered in the reply and not kept; a longer one is kept and
// pointed to, and its rows are not repeated. One or the other, never both.
const InlineRows = 12

func tabular(kind string) bool {
	return kind == "table_view" || kind == "report_preview"
}

func artifactNoun(kind string) string {
	switch kind {
	case "table_view", "report_preview":
		return "a table"
	case "entity_card":
		return "a record card"
	case "report_run":
		return "a report run"
	case "rate_explanation":
		return "a rate ledger"
	case "run_diff":
		return "a comparison of two runs"
	case "email_draft":
		return "a draft"
	case "plan":
		return "a plan"
	default:
		return "an artifact"
	}
}

// unpublishableRefusal answers publish_artifact where there is nowhere to put
// a document: a background run, or a surface without a pane.
const unpublishableRefusal = "There is nowhere to publish a document from here. Put the text " +
	"in your reply instead."
