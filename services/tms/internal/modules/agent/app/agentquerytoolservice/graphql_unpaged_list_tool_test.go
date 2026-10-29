package agentquerytoolservice

import (
	"strconv"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/fieldgate"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/graphqllisttool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/graphqlread"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/readspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/workerquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/graphqlreadports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func keyedSpec(t *testing.T, name string) *readspec.List {
	t.Helper()

	return querytooltest.ListSpecNamed(t, graphQLListSpecs(), name)
}

func checklistAnswer() *graphqlreadports.GraphQLReadResult {
	return &graphqlreadports.GraphQLReadResult{
		Data: map[string]any{"workerChecklists": []any{map[string]any{
			"id": "wcl_1", "name": "Onboarding", "status": "Open", "__typename": "WorkerChecklist",
			"items": []any{map[string]any{"id": "wcli_1", "label": "Sign handbook", "note": nil}},
		}}},
		Withheld: []string{"workerChecklists.items.note"},
	}
}

func templateAnswer(more bool) *graphqlreadports.GraphQLReadResult {
	return &graphqlreadports.GraphQLReadResult{Data: map[string]any{
		"workerChecklistTemplates": map[string]any{
			"edges": []any{map[string]any{"node": map[string]any{
				"id": "wclt_1", "code": "ONB", "name": "Onboarding",
			}}},
			"pageInfo": map[string]any{"hasNextPage": more, "endCursor": "c1"},
		},
	}}
}

func TestUnpagedList_SendsTheWorkerAndReadsOpenOnesByDefault(t *testing.T) {
	t.Parallel()

	reader := &querytooltest.KeyedReader{Answers: map[string]*graphqlreadports.GraphQLReadResult{
		"DeskWorkerChecklists":         checklistAnswer(),
		"DeskWorkerChecklistTemplates": templateAnswer(false),
	}}
	tool, err := graphqllisttool.NewUnpaged(
		keyedSpec(t, workerquerytools.ToolListWorkerChecklists),
		reader,
		&mocks.DenyResourcesPermissionEngine{},
	)
	require.NoError(t, err)

	workerID := pulid.MustNew("wrk_").String()
	data, err := tool.Query(
		t.Context(),
		querytooltest.KeyedCall(map[string]any{querytool.ParamWorkerID: workerID}),
	)
	require.NoError(t, err)

	sent := reader.Sent("DeskWorkerChecklists")
	require.NotNil(t, sent)
	assert.Equal(t, map[string]any{"workerId": workerID, "includeClosed": false}, sent.Variables)

	outcome, ok := data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 1, outcome["count"])
	items, _ := outcome["items"].([]map[string]any)
	require.Len(t, items, 1)
	assert.NotContains(t, items[0], "__typename")
	lines, _ := items[0]["items"].([]any)
	require.Len(t, lines, 1)
	assert.NotContains(t, lines[0], "note", "a withheld value is never carried, even as null")
	assert.Equal(t, []string{"items.note"}, outcome["withheldByAccess"])
	assert.Equal(t, fieldgate.WithheldNote, outcome["withheldNote"])
}

func TestUnpagedList_ReadsTheTemplatesAPersonStartsByHand(t *testing.T) {
	t.Parallel()

	reader := &querytooltest.KeyedReader{Answers: map[string]*graphqlreadports.GraphQLReadResult{
		"DeskWorkerChecklists":         checklistAnswer(),
		"DeskWorkerChecklistTemplates": templateAnswer(true),
	}}
	tool, err := graphqllisttool.NewUnpaged(
		keyedSpec(t, workerquerytools.ToolListWorkerChecklists),
		reader,
		&mocks.DenyResourcesPermissionEngine{},
	)
	require.NoError(t, err)

	data, err := tool.Query(t.Context(), querytooltest.KeyedCall(map[string]any{
		querytool.ParamWorkerID: pulid.MustNew("wrk_").String(),
	}))
	require.NoError(t, err)

	sent := reader.Sent("DeskWorkerChecklistTemplates")
	require.NotNil(t, sent)
	assert.Equal(t, map[string]any{"input": map[string]any{
		"status": "Active", "trigger": "Manual", "first": graphqlread.AlongsideMost,
	}}, sent.Variables)

	outcome, _ := data.(map[string]any)
	templates, _ := outcome["templatesStartedByHand"].([]map[string]any)
	require.Len(t, templates, 1)
	assert.Equal(t, "wclt_1", templates[0]["id"])
	assert.Contains(t, outcome["note"], "Only the first 100 checklist templates")
}

func TestUnpagedList_LeavesOutTheTemplatesAReaderCannotRead(t *testing.T) {
	t.Parallel()

	reader := &querytooltest.KeyedReader{Answers: map[string]*graphqlreadports.GraphQLReadResult{
		"DeskWorkerChecklists": checklistAnswer(),
	}}
	engine := &mocks.DenyResourcesPermissionEngine{
		Denied: []string{permission.ResourceWorkerChecklistTemplate.String()},
	}
	tool, err := graphqllisttool.NewUnpaged(
		keyedSpec(t, workerquerytools.ToolListWorkerChecklists),
		reader,
		engine,
	)
	require.NoError(t, err)

	data, err := tool.Query(t.Context(), querytooltest.KeyedCall(map[string]any{
		querytool.ParamWorkerID: pulid.MustNew("wrk_").String(),
	}))
	require.NoError(t, err)

	assert.Nil(
		t,
		reader.Sent("DeskWorkerChecklistTemplates"),
		"nothing is read that would be refused",
	)
	outcome, _ := data.(map[string]any)
	assert.NotContains(t, outcome, "templatesStartedByHand")
}

func TestUnpagedList_RefusesACallWithoutItsWorkerOrWithAName(t *testing.T) {
	t.Parallel()

	reader := &querytooltest.KeyedReader{}
	tool, err := graphqllisttool.NewUnpaged(
		keyedSpec(t, workerquerytools.ToolListWorkerTraining),
		reader,
		&mocks.DenyResourcesPermissionEngine{},
	)
	require.NoError(t, err)

	_, err = tool.Query(t.Context(), querytooltest.KeyedCall(map[string]any{}))
	require.ErrorContains(t, err, `"workerId" is required`)

	_, err = tool.Query(
		t.Context(),
		querytooltest.KeyedCall(map[string]any{querytool.ParamWorkerID: "Jane Smith"}),
	)
	require.Error(t, err)
	assert.Empty(t, reader.Requests, "a refused call reads nothing")
}

func TestUnpagedList_TakesTheWorkerAndTheSwitchAsItsParameters(t *testing.T) {
	t.Parallel()

	tool, err := graphqllisttool.NewUnpaged(
		keyedSpec(t, workerquerytools.ToolListWorkerTraining),
		&querytooltest.KeyedReader{},
		&mocks.DenyResourcesPermissionEngine{},
	)
	require.NoError(t, err)

	schema := tool.ParamSchema()
	assert.Equal(t, []string{querytool.ParamWorkerID}, schema["required"])
	properties, _ := schema["properties"].(map[string]any)
	assert.Len(t, properties, 2)
	closed, _ := properties[querytool.WfParamClosed].(map[string]any)
	assert.Contains(t, closed["description"], "expired assignments")
	assert.Contains(t, closed["description"], workerquerytools.NoteOpenOnly)
	worker, _ := properties[querytool.ParamWorkerID].(map[string]any)
	assert.Equal(t, permission.ResourceWorker.String(), worker[toolschema.KeyRecordOf])
}

func TestUnpagedList_SpecsMatchWhatTheGeneratorProved(t *testing.T) {
	t.Parallel()

	spec := *keyedSpec(t, workerquerytools.ToolListWorkerChecklists)
	spec.Alongside = nil
	_, err := graphqllisttool.NewUnpaged(&spec, &querytooltest.KeyedReader{}, nil)
	require.ErrorContains(t, err, "reads.yml lists 1 reads alongside it, its spec 0")

	spec = *keyedSpec(t, workerquerytools.ToolListWorkerTraining)
	spec.Alongside = []readspec.Alongside{
		{Key: "templates", Operation: workerquerytools.OpDeskReviewTemplates},
	}
	_, err = graphqllisttool.NewUnpaged(&spec, &querytooltest.KeyedReader{}, nil)
	require.ErrorContains(t, err, "which reads.yml does not list for it")

	spec = *keyedSpec(t, workerquerytools.ToolListWorkerTraining)
	spec.References = nil
	_, err = graphqllisttool.NewUnpaged(&spec, &querytooltest.KeyedReader{}, nil)
	require.ErrorContains(t, err, "requires workerId, which no tool supplies")
}

func TestUnpagedList_ReadsTheWholeOrganizationWithAPageAndSaysWhenItIsFull(t *testing.T) {
	t.Parallel()

	rows := make([]any, 0, querytool.GraphQLListLimit)
	for idx := range querytool.GraphQLListLimit {
		rows = append(rows, map[string]any{"id": "wdt_" + strconv.Itoa(idx), "status": "Scheduled"})
	}
	reader := &querytooltest.KeyedReader{Answers: map[string]*graphqlreadports.GraphQLReadResult{
		"DeskWorkerDotTests": {Data: map[string]any{"workerDotTests": rows}},
	}}
	tool, err := graphqllisttool.NewUnpaged(
		keyedSpec(t, workerquerytools.ToolListWorkerDOTTests),
		reader,
		&mocks.DenyResourcesPermissionEngine{},
	)
	require.NoError(t, err)
	assert.NotContains(t, tool.ParamSchema(), "required", "the worker is optional")

	data, err := tool.Query(
		t.Context(),
		querytooltest.KeyedCall(map[string]any{querytool.WfParamOpenOnly: true}),
	)
	require.NoError(t, err)

	sent := reader.Sent("DeskWorkerDotTests")
	require.NotNil(t, sent)
	assert.Equal(
		t,
		map[string]any{"openOnly": true, "limit": querytool.GraphQLListLimit},
		sent.Variables,
	)
	outcome, _ := data.(map[string]any)
	assert.Contains(t, outcome["note"], "more may exist")
}

func TestUnpagedList_LeavesOutTheWorkerOfAPrivacyCase(t *testing.T) {
	t.Parallel()

	reader := &querytooltest.KeyedReader{Answers: map[string]*graphqlreadports.GraphQLReadResult{
		"DeskWorkerInjuries": {Data: map[string]any{"workerInjuries": []any{
			map[string]any{
				"id": "winj_1", "workerId": "wrk_1", "privacyCase": true, "logName": "Privacy Case",
				"worker": map[string]any{"id": "wrk_1", "wholeName": "Jane Smith"},
			},
			map[string]any{
				"id": "winj_2", "workerId": "wrk_2", "privacyCase": false,
				"worker": map[string]any{"id": "wrk_2", "wholeName": "Sam Lee"},
			},
		}}},
	}}
	tool, err := graphqllisttool.NewUnpaged(
		keyedSpec(t, workerquerytools.ToolListWorkerInjuries),
		reader,
		&mocks.DenyResourcesPermissionEngine{},
	)
	require.NoError(t, err)

	data, err := tool.Query(
		t.Context(),
		querytooltest.KeyedCall(map[string]any{"caseYear": float64(2026)}),
	)
	require.NoError(t, err)

	sent := reader.Sent("DeskWorkerInjuries")
	require.NotNil(t, sent)
	assert.Equal(t, float64(2026), sent.Variables["caseYear"])
	outcome, _ := data.(map[string]any)
	items, _ := outcome["items"].([]map[string]any)
	require.Len(t, items, 2)
	assert.NotContains(t, items[0], "worker", "a privacy case never carries its worker's name")
	assert.Equal(t, "wrk_1", items[0]["workerId"])
	assert.Contains(t, items[1], "worker")
}

func safetyAnswers() map[string]*graphqlreadports.GraphQLReadResult {
	return map[string]*graphqlreadports.GraphQLReadResult{
		"DeskWorkerSafetyEvents": {Data: map[string]any{"workerSafetyEvents": []any{
			map[string]any{
				"id": "wsev_1", "kind": "Inspection", "status": "Open",
				"violations": []any{map[string]any{"id": "wsvi_1", "code": "393.9"}},
			},
		}}},
		"DeskWorkerRecognitions": {
			Data: map[string]any{"workerRecognitions": []any{
				map[string]any{
					"id":         "wrec_1",
					"title":      "Ten years",
					"__typename": "WorkerRecognition",
				},
			}},
			Withheld: []string{"workerRecognitions.title"},
		},
	}
}

func TestUnpagedList_ReadsTheWorkersRecognitionsByTheWorkerTheCallNames(t *testing.T) {
	t.Parallel()

	reader := &querytooltest.KeyedReader{Answers: safetyAnswers()}
	tool, err := graphqllisttool.NewUnpaged(
		keyedSpec(t, workerquerytools.ToolListWorkerSafetyEvents),
		reader,
		&mocks.DenyResourcesPermissionEngine{},
	)
	require.NoError(t, err)

	workerID := pulid.MustNew("wrk_").String()
	data, err := tool.Query(
		t.Context(),
		querytooltest.KeyedCall(map[string]any{querytool.ParamWorkerID: workerID}),
	)
	require.NoError(t, err)

	sent := reader.Sent("DeskWorkerRecognitions")
	require.NotNil(t, sent)
	assert.Equal(t, map[string]any{"workerId": workerID}, sent.Variables)
	outcome, _ := data.(map[string]any)
	recognitions, _ := outcome["recognitions"].([]map[string]any)
	require.Len(t, recognitions, 1)
	assert.Equal(t, "wrec_1", recognitions[0]["id"])
	assert.NotContains(t, recognitions[0], "__typename")
	assert.Contains(t, outcome["withheldByAccess"], "recognitions.title")
	items, _ := outcome["items"].([]map[string]any)
	require.Len(t, items, 1)
	assert.NotEmpty(t, items[0]["violations"])
}

func TestUnpagedList_ReadsNoRecognitionsWithoutAWorker(t *testing.T) {
	t.Parallel()

	reader := &querytooltest.KeyedReader{Answers: safetyAnswers()}
	tool, err := graphqllisttool.NewUnpaged(
		keyedSpec(t, workerquerytools.ToolListWorkerSafetyEvents),
		reader,
		&mocks.DenyResourcesPermissionEngine{},
	)
	require.NoError(t, err)

	data, err := tool.Query(
		t.Context(),
		querytooltest.KeyedCall(map[string]any{querytool.WfParamOpenOnly: true}),
	)
	require.NoError(t, err)

	assert.Nil(t, reader.Sent("DeskWorkerRecognitions"), "a read keyed by the worker needs one")
	outcome, _ := data.(map[string]any)
	assert.NotContains(t, outcome, "recognitions")
	sent := reader.Sent("DeskWorkerSafetyEvents")
	require.NotNil(t, sent)
	assert.Equal(
		t,
		map[string]any{"openOnly": true, "limit": querytool.GraphQLListLimit},
		sent.Variables,
	)
}

func TestUnpagedList_AKeyedReadAlongsideTakesOnlyTheVariablesItLeavesToTheSpec(t *testing.T) {
	t.Parallel()

	spec := keyedSpec(t, workerquerytools.ToolListWorkerSafetyEvents)
	spec.Alongside[0].Inputs = map[string]any{querytool.InputStatus: querytool.ActiveStatus}
	_, err := graphqllisttool.NewUnpaged(
		spec,
		&querytooltest.KeyedReader{},
		&mocks.DenyResourcesPermissionEngine{},
	)
	require.ErrorContains(t, err, "takes no status of the spec's")
}

func TestUnpagedList_ReadsTheFleetsOpenEventsByDefaultAndOneWorkersWhole(t *testing.T) {
	t.Parallel()

	workerID := pulid.MustNew("wrk_").String()
	cases := map[string]struct {
		params map[string]any
		want   map[string]any
	}{
		"no worker reads what is still open across the fleet": {
			params: map[string]any{},
			want:   map[string]any{"openOnly": true, "limit": querytool.GraphQLListLimit},
		},
		"no worker, and the closed ones asked for too": {
			params: map[string]any{querytool.WfParamOpenOnly: false},
			want:   map[string]any{"openOnly": false, "limit": querytool.GraphQLListLimit},
		},
		"one worker's record is read whole": {
			params: map[string]any{querytool.ParamWorkerID: workerID},
			want:   map[string]any{"workerId": workerID, "limit": querytool.GraphQLListLimit},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			reader := &querytooltest.KeyedReader{Answers: safetyAnswers()}
			tool, err := graphqllisttool.NewUnpaged(
				keyedSpec(t, workerquerytools.ToolListWorkerSafetyEvents),
				reader,
				&mocks.DenyResourcesPermissionEngine{},
			)
			require.NoError(t, err)

			_, err = tool.Query(t.Context(), querytooltest.KeyedCall(tc.params))
			require.NoError(t, err)

			sent := reader.Sent("DeskWorkerSafetyEvents")
			require.NotNil(t, sent)
			assert.Equal(t, tc.want, sent.Variables)
		})
	}
}

func offsetPageRecord(t *testing.T, data any) map[string]any {
	t.Helper()

	page, ok := data.(*graphqllisttool.OffsetPage)
	require.True(t, ok, "a list that pages by cursor answers with its page")

	return page.Record
}

func credentialAnswers() map[string]*graphqlreadports.GraphQLReadResult {
	return map[string]*graphqlreadports.GraphQLReadResult{
		"DeskWorkerCredentialList": {Data: map[string]any{"workerCredentials": []any{
			map[string]any{"id": "wcr_1", "status": "Active", "daysUntilExpiry": float64(12)},
		}}},
		"DeskWorkerCredentialTypes": {Data: map[string]any{"workerCredentialTypes": map[string]any{
			"edges": []any{map[string]any{"node": map[string]any{
				"id": "wct_1", "code": "MED_CARD", "isRequired": true,
			}}},
			"pageInfo": map[string]any{"hasNextPage": false},
		}}},
	}
}

func TestUnpagedList_ReadsTheCredentialTypesBesideOneWorkersCredentialsOnly(t *testing.T) {
	t.Parallel()

	workerID := pulid.MustNew("wrk_").String()
	cases := map[string]struct {
		params map[string]any
		types  bool
		sent   map[string]any
	}{
		"one worker's credentials come with the types the organization tracks": {
			params: map[string]any{"workerIds": []any{workerID}},
			types:  true,
			sent: map[string]any{
				"workerIds": []any{workerID}, "includeArchived": false,
				"limit": querytool.GraphQLListLimit + 1, "offset": 0,
			},
		},
		"what falls due across the organization does not": {
			params: map[string]any{"expiringWithinDays": float64(14)},
			sent: map[string]any{
				"expiringWithinDays": float64(14), "includeArchived": false,
				"limit": querytool.GraphQLListLimit + 1, "offset": 0,
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			reader := &querytooltest.KeyedReader{Answers: credentialAnswers()}
			tool, err := graphqllisttool.NewUnpaged(
				keyedSpec(t, workerquerytools.ToolWorkerListCredentials),
				reader,
				&mocks.DenyResourcesPermissionEngine{},
			)
			require.NoError(t, err)

			data, err := tool.Query(t.Context(), querytooltest.KeyedCall(tc.params))
			require.NoError(t, err)

			sent := reader.Sent("DeskWorkerCredentialList")
			require.NotNil(t, sent)
			assert.Equal(t, tc.sent, sent.Variables)
			outcome := offsetPageRecord(t, data)
			if tc.types {
				types, _ := outcome["credentialTypes"].([]map[string]any)
				require.Len(t, types, 1)
				assert.Equal(t, "MED_CARD", types[0]["code"])
				return
			}
			assert.Nil(t, reader.Sent("DeskWorkerCredentialTypes"))
			assert.NotContains(t, outcome, "credentialTypes")
		})
	}
}

// A credential's number is the licence or card number itself. Who needs a
// new medical card is answered without it, and the operation the tool reads
// never asks for it.
func TestCredentialList_NeverSelectsTheCredentialNumber(t *testing.T) {
	t.Parallel()

	read := graphqlread.Reads[workerquerytools.ToolWorkerListCredentials]
	for _, field := range read.Fields {
		if field.Name == "number" {
			assert.False(t, field.Selected)
		}
	}
}

func credentialRows(from, count int) []any {
	rows := make([]any, 0, count)
	for idx := range count {
		rows = append(rows, map[string]any{"id": "wcr_" + strconv.Itoa(from+idx)})
	}

	return rows
}

// What falls due across a fleet can pass a page; the list pages by cursor,
// and the offset it pages by never reaches the model.
func TestUnpagedList_PagesWhatFallsDueByCursor(t *testing.T) {
	t.Parallel()

	reader := &querytooltest.KeyedReader{Answers: map[string]*graphqlreadports.GraphQLReadResult{
		"DeskWorkerCredentialList": {Data: map[string]any{
			"workerCredentials": credentialRows(0, 3),
		}},
	}}
	tool, err := graphqllisttool.NewUnpaged(
		keyedSpec(t, workerquerytools.ToolWorkerListCredentials),
		reader,
		&mocks.DenyResourcesPermissionEngine{},
	)
	require.NoError(t, err)
	properties, _ := tool.ParamSchema()[toolschema.KeyProperties].(map[string]any)
	assert.Contains(t, properties, querytool.ParamAfter)
	assert.NotContains(t, properties, querytool.ParamOffset)

	args := map[string]any{"expiringWithinDays": float64(30), querytool.ParamLimit: float64(2)}
	data, err := tool.Query(t.Context(), querytooltest.KeyedCall(args))
	require.NoError(t, err)
	first := offsetPageRecord(t, data)
	items, _ := first["items"].([]map[string]any)
	require.Len(t, items, 2, "the row past the page only says there is more")
	assert.Equal(t, true, first["hasMore"])
	cursor, _ := first["nextCursor"].(string)
	require.NotEmpty(t, cursor)
	sent := reader.Sent("DeskWorkerCredentialList")
	assert.Equal(t, 3, sent.Variables["limit"])
	assert.Equal(t, 0, sent.Variables["offset"])

	next := &querytooltest.KeyedReader{Answers: map[string]*graphqlreadports.GraphQLReadResult{
		"DeskWorkerCredentialList": {Data: map[string]any{
			"workerCredentials": credentialRows(2, 1),
		}},
	}}
	tool, err = graphqllisttool.NewUnpaged(
		keyedSpec(t, workerquerytools.ToolWorkerListCredentials),
		next,
		&mocks.DenyResourcesPermissionEngine{},
	)
	require.NoError(t, err)
	data, err = tool.Query(t.Context(), querytooltest.KeyedCall(map[string]any{
		"expiringWithinDays": float64(
			30,
		), querytool.ParamLimit: float64(2), querytool.ParamAfter: cursor,
	}))
	require.NoError(t, err)
	assert.Equal(t, 2, next.Sent("DeskWorkerCredentialList").Variables["offset"])
	last := offsetPageRecord(t, data)
	assert.NotContains(t, last, "hasMore")
	assert.NotContains(t, last, "nextCursor")

	_, err = tool.Query(t.Context(), querytooltest.KeyedCall(map[string]any{
		"expiringWithinDays": float64(
			60,
		), querytool.ParamLimit: float64(2), querytool.ParamAfter: cursor,
	}))
	require.ErrorIs(
		t,
		err,
		querytool.ErrStaleCursor,
		"a cursor is bound to the call's other arguments",
	)
	_, err = tool.Query(
		t.Context(),
		querytooltest.KeyedCall(map[string]any{querytool.ParamOffset: float64(2)}),
	)
	require.ErrorIs(t, err, querytool.ErrOffsetOnCursorList)
}
