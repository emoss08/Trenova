package workerquerytools

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/gettool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/graphqlread"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/readspec"
	"github.com/emoss08/trenova/internal/modules/agent/ports/graphqlreadports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func workerGetTool(
	t *testing.T,
	spec readspec.Get,
	reader graphqlreadports.GraphQLReader,
) *gettool.GraphQLTool {
	t.Helper()

	tool, err := gettool.NewGraphQL(&spec, reader, &mocks.DenyResourcesPermissionEngine{})
	require.NoError(t, err)

	return tool
}

func TestWorkerGet_ReadsTheCardAndFoldsComplianceIntoAssignability(t *testing.T) {
	t.Parallel()

	workerID := pulid.MustNew("wrk_").String()
	reader := &querytooltest.KeyedReader{Answers: map[string]*graphqlreadports.GraphQLReadResult{
		"DeskWorkerCard": {
			Data: map[string]any{"worker": map[string]any{
				"id": workerID, "firstName": "Sarah", "lastName": "Williams",
				"canBeAssigned": true, "assignmentBlocked": "",
				"profile": map[string]any{
					"complianceStatus": "NonCompliant", "licenseExpiry": 1_800_000_000,
					"eldExempt": false,
				},
			}},
			Withheld: []string{"worker.email", "worker.profile.hazmatExpiry"},
		},
	}}
	tool := workerGetTool(t, GetSpec(), reader)

	data, err := tool.Query(
		t.Context(),
		querytooltest.KeyedCall(map[string]any{querytool.ParamWorkerID: workerID}),
	)
	require.NoError(t, err)

	assert.Equal(t, map[string]any{"id": workerID}, reader.Sent("DeskWorkerCard").Variables)
	record, ok := data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, false, record["canBeAssigned"], "a NonCompliant driver cannot be assigned")
	assert.Equal(t, "Credentials are not compliant", record["assignmentBlocked"])
	profile, _ := record["profile"].(map[string]any)
	assert.Contains(t, profile, "medicalCardExpiry", "a date not on file is present and null")
	assert.Nil(t, profile["medicalCardExpiry"])
	assert.NotContains(t, profile, "hazmatExpiry", "a withheld date is not reported as missing")
	assert.Equal(t, []string{"email", "profile.hazmatExpiry"}, record[graphqlread.OutcomeWithheld])
}

// The card selects what the repository row carried, so the conversion loses
// no field a reader whose access reaches it was shown.
func TestWorkerGet_TheCardSelectsWhatTheRepositoryRowCarried(t *testing.T) {
	t.Parallel()

	selected := func(fields []graphqlread.Field) map[string]bool {
		out := make(map[string]bool, len(fields))
		for _, field := range fields {
			out[field.Name] = field.Selected
		}
		return out
	}
	read := graphqlread.Reads[toolWorkerGet]
	own := selected(read.Fields)
	for _, name := range []string{
		"externalId", "city", "postalCode", "email", "phoneNumber", "emergencyContactName",
		"emergencyContactPhone", "canBeAssigned", "assignmentBlocked",
		"availableForDispatch", "leaveType", "fleetCode", "state",
	} {
		assert.True(t, own[name], "worker_get selects %s", name)
	}
	var profile []graphqlread.Field
	for _, field := range read.Fields {
		if field.Name == workerProfileField {
			profile = field.Fields
		}
	}
	nested := selected(profile)
	for _, name := range []string{
		"cdlClass", "cdlRestrictions", "endorsement", "complianceStatus", "isQualified",
		"hazmatExpiry", "licenseExpiry", "medicalCardExpiry", "twicExpiry", "physicalDueDate",
		"mvrDueDate", "lastMvrCheck", "nextTrainingDue", "hireDate", "terminationDate",
		"disqualificationReason", "safetyRating", "eldExempt", "shortHaulExempt",
	} {
		assert.True(t, nested[name], "worker_get selects profile.%s", name)
	}
	for _, name := range []string{"dob", "twicCardNumber", "drugAlcoholStatus", "lastDrugTest"} {
		assert.False(t, nested[name], "worker_get never selects the Confidential %s", name)
	}
}

func TestWorkerGetHOS_ReadsTheClocksByTheWorkerInMinutes(t *testing.T) {
	t.Parallel()

	workerID := pulid.MustNew("wrk_").String()
	recordedAt := timeutils.NowUnix() - 600
	reader := &querytooltest.KeyedReader{Answers: map[string]*graphqlreadports.GraphQLReadResult{
		"WorkerHosState": {Data: map[string]any{"workerHosState": map[string]any{
			"workerId": workerID, "workerName": "Maria Ortiz", "dutyStatus": "Driving",
			"driveRemainingMs": float64(90 * 60_000), "shiftRemainingMs": float64(5 * 3_600_000),
			"cycleRemainingMs": float64(0), "driveLimitMs": float64(11 * 3_600_000),
			"recordedAt": float64(recordedAt),
		}}},
	}}
	tool := workerGetTool(t, HOSGetSpec(), reader)

	data, err := tool.Query(
		t.Context(),
		querytooltest.KeyedCall(map[string]any{querytool.ParamWorkerID: workerID}),
	)
	require.NoError(t, err)

	assert.Equal(t, map[string]any{querytool.ParamWorkerID: workerID},
		reader.Sent("WorkerHosState").Variables, "the clocks are read by the worker's id")
	record, ok := data.(map[string]any)
	require.True(t, ok)
	assert.EqualValues(t, 90, record["driveRemainingMinutes"])
	assert.EqualValues(t, 300, record["shiftRemainingMinutes"])
	assert.EqualValues(t, 0, record["cycleRemainingMinutes"], "a spent clock is zero, not absent")
	assert.EqualValues(t, 660, record["driveLimitMinutes"])
	assert.NotContains(t, record, "driveRemainingMs")
	assert.Equal(t, false, record[hosStale])
	assert.Equal(t, timeutils.FormatUnixDateTimeIn(recordedAt, "America/Chicago"),
		record[hosRecordedAt])
	assert.NotContains(t, record, graphqlread.OutcomeNote)
}

func TestWorkerGetHOS_MarksAStaleReading(t *testing.T) {
	t.Parallel()

	reader := &querytooltest.KeyedReader{Answers: map[string]*graphqlreadports.GraphQLReadResult{
		"WorkerHosState": {Data: map[string]any{"workerHosState": map[string]any{
			"dutyStatus": "OffDuty", "driveRemainingMs": float64(60_000),
			"recordedAt": float64(1),
		}}},
	}}
	tool := workerGetTool(t, HOSGetSpec(), reader)

	data, err := tool.Query(
		t.Context(),
		querytooltest.KeyedCall(
			map[string]any{querytool.ParamWorkerID: pulid.MustNew("wrk_").String()},
		),
	)
	require.NoError(t, err)

	record, _ := data.(map[string]any)
	assert.Equal(t, true, record[hosStale])
	assert.Contains(t, record[graphqlread.OutcomeNote], "more than twelve hours old")
	assert.Contains(t, record[graphqlread.OutcomeNote], "do not stop")
}

/*
gpt-6-luna read "plan on nothing" and zeros on every clock as "no hours left"
and proposed no assignment for a driver the person named. A driver no ELD has
reported is not a missing record: the clocks are left out, and the note says
unreported hours are a concern to name beside the proposal, not a reason to
withhold it.
*/
func TestWorkerGetHOS_ADriverNoELDReportedIsUnknownNotMissing(t *testing.T) {
	t.Parallel()

	workerID := pulid.MustNew("wrk_").String()
	reader := &querytooltest.KeyedReader{Answers: map[string]*graphqlreadports.GraphQLReadResult{
		"WorkerHosState": {Data: map[string]any{"workerHosState": nil}},
	}}
	tool := workerGetTool(t, HOSGetSpec(), reader)

	data, err := tool.Query(
		t.Context(),
		querytooltest.KeyedCall(map[string]any{querytool.ParamWorkerID: workerID}),
	)
	require.NoError(t, err)

	record, _ := data.(map[string]any)
	assert.Equal(t, workerID, record[querytool.ParamWorkerID])
	assert.Equal(t, "Unknown", record[hosDutyStatus])
	assert.Equal(t, true, record[hosStale])
	assert.NotContains(t, record, "driveRemainingMinutes", "an unknown clock is absent, not zero")
	assert.Contains(t, record[graphqlread.OutcomeNote], "No ELD has reported hours of service")
	assert.Contains(t, record[graphqlread.OutcomeNote], "do not stop")
	assert.NotContains(t, record[graphqlread.OutcomeNote], "plan on nothing")
}

func TestWorkerGetDOTRandomDraw_ReadsTheRoundWithItsSelections(t *testing.T) {
	t.Parallel()

	drawID := pulid.MustNew("drdraw_").String()
	reader := &querytooltest.KeyedReader{Answers: map[string]*graphqlreadports.GraphQLReadResult{
		"DotRandomDraw": {Data: map[string]any{"dotRandomDraw": map[string]any{
			"id": drawID, "periodKey": "2026-Q3", "status": "Final",
			"entries": []any{map[string]any{
				"id": pulid.MustNew("drde_").String(), "substance": "Alcohol", "rank": 1,
				"status": "Selected", "worker": map[string]any{"firstName": "Maria"},
			}},
		}}},
	}}
	tool := workerGetTool(t, DOTRandomDrawGetSpec(), reader)

	data, err := tool.Query(
		t.Context(),
		querytooltest.KeyedCall(map[string]any{paramDrawID: drawID}),
	)
	require.NoError(t, err)

	assert.Equal(t, map[string]any{"id": drawID}, reader.Sent("DotRandomDraw").Variables)
	record, _ := data.(map[string]any)
	entries, _ := record["entries"].([]any)
	require.Len(t, entries, 1)
	assert.Equal(t, permission.ResourceDOTRandomPool, tool.Policy().Resource)
	property, _ := tool.ParamSchema()["properties"].(map[string]any)[paramDrawID].(map[string]any)
	assert.NotEmpty(t, property, "the round's id is held to the draw's prefix")
}

func TestWorkforceGets_ReadThroughTheirScreensOperations(t *testing.T) {
	t.Parallel()

	for tool, operation := range map[string]string{
		toolWorkerGet:           "DeskWorkerCard",
		ToolWorkerGetCredential: "DeskWorkerCredential",
		toolWorkerGetDOTDraw:    "DotRandomDraw",
		toolWorkerGetHOS:        "WorkerHosState",
	} {
		read, ok := graphqlread.Reads[tool]
		require.True(t, ok, tool)
		assert.Equal(t, graphqlread.VerbGet, read.Verb, tool)
		assert.Equal(t, operation, read.Operation, tool)
	}
	hours, card := graphqlread.Reads[toolWorkerGetHOS], graphqlread.Reads[toolWorkerGet]
	assert.Equal(t, querytool.ParamWorkerID, hours.GetKey())
	assert.Equal(t, graphqlread.IDVariable, card.GetKey())
}
