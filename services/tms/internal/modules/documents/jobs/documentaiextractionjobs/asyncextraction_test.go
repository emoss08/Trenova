package documentaiextractionjobs

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/aiconfig"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/documentanalysis"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/documentextractionsupport"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentaiextractionports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/documentcontrol"
	"github.com/emoss08/trenova/internal/testutil/mocks/documentcontentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/documentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

/*
The AI reading is queued when a document is analyzed and sent to the model
later. A tenant that turns "Read with AI" or document intelligence off in
between is not sent to the model: the submit settles the extraction as
rejected with nothing recorded or submitted (the AI extraction repository and
model service are nil, so reaching either panics).
*/
func TestSubmitAIExtraction_SendsNothingOnceAIReadingIsOff(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		disable func(*documentcontrol.DocumentControl)
	}{
		{
			name:    "read with AI off",
			disable: func(c *documentcontrol.DocumentControl) { c.EnableAIAssistedExtraction = false },
		},
		{
			name:    "document intelligence off",
			disable: func(c *documentcontrol.DocumentControl) { c.EnableDocumentIntelligence = false },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			orgID := pulid.MustNew("org_")
			buID := pulid.MustNew("bu_")
			extractedAt := int64(1_700_000_000)
			doc := &document.Document{
				ID:             pulid.MustNew("doc_"),
				OrganizationID: orgID,
				BusinessUnitID: buID,
				OriginalName:   "rate-confirmation.pdf",
			}
			control := documentcontrol.NewDefaultDocumentControl(orgID, buID)
			tt.disable(control)

			docRepo := documentportsmock.NewMockDocumentRepository(t)
			docRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(doc, nil)
			contentRepo := documentcontentportsmock.NewMockDocumentContentRepository(t)
			contentRepo.EXPECT().
				GetByDocumentID(mock.Anything, doc.ID, mock.Anything).
				Return(&documentcontent.Content{
					DocumentID:      doc.ID,
					LastExtractedAt: &extractedAt,
				}, nil)
			controlRepo := tenantportsmock.NewMockDocumentControlRepository(t)
			controlRepo.EXPECT().GetOrCreate(mock.Anything, orgID, buID).Return(control, nil)

			activities := &Activities{
				logger:       zap.NewNop(),
				cfg:          &aiconfig.Config{DocumentExtraction: true},
				documentRepo: docRepo,
				contentRepo:  contentRepo,
				support: documentextractionsupport.New(documentextractionsupport.Params{
					Logger:              zap.NewNop(),
					DocumentControlRepo: controlRepo,
				}),
			}

			completion, err := activities.submitAIExtraction(
				t.Context(),
				&documentanalysis.ProcessDocumentAIExtractionPayload{
					BasePayload: temporaltype.BasePayload{
						OrganizationID: orgID,
						BusinessUnitID: buID,
					},
					DocumentID:  doc.ID,
					ExtractedAt: extractedAt,
				},
			)

			require.NoError(t, err)
			require.NotNil(t, completion)
			assert.Equal(
				t,
				documentaiextractionports.AIBackgroundExtractionStatusFailed,
				completion.Status,
			)
			assert.Equal(t, FailureAIExtractionDisabled, completion.FailureCode)
			assert.Equal(
				t,
				string(documentanalysis.AIAcceptanceStatusRejected),
				completion.AcceptanceState,
			)
			assert.Equal(t, extractedAt, completion.ExtractedAt)
		})
	}
}
