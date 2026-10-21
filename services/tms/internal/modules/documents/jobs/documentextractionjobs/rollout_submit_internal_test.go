package documentextractionjobs

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentaiextraction"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentaiextractionports"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"go.uber.org/zap"
)

type memoryAIExtractionRepository struct {
	documentaiextractionports.DocumentAIExtractionRepository
}

func (memoryAIExtractionRepository) SavePending(
	_ context.Context,
	entity *documentaiextraction.Extraction,
) (*documentaiextraction.Extraction, error) {
	entity.ID = pulid.MustNew("dae_")
	return entity, nil
}

func (memoryAIExtractionRepository) Update(
	_ context.Context,
	entity *documentaiextraction.Extraction,
) (*documentaiextraction.Extraction, error) {
	return entity, nil
}

type inlineExtraction struct {
	noopAIDocumentService
	preferred pulid.ID
}

func (s *inlineExtraction) SubmitRateConfirmationBackgroundExtraction(
	_ context.Context,
	req *documentaiextractionports.AIExtractRequest,
) (*documentaiextractionports.AIBackgroundExtractSubmission, error) {
	s.preferred = req.PreferredProviderID
	return &documentaiextractionports.AIBackgroundExtractSubmission{
		ProviderID:    pulid.MustNew("aip_"),
		Model:         "production-model",
		ExtractResult: &documentaiextractionports.AIExtractResult{},
	}, nil
}

func submitExtraction(t *testing.T) (*fakeRolloutRouter, *inlineExtraction) {
	t.Helper()

	extractedAt := int64(1_800_000_000)
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	docID := pulid.MustNew("doc_")
	router := &fakeRolloutRouter{preferred: pulid.MustNew("aip_")}
	ai := &inlineExtraction{}
	activities := &Activities{
		logger: zap.NewNop(),
		cfg:    &config.AIConfig{},
		documentRepo: &memoryDocumentRepository{doc: &document.Document{
			ID:             docID,
			OrganizationID: orgID,
			BusinessUnitID: buID,
			OriginalName:   "rate-confirmation.pdf",
		}},
		contentRepo: &memoryContentRepository{
			content: &documentcontent.Content{
				ID:              pulid.MustNew("dc_"),
				DocumentID:      docID,
				OrganizationID:  orgID,
				BusinessUnitID:  buID,
				ContentText:     "Load 48213",
				LastExtractedAt: &extractedAt,
			},
			pages: []*documentcontent.Page{{PageNumber: 1, ExtractedText: "Load 48213"}},
		},
		aiExtractionRepo:  memoryAIExtractionRepository{},
		aiDocumentService: ai,
		rollout:           router,
	}

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestActivityEnvironment()
	env.RegisterActivity(activities)
	_, err := env.ExecuteActivity(
		activities.SubmitDocumentAIExtractionActivity,
		&ProcessDocumentAIExtractionPayload{
			BasePayload: temporaltype.BasePayload{OrganizationID: orgID, BusinessUnitID: buID},
			DocumentID:  docID,
			ExtractedAt: extractedAt,
		},
	)
	require.NoError(t, err)

	return router, ai
}

func TestEveryExtractionIsOfferedToTheRollout(t *testing.T) {
	t.Parallel()

	router, ai := submitExtraction(t)

	require.NotNil(t, router.assigned)
	assert.Equal(t, int64(1_800_000_000), router.assigned.ExtractedAt)
	assert.Equal(t, router.preferred, ai.preferred, "the assigned side is the provider asked first")
}
