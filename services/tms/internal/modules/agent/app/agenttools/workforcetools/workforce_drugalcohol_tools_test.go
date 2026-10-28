package workforcetools

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerdrugalcoholservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerdot"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDrugAlcohol struct {
	DrugAlcoholKeeper

	guard *agenttoolstest.WriteGuard
	test  *workerdot.WorkerDOTTest
	draw  *workerdot.DOTRandomDraw
	entry *workerdot.DOTRandomDrawEntry

	recorded  *workerdot.WorkerDOTTest
	cancelled []string
	drawn     *workerdrugalcoholservice.RunDrawRequest
	finalized []pulid.ID
	moved     *workerdrugalcoholservice.UpdateEntryRequest
}

func newFakeDrugAlcohol() *fakeDrugAlcohol {
	return &fakeDrugAlcohol{
		guard: &agenttoolstest.WriteGuard{},
		test: &workerdot.WorkerDOTTest{
			ID:        pulid.MustNew("wdt_"),
			WorkerID:  pulid.MustNew("wrk_"),
			TestType:  workerdot.DOTTestRandom,
			Substance: workerdot.DOTSubstanceDrug,
			Status:    workerdot.DOTTestStatusScheduled,
			Result:    workerdot.DOTResultPending,
			Version:   2,
		},
		draw: &workerdot.DOTRandomDraw{
			ID:        pulid.MustNew("drdraw_"),
			PeriodKey: "2026-Q3",
			Status:    workerdot.RandomDrawStatusDraft,
			Version:   1,
		},
		entry: &workerdot.DOTRandomDrawEntry{
			ID:        pulid.MustNew("drde_"),
			Substance: workerdot.DOTSubstanceDrug,
			Rank:      1,
			Status:    workerdot.RandomEntrySelected,
			Version:   1,
		},
	}
}

func (f *fakeDrugAlcohol) PlanRecordTest(
	_ context.Context,
	entity *workerdot.WorkerDOTTest,
	userID pulid.ID,
) (*workerdot.WorkerDOTTest, error) {
	if entity.TestType == workerdot.DOTTestPostAccident && entity.Reason == "" {
		return nil, errortypes.NewValidationError("reason", errortypes.ErrRequired,
			"A post-accident test records what happened")
	}
	planned := *entity
	planned.OrderedByID = userID
	return &planned, nil
}

func (f *fakeDrugAlcohol) RecordTest(
	ctx context.Context,
	entity *workerdot.WorkerDOTTest,
	userID pulid.ID,
) (*workerdot.WorkerDOTTest, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	planned, err := f.PlanRecordTest(ctx, entity, userID)
	if err != nil {
		return nil, err
	}
	planned.ID = pulid.MustNew("wdt_")
	f.recorded = planned
	return planned, nil
}

func (f *fakeDrugAlcohol) PlanCancelTest(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
	string,
) (*workerdrugalcoholservice.TestChange, error) {
	if f.test.Result != workerdot.DOTResultPending {
		return nil, errortypes.NewValidationError("status", errortypes.ErrInvalidOperation,
			"A test with a result cannot be cancelled")
	}
	after := *f.test
	after.Status = workerdot.DOTTestStatusCancelled
	return &workerdrugalcoholservice.TestChange{Before: f.test, After: &after}, nil
}

func (f *fakeDrugAlcohol) CancelTest(
	_ context.Context,
	_ pagination.TenantInfo,
	_ pulid.ID,
	reason string,
	_ pulid.ID,
) (*workerdot.WorkerDOTTest, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.cancelled = append(f.cancelled, reason)
	return f.test, nil
}

func (f *fakeDrugAlcohol) PlanRunDraw(
	_ context.Context,
	req *workerdrugalcoholservice.RunDrawRequest,
) (*workerdot.DOTRandomDraw, error) {
	return &workerdot.DOTRandomDraw{
		PoolID:          req.PoolID,
		PeriodKey:       "2026-Q3",
		Status:          workerdot.RandomDrawStatusDraft,
		PoolSize:        40,
		DrugSelected:    5,
		AlcoholSelected: 1,
	}, nil
}

func (f *fakeDrugAlcohol) RunDraw(
	_ context.Context,
	req *workerdrugalcoholservice.RunDrawRequest,
) (*workerdot.DOTRandomDraw, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.drawn = req
	return f.draw, nil
}

func (f *fakeDrugAlcohol) PlanFinalizeDraw(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*workerdrugalcoholservice.DrawChange, error) {
	after := *f.draw
	after.Status = workerdot.RandomDrawStatusFinal
	return &workerdrugalcoholservice.DrawChange{Before: f.draw, After: &after}, nil
}

func (f *fakeDrugAlcohol) FinalizeDraw(
	_ context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
	_ pulid.ID,
) (*workerdot.DOTRandomDraw, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.finalized = append(f.finalized, id)
	return f.draw, nil
}

func (f *fakeDrugAlcohol) PlanUpdateDrawEntry(
	_ context.Context,
	req *workerdrugalcoholservice.UpdateEntryRequest,
) (*workerdrugalcoholservice.EntryChange, error) {
	if req.Status == workerdot.RandomEntryExcused && req.ExcuseReason == "" {
		return nil, errortypes.NewValidationError("excuseReason", errortypes.ErrRequired,
			"An excused selection says why")
	}
	after := *f.entry
	after.Status = req.Status
	after.ExcuseReason = req.ExcuseReason
	return &workerdrugalcoholservice.EntryChange{Before: f.entry, After: &after}, nil
}

func (f *fakeDrugAlcohol) UpdateDrawEntry(
	_ context.Context,
	req *workerdrugalcoholservice.UpdateEntryRequest,
) (*workerdot.DOTRandomDrawEntry, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.moved = req
	return f.entry, nil
}

func TestScheduleDOTTest_RecordsAnOrderAndNoResult(t *testing.T) {
	t.Parallel()

	tests := newFakeDrugAlcohol()
	tool := newScheduleDOTTestTool(tests)
	params := agenttoolstest.ExecuteParams(map[string]any{
		toolparams.ParamWorkerID:  tests.test.WorkerID.String(),
		paramDOTTestType:          "PostAccident",
		paramSubstance:            "Alcohol",
		paramScheduledAt:          "2026-09-21T09:00:00-05:00",
		changepreview.FieldReason: "Jackknife on I-80 with a tow-away",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		tests.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Contains(t, preview.Summary, "No result is recorded")
	require.NoError(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), params))

	result, err := tool.(agenttoolports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	require.NotNil(t, tests.recorded)
	assert.Equal(t, workerdot.DOTTestStatusScheduled, tests.recorded.Status)
	assert.Equal(t, workerdot.DOTResultPending, tests.recorded.Result)
	assert.True(t, tests.recorded.IsDOT)
	assert.Contains(t, result.IDs, paramDOTTestID)
	assert.Equal(t, permission.ResourceWorkerDOTTest, tool.Policy().Resource)
	assert.Equal(t, permission.OpCreate, tool.Policy().Operation)

	missingReason := agenttoolstest.ExecuteParams(map[string]any{
		toolparams.ParamWorkerID: tests.test.WorkerID.String(),
		paramDOTTestType:         "PostAccident",
		paramSubstance:           "Drug",
	})
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), missingReason),
		"validation runs the service's plan")
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(),
		agenttoolstest.ExecuteParams(map[string]any{
			toolparams.ParamWorkerID: tests.test.WorkerID.String(),
			paramDOTTestType:         "Positive",
			paramSubstance:           "Drug",
		})))
}

func TestCancelDOTTest_RefusesATestWithAResult(t *testing.T) {
	t.Parallel()

	tests := newFakeDrugAlcohol()
	tool := newCancelDOTTestTool(tests)
	params := agenttoolstest.ExecuteParams(map[string]any{
		paramDOTTestID:            tests.test.ID.String(),
		changepreview.FieldReason: "Driver was on leave",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		tests.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Equal(
		t,
		string(workerdot.DOTTestStatusCancelled),
		agenttoolstest.FieldByPath(
			t,
			agenttoolstest.PreviewChange(t, preview, 0),
			changepreview.FieldStatus,
		).After,
	)
	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, []string{"Driver was on leave"}, tests.cancelled)
	assert.Equal(t, agent.TierPropose, tool.Policy().MaxTier)
	assert.False(t, tool.Policy().Reversible)

	tests.test.Result = workerdot.DOTResultNegative
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), params))
}

func TestRunDOTRandomDraw_PreviewNamesNoOne(t *testing.T) {
	t.Parallel()

	draws := newFakeDrugAlcohol()
	tool := newRunDOTRandomDrawTool(draws)
	params := agenttoolstest.ExecuteParams(map[string]any{paramDrawDay: "2026-08-15"})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		draws.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Contains(t, preview.Summary, "5 for drug testing and 1 for alcohol")
	assert.Contains(t, preview.Summary, "decided when it is drawn")

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, draws.drawn)
	assert.Equal(t, int64(1_786_752_000), draws.drawn.At)
	assert.Equal(t, permission.OpManage, tool.Policy().Operation)
	_, reports := tool.(agenttoolports.ToolResultReporter)
	assert.False(t, reports, "a round has no record page to link")
}

func TestFinalizeDOTRandomDraw_RunsOnlyFromAPersonsApproval(t *testing.T) {
	t.Parallel()

	draws := newFakeDrugAlcohol()
	tool := newFinalizeDOTRandomDrawTool(draws)
	raw := map[string]any{paramDrawID: draws.draw.ID.String()}

	err := tool.Execute(t.Context(), agenttoolstest.ExecuteParams(raw))
	require.True(t, errors.Is(err, toolspec.ErrNeedsAPersonsApproval))
	assert.Empty(t, draws.finalized)

	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.ApprovedParams(raw)))
	assert.Equal(t, []pulid.ID{draws.draw.ID}, draws.finalized)
	assert.Equal(t, agent.TierPropose, tool.Policy().MaxTier)
}

func TestUpdateDOTRandomSelection_AnExcuseSaysWhy(t *testing.T) {
	t.Parallel()

	draws := newFakeDrugAlcohol()
	tool := newUpdateDOTRandomSelectionTool(draws)
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(),
		agenttoolstest.ExecuteParams(map[string]any{
			paramDrawEntryID:   draws.entry.ID.String(),
			paramSelectionMove: "Excused",
		})))
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(),
		agenttoolstest.ExecuteParams(map[string]any{
			paramDrawEntryID:   draws.entry.ID.String(),
			paramSelectionMove: "Completed",
		})), "a selection completes only when its test is recorded")

	params := agenttoolstest.ExecuteParams(map[string]any{
		paramDrawEntryID:   draws.entry.ID.String(),
		paramSelectionMove: "Excused",
		paramExcuseReason:  "Extended medical leave",
	})
	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		draws.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Contains(t, preview.Summary, "Excused")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, draws.moved)
	assert.Equal(t, workerdot.RandomEntryExcused, draws.moved.Status)
}
