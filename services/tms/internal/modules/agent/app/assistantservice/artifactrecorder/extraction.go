package artifactrecorder

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/artifactpayload"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/extractionartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/typeutils"
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
	if r.reads == nil {
		return extractionartifact.ExtractionSource{}
	}

	return extractionartifact.LoadSource(r.ctx, &extractionartifact.SourceReads{
		Documents: r.reads.Documents,
		Contents:  r.reads.Contents,
		Logger:    r.logger,
	}, documentID, r.tenant).Source
}
