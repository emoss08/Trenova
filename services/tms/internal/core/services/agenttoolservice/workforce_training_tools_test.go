package agenttoolservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/workertrainingservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeTraining struct {
	trainingKeeper

	guard  *writeGuard
	record *worker.WorkerTrainingRecord
	course *worker.TrainingCourse
	failed []workertrainingservice.BulkAssignOutcome

	assigned     *workertrainingservice.AssignRequest
	bulkAssigned *workertrainingservice.BulkAssignRequest
	required     []pulid.ID
	completed    *workertrainingservice.CompleteRequest
	waived       *workertrainingservice.StatusRequest
	cancelled    *workertrainingservice.StatusRequest
}

func newFakeTraining() *fakeTraining {
	course := &worker.TrainingCourse{ID: pulid.MustNew("trc_"), Name: "Hazmat awareness"}
	return &fakeTraining{
		guard:  &writeGuard{},
		course: course,
		record: &worker.WorkerTrainingRecord{
			ID:       pulid.MustNew("wtr_"),
			WorkerID: pulid.MustNew("wrk_"),
			CourseID: course.ID,
			Course:   course,
			Status:   worker.TrainingStatusAssigned,
			Version:  2,
		},
	}
}

func (f *fakeTraining) assignedRecord(workerID, courseID pulid.ID) *worker.WorkerTrainingRecord {
	return &worker.WorkerTrainingRecord{
		WorkerID: workerID,
		CourseID: courseID,
		Course:   f.course,
		Status:   worker.TrainingStatusAssigned,
	}
}

func (f *fakeTraining) PlanAssign(
	_ context.Context,
	req *workertrainingservice.AssignRequest,
) (*worker.WorkerTrainingRecord, error) {
	return f.assignedRecord(req.WorkerID, req.CourseID), nil
}

func (f *fakeTraining) Assign(
	_ context.Context,
	req *workertrainingservice.AssignRequest,
) (*worker.WorkerTrainingRecord, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.assigned = req
	return f.record, nil
}

func (f *fakeTraining) PlanBulkAssign(
	_ context.Context,
	req *workertrainingservice.BulkAssignRequest,
) (*workertrainingservice.BulkAssignPlan, error) {
	plan := &workertrainingservice.BulkAssignPlan{Failed: f.failed}
	if len(f.failed) > 0 {
		return plan, nil
	}
	for _, workerID := range req.WorkerIDs {
		for _, courseID := range req.CourseIDs {
			plan.Records = append(plan.Records, f.assignedRecord(workerID, courseID))
		}
	}
	return plan, nil
}

func (f *fakeTraining) BulkAssign(
	_ context.Context,
	req *workertrainingservice.BulkAssignRequest,
) (*workertrainingservice.BulkAssignResult, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.bulkAssigned = req
	return &workertrainingservice.BulkAssignResult{}, nil
}

func (f *fakeTraining) PlanAssignRequired(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) ([]*worker.TrainingCourse, error) {
	return []*worker.TrainingCourse{f.course}, nil
}

func (f *fakeTraining) AssignRequired(
	_ context.Context,
	_ pagination.TenantInfo,
	workerID pulid.ID,
	_ pulid.ID,
) ([]*worker.WorkerTrainingRecord, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.required = append(f.required, workerID)
	return []*worker.WorkerTrainingRecord{f.record}, nil
}

func (f *fakeTraining) PlanComplete(
	_ context.Context,
	req *workertrainingservice.CompleteRequest,
) (*workertrainingservice.RecordChange, error) {
	if req.ID.IsNil() && (req.WorkerID.IsNil() || req.CourseID.IsNil()) {
		return nil, errortypes.NewValidationError("id", errortypes.ErrRequired,
			"Name the assignment, or the worker and course")
	}
	after := *f.record
	after.Status = worker.TrainingStatusCompleted
	if req.Score.Valid && req.Score.Decimal.IntPart() < 80 {
		after.Status = worker.TrainingStatusFailed
	}
	if req.ID.IsNil() {
		return &workertrainingservice.RecordChange{After: &after}, nil
	}
	return &workertrainingservice.RecordChange{Before: f.record, After: &after}, nil
}

func (f *fakeTraining) Complete(
	_ context.Context,
	req *workertrainingservice.CompleteRequest,
) (*worker.WorkerTrainingRecord, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.completed = req
	return f.record, nil
}

func (f *fakeTraining) planClose(status worker.TrainingStatus) *workertrainingservice.RecordChange {
	after := *f.record
	after.Status = status
	return &workertrainingservice.RecordChange{Before: f.record, After: &after}
}

func (f *fakeTraining) PlanWaive(
	_ context.Context,
	req *workertrainingservice.StatusRequest,
) (*workertrainingservice.RecordChange, error) {
	if req.Reason == "" {
		return nil, errortypes.NewValidationError("reason", errortypes.ErrRequired,
			"A waiver says why")
	}
	return f.planClose(worker.TrainingStatusWaived), nil
}

func (f *fakeTraining) Waive(
	_ context.Context,
	req *workertrainingservice.StatusRequest,
) (*worker.WorkerTrainingRecord, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.waived = req
	return f.record, nil
}

func (f *fakeTraining) PlanCancel(
	context.Context,
	*workertrainingservice.StatusRequest,
) (*workertrainingservice.RecordChange, error) {
	return f.planClose(worker.TrainingStatusCancelled), nil
}

func (f *fakeTraining) Cancel(
	_ context.Context,
	req *workertrainingservice.StatusRequest,
) (*worker.WorkerTrainingRecord, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.cancelled = req
	return f.record, nil
}

func TestAssignWorkerTraining_RoutesOnePairToAssign(t *testing.T) {
	t.Parallel()

	training := newFakeTraining()
	tool := newAssignWorkerTrainingTool(training)
	params := executeParams(map[string]any{
		paramTrainingWorkers: []any{training.record.WorkerID.String()},
		paramCourseIDs:       []any{training.course.ID.String()},
		wfParamDueDate:       "2026-10-31",
	})

	preview := previewWithoutWrites(t, training.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Would assign 1 course(s)")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, training.assigned)
	assert.Nil(t, training.bulkAssigned)
	require.NotNil(t, training.assigned.DueAt)
	assert.Equal(t, []agent.EgressClass{agent.EgressDriverVisible}, tool.Policy().Egress)
	assert.Equal(t, permission.OpAssign, tool.Policy().Operation)
}

func TestAssignWorkerTraining_SendsSeveralToBulkAssign(t *testing.T) {
	t.Parallel()

	training := newFakeTraining()
	tool := newAssignWorkerTrainingTool(training)
	params := executeParams(map[string]any{
		paramTrainingWorkers: []any{
			pulid.MustNew("wrk_").String(),
			pulid.MustNew("wrk_").String(),
		},
		paramCourseIDs: []any{training.course.ID.String()},
	})

	preview := previewWithoutWrites(t, training.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Len(t, preview.Changes, 2)
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, training.bulkAssigned)
	assert.Len(t, training.bulkAssigned.WorkerIDs, 2)

	training.failed = []workertrainingservice.BulkAssignOutcome{{Error: "Course is retired"}}
	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(), params),
		"a run that assigns nothing is refused")
}

func TestAssignRequiredWorkerTraining_CountsTheMissingCourses(t *testing.T) {
	t.Parallel()

	training := newFakeTraining()
	tool := newAssignRequiredWorkerTrainingTool(training)
	params := executeParams(map[string]any{paramWorkerID: training.record.WorkerID.String()})

	preview := previewWithoutWrites(t, training.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "1 required course(s)")
	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, []pulid.ID{training.record.WorkerID}, training.required)
}

func TestRecordTrainingCompletion_SaysWhetherItPassed(t *testing.T) {
	t.Parallel()

	training := newFakeTraining()
	tool := newRecordTrainingCompletionTool(training)
	params := executeParams(map[string]any{
		paramTrainingID: training.record.ID.String(),
		"completedAt":   "2026-09-18",
		paramScore:      "72",
	})

	preview := previewWithoutWrites(t, training.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "a fail below the passing mark")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, training.completed)
	assert.True(t, training.completed.Score.Valid)
	assert.Equal(t, time.Date(2026, time.September, 18, 0, 0, 0, 0, time.UTC).Unix(),
		training.completed.CompletedAt)

	direct := executeParams(map[string]any{
		paramWorkerID: training.record.WorkerID.String(),
		paramCourseID: training.course.ID.String(),
	})
	preview = previewWithoutWrites(t, training.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), direct)
	})
	assert.Equal(t, agent.PreviewOperationCreate, previewChange(t, preview, 0).Operation)

	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
		executeParams(map[string]any{paramWorkerID: training.record.WorkerID.String()})))
}

func TestCloseWorkerTraining_WaiveNeedsAReason(t *testing.T) {
	t.Parallel()

	training := newFakeTraining()
	tool := newCloseWorkerTrainingTool(training)
	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
		executeParams(map[string]any{
			paramTrainingID:    training.record.ID.String(),
			paramTrainingClose: "Waive",
		})))

	require.NoError(t, tool.Execute(t.Context(), executeParams(map[string]any{
		paramTrainingID:    training.record.ID.String(),
		paramTrainingClose: "Waive",
		fieldReason:        "Equivalent certificate on file",
	})))
	require.NotNil(t, training.waived)
	assert.Nil(t, training.cancelled)

	require.NoError(t, tool.Execute(t.Context(), executeParams(map[string]any{
		paramTrainingID:    training.record.ID.String(),
		paramTrainingClose: "Cancel",
	})))
	require.NotNil(t, training.cancelled)
	assert.Equal(t, agent.TierPropose, tool.Policy().MaxTier)
}
