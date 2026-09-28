package agenttoolservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/workerchecklistservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeChecklists struct {
	checklistKeeper

	guard     *writeGuard
	checklist *worker.WorkerChecklist
	item      *worker.WorkerChecklistItem
	existing  bool

	started   *workerchecklistservice.StartRequest
	moves     []string
	cancelled *workerchecklistservice.CancelRequest
}

func newFakeChecklists() *fakeChecklists {
	checklistID := pulid.MustNew("wcl_")
	item := &worker.WorkerChecklistItem{
		ID:          pulid.MustNew("wcli_"),
		ChecklistID: checklistID,
		Label:       "Road test",
		Status:      worker.ChecklistItemPending,
		Version:     1,
	}
	return &fakeChecklists{
		guard: &writeGuard{},
		item:  item,
		checklist: &worker.WorkerChecklist{
			ID:       checklistID,
			WorkerID: pulid.MustNew("wrk_"),
			Name:     "Driver onboarding",
			Status:   worker.ChecklistStatusOpen,
			Items:    []*worker.WorkerChecklistItem{item},
			Version:  2,
		},
	}
}

func (f *fakeChecklists) PlanStart(
	context.Context,
	*workerchecklistservice.StartRequest,
) (*workerchecklistservice.StartPlan, error) {
	return &workerchecklistservice.StartPlan{Checklist: f.checklist, Existing: f.existing}, nil
}

func (f *fakeChecklists) Start(
	_ context.Context,
	req *workerchecklistservice.StartRequest,
) (*worker.WorkerChecklist, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.started = req
	return f.checklist, nil
}

func (f *fakeChecklists) planItem(
	req *workerchecklistservice.ItemRequest,
	status worker.ChecklistItemStatus,
	noteRequired bool,
) (*workerchecklistservice.ItemPlan, error) {
	if noteRequired && req.Note == "" {
		return nil, errortypes.NewValidationError("note", errortypes.ErrRequired,
			"A note says why")
	}
	after := *f.item
	after.Status = status
	return &workerchecklistservice.ItemPlan{
		Item:      workerchecklistservice.ItemChange{Before: f.item, After: &after},
		Checklist: f.checklist,
	}, nil
}

func (f *fakeChecklists) settle(move string) (*worker.WorkerChecklist, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.moves = append(f.moves, move)
	return f.checklist, nil
}

func (f *fakeChecklists) PlanCompleteItem(
	_ context.Context,
	req *workerchecklistservice.ItemRequest,
) (*workerchecklistservice.ItemPlan, error) {
	return f.planItem(req, worker.ChecklistItemDone, false)
}

func (f *fakeChecklists) CompleteItem(
	context.Context,
	*workerchecklistservice.ItemRequest,
) (*worker.WorkerChecklist, error) {
	return f.settle("complete")
}

func (f *fakeChecklists) PlanSkipItem(
	_ context.Context,
	req *workerchecklistservice.ItemRequest,
) (*workerchecklistservice.ItemPlan, error) {
	return f.planItem(req, worker.ChecklistItemSkipped, true)
}

func (f *fakeChecklists) SkipItem(
	context.Context,
	*workerchecklistservice.ItemRequest,
) (*worker.WorkerChecklist, error) {
	return f.settle("skip")
}

func (f *fakeChecklists) PlanReopenItem(
	context.Context,
	*workerchecklistservice.ReopenItemRequest,
) (*workerchecklistservice.ItemPlan, error) {
	before := *f.item
	before.Status = worker.ChecklistItemDone
	return &workerchecklistservice.ItemPlan{
		Item:      workerchecklistservice.ItemChange{Before: &before, After: f.item},
		Checklist: f.checklist,
	}, nil
}

func (f *fakeChecklists) ReopenItem(
	context.Context,
	*workerchecklistservice.ReopenItemRequest,
) (*worker.WorkerChecklist, error) {
	return f.settle("reopen")
}

func (f *fakeChecklists) PlanCancel(
	context.Context,
	*workerchecklistservice.CancelRequest,
) (*workerchecklistservice.ChecklistChange, error) {
	after := *f.checklist
	after.Status = worker.ChecklistStatusCancelled
	return &workerchecklistservice.ChecklistChange{Before: f.checklist, After: &after}, nil
}

func (f *fakeChecklists) Cancel(
	_ context.Context,
	req *workerchecklistservice.CancelRequest,
) (*worker.WorkerChecklist, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.cancelled = req
	return f.checklist, nil
}

func TestStartWorkerChecklist_SaysWhenOneIsAlreadyOpen(t *testing.T) {
	t.Parallel()

	checklists := newFakeChecklists()
	tool := newStartWorkerChecklistTool(checklists)
	params := executeParams(map[string]any{
		paramWorkerID:          checklists.checklist.WorkerID.String(),
		paramChecklistTemplate: pulid.MustNew("wclt_").String(),
		"startedAt":            "2026-09-21",
	})

	preview := previewWithoutWrites(t, checklists.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "with 1 item(s)")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, checklists.started)
	assert.Equal(t, time.Date(2026, time.September, 21, 0, 0, 0, 0, time.UTC).Unix(),
		checklists.started.StartedAt)

	checklists.existing = true
	preview = previewWithoutWrites(t, checklists.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "already open")
	assert.Empty(t, preview.Changes)
}

func TestUpdateWorkerChecklistItem_RoutesEachMove(t *testing.T) {
	t.Parallel()

	checklists := newFakeChecklists()
	tool := newUpdateWorkerChecklistItemTool(checklists)
	complete := executeParams(map[string]any{
		paramChecklistItemID: checklists.item.ID.String(),
		paramItemMove:        "Complete",
		fieldNote:            "Passed with the safety manager",
	})

	preview := previewWithoutWrites(t, checklists.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), complete)
	})
	assert.Contains(t, preview.Summary, `complete "Road test" on Driver onboarding`)
	result, err := tool.(serviceports.ToolResultReporter).ExecuteWithResult(t.Context(), complete)
	require.NoError(t, err)
	assert.Equal(t, "completed", result.Action)
	assert.Equal(t, checklists.item.ID.String(), result.IDs[paramChecklistItemID])

	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
		executeParams(map[string]any{
			paramChecklistItemID: checklists.item.ID.String(),
			paramItemMove:        "Skip",
		})), "skipping needs a note")
	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
		executeParams(map[string]any{
			paramChecklistItemID: checklists.item.ID.String(),
			paramItemMove:        "Reopen",
			fieldNote:            "x",
		})), "a reopened item takes no note")

	require.NoError(t, tool.Execute(t.Context(), executeParams(map[string]any{
		paramChecklistItemID: checklists.item.ID.String(),
		paramItemMove:        "Reopen",
	})))
	assert.Equal(t, []string{"complete", "reopen"}, checklists.moves)
	assert.Equal(t, permission.ResourceWorkerChecklist, tool.Policy().Resource)
}

func TestCancelWorkerChecklist_KeepsTheReason(t *testing.T) {
	t.Parallel()

	checklists := newFakeChecklists()
	tool := newCancelWorkerChecklistTool(checklists)
	params := executeParams(map[string]any{
		paramChecklistID: checklists.checklist.ID.String(),
		fieldReason:      "Hire withdrawn",
	})

	preview := previewWithoutWrites(t, checklists.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Equal(t, "Would cancel Driver onboarding.", preview.Summary)
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, checklists.cancelled)
	assert.Equal(t, "Hire withdrawn", checklists.cancelled.Reason)
	assert.Equal(t, agent.TierPropose, tool.Policy().MaxTier)
	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
		executeParams(map[string]any{paramChecklistID: checklists.checklist.ID.String()})))
}
