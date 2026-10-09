package agenttoolservice

import (
	"testing"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
A model that found the open assignment with list_worker_training, filtered to
the worker and the course, sent the assignment's id with the worker and the
course it had filtered by, and the completion was refused for naming the
assignment both ways. All three named the same record, so it is recorded.
*/
func TestRecordTrainingCompletion_TakesTheAssignmentsOwnWorkerAndCourseBesideIt(t *testing.T) {
	t.Parallel()

	training := newFakeTraining()
	tool := newRecordTrainingCompletionTool(training)
	params := executeParams(map[string]any{
		paramTrainingID: training.record.ID.String(),
		paramWorkerID:   training.record.WorkerID.String(),
		paramCourseID:   training.record.CourseID.String(),
	})

	require.NoError(t, tool.(serviceports.ToolValidator).Validate(t.Context(), params))
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, training.completed)
	assert.Equal(t, training.record.ID, training.completed.ID)
}

func TestRecordTrainingCompletion_ARefusalForAnotherWorkersAssignmentShowsBothCalls(t *testing.T) {
	t.Parallel()

	training := newFakeTraining()
	tool := newRecordTrainingCompletionTool(training)
	params := executeParams(map[string]any{
		paramTrainingID: training.record.ID.String(),
		paramWorkerID:   pulid.MustNew("wrk_").String(),
		paramCourseID:   training.record.CourseID.String(),
	})

	err := tool.(serviceports.ToolValidator).Validate(t.Context(), params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "send trainingId without workerId and courseId")
	assert.Contains(t, err.Error(), "send workerId with courseId and no trainingId")

	require.Error(t, tool.Execute(t.Context(), params))
	assert.Nil(t, training.completed, "a completion never lands on another worker's record")
}
