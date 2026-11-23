package desktranscriptjobs

import (
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/desktranscriptports"
	"github.com/emoss08/trenova/internal/modules/workspace/jobs/reportjobs"
	"github.com/emoss08/trenova/internal/testutil/entitlementtest"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/enums/v1"
)

func functionName(fn any) string {
	full := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()

	return full[strings.LastIndex(full, ".")+1:]
}

func TestTheExportWorkflowIsRegisteredUnderTheNameTheServiceStarts(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		desktranscriptports.ExportWorkflowName,
		functionName(DeskTranscriptExportWorkflow),
	)
	for _, wf := range RegisterWorkflows() {
		assert.Equal(t, wf.Name, functionName(wf.Fn))
		assert.Equal(t, temporaltype.ReportTaskQueue, wf.TaskQueue)
		assert.NotEmpty(t, wf.Description)
	}
}

func TestTheExportWorkerSharesTheReportQueuesOptions(t *testing.T) {
	t.Parallel()

	assert.Equal(t, temporaltype.ReportTaskQueue, DomainConfig.TaskQueue)
	assert.Equal(t, reportjobs.DomainConfig.WorkerConfig, DomainConfig.WorkerConfig)
}

func TestExpiredExportsAreCleanedUpHourly(t *testing.T) {
	t.Parallel()

	schedules := NewScheduleProvider().GetSchedules()
	require.Len(t, schedules, 1)
	require.NoError(t, schedules[0].Validate())
	assert.Equal(t, CleanupScheduleID, schedules[0].ID)
	assert.Equal(t, cleanupEvery, schedules[0].Spec.Interval)
	assert.Equal(t, enums.SCHEDULE_OVERLAP_POLICY_SKIP, schedules[0].OverlapPolicy)
}

func TestSchedulesServeOnlyEntitledTenants(t *testing.T) {
	t.Parallel()

	entitlementtest.RunSchedules(t, entitlementtest.Package{
		Schedules:  NewScheduleProvider().GetSchedules(),
		Activities: []any{&Activities{}},
		Workflows:  convertWorkflows(RegisterWorkflows()),
		Shapes: map[string]entitlementtest.Shape{
			CleanupScheduleID: entitlementtest.InOneActivity,
		},
	})
}
