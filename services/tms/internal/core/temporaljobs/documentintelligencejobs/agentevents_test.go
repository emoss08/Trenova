package documentintelligencejobs

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documentshipmentdraft"
	"github.com/emoss08/trenova/internal/testutil/agenteventstest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestUpsertDraft_PublishesDocumentExtractedEvent(t *testing.T) {
	t.Parallel()

	draftRepo := mocks.NewMockDocumentShipmentDraftRepository(t)
	draftRepo.EXPECT().
		Upsert(mock.Anything, mock.AnythingOfType("*documentshipmentdraft.DocumentShipmentDraft")).
		RunAndReturn(func(_ context.Context, entity *documentshipmentdraft.DocumentShipmentDraft) (*documentshipmentdraft.DocumentShipmentDraft, error) {
			return entity, nil
		}).
		Once()

	recorder := &agenteventstest.Recorder{}
	activities := &Activities{
		logger:      zap.NewNop(),
		draftRepo:   draftRepo,
		agentEvents: recorder,
	}
	doc := &document.Document{
		ID:             pulid.MustNew("doc_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}

	status, err := activities.upsertDraft(
		t.Context(), doc, kindRateConfirmation, &DocumentIntelligenceAnalysis{}, true,
	)

	require.NoError(t, err)
	require.Equal(t, document.ShipmentDraftStatusReady, status)
	events := recorder.Published()
	require.Len(t, events, 1)
	require.Equal(t, agent.EventDocumentExtracted, events[0].Kind)
	require.Equal(t, doc.ID, events[0].SubjectID)
	require.Equal(t, doc.OrganizationID, events[0].TenantInfo.OrgID)
	require.Equal(t, doc.BusinessUnitID, events[0].TenantInfo.BuID)
}
