package artifactrecorder

import (
	"cmp"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/artifactpayload"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/extractionartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/typeutils"
	"go.uber.org/zap"
)

func (r *Recorder) extraction(callID string, data any) *assistantartifact.Artifact {
	document, ok := artifactpayload.ToJSONDocument(data)
	if !ok {
		return nil
	}
	draft := document.Fields
	documentID, err := pulid.Parse(
		typeutils.StringOfTrimmed(draft[extractionartifact.DraftDocumentIDParam]),
	)
	if err != nil {
		return nil
	}

	source := r.extractionSource(documentID)

	return extractionartifact.ExtractionArtifact(callID, draft, &source)
}

func (r *Recorder) extractionSource(documentID pulid.ID) extractionartifact.ExtractionSource {
	source := extractionartifact.ExtractionSource{}
	if r.reads == nil {
		return source
	}

	if r.reads.Documents != nil {
		doc, err := r.reads.Documents.GetByID(r.ctx, documentports.GetDocumentByIDRequest{
			ID:         documentID,
			TenantInfo: r.tenant,
		})
		if err == nil && doc != nil {
			source.FileName = cmp.Or(doc.OriginalName, doc.FileName)
			source.ContentType = strings.ToLower(doc.FileType)
		}
	}

	if r.reads.Contents == nil {
		return source
	}
	content, err := r.reads.Contents.GetContent(r.ctx, documentID, r.tenant)
	if err != nil {
		r.logger.Debug("extraction pages could not be read", zap.Error(err))
	} else if content != nil {
		source.KindConfidence = content.ClassificationConfidence
		source.Pages = make([]extractionartifact.ExtractionPage, 0, len(content.Pages))
		for _, page := range content.Pages {
			if page == nil {
				continue
			}
			source.Pages = append(source.Pages, extractionartifact.ExtractionPage{
				Number: page.PageNumber,
				Width:  page.Width,
				Height: page.Height,
				Lines:  documentcontent.LinesOf(page.Metadata),
			})
		}
	}

	stored, err := r.reads.Contents.GetShipmentDraft(r.ctx, documentID, r.tenant)
	if err != nil {
		r.logger.Debug("extraction evidence could not be read", zap.Error(err))
	} else if stored != nil {
		source.FieldEvidence, source.StopEvidence = extractionartifact.DraftEvidence(
			stored.DraftData,
		)
	}

	return source
}
