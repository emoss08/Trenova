package agenttoolservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/workerdrugalcoholservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDrugAlcohol struct {
	drugAlcoholKeeper

	guard *writeGuard
	test  *worker.WorkerDOTTest
	draw  *worker.DOTRandomDraw
	entry *worker.DOTRandomDrawEntry

	recorded  *worker.WorkerDOTTest
	cancelled []string
	drawn     *workerdrugalcoholservice.RunDrawRequest
	finalized []pulid.ID
	moved     *workerdrugalcoholservice.UpdateEntryRequest
}

func newFakeDrugAlcohol() *fakeDrugAlcohol {
	return &fakeDrugAlcohol{
		guard: &writeGuard{},
		test: &worker.WorkerDOTTest{
			ID:        pulid.MustNew("wdt_"),
			WorkerID:  pulid.MustNew("wrk_"),
			TestType:  worker.DOTTestRandom,
			Substance: worker.DOTSubstanceDrug,
			Status:    worker.DOTTestStatusScheduled,
			Result:    worker.DOTResultPending,
			Version:   2,
		},
		draw: &worker.DOTRandomDraw{
			ID:        pulid.MustNew("drdraw_"),
			PeriodKey: "2026-Q3",
			Status:    worker.RandomDrawStatusDraft,
			Version:   1,
		},
		entry: &worker.DOTRandomDrawEntry{
			ID:        pulid.MustNew("drde_"),
			Substance: worker.DOTSubstanceDrug,
			Rank:      1,
			Status:    worker.RandomEntrySelected,
			Version:   1,
		},
	}
}

func (f *fakeDrugAlcohol) PlanRecordTest(
	_ context.Context,
	entity *worker.WorkerDOTTest,
	userID pulid.ID,
) (*worker.WorkerDOTTest, error) {
	if entity.TestType == worker.DOTTestPostAccident && entity.Reason == "" {
		return nil, errortypes.NewValidationError("reason", errortypes.ErrRequired,
			"A post-accident test records what happened")
	}
	planned := *entity
	planned.OrderedByID = userID
	return &planned, nil
}

func (f *fakeDrugAlcohol) RecordTest(
	ctx context.Context,
	entity *worker.WorkerDOTTest,
	userID pulid.ID,
) (*worker.WorkerDOTTest, error) {
	if err := f.guard.write(); err != nil {
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
	if f.test.Result != worker.DOTResultPending {
		return nil, errortypes.NewValidationError("status", errortypes.ErrInvalidOperation,
			"A test with a result cannot be cancelled")
	}
	after := *f.test
	after.Status = worker.DOTTestStatusCancelled
	return &workerdrugalcoholservice.TestChange{Before: f.test, After: &after}, nil
}

func (f *fakeDrugAlcohol) CancelTest(
	_ context.Context,
	_ pagination.TenantInfo,
	_ pulid.ID,
	reason string,
	_ pulid.ID,
) (*worker.WorkerDOTTest, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.cancelled = append(f.cancelled, reason)
	return f.test, nil
}

func (f *fakeDrugAlcohol) PlanRunDraw(
	_ context.Context,
	req *workerdrugalcoholservice.RunDrawRequest,
) (*worker.DOTRandomDraw, error) {
	return &worker.DOTRandomDraw{
		PoolID:          req.PoolID,
		PeriodKey:       "2026-Q3",
		Status:          worker.RandomDrawStatusDraft,
		PoolSize:        40,
		DrugSelected:    5,
		AlcoholSelected: 1,
	}, nil
}

func (f *fakeDrugAlcohol) RunDraw(
	_ context.Context,
	req *workerdrugalcoholservice.RunDrawRequest,
) (*worker.DOTRandomDraw, error) {
	if err := f.guard.write(); err != nil {
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
	after.Status = worker.RandomDrawStatusFinal
	return &workerdrugalcoholservice.DrawChange{Before: f.draw, After: &after}, nil
}

func (f *fakeDrugAlcohol) FinalizeDraw(
	_ context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
	_ pulid.ID,
) (*worker.DOTRandomDraw, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.finalized = append(f.finalized, id)
	return f.draw, nil
}

func (f *fakeDrugAlcohol) PlanUpdateDrawEntry(
	_ context.Context,
	req *workerdrugalcoholservice.UpdateEntryRequest,
) (*workerdrugalcoholservice.EntryChange, error) {
	if req.Status == worker.RandomEntryExcused && req.ExcuseReason == "" {
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
) (*worker.DOTRandomDrawEntry, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.moved = req
	return f.entry, nil
}

func TestScheduleDOTTest_RecordsAnOrderAndNoResult(t *testing.T) {
	t.Parallel()

	tests := newFakeDrugAlcohol()
	tool := newScheduleDOTTestTool(tests)
	params := executeParams(map[string]any{
		paramWorkerID:    tests.test.WorkerID.String(),
		paramDOTTestType: "PostAccident",
		paramSubstance:   "Alcohol",
		paramScheduledAt: "2026-09-21T09:00:00-05:00",
		fieldReason:      "Jackknife on I-80 with a tow-away",
	})

	preview := previewWithoutWrites(t, tests.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "No result is recorded")
	require.NoError(t, tool.(serviceports.ToolValidator).Validate(t.Context(), params))

	result, err := tool.(serviceports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	require.NotNil(t, tests.recorded)
	assert.Equal(t, worker.DOTTestStatusScheduled, tests.recorded.Status)
	assert.Equal(t, worker.DOTResultPending, tests.recorded.Result)
	assert.True(t, tests.recorded.IsDOT)
	assert.Contains(t, result.IDs, paramDOTTestID)
	assert.Equal(t, permission.ResourceWorkerDOTTest, tool.Policy().Resource)
	assert.Equal(t, permission.OpCreate, tool.Policy().Operation)

	missingReason := executeParams(map[string]any{
		paramWorkerID:    tests.test.WorkerID.String(),
		paramDOTTestType: "PostAccident",
		paramSubstance:   "Drug",
	})
	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(), missingReason),
		"validation runs the service's plan")
	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
		executeParams(map[string]any{
			paramWorkerID:    tests.test.WorkerID.String(),
			paramDOTTestType: "Positive",
			paramSubstance:   "Drug",
		})))
}

func TestCancelDOTTest_RefusesATestWithAResult(t *testing.T) {
	t.Parallel()

	tests := newFakeDrugAlcohol()
	tool := newCancelDOTTestTool(tests)
	params := executeParams(map[string]any{
		paramDOTTestID: tests.test.ID.String(),
		fieldReason:    "Driver was on leave",
	})

	preview := previewWithoutWrites(t, tests.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Equal(t, string(worker.DOTTestStatusCancelled),
		fieldByPath(t, previewChange(t, preview, 0), fieldStatus).After)
	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, []string{"Driver was on leave"}, tests.cancelled)
	assert.Equal(t, agent.TierPropose, tool.Policy().MaxTier)
	assert.False(t, tool.Policy().Reversible)

	tests.test.Result = worker.DOTResultNegative
	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(), params))
}

func TestRunDOTRandomDraw_PreviewNamesNoOne(t *testing.T) {
	t.Parallel()

	draws := newFakeDrugAlcohol()
	tool := newRunDOTRandomDrawTool(draws)
	params := executeParams(map[string]any{paramDrawDay: "2026-08-15"})

	preview := previewWithoutWrites(t, draws.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "5 for drug testing and 1 for alcohol")
	assert.Contains(t, preview.Summary, "decided when it is drawn")

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, draws.drawn)
	assert.Equal(t, int64(1_786_752_000), draws.drawn.At)
	assert.Equal(t, permission.OpManage, tool.Policy().Operation)
	_, reports := tool.(serviceports.ToolResultReporter)
	assert.False(t, reports, "a round has no record page to link")
}

func TestFinalizeDOTRandomDraw_RunsOnlyFromAPersonsApproval(t *testing.T) {
	t.Parallel()

	draws := newFakeDrugAlcohol()
	tool := newFinalizeDOTRandomDrawTool(draws)
	raw := map[string]any{paramDrawID: draws.draw.ID.String()}

	err := tool.Execute(t.Context(), executeParams(raw))
	require.True(t, errors.Is(err, ErrNeedsAPersonsApproval))
	assert.Empty(t, draws.finalized)

	require.NoError(t, tool.Execute(t.Context(), approvedParams(raw)))
	assert.Equal(t, []pulid.ID{draws.draw.ID}, draws.finalized)
	assert.Equal(t, agent.TierPropose, tool.Policy().MaxTier)
}

func TestUpdateDOTRandomSelection_AnExcuseSaysWhy(t *testing.T) {
	t.Parallel()

	draws := newFakeDrugAlcohol()
	tool := newUpdateDOTRandomSelectionTool(draws)
	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
		executeParams(map[string]any{
			paramDrawEntryID:   draws.entry.ID.String(),
			paramSelectionMove: "Excused",
		})))
	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
		executeParams(map[string]any{
			paramDrawEntryID:   draws.entry.ID.String(),
			paramSelectionMove: "Completed",
		})), "a selection completes only when its test is recorded")

	params := executeParams(map[string]any{
		paramDrawEntryID:   draws.entry.ID.String(),
		paramSelectionMove: "Excused",
		paramExcuseReason:  "Extended medical leave",
	})
	preview := previewWithoutWrites(t, draws.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Excused")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, draws.moved)
	assert.Equal(t, worker.RandomEntryExcused, draws.moved.Status)
}
