package documentaiextractionjobs

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documentaiextraction"
	"github.com/emoss08/trenova/internal/modules/documents/domain/extractionrollout"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/documentanalysis"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentaiextractionports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionrolloutports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeRolloutRouter struct {
	preferred pulid.ID
	assignErr error
	settleErr error
	assigned  *extractionrolloutports.AssignExtractionRolloutRequest
	settled   *extractionrolloutports.SettleExtractionRolloutRequest
}

func (f *fakeRolloutRouter) AssignExtraction(
	_ context.Context,
	req *extractionrolloutports.AssignExtractionRolloutRequest,
) (pulid.ID, error) {
	f.assigned = req
	return f.preferred, f.assignErr
}

func (f *fakeRolloutRouter) SettleExtraction(
	_ context.Context,
	req *extractionrolloutports.SettleExtractionRolloutRequest,
) error {
	f.settled = req
	return f.settleErr
}

func rolloutTenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func TestRolloutPreferenceAsksTheRouterForTheExtraction(t *testing.T) {
	t.Parallel()

	candidate := pulid.MustNew("aip_")
	router := &fakeRolloutRouter{preferred: candidate}
	activities := &Activities{logger: zap.NewNop(), rollout: router}
	payload := &documentanalysis.ProcessDocumentAIExtractionPayload{
		DocumentID:  pulid.MustNew("doc_"),
		ExtractedAt: 1_800_000_000,
	}

	preferred := activities.rolloutPreference(t.Context(), rolloutTenant(), payload)

	assert.Equal(t, candidate, preferred)
	require.NotNil(t, router.assigned)
	assert.Equal(t, payload.DocumentID, router.assigned.DocumentID)
	assert.Equal(t, payload.ExtractedAt, router.assigned.ExtractedAt)
}

func TestRolloutPreferenceNeverBlocksProduction(t *testing.T) {
	t.Parallel()

	payload := &documentanalysis.ProcessDocumentAIExtractionPayload{
		DocumentID: pulid.MustNew("doc_"),
	}

	withoutRouter := &Activities{logger: zap.NewNop()}
	assert.True(t, withoutRouter.rolloutPreference(t.Context(), rolloutTenant(), payload).IsNil())

	failing := &Activities{
		logger: zap.NewNop(),
		rollout: &fakeRolloutRouter{
			preferred: pulid.MustNew("aip_"),
			assignErr: errors.New("database unavailable"),
		},
	}
	assert.True(t, failing.rolloutPreference(t.Context(), rolloutTenant(), payload).IsNil(),
		"a rollout that cannot be read serves from the usual providers")
}

func TestSettleRolloutRecordsWhoServedTheExtraction(t *testing.T) {
	t.Parallel()

	router := &fakeRolloutRouter{settleErr: errors.New("database unavailable")}
	activities := &Activities{logger: zap.NewNop(), rollout: router}
	payload := &ApplyDocumentAIExtractionPayload{
		DocumentID:  pulid.MustNew("doc_"),
		ExtractedAt: 1_800_000_000,
	}
	row := &documentaiextraction.Extraction{
		ProviderID: pulid.MustNew("aip_"),
		Model:      "trenova-extract",
	}

	activities.settleRollout(
		t.Context(), payload, rolloutTenant(), extractionrollout.OutcomeAccepted, row,
	)

	require.NotNil(t, router.settled)
	assert.Equal(t, extractionrollout.OutcomeAccepted, router.settled.Outcome)
	assert.Equal(t, row.ProviderID, router.settled.ServedProviderID)
	assert.Equal(t, "trenova-extract", router.settled.ServedModel)
	assert.Equal(t, payload.DocumentID, router.settled.DocumentID)

	activities.settleRollout(
		t.Context(), payload, rolloutTenant(), extractionrollout.OutcomeSuperseded, nil,
	)
	assert.True(t, router.settled.ServedProviderID.IsNil())
}

func TestRolloutOutcomeFollowsTheAppliedAnswer(t *testing.T) {
	t.Parallel()

	completed := &documentanalysis.AsyncAIExtractionCompletion{
		Status: documentaiextractionports.AIBackgroundExtractionStatusCompleted,
	}
	failed := &documentanalysis.AsyncAIExtractionCompletion{
		Status: documentaiextractionports.AIBackgroundExtractionStatusFailed,
	}

	assert.Equal(t, extractionrollout.OutcomeAccepted,
		rolloutOutcome(completed, documentanalysis.AIAcceptanceStatusAccepted))
	assert.Equal(t, extractionrollout.OutcomeRejected,
		rolloutOutcome(completed, documentanalysis.AIAcceptanceStatusRejected))
	assert.Equal(t, extractionrollout.OutcomeFailed,
		rolloutOutcome(failed, documentanalysis.AIAcceptanceStatusRejected))
}
