package artifactrecorder

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/operations/domain/documentshipmentdraft"
	"github.com/emoss08/trenova/internal/testutil/mocks/documentportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestArtifactRecorder_ExtractionReadsTheFileItsPagesAndEvidence(t *testing.T) {
	t.Parallel()

	documentID := pulid.MustNew("doc_")
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	documents := documentportsmock.NewMockDocumentRepository(t)
	documents.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&document.Document{
		ID: documentID, OriginalName: "Northline_RateCon.pdf", FileName: "stored.pdf",
		FileType: "Application/PDF",
	}, nil)

	contents := documentportsmock.NewMockDocumentContentService(t)
	contents.EXPECT().GetContent(mock.Anything, documentID, tenant).Return(&documentcontent.Content{
		ClassificationConfidence: 0.94,
		Pages: []*documentcontent.Page{{
			PageNumber: 1, Width: 1275, Height: 1650,
			Metadata: map[string]any{documentcontent.MetadataLines: []any{
				map[string]any{
					"t": "NORTHLINE LOGISTICS",
					"x": 0.08,
					"y": 0.05,
					"w": 0.3,
					"h": 0.02,
				},
				map[string]any{
					"t": "Northline Logistics remit",
					"x": 0.1,
					"y": 0.7,
					"w": 0.4,
					"h": 0.014,
				},
			}},
		}},
	}, nil)
	contents.EXPECT().GetShipmentDraft(mock.Anything, documentID, tenant).Return(
		&documentshipmentdraft.DocumentShipmentDraft{DraftData: map[string]any{
			"fields": map[string]any{
				"billTo": map[string]any{"evidenceExcerpt": "Northline Logistics remit to"},
			},
		}}, nil)

	recorder := &Recorder{
		ctx:    t.Context(),
		logger: zap.NewNop(),
		tenant: tenant,
		reads:  &Reads{Documents: documents, Contents: contents},
	}

	artifact := recorder.extraction("call_1", map[string]any{
		"documentId":   documentID.String(),
		"documentKind": "RateConfirmation",
		"fields": map[string]any{
			"billTo": map[string]any{
				"label": "Bill To", "value": "Northline Logistics", "confidence": 0.9, "pageNumber": 1.0,
			},
		},
	})
	require.NotNil(t, artifact)

	assert.Equal(t, "Northline_RateCon.pdf", artifact.Payload["fileName"])
	assert.Equal(t, "application/pdf", artifact.Payload["contentType"])
	assert.InDelta(t, 0.94, artifact.Payload["kindConfidence"], 0.0001)
	box := assistanttest.FieldsByKey(t, artifact)["billTo"]["box"].(map[string]float64)
	assert.InDelta(t, 0.7, box["y"], 0.0001, "the line the reading quoted")
}
