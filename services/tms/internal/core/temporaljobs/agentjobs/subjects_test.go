package agentjobs

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/documentshipmentdraft"
	"github.com/emoss08/trenova/internal/core/domain/insight"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
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

type fakeInsightRepo struct {
	repositories.InsightRepository

	found  *insight.Insight
	lastID pulid.ID
}

func (f *fakeInsightRepo) GetByID(
	_ context.Context,
	req repositories.GetInsightByIDRequest,
) (*insight.Insight, error) {
	f.lastID = req.ID

	return f.found, nil
}

// A run woken by insight.detected starts with the finding in front of it:
// what was measured, about whom, and what the detector suggests.
func TestSubjectContext_DescribesAnInsightByItsFinding(t *testing.T) {
	t.Parallel()

	found := &insight.Insight{
		ID:             pulid.MustNew("inst_"),
		Category:       insight.CategoryCashFlow,
		Severity:       insight.SeverityCritical,
		Status:         insight.StatusActive,
		Subject:        "Acme Foods",
		Headline:       "$48,200 of delivered work for Acme Foods is not yet billed",
		Recommendation: "Move the ready items through the billing queue.",
		Metrics:        []insight.Metric{{Key: "unbilled", Label: "Unbilled", Value: decimal.NewFromInt(48200)}},
	}
	repo := &fakeInsightRepo{found: found}
	subjects := &SubjectContext{insights: repo, logger: zap.NewNop()}

	subject, err := subjects.Describe(t.Context(), pagination.TenantInfo{}, agent.SubjectInsight, found.ID)
	require.NoError(t, err)

	assert.Equal(t, found.ID, repo.lastID)
	assert.Equal(t, "Insight: "+found.Headline, subject.Label)
	assert.Contains(t, subject.Notes, "Acme Foods")
	assert.Contains(t, subject.Notes, "48200")
	assert.Contains(t, subject.Notes, "billing queue")
	assert.NotContains(t, subject.Notes, "no longer active")
}

func TestSubjectContext_WarnsWhenTheInsightIsNoLongerActive(t *testing.T) {
	t.Parallel()

	found := &insight.Insight{
		ID:       pulid.MustNew("inst_"),
		Status:   insight.StatusResolved,
		Headline: "Detention at Acme Foods dock 4 has stopped",
	}
	subjects := &SubjectContext{insights: &fakeInsightRepo{found: found}, logger: zap.NewNop()}

	subject, err := subjects.Describe(t.Context(), pagination.TenantInfo{}, agent.SubjectInsight, found.ID)
	require.NoError(t, err)
	assert.Contains(t, subject.Notes, "no longer active")
}

func TestSubjectContext_InsightWithoutRepositoryStillHasAnID(t *testing.T) {
	t.Parallel()

	subjects := &SubjectContext{logger: zap.NewNop()}
	id := pulid.MustNew("inst_")

	subject, err := subjects.Describe(t.Context(), pagination.TenantInfo{}, agent.SubjectInsight, id)
	require.NoError(t, err)
	assert.Equal(t, id.String(), subject.ID)
	assert.Equal(t, "Insight", subject.Label)
}
