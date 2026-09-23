package schedule

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/workflow"
)

type argsTestInput struct {
	Limit int
}

func contextWorkflowWithInput(workflow.Context, argsTestInput) error {
	return nil
}

func contextWorkflowWithoutInput(workflow.Context) error {
	return nil
}

func variadicWorkflow(workflow.Context, string, ...argsTestInput) error {
	return nil
}

func dummyWorkflow() error {
	return nil
}

func TestCron(t *testing.T) {
	spec := Cron("0 9 * * *")

	assert.Equal(t, "0 9 * * *", spec.Cron)
	assert.Equal(t, "UTC", spec.Timezone)
	assert.True(t, spec.IsCron())
	assert.False(t, spec.IsInterval())
}

func TestEvery(t *testing.T) {
	spec := Every(30 * time.Minute)

	assert.Equal(t, 30*time.Minute, spec.Interval)
	assert.Equal(t, "UTC", spec.Timezone)
	assert.True(t, spec.IsInterval())
	assert.False(t, spec.IsCron())
}

func TestSpec_WithTimezone(t *testing.T) {
	spec := Cron("0 9 * * *").WithTimezone("America/New_York")

	assert.Equal(t, "America/New_York", spec.Timezone)
}

func TestSpec_WithJitter(t *testing.T) {
	spec := Every(1 * time.Hour).WithJitter(5 * time.Minute)

	assert.Equal(t, 5*time.Minute, spec.Jitter)
}

func TestSpec_WithStartAt(t *testing.T) {
	startTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	spec := Cron("0 0 * * *").WithStartAt(startTime)

	require.NotNil(t, spec.StartAt)
	assert.Equal(t, startTime, *spec.StartAt)
}

func TestSpec_WithEndAt(t *testing.T) {
	endTime := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)
	spec := Cron("0 0 * * *").WithEndAt(endTime)

	require.NotNil(t, spec.EndAt)
	assert.Equal(t, endTime, *spec.EndAt)
}

func TestSchedule_Validate_Valid(t *testing.T) {
	tests := []struct {
		name     string
		schedule *Schedule
	}{
		{
			name: "valid cron schedule",
			schedule: &Schedule{
				ID:        "test-schedule",
				Spec:      Cron("0 9 * * *"),
				Workflow:  dummyWorkflow,
				TaskQueue: "test-queue",
			},
		},
		{
			name: "valid interval schedule",
			schedule: &Schedule{
				ID:        "test-schedule",
				Spec:      Every(30 * time.Minute),
				Workflow:  dummyWorkflow,
				TaskQueue: "test-queue",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.schedule.Validate()
			assert.NoError(t, err)
		})
	}
}

func TestSchedule_Validate_Invalid(t *testing.T) {
	tests := []struct {
		name        string
		schedule    *Schedule
		expectedErr error
	}{
		{
			name: "missing ID",
			schedule: &Schedule{
				Spec:      Cron("0 9 * * *"),
				Workflow:  dummyWorkflow,
				TaskQueue: "test-queue",
			},
			expectedErr: ErrScheduleIDRequired,
		},
		{
			name: "missing workflow",
			schedule: &Schedule{
				ID:        "test-schedule",
				Spec:      Cron("0 9 * * *"),
				TaskQueue: "test-queue",
			},
			expectedErr: ErrWorkflowRequired,
		},
		{
			name: "missing task queue",
			schedule: &Schedule{
				ID:       "test-schedule",
				Spec:     Cron("0 9 * * *"),
				Workflow: dummyWorkflow,
			},
			expectedErr: ErrTaskQueueRequired,
		},
		{
			name: "missing spec",
			schedule: &Schedule{
				ID:        "test-schedule",
				Workflow:  dummyWorkflow,
				TaskQueue: "test-queue",
			},
			expectedErr: ErrInvalidScheduleSpec,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.schedule.Validate()
			assert.ErrorIs(t, err, tt.expectedErr)
		})
	}
}

func TestSchedule_Validate_Args(t *testing.T) {
	t.Parallel()

	base := func(wf any, args ...any) *Schedule {
		return &Schedule{
			ID:        "args-schedule",
			Spec:      Every(time.Minute),
			Workflow:  wf,
			TaskQueue: "test-queue",
			Args:      args,
		}
	}

	tests := []struct {
		name    string
		sched   *Schedule
		wantErr bool
	}{
		{
			name:  "context and input with one arg",
			sched: base(contextWorkflowWithInput, argsTestInput{}),
		},
		{
			name:    "context and input without args",
			sched:   base(contextWorkflowWithInput),
			wantErr: true,
		},
		{
			name:    "context and input with extra args",
			sched:   base(contextWorkflowWithInput, argsTestInput{}, argsTestInput{}),
			wantErr: true,
		},
		{name: "context only without args", sched: base(contextWorkflowWithoutInput)},
		{
			name:    "context only with args",
			sched:   base(contextWorkflowWithoutInput, 1),
			wantErr: true,
		},
		{name: "no context without args", sched: base(dummyWorkflow)},
		{name: "variadic with required arg", sched: base(variadicWorkflow, "x")},
		{
			name:  "variadic with extra args",
			sched: base(variadicWorkflow, "x", argsTestInput{}, argsTestInput{}),
		},
		{name: "variadic missing required arg", sched: base(variadicWorkflow), wantErr: true},
		{name: "workflow referenced by name", sched: base("NamedWorkflow")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.sched.Validate()
			if tt.wantErr {
				require.ErrorIs(t, err, ErrWorkflowArgsInvalid)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestSchedule_Hash(t *testing.T) {
	schedule1 := &Schedule{
		ID:        "test-schedule",
		Spec:      Cron("0 9 * * *"),
		Workflow:  dummyWorkflow,
		TaskQueue: "test-queue",
	}

	schedule2 := &Schedule{
		ID:        "test-schedule",
		Spec:      Cron("0 9 * * *"),
		Workflow:  dummyWorkflow,
		TaskQueue: "test-queue",
	}

	schedule3 := &Schedule{
		ID:        "test-schedule",
		Spec:      Cron("0 10 * * *"),
		Workflow:  dummyWorkflow,
		TaskQueue: "test-queue",
	}

	assert.Equal(t, schedule1.Hash(), schedule2.Hash())
	assert.NotEqual(t, schedule1.Hash(), schedule3.Hash())
	assert.Len(t, schedule1.Hash(), 16)
}

func TestSchedule_ToScheduleOptions(t *testing.T) {
	schedule := &Schedule{
		ID:            "test-schedule",
		Description:   "Test schedule",
		Spec:          Every(30 * time.Minute).WithTimezone("America/Chicago"),
		Workflow:      dummyWorkflow,
		TaskQueue:     "test-queue",
		OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_BUFFER_ONE,
		Memo: map[string]any{
			"custom": "value",
		},
	}

	opts := schedule.ToScheduleOptions()

	assert.Equal(t, "test-schedule", opts.ID)
	assert.Equal(t, "America/Chicago", opts.Spec.TimeZoneName)
	assert.Len(t, opts.Spec.Intervals, 1)
	assert.Equal(t, 30*time.Minute, opts.Spec.Intervals[0].Every)
	assert.Equal(t, enums.SCHEDULE_OVERLAP_POLICY_BUFFER_ONE, opts.Overlap)
}

func TestSchedule_ToScheduleOptions_DefaultOverlapPolicy(t *testing.T) {
	schedule := &Schedule{
		ID:        "test-schedule",
		Spec:      Every(30 * time.Minute),
		Workflow:  dummyWorkflow,
		TaskQueue: "test-queue",
	}

	opts := schedule.ToScheduleOptions()

	assert.Equal(t, enums.SCHEDULE_OVERLAP_POLICY_SKIP, opts.Overlap)
}
