package assistantservice

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/docrender"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/typeutils"
)

const (
	DocumentExportPDF  = "pdf"
	DocumentExportDOCX = "docx"

	docxContentType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	wordsPerMinute  = 220
)

// ExportDocument renders the version of a document a person is reading as a
// PDF, printed by the same renderer as invoices, or as a Word file.
func (s *Service) ExportDocument(
	ctx context.Context,
	req repositories.GetThreadRequest,
	artifactID pulid.ID,
	format string,
) (*services.ArtifactFile, error) {
	versions, err := s.documentLineage(ctx, req, artifactID)
	if err != nil {
		return nil, err
	}
	version := latestOf(versions)
	for _, candidate := range versions {
		if candidate.ID == artifactID {
			version = candidate
		}
	}
	doc := renderableDocument(version)
	name := assistantartifact.SlugBase(version.Title)

	switch format {
	case DocumentExportPDF:
		if s.pdfs == nil {
			return nil, errortypes.NewBusinessError("PDF export is not available here")
		}
		body, renderErr := s.pdfs.Render(ctx, &services.PDFRenderRequest{
			HTML:  docrender.HTML(doc),
			Title: version.Title,
		})
		if renderErr != nil {
			if errors.Is(renderErr, services.ErrPDFRendererUnavailable) {
				return nil, errortypes.NewBusinessError("PDF export is not available right now")
			}
			return nil, renderErr
		}

		return &services.ArtifactFile{
			FileName: name + ".pdf", ContentType: "application/pdf", Body: body,
		}, nil
	case DocumentExportDOCX:
		body, renderErr := docrender.DOCX(doc)
		if renderErr != nil {
			return nil, renderErr
		}

		return &services.ArtifactFile{
			FileName: name + ".docx", ContentType: docxContentType, Body: body,
		}, nil
	default:
		return nil, errortypes.NewValidationError(
			"format", errortypes.ErrInvalid, "A document downloads as pdf or docx",
		)
	}
}

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
	if editor := typeutils.StringOfTrimmed(version.Payload[assistantartifact.DocumentEditor]); editor != "" {
		author = editor
	}
	byline := strings.Join(nonEmpty(
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
		Title:   version.Title,
		Kicker:  fmt.Sprintf("%s · %d words · %d min read", docType, words, max(1, (words+wordsPerMinute/2)/wordsPerMinute)),
		Byline:  byline,
		Body:    body,
		Sources: rendered,
	}
}
