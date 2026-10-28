package ratetools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/rating/app/rateimportservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/ratesimulationservice"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateimport"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratesimulation"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRateImports struct {
	guard     *agenttoolstest.WriteGuard
	batch     *rateimport.RateImportBatch
	committed *rateimportservice.CommitRequest
	discarded *rateimportservice.CommitRequest
}

func (f *fakeRateImports) PlanCommit(
	context.Context,
	*rateimportservice.CommitRequest,
) (*rateimportservice.CommitPlan, error) {
	return &rateimportservice.CommitPlan{
		Batch:         f.batch,
		SupersededIDs: []pulid.ID{pulid.MustNew("rar_")},
		Rules: []*rateagreement.RateAgreementRule{
			agreementRule("Texas out", "2.49"),
			agreementRule("Texas back", "1.90"),
		},
	}, nil
}

func (f *fakeRateImports) Commit(
	_ context.Context,
	req *rateimportservice.CommitRequest,
) (*rateimport.RateImportBatch, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.committed = req

	return f.batch, nil
}

func (f *fakeRateImports) PlanDiscard(
	context.Context,
	*rateimportservice.CommitRequest,
) (*rateimportservice.BatchChange, error) {
	after := *f.batch
	after.Status = rateimport.StatusDiscarded

	return &rateimportservice.BatchChange{Before: f.batch, After: &after}, nil
}

func (f *fakeRateImports) Discard(
	_ context.Context,
	req *rateimportservice.CommitRequest,
) (*rateimport.RateImportBatch, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.discarded = req

	return f.batch, nil
}

func TestRateImportTools_ApplyingASheetIsAPersonsCallAndDiscardingIsNot(t *testing.T) {
	t.Parallel()

	imports := &fakeRateImports{
		guard: &agenttoolstest.WriteGuard{},
		batch: &rateimport.RateImportBatch{
			ID:            pulid.MustNew("rib_"),
			FileName:      "acme-2027.csv",
			Status:        rateimport.StatusParsed,
			EffectiveFrom: 1_798_761_600,
			Version:       2,
		},
	}
	params := agenttoolstest.ExecuteParams(
		map[string]any{paramRateImportID: imports.batch.ID.String()},
	)

	commit := NewCommitRateImportTool(imports)
	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		imports.guard,
		func() (*agent.ToolPreview, error) {
			return commit.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Contains(t, preview.Summary, "Would apply the rate sheet acme-2027.csv from ")
	assert.Contains(t, preview.Summary, "close out 1 lane and add 2 lanes")
	assert.NotContains(t, preview.Summary, "2.49", "negotiated rates are confidential")
	assert.Nil(t, agenttoolstest.PreviewChange(t, preview, 0).Money)
	require.ErrorIs(t, commit.Execute(t.Context(), params), toolspec.ErrNeedsAPersonsApproval)
	assert.Nil(t, imports.committed)

	approved := params
	approved.ProposalID = pulid.MustNew("ap_")
	require.NoError(t, commit.Execute(t.Context(), approved))
	assert.Equal(t, imports.batch.ID, imports.committed.RateImportBatchID)
	assert.Equal(t, []agent.EgressClass{agent.EgressMoney}, commit.Policy().Egress)

	discard := NewDiscardRateImportTool(imports)
	preview = agenttoolstest.PreviewWithoutWrites(
		t,
		imports.guard,
		func() (*agent.ToolPreview, error) {
			return discard.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Equal(
		t,
		"Discarded",
		agenttoolstest.FieldByPath(t, agenttoolstest.PreviewChange(t, preview, 0), "status").After,
	)
	require.NoError(t, discard.Execute(t.Context(), params))
	assert.Equal(t, imports.batch.ID, imports.discarded.RateImportBatchID)
	assert.Equal(t, []agent.EgressClass{agent.EgressInternal}, discard.Policy().Egress)
}

type fakeSimulations struct {
	guard   *agenttoolstest.WriteGuard
	created *ratesimulation.RateSimulation
}

func (f *fakeSimulations) PlanCreate(
	_ context.Context,
	entity *ratesimulation.RateSimulation,
) (*ratesimulationservice.CreatePlan, error) {
	return &ratesimulationservice.CreatePlan{
		Simulation: entity,
		Agreement:  storedAgreement(rateagreement.StatusDraft),
	}, nil
}

func (f *fakeSimulations) Create(
	_ context.Context,
	entity *ratesimulation.RateSimulation,
	_ pulid.ID,
) (*ratesimulation.RateSimulation, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.created = entity
	saved := *entity
	saved.ID = pulid.MustNew("rsim_")

	return &saved, nil
}

func TestRunRateSimulation_ReplaysTheWholeLastDayAndChangesNothing(t *testing.T) {
	t.Parallel()

	simulations := &fakeSimulations{guard: &agenttoolstest.WriteGuard{}}
	tool := NewRunRateSimulationTool(simulations)
	agreementID := pulid.MustNew("rag_")
	params := agenttoolstest.ExecuteParams(map[string]any{
		paramRateAgreementID: agreementID.String(),
		paramSimulationName:  "Acme renewal against Q3",
		paramSampleFrom:      "2026-07-01",
		paramSampleTo:        "2026-09-30",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		simulations.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Contains(t, preview.Summary, "Would replay Customer shipments shipped ")
	assert.Contains(t, preview.Summary, "against the Draft rate agreement ACME-2026")

	result, err := tool.(agenttoolports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t, "started", result.Action)
	assert.Equal(t, agreementID.String(), result.IDs[paramRateAgreementID])
	assert.Equal(
		t,
		simulations.created.SampleFrom+92*toolparams.SecondsPerDay,
		simulations.created.SampleTo,
	)
	assert.Equal(t, rateagreement.PartyTypeCustomer, simulations.created.PartyType)

	policy := tool.Policy()
	assert.Equal(t, permission.ResourceRateSimulation, policy.Resource)
	assert.Equal(t, agent.TierAutoExecute, policy.MaxTier)

	params.Params[paramSampleLimit] = maxSampleShipments + 1
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), params))
}
