package agentquerytoolservice

import (
	"context"
	"strconv"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// keyedReader answers each operation with what a test gives it, and keeps
// every request it was sent.
type keyedReader struct {
	mu       sync.Mutex
	answers  map[string]*agentports.GraphQLReadResult
	requests []*agentports.GraphQLReadRequest
}

func (r *keyedReader) Read(
	_ context.Context,
	req *agentports.GraphQLReadRequest,
) (*agentports.GraphQLReadResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.requests = append(r.requests, req)
	if answer, ok := r.answers[req.Operation]; ok {
		return answer, nil
	}

	return &agentports.GraphQLReadResult{Data: map[string]any{}}, nil
}

func (r *keyedReader) sent(operation string) *agentports.GraphQLReadRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, req := range r.requests {
		if req.Operation == operation {
			return req
		}
	}

	return nil
}

func keyedSpec(t *testing.T, name string) *graphQLListSpec {
	t.Helper()

	for _, spec := range workerRecordReadSpecs() {
		if spec.name == name {
			return &spec
		}
	}
	require.FailNow(t, "no spec "+name)

	return nil
}

func keyedCall(params map[string]any) *agentports.QueryToolParams {
	actor := &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}

	return &agentports.QueryToolParams{
		OrganizationID: actor.OrganizationID,
		BusinessUnitID: actor.BusinessUnitID,
		Actor:          actor,
		Timezone:       "America/Chicago",
		Params:         params,
		RunID:          pulid.MustNew("ar_"),
	}
}

func checklistAnswer() *agentports.GraphQLReadResult {
	return &agentports.GraphQLReadResult{
		Data: map[string]any{"workerChecklists": []any{map[string]any{
			"id": "wcl_1", "name": "Onboarding", "status": "Open", "__typename": "WorkerChecklist",
			"items": []any{map[string]any{"id": "wcli_1", "label": "Sign handbook", "note": nil}},
		}}},
		Withheld: []string{"workerChecklists.items.note"},
	}
}

func templateAnswer(more bool) *agentports.GraphQLReadResult {
	return &agentports.GraphQLReadResult{Data: map[string]any{
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

	reader := &keyedReader{answers: map[string]*agentports.GraphQLReadResult{
		"DeskWorkerChecklists":         checklistAnswer(),
		"DeskWorkerChecklistTemplates": templateAnswer(false),
	}}
	tool, err := newGraphQLUnpagedListTool(keyedSpec(t, toolListWorkerChecklists), reader,
		&mocks.DenyResourcesPermissionEngine{})
	require.NoError(t, err)

	workerID := pulid.MustNew("wrk_").String()
	data, err := tool.Query(t.Context(), keyedCall(map[string]any{paramWorkerID: workerID}))
	require.NoError(t, err)

	sent := reader.sent("DeskWorkerChecklists")
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
	assert.Equal(t, withheldNote, outcome["withheldNote"])
}

func TestUnpagedList_ReadsTheTemplatesAPersonStartsByHand(t *testing.T) {
	t.Parallel()

	reader := &keyedReader{answers: map[string]*agentports.GraphQLReadResult{
		"DeskWorkerChecklists":         checklistAnswer(),
		"DeskWorkerChecklistTemplates": templateAnswer(true),
	}}
	tool, err := newGraphQLUnpagedListTool(keyedSpec(t, toolListWorkerChecklists), reader,
		&mocks.DenyResourcesPermissionEngine{})
	require.NoError(t, err)

	data, err := tool.Query(t.Context(), keyedCall(map[string]any{
		paramWorkerID: pulid.MustNew("wrk_").String(),
	}))
	require.NoError(t, err)

	sent := reader.sent("DeskWorkerChecklistTemplates")
	require.NotNil(t, sent)
	assert.Equal(t, map[string]any{"input": map[string]any{
		"status": "Active", "trigger": "Manual", "first": alongsideMost,
	}}, sent.Variables)

	outcome, _ := data.(map[string]any)
	templates, _ := outcome["templatesStartedByHand"].([]map[string]any)
	require.Len(t, templates, 1)
	assert.Equal(t, "wclt_1", templates[0]["id"])
	assert.Contains(t, outcome["note"], "Only the first 100 checklist templates")
}

func TestUnpagedList_LeavesOutTheTemplatesAReaderCannotRead(t *testing.T) {
	t.Parallel()

	reader := &keyedReader{answers: map[string]*agentports.GraphQLReadResult{
		"DeskWorkerChecklists": checklistAnswer(),
	}}
	engine := &mocks.DenyResourcesPermissionEngine{
		Denied: []string{permission.ResourceWorkerChecklistTemplate.String()},
	}
	tool, err := newGraphQLUnpagedListTool(keyedSpec(t, toolListWorkerChecklists), reader, engine)
	require.NoError(t, err)

	data, err := tool.Query(t.Context(), keyedCall(map[string]any{
		paramWorkerID: pulid.MustNew("wrk_").String(),
	}))
	require.NoError(t, err)

	assert.Nil(t, reader.sent("DeskWorkerChecklistTemplates"), "nothing is read that would be refused")
	outcome, _ := data.(map[string]any)
	assert.NotContains(t, outcome, "templatesStartedByHand")
}

func TestUnpagedList_RefusesACallWithoutItsWorkerOrWithAName(t *testing.T) {
	t.Parallel()

	reader := &keyedReader{}
	tool, err := newGraphQLUnpagedListTool(keyedSpec(t, toolListWorkerTraining), reader,
		&mocks.DenyResourcesPermissionEngine{})
	require.NoError(t, err)

	_, err = tool.Query(t.Context(), keyedCall(map[string]any{}))
	require.ErrorContains(t, err, `"workerId" is required`)

	_, err = tool.Query(t.Context(), keyedCall(map[string]any{paramWorkerID: "Jane Smith"}))
	require.Error(t, err)
	assert.Empty(t, reader.requests, "a refused call reads nothing")
}

func TestUnpagedList_TakesTheWorkerAndTheSwitchAsItsParameters(t *testing.T) {
	t.Parallel()

	tool, err := newGraphQLUnpagedListTool(keyedSpec(t, toolListWorkerTraining), &keyedReader{},
		&mocks.DenyResourcesPermissionEngine{})
	require.NoError(t, err)

	schema := tool.ParamSchema()
	assert.Equal(t, []string{paramWorkerID}, schema["required"])
	properties, _ := schema["properties"].(map[string]any)
	assert.Len(t, properties, 2)
	closed, _ := properties[wfParamClosed].(map[string]any)
	assert.Contains(t, closed["description"], "expired assignments")
	assert.Contains(t, closed["description"], noteOpenOnly)
	worker, _ := properties[paramWorkerID].(map[string]any)
	assert.Equal(t, permission.ResourceWorker.String(), worker[toolschema.KeyRecordOf])
}

func TestUnpagedList_SpecsMatchWhatTheGeneratorProved(t *testing.T) {
	t.Parallel()

	spec := *keyedSpec(t, toolListWorkerChecklists)
	spec.alongside = nil
	_, err := newGraphQLUnpagedListTool(&spec, &keyedReader{}, nil)
	require.ErrorContains(t, err, "reads.yml lists 1 reads alongside it, its spec 0")

	spec = *keyedSpec(t, toolListWorkerTraining)
	spec.alongside = []alongsideSpec{{key: "templates", operation: opDeskReviewTemplates}}
	_, err = newGraphQLUnpagedListTool(&spec, &keyedReader{}, nil)
	require.ErrorContains(t, err, "which reads.yml does not list for it")

	spec = *keyedSpec(t, toolListWorkerTraining)
	spec.references = nil
	_, err = newGraphQLUnpagedListTool(&spec, &keyedReader{}, nil)
	require.ErrorContains(t, err, "requires workerId, which no tool supplies")
}

func TestUnpagedList_ReadsTheWholeOrganizationWithAPageAndSaysWhenItIsFull(t *testing.T) {
	t.Parallel()

	rows := make([]any, 0, graphQLListLimit)
	for idx := range graphQLListLimit {
		rows = append(rows, map[string]any{"id": "wdt_" + strconv.Itoa(idx), "status": "Scheduled"})
	}
	reader := &keyedReader{answers: map[string]*agentports.GraphQLReadResult{
		"DeskWorkerDotTests": {Data: map[string]any{"workerDotTests": rows}},
	}}
	tool, err := newGraphQLUnpagedListTool(keyedSpec(t, toolListWorkerDOTTests), reader,
		&mocks.DenyResourcesPermissionEngine{})
	require.NoError(t, err)
	assert.NotContains(t, tool.ParamSchema(), "required", "the worker is optional")

	data, err := tool.Query(t.Context(), keyedCall(map[string]any{wfParamOpenOnly: true}))
	require.NoError(t, err)

	sent := reader.sent("DeskWorkerDotTests")
	require.NotNil(t, sent)
	assert.Equal(t, map[string]any{"openOnly": true, "limit": graphQLListLimit}, sent.Variables)
	outcome, _ := data.(map[string]any)
	assert.Contains(t, outcome["note"], "more may exist")
}

func TestUnpagedList_LeavesOutTheWorkerOfAPrivacyCase(t *testing.T) {
	t.Parallel()

	reader := &keyedReader{answers: map[string]*agentports.GraphQLReadResult{
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
	tool, err := newGraphQLUnpagedListTool(keyedSpec(t, toolListWorkerInjuries), reader,
		&mocks.DenyResourcesPermissionEngine{})
	require.NoError(t, err)

	data, err := tool.Query(t.Context(), keyedCall(map[string]any{"caseYear": float64(2026)}))
	require.NoError(t, err)

	sent := reader.sent("DeskWorkerInjuries")
	require.NotNil(t, sent)
	assert.Equal(t, float64(2026), sent.Variables["caseYear"])
	outcome, _ := data.(map[string]any)
	items, _ := outcome["items"].([]map[string]any)
	require.Len(t, items, 2)
	assert.NotContains(t, items[0], "worker", "a privacy case never carries its worker's name")
	assert.Equal(t, "wrk_1", items[0]["workerId"])
	assert.Contains(t, items[1], "worker")
}

func safetyAnswers() map[string]*agentports.GraphQLReadResult {
	return map[string]*agentports.GraphQLReadResult{
		"DeskWorkerSafetyEvents": {Data: map[string]any{"workerSafetyEvents": []any{
			map[string]any{
				"id": "wsev_1", "kind": "Inspection", "status": "Open",
				"violations": []any{map[string]any{"id": "wsvi_1", "code": "393.9"}},
			},
		}}},
		"DeskWorkerRecognitions": {
			Data: map[string]any{"workerRecognitions": []any{
				map[string]any{"id": "wrec_1", "title": "Ten years", "__typename": "WorkerRecognition"},
			}},
			Withheld: []string{"workerRecognitions.title"},
		},
	}
}

func TestUnpagedList_ReadsTheWorkersRecognitionsByTheWorkerTheCallNames(t *testing.T) {
	t.Parallel()

	reader := &keyedReader{answers: safetyAnswers()}
	tool, err := newGraphQLUnpagedListTool(keyedSpec(t, toolListWorkerSafetyEvents), reader,
		&mocks.DenyResourcesPermissionEngine{})
	require.NoError(t, err)

	workerID := pulid.MustNew("wrk_").String()
	data, err := tool.Query(t.Context(), keyedCall(map[string]any{paramWorkerID: workerID}))
	require.NoError(t, err)

	sent := reader.sent("DeskWorkerRecognitions")
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

	reader := &keyedReader{answers: safetyAnswers()}
	tool, err := newGraphQLUnpagedListTool(keyedSpec(t, toolListWorkerSafetyEvents), reader,
		&mocks.DenyResourcesPermissionEngine{})
	require.NoError(t, err)

	data, err := tool.Query(t.Context(), keyedCall(map[string]any{wfParamOpenOnly: true}))
	require.NoError(t, err)

	assert.Nil(t, reader.sent("DeskWorkerRecognitions"), "a read keyed by the worker needs one")
	outcome, _ := data.(map[string]any)
	assert.NotContains(t, outcome, "recognitions")
	sent := reader.sent("DeskWorkerSafetyEvents")
	require.NotNil(t, sent)
	assert.Equal(t, map[string]any{"openOnly": true, "limit": graphQLListLimit}, sent.Variables)
}

func TestUnpagedList_AKeyedReadAlongsideTakesNoTableInput(t *testing.T) {
	t.Parallel()

	spec := keyedSpec(t, toolListWorkerSafetyEvents)
	spec.alongside[0].inputs = map[string]any{inputStatus: activeStatus}
	_, err := newGraphQLUnpagedListTool(spec, &keyedReader{}, &mocks.DenyResourcesPermissionEngine{})
	require.ErrorContains(t, err, "takes no table input")
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
			want:   map[string]any{"openOnly": true, "limit": graphQLListLimit},
		},
		"no worker, and the closed ones asked for too": {
			params: map[string]any{wfParamOpenOnly: false},
			want:   map[string]any{"openOnly": false, "limit": graphQLListLimit},
		},
		"one worker's record is read whole": {
			params: map[string]any{paramWorkerID: workerID},
			want:   map[string]any{"workerId": workerID, "limit": graphQLListLimit},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			reader := &keyedReader{answers: safetyAnswers()}
			tool, err := newGraphQLUnpagedListTool(keyedSpec(t, toolListWorkerSafetyEvents), reader,
				&mocks.DenyResourcesPermissionEngine{})
			require.NoError(t, err)

			_, err = tool.Query(t.Context(), keyedCall(tc.params))
			require.NoError(t, err)

			sent := reader.sent("DeskWorkerSafetyEvents")
			require.NotNil(t, sent)
			assert.Equal(t, tc.want, sent.Variables)
		})
	}
}

func credentialAnswers() map[string]*agentports.GraphQLReadResult {
	return map[string]*agentports.GraphQLReadResult{
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
			params: map[string]any{paramWorkerID: workerID},
			types:  true,
			sent: map[string]any{
				"workerId": workerID, "includeArchived": false, "limit": graphQLListLimit,
			},
		},
		"what falls due across the organization does not": {
			params: map[string]any{"expiringWithinDays": float64(14)},
			sent: map[string]any{
				"expiringWithinDays": float64(14), "includeArchived": false,
				"limit": graphQLListLimit,
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			reader := &keyedReader{answers: credentialAnswers()}
			tool, err := newGraphQLUnpagedListTool(keyedSpec(t, toolWorkerListCredentials), reader,
				&mocks.DenyResourcesPermissionEngine{})
			require.NoError(t, err)

			data, err := tool.Query(t.Context(), keyedCall(tc.params))
			require.NoError(t, err)

			sent := reader.sent("DeskWorkerCredentialList")
			require.NotNil(t, sent)
			assert.Equal(t, tc.sent, sent.Variables)
			outcome, _ := data.(map[string]any)
			if tc.types {
				types, _ := outcome["credentialTypes"].([]map[string]any)
				require.Len(t, types, 1)
				assert.Equal(t, "MED_CARD", types[0]["code"])
				return
			}
			assert.Nil(t, reader.sent("DeskWorkerCredentialTypes"))
			assert.NotContains(t, outcome, "credentialTypes")
		})
	}
}

// A credential's number is the licence or card number itself. Who needs a
// new medical card is answered without it, and the operation the tool reads
// never asks for it.
func TestCredentialList_NeverSelectsTheCredentialNumber(t *testing.T) {
	t.Parallel()

	read := graphQLReads[toolWorkerListCredentials]
	for _, field := range read.Fields {
		if field.Name == "number" {
			assert.False(t, field.Selected)
		}
	}
}
