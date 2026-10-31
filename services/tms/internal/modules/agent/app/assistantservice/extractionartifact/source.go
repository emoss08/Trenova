package extractionartifact

import (
	"cmp"
	"context"
	"maps"
	"strings"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/typeutils"
	"go.uber.org/zap"
)

type SourceReads struct {
	Documents documentports.DocumentRepository
	Contents  documentports.DocumentContentService
	Logger    *zap.Logger
}

// Loaded is what a document's extraction is drawn from: the file, its pages
// with their layout lines, docling's reading of it, and the shipment draft
// when one was made from it.
type Loaded struct {
	Source        ExtractionSource
	DocumentKind  string
	Intelligence  map[string]any
	ShipmentDraft map[string]any
}

func LoadSource(
	ctx context.Context,
	reads *SourceReads,
	documentID pulid.ID,
	tenant pagination.TenantInfo,
) Loaded {
	loaded := Loaded{}
	logger := reads.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	if reads.Documents != nil {
		doc, err := reads.Documents.GetByID(ctx, documentports.GetDocumentByIDRequest{
			ID:         documentID,
			TenantInfo: tenant,
		})
		if err == nil && doc != nil {
			loaded.Source.FileName = cmp.Or(doc.OriginalName, doc.FileName)
			loaded.Source.ContentType = strings.ToLower(doc.FileType)
		}
	}

	if reads.Contents == nil {
		return loaded
	}

	content, err := reads.Contents.GetContent(ctx, documentID, tenant)
	if err != nil {
		logger.Debug("extraction pages could not be read", zap.Error(err))
	} else if content != nil {
		loaded.Source.KindConfidence = content.ClassificationConfidence
		loaded.DocumentKind = content.DetectedDocumentKind
		loaded.Source.Pages = make([]ExtractionPage, 0, len(content.Pages))
		for _, page := range content.Pages {
			if page == nil {
				continue
			}
			loaded.Source.Pages = append(loaded.Source.Pages, ExtractionPage{
				Number: page.PageNumber,
				Width:  page.Width,
				Height: page.Height,
				Lines:  documentcontent.LinesOf(page.Metadata),
			})
		}
		if intelligence, ok := content.StructuredData["intelligence"].(map[string]any); ok {
			loaded.Intelligence = intelligence
		}
	}

	stored, err := reads.Contents.GetShipmentDraft(ctx, documentID, tenant)
	if err != nil {
		logger.Debug("extraction evidence could not be read", zap.Error(err))
	} else if stored != nil {
		loaded.ShipmentDraft = stored.DraftData
		loaded.Source.FieldEvidence, loaded.Source.StopEvidence = DraftEvidence(stored.DraftData)
	}

	return loaded
}

// ReadDocument is a document as it was read, in the extraction card's payload:
// each value with the page and box it was found in. A document no shipment
// draft was made from (a proof of delivery, a receipt) is read from docling's
// own intelligence. A document nothing was read from has no fields.
func ReadDocument(
	ctx context.Context,
	reads *SourceReads,
	documentID pulid.ID,
	tenant pagination.TenantInfo,
) map[string]any {
	loaded := LoadSource(ctx, reads, documentID, tenant)

	draft := loaded.ShipmentDraft
	if !hasFields(draft) {
		draft = intelligenceDraft(loaded.Intelligence, loaded.DocumentKind)
		loaded.Source.FieldEvidence, loaded.Source.StopEvidence = DraftEvidence(draft)
	}
	draft[DraftDocumentIDParam] = documentID.String()

	if artifact := ExtractionArtifact("", draft, &loaded.Source); artifact != nil {
		return artifact.Payload
	}

	return map[string]any{
		"documentId":  documentID.String(),
		"fileName":    loaded.Source.FileName,
		"contentType": loaded.Source.ContentType,
		"pageCount":   max(len(loaded.Source.Pages), 1),
		"pages":       pageSizes(loaded.Source.Pages),
		"fields":      []map[string]any{},
		"missing":     []string{},
	}
}

func hasFields(draft map[string]any) bool {
	fields, _ := draft["fields"].(map[string]any)
	stops, _ := draft["stops"].([]any)
	return len(fields) > 0 || len(stops) > 0
}

// intelligenceDraft names docling's reading in the draft's terms, so one
// reader serves both.
func intelligenceDraft(intelligence map[string]any, detectedKind string) map[string]any {
	draft := make(map[string]any, len(intelligence)+3)
	maps.Copy(draft, intelligence)
	draft["documentKind"] = cmp.Or(typeutils.StringOfTrimmed(intelligence["kind"]), detectedKind)
	draft["confidence"] = intelligence["overallConfidence"]

	return draft
}
