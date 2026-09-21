package agentjobs

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/documentshipmentdraft"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeContent struct {
	serviceports.DocumentContentService

	draft  *documentshipmentdraft.DocumentShipmentDraft
	lastID pulid.ID
}

func (f *fakeContent) GetShipmentDraft(
	_ context.Context,
	documentID pulid.ID,
	_ pagination.TenantInfo,
) (*documentshipmentdraft.DocumentShipmentDraft, error) {
	f.lastID = documentID

	return f.draft, nil
}

// A run woken by document.extracted starts with the draft in front of it,
// the way a run woken by a service failure starts with the shipment.
func TestSubjectContext_DescribesADocumentByItsDraft(t *testing.T) {
	t.Parallel()

	attached := pulid.MustNew("shp_")
	content := &fakeContent{draft: &documentshipmentdraft.DocumentShipmentDraft{
		Status:             documentshipmentdraft.StatusReady,
		DocumentKind:       "rate_confirmation",
		Confidence:         0.82,
		AttachedShipmentID: &attached,
		DraftData: map[string]any{
			"fields": map[string]any{"bol": map[string]any{"value": "BOL-778", "confidence": 0.97}},
		},
	}}
	subjects := &SubjectContext{content: content, logger: zap.NewNop()}

	docID := pulid.MustNew("doc_")
	subject, err := subjects.Describe(t.Context(), pagination.TenantInfo{}, agent.SubjectDocument, docID)
	require.NoError(t, err)

	assert.Equal(t, docID, content.lastID)
	assert.Equal(t, "Document (rate confirmation)", subject.Label)
	assert.Contains(t, subject.Notes, "BOL-778")
	assert.Contains(t, subject.Notes, attached.String())
	assert.Contains(t, subject.Notes, "do not create another")
}

func TestSubjectContext_DocumentWithoutContentServiceStillHasAnID(t *testing.T) {
	t.Parallel()

	subjects := &SubjectContext{logger: zap.NewNop()}
	docID := pulid.MustNew("doc_")

	subject, err := subjects.Describe(t.Context(), pagination.TenantInfo{}, agent.SubjectDocument, docID)
	require.NoError(t, err)
	assert.Equal(t, docID.String(), subject.ID)
	assert.Equal(t, "Document", subject.Label)
	assert.Empty(t, subject.Notes)
}
