package assistantartifacts

import (
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/pkg/docrender"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/typeutils"
)

const (
	DocumentExportPDF  = "pdf"
	DocumentExportDOCX = "docx"

	docxContentType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	wordsPerMinute  = 220
)

// renderableDocument is a version as a file prints it: the kicker and byline
// the Desk shows above it, and its sources below.
func renderableDocument(version *assistantartifact.Artifact) *docrender.Document {
	body := typeutils.StringOfTrimmed(version.Payload[assistantartifact.DocumentBody])
	words := len(strings.Fields(docrender.PlainText(body)))
	docType := typeutils.StringOfTrimmed(version.Payload[assistantartifact.DocumentType])
	if docType == "" {
		docType = "Document"
	}
	author := typeutils.StringOfTrimmed(version.Payload[assistantartifact.DocumentAuthor])
	if editor := typeutils.StringOfTrimmed(
		version.Payload[assistantartifact.DocumentEditor],
	); editor != "" {
		author = editor
	}
	byline := strings.Join(stringutils.NonEmptyStrings(
		author,
		"written "+time.Unix(version.CreatedAt, 0).UTC().Format("Jan 2, 2006"),
		typeutils.StringOfTrimmed(version.Payload[assistantartifact.DocumentBasis]),
	), " · ")

	var sources []assistantartifact.DocumentSource
	_ = jsonutils.Convert(version.Payload[assistantartifact.DocumentSources], &sources)
	rendered := make([]docrender.Source, 0, len(sources))
	for _, source := range sources {
		rendered = append(rendered, docrender.Source{
			N: source.N, Label: source.Label, Tool: source.Tool, Detail: source.Detail,
		})
	}

	return &docrender.Document{
		Title: version.Title,
		Kicker: fmt.Sprintf(
			"%s · %d words · %d min read",
			docType,
			words,
			max(1, (words+wordsPerMinute/2)/wordsPerMinute),
		),
		Byline:  byline,
		Body:    body,
		Sources: rendered,
	}
}
