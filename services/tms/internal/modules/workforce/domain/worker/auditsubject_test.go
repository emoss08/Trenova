package worker_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/dotclearinghouse"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/employment"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/profilechange"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/pto"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/timesheet"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerchecklist"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercredential"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerdiscipline"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerdot"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerinjury"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerleave"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerreview"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workersafety"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerschedule"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workertraining"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

type subjectOwner interface {
	AuditSubject() (permission.Resource, pulid.ID)
}

func TestAuditSubject_EveryRecordOfAWorkersFileNamesTheWorker(t *testing.T) {
	t.Parallel()

	workerID := pulid.MustNew("wrk_")
	cases := []struct {
		name   string
		record subjectOwner
	}{
		{
			"workercredential.WorkerCredential",
			&workercredential.WorkerCredential{WorkerID: workerID},
		},
		{"employment.WorkerEmploymentEvent", &employment.WorkerEmploymentEvent{WorkerID: workerID}},
		{
			"employment.WorkerEmploymentVerification",
			&employment.WorkerEmploymentVerification{WorkerID: workerID},
		},
		{
			"workertraining.WorkerTrainingRecord",
			&workertraining.WorkerTrainingRecord{WorkerID: workerID},
		},
		{"workersafety.WorkerSafetyEvent", &workersafety.WorkerSafetyEvent{WorkerID: workerID}},
		{
			"workerdiscipline.WorkerDisciplinaryAction",
			&workerdiscipline.WorkerDisciplinaryAction{WorkerID: workerID},
		},
		{
			"workerdiscipline.WorkerRecognition",
			&workerdiscipline.WorkerRecognition{WorkerID: workerID},
		},
		{"workerinjury.WorkerInjury", &workerinjury.WorkerInjury{WorkerID: workerID}},
		{"workerchecklist.WorkerChecklist", &workerchecklist.WorkerChecklist{WorkerID: workerID}},
		{"workerreview.PerformanceReview", &workerreview.PerformanceReview{WorkerID: workerID}},
		{"workerleave.WorkerLeaveCase", &workerleave.WorkerLeaveCase{WorkerID: workerID}},
		{"workerleave.WorkerLeaveEntry", &workerleave.WorkerLeaveEntry{WorkerID: workerID}},
		{"workerdot.WorkerDOTTest", &workerdot.WorkerDOTTest{WorkerID: workerID}},
		{"workerdot.WorkerDOTViolation", &workerdot.WorkerDOTViolation{WorkerID: workerID}},
		{
			"dotclearinghouse.WorkerClearinghouseQuery",
			&dotclearinghouse.WorkerClearinghouseQuery{WorkerID: workerID},
		},
		{
			"workerschedule.WorkerShiftAssignment",
			&workerschedule.WorkerShiftAssignment{WorkerID: workerID},
		},
		{
			"workerschedule.WorkerAvailabilityPreference",
			&workerschedule.WorkerAvailabilityPreference{WorkerID: workerID},
		},
		{
			"profilechange.WorkerProfileChangeRequest",
			&profilechange.WorkerProfileChangeRequest{WorkerID: workerID},
		},
		{"pto.WorkerPTOPolicyAssignment", &pto.WorkerPTOPolicyAssignment{WorkerID: workerID}},
		{"pto.WorkerPTOLedgerEntry", &pto.WorkerPTOLedgerEntry{WorkerID: workerID}},
		{"timesheet.Timesheet", &timesheet.Timesheet{WorkerID: workerID}},
		{"timesheet.TimeClockEntry", &timesheet.TimeClockEntry{WorkerID: workerID}},
		{"worker.WorkerPTO", &worker.WorkerPTO{WorkerID: workerID}},
	}
	for _, tc := range cases {
		resource, id := tc.record.AuditSubject()
		assert.Equal(t, permission.ResourceWorker, resource, tc.name)
		assert.Equal(t, workerID, id, tc.name)
	}
}

func TestAuditSubject_TheWorkerIsItsOwnSubject(t *testing.T) {
	t.Parallel()

	wrk := &worker.Worker{ID: pulid.MustNew("wrk_")}
	resource, id := wrk.AuditSubject()
	assert.Equal(t, permission.ResourceWorker, resource)
	assert.Equal(t, wrk.ID, id)
}

func TestAuditSubject_ANilRecordNamesNoSubject(t *testing.T) {
	t.Parallel()

	for _, record := range []subjectOwner{
		(*worker.Worker)(nil),
		(*workercredential.WorkerCredential)(nil),
		(*employment.WorkerEmploymentEvent)(nil),
		(*employment.WorkerEmploymentVerification)(nil),
		(*workertraining.WorkerTrainingRecord)(nil),
		(*workersafety.WorkerSafetyEvent)(nil),
		(*workerdiscipline.WorkerDisciplinaryAction)(nil),
		(*workerdiscipline.WorkerRecognition)(nil),
		(*workerinjury.WorkerInjury)(nil),
		(*workerchecklist.WorkerChecklist)(nil),
		(*workerreview.PerformanceReview)(nil),
		(*workerleave.WorkerLeaveCase)(nil),
		(*workerleave.WorkerLeaveEntry)(nil),
		(*workerdot.WorkerDOTTest)(nil),
		(*workerdot.WorkerDOTViolation)(nil),
		(*dotclearinghouse.WorkerClearinghouseQuery)(nil),
		(*workerschedule.WorkerShiftAssignment)(nil),
		(*workerschedule.WorkerAvailabilityPreference)(nil),
		(*profilechange.WorkerProfileChangeRequest)(nil),
		(*pto.WorkerPTOPolicyAssignment)(nil),
		(*pto.WorkerPTOLedgerEntry)(nil),
		(*timesheet.Timesheet)(nil),
		(*timesheet.TimeClockEntry)(nil),
		(*worker.WorkerPTO)(nil),
	} {
		resource, id := record.AuditSubject()
		assert.Empty(t, resource)
		assert.True(t, id.IsNil())
	}
}
