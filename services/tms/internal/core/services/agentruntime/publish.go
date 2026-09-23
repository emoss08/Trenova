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
	"pass its artifactId with the whole new text."

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
			"body is longer than %d characters; publish the part that matters", maxDocumentBodyBytes))
	}

	var revises pulid.ID
	if raw := strings.TrimSpace(stringArg(arguments, "artifactId")); raw != "" {
		parsed, err := pulid.Parse(raw)
		if err != nil {
			problems = append(problems, "artifactId is not an id a publish_artifact result gave you")
		} else {
			revises = parsed
		}
	}

	if len(problems) > 0 {
		return failedOutcome("Tool %q was not run: %s.", publishArtifactName, strings.Join(problems, "; "))
	}

	return toolOutcome{
		publishes: true,
		data: serviceports.PublishedDocument{
			Title:      title,
			Body:       body,
			ArtifactID: revises,
		},
	}
}

// publishedContent is what the model reads after its document was kept.
func publishedContent(shown *serviceports.ShownArtifact) string {
	encoded := fmt.Sprintf(`{"artifactId":%q,"title":%q,"status":"published"}`,
		shown.ID.String(), shown.Title)

	return FenceToolResult(publishArtifactName, encoded) +
		"\n\n[The person can open this document beside the conversation. Reply in two or " +
		"three sentences that point to it; do not repeat its text.]"
}

// shownNote tells the model that a result it is reading is already in front
// of the person. A small model otherwise reprints a twenty-five row list as a
// markdown table under the table the person is already looking at.
func shownNote(shown *serviceports.ShownArtifact) string {
	return fmt.Sprintf("\n\n[Shown to the person as %s titled %q, which they can open beside "+
		"the conversation. Answer with what matters (the count, the few rows or fields that "+
		"answer the question, anything that needs attention) and refer to it rather than "+
		"repeating it.]", artifactNoun(shown.Kind), shown.Title)
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
