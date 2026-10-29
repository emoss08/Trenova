package agentquerytoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/graphqlreadports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func briefingTool(
	t *testing.T,
	reader graphqlreadports.GraphQLReader,
) agenttoolports.AgentQueryTool {
	t.Helper()

	spec := dailyBriefingGetSpec()
	tool, err := newGraphQLGetTool(&spec, reader, &mocks.DenyResourcesPermissionEngine{})
	require.NoError(t, err)

	return tool
}

func briefingAnswer(page any) map[string]*graphqlreadports.GraphQLReadResult {
	return map[string]*graphqlreadports.GraphQLReadResult{
		"TodaysBriefing": {Data: map[string]any{"todaysBriefing": page}},
	}
}

/*
The morning's page is read through the Desk's own operation.

The briefing service cuts the page to the sections the caller may read, so the
tool sends the role and the day the call names on the screen's input and shows
what comes back, with a note when anything was withheld.
*/
func TestGetDailyBriefing_SendsTheRoleAndDayOnTheScreensInput(t *testing.T) {
	t.Parallel()

	reader := &keyedReader{answers: briefingAnswer(map[string]any{
		"id": "brf_1", "roleKey": "Billing", "briefingDate": "2026-09-23", "status": "Ready",
		"headline": "Twelve items open.", "narrated": true,
		"sections": []any{}, "withheldSections": []any{},
	})}
	tool := briefingTool(t, reader)

	result, err := tool.Query(t.Context(), testParams(map[string]any{
		paramRole: "Billing", paramDate: "2026-09-23",
	}))
	require.NoError(t, err)

	assert.Equal(t, map[string]any{
		"input": map[string]any{"roleKey": "Billing", "briefingDate": "2026-09-23"},
	}, reader.sent("TodaysBriefing").Variables)
	page := encodedDocument(t, result)
	assert.Equal(t, "Twelve items open.", page["headline"])
	assert.NotContains(t, page, "note")
}

func TestGetDailyBriefing_ACallThatNamesNothingReadsEveryonesPageForToday(t *testing.T) {
	t.Parallel()

	reader := &keyedReader{answers: briefingAnswer(map[string]any{
		"id": "brf_1", "roleKey": "General", "status": "Ready",
		"sections": []any{}, "withheldSections": []any{},
	})}
	tool := briefingTool(t, reader)
	assert.Empty(t, tool.ParamSchema()[toolschema.KeyRequired], "both role and day are optional")

	_, err := tool.Query(t.Context(), testParams(map[string]any{}))
	require.NoError(t, err)

	assert.Equal(t, map[string]any{"input": map[string]any{}},
		reader.sent("TodaysBriefing").Variables)
}

func TestGetDailyBriefing_SaysWhatWasWithheldAndWhyThePageReadsPlainer(t *testing.T) {
	t.Parallel()

	reader := &keyedReader{answers: briefingAnswer(map[string]any{
		"id": "brf_1", "roleKey": "General", "status": "Failed", "headline": "",
		"sections": []any{},
		"withheldSections": []any{map[string]any{
			"key": "Cash", "title": "Cash received yesterday", "partial": false,
		}},
	})}

	result, err := briefingTool(t, reader).Query(t.Context(), testParams(map[string]any{}))
	require.NoError(t, err)

	assert.Equal(t, briefingFailedNote+" "+briefingWithheldNote, encodedDocument(t, result)["note"])
}

func TestGetDailyBriefing_SaysSoWhenNothingIsWrittenYet(t *testing.T) {
	t.Parallel()

	reader := &keyedReader{answers: briefingAnswer(nil)}

	result, err := briefingTool(t, reader).Query(t.Context(), testParams(map[string]any{}))
	require.NoError(t, err)

	page := encodedDocument(t, result)
	assert.Equal(t, false, page["found"])
	assert.Equal(t, briefingMissingNote, page["note"])
}

func TestGetDailyBriefing_TakesOnlyARoleTheSchemaNamesAndACalendarDay(t *testing.T) {
	t.Parallel()

	properties, _ := briefingTool(t, &keyedReader{}).ParamSchema()[toolschema.KeyProperties].(map[string]any)
	role, _ := properties[paramRole].(map[string]any)
	assert.Equal(t, []string{"Dispatch", "Billing", "Compliance", "Leadership", "General"},
		role[toolschema.KeyEnum])
	assert.Equal(t, "Whose morning to read; omitted reads the one written for everybody.",
		role[toolschema.KeyDescription], "a get's role picks the page, it filters nothing")
	day, _ := properties[paramDate].(map[string]any)
	assert.Equal(t, "date", day["format"])
}

func TestGetDailyBriefing_PolicyIsAPlainRead(t *testing.T) {
	t.Parallel()

	policy := briefingTool(t, &keyedReader{}).Policy()

	assert.Equal(t, permission.ResourceBriefing, policy.Resource)
	assert.Equal(t, agenttooltrust.ExternalReadNever, policy.ReadsExternal)
}
