package assistantservice

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const pagedTool = "billing_list_queue_items"

func queueItem(n int) map[string]any {
	return map[string]any{
		"id":     "bqi_" + strings.Repeat("0", 3) + string(rune('A'+n)),
		"number": "BQ-" + string(rune('A'+n)),
		"status": "ReadyForReview",
	}
}

func queuePage(from, count int, more map[string]any) map[string]any {
	items := make([]any, 0, count)
	for n := from; n < from+count; n++ {
		items = append(items, queueItem(n))
	}
	page := map[string]any{
		"items":   items,
		"columns": []any{"number", "status"},
		"count":   float64(count),
	}
	for key, value := range more {
		page[key] = value
	}

	return page
}

func pageRead(callID string, arguments, data map[string]any) agentruntimeports.ToolObservation {
	return agentruntimeports.ToolObservation{
		Call: llmports.ToolCall{ID: callID, Name: pagedTool, Arguments: arguments},
		Data: data,
	}
}

func newPagingRecorder(t *testing.T, repo *stubArtifactRepo) *artifactRecorder {
	t.Helper()

	svc := &Service{logger: zap.NewNop(), artifacts: repo}

	return svc.newArtifactRecorder(
		t.Context(),
		&conversation.Thread{ID: pulid.MustNew("athr_")},
		pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		testActor(),
		nil,
	)
}

func rowNumbers(artifact *assistantartifact.Artifact) []string {
	rows := rowsFrom(artifact.Payload[payloadRows])
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		record, _ := row.(map[string]any)
		number, _ := record["number"].(string)
		out = append(out, number)
	}

	return out
}

func TestTableArtifact_KeepsWhereTheListGoesOn(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		result     map[string]any
		hasMore    bool
		nextCursor any
		nextOffset any
		total      any
		estimated  any
	}{
		"a cursor list with more": {
			result:     queuePage(0, 2, map[string]any{"hasMore": true, "nextCursor": "c-2"}),
			hasMore:    true,
			nextCursor: "c-2",
		},
		"an offset list with more and a total": {
			result: queuePage(0, 2, map[string]any{
				"hasMore": true, "nextOffset": float64(2), "total": float64(9),
			}),
			hasMore:    true,
			nextOffset: 2,
			total:      9,
		},
		"a list with an estimated total": {
			result: queuePage(0, 2, map[string]any{
				"hasMore": true, "nextCursor": "c-2", "total": float64(1_250_000),
				"totalEstimated": true,
			}),
			hasMore:    true,
			nextCursor: "c-2",
			total:      1_250_000,
			estimated:  true,
		},
		"the whole list": {
			result: queuePage(0, 2, map[string]any{"hasMore": false}),
		},
		"a page with no word on what follows": {
			result: queuePage(0, 2, nil),
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			artifact := tableArtifact("call_1", pagedTool, tc.result)
			require.NotNil(t, artifact)
			assert.Equal(t, tc.hasMore, artifact.Payload[payloadHasMore])
			assert.Equal(t, tc.nextCursor, artifact.Payload[payloadNextCursor])
			assert.Equal(t, tc.nextOffset, artifact.Payload[payloadNextOffset])
			assert.Equal(t, tc.total, artifact.Payload[payloadTotal])
			assert.Equal(t, tc.estimated, artifact.Payload[payloadTotalEstimated])
			assert.Equal(t, 1, artifact.Payload[payloadPages])
			assert.Equal(t, tc.hasMore, pagedTable(artifact))
		})
	}
}

func TestTableArtifact_ATableCutToFitHasMoreButNoPlaceToGoOnFrom(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("x", assistantartifact.MaxPayloadBytes/10)
	items := make([]any, 0, 20)
	for n := range 20 {
		item := queueItem(n)
		item["status"] = long
		items = append(items, item)
	}
	result := map[string]any{
		"items":      items,
		"columns":    []any{"number", "status"},
		"count":      float64(20),
		"hasMore":    false,
		"nextCursor": "",
	}

	artifact := tableArtifact("call_1", pagedTool, result)
	require.NotNil(t, artifact)
	assert.Equal(t, true, artifact.Payload[payloadTruncated])
	assert.Equal(t, true, artifact.Payload[payloadHasMore])
	assert.NotContains(t, artifact.Payload, payloadNextCursor)
	assert.NotContains(t, artifact.Payload, payloadNextOffset)
	assert.True(t, pagedTable(artifact))
}

func TestArtifactRecorder_APageReadOnJoinsTheTable(t *testing.T) {
	t.Parallel()

	repo := &stubArtifactRepo{}
	recorder := newPagingRecorder(t, repo)
	filters := map[string]any{"status": "Ready", "limit": float64(2)}

	first, err := recorder.observe(pageRead("call_1", filters,
		queuePage(0, 2, map[string]any{"hasMore": true, "nextCursor": "c-2"})))
	require.NoError(t, err)
	require.NotNil(t, first)
	assert.True(t, first.Paged)

	second, err := recorder.observe(pageRead("call_2",
		map[string]any{"status": "Ready", "limit": float64(2), "after": "c-2"},
		queuePage(2, 2, map[string]any{"hasMore": true, "nextCursor": "c-4"})))
	require.NoError(t, err)
	require.NotNil(t, second)

	require.Len(t, repo.upserts, 2)
	table := repo.upserts[1]
	assert.Equal(t, first.ID, second.ID)
	assert.Equal(t, repo.upserts[0].ID, table.ID)
	assert.Equal(t, "call_1", table.SourceToolCallID)
	assert.True(t, table.LineageID.IsNil())
	assert.Equal(t, repo.upserts[0].LineageSeq, table.LineageSeq)
	assert.Equal(t, []string{"BQ-A", "BQ-B", "BQ-C", "BQ-D"}, rowNumbers(table))
	assert.Equal(t, 4, table.Payload[payloadRowCount])
	assert.Equal(t, 2, table.Payload[payloadPages])
	assert.Equal(t, "c-4", table.Payload[payloadNextCursor])
	assert.Equal(t, 4, second.Rows)
	assert.True(t, second.Paged)
	assert.True(t, second.MustBeLinked())
	require.Len(t, recorder.recorded, 1)
	assert.Equal(t, []string{"BQ-A", "BQ-B", "BQ-C", "BQ-D"}, rowNumbers(recorder.recorded[0]))
}

func TestArtifactRecorder_APageFromALaterActivityFindsItsTable(t *testing.T) {
	t.Parallel()

	repo := &stubArtifactRepo{}
	_, err := newPagingRecorder(t, repo).observe(pageRead("call_1",
		map[string]any{"status": "Ready"},
		queuePage(0, 2, map[string]any{"hasMore": true, "nextOffset": float64(2)})))
	require.NoError(t, err)

	later := newPagingRecorder(t, repo)
	shown, err := later.observe(pageRead("call_2",
		map[string]any{"status": "Ready", "offset": float64(2)},
		queuePage(2, 1, map[string]any{"hasMore": false})))
	require.NoError(t, err)
	require.NotNil(t, shown)

	require.Len(t, repo.upserts, 2)
	table := repo.upserts[1]
	assert.Equal(t, repo.upserts[0].ID, shown.ID)
	assert.Equal(t, []string{"BQ-A", "BQ-B", "BQ-C"}, rowNumbers(table))
	assert.Equal(t, false, table.Payload[payloadHasMore])
	assert.NotContains(t, table.Payload, payloadNextOffset)
	assert.True(t, shown.Paged, "a table read in two pages is paged though nothing follows")
}

func TestArtifactRecorder_ALastPageOfOneRecordStaysInItsTable(t *testing.T) {
	t.Parallel()

	repo := &stubArtifactRepo{}
	recorder := newPagingRecorder(t, repo)
	template := func(id, name string) map[string]any {
		return map[string]any{"id": id, "name": name, "status": "Active"}
	}
	read := func(callID string, arguments, data map[string]any) agentruntimeports.ToolObservation {
		return agentruntimeports.ToolObservation{
			Call: llmports.ToolCall{
				ID: callID, Name: "formula_list_templates", Arguments: arguments,
			},
			Data: data,
		}
	}

	_, err := recorder.observe(read("call_1", map[string]any{}, map[string]any{
		"count": float64(2), "hasMore": true, "nextCursor": "c-2",
		"columns": []any{"id", "name", "status"},
		"items": []any{
			template("ft_01JPERMILE00000000000000", "Per mile"),
			template("ft_01JPERSTOP00000000000000", "Per stop"),
		},
	}))
	require.NoError(t, err)
	shown, err := recorder.observe(read("call_2", map[string]any{"after": "c-2"}, map[string]any{
		"count": float64(1), "hasMore": false,
		"columns": []any{"id", "name", "status"},
		"items":   []any{template("ft_01JPERCWT000000000000000", "Per CWT")},
	}))
	require.NoError(t, err)

	require.Len(t, repo.upserts, 2)
	assert.Equal(t, string(assistantartifact.KindTableView), shown.Kind)
	assert.Equal(t, repo.upserts[0].ID, shown.ID)
	assert.Len(t, rowsFrom(repo.upserts[1].Payload[payloadRows]), 3)
}

func TestArtifactRecorder_ACursorTheTableDidNotHandOutIsKeptAsBefore(t *testing.T) {
	t.Parallel()

	repo := &stubArtifactRepo{}
	recorder := newPagingRecorder(t, repo)

	_, err := recorder.observe(pageRead("call_1", map[string]any{"status": "Ready"},
		queuePage(0, 2, map[string]any{"hasMore": true, "nextCursor": "c-2"})))
	require.NoError(t, err)
	_, err = recorder.observe(pageRead("call_2",
		map[string]any{"status": "Ready", "after": "c-9"},
		queuePage(8, 2, map[string]any{"hasMore": false})))
	require.NoError(t, err)

	require.Len(t, repo.upserts, 2)
	assert.NotEqual(t, repo.upserts[0].ID, repo.upserts[1].ID)
	assert.Equal(t, repo.upserts[0].ID, repo.upserts[1].LineageID)
	assert.Equal(t, 2, repo.upserts[1].LineageSeq)
}

func TestArtifactRecorder_APageThatWouldNotFitLeavesTheTableAsItIs(t *testing.T) {
	t.Parallel()

	repo := &stubArtifactRepo{}
	recorder := newPagingRecorder(t, repo)
	_, err := recorder.observe(pageRead("call_1", map[string]any{},
		queuePage(0, 2, map[string]any{"hasMore": true, "nextCursor": "c-2"})))
	require.NoError(t, err)

	long := strings.Repeat("x", assistantartifact.MaxPayloadBytes)
	page := queuePage(2, 1, map[string]any{"hasMore": true, "nextCursor": "c-3"})
	page["items"].([]any)[0].(map[string]any)["status"] = long
	shown, err := recorder.observe(pageRead("call_2", map[string]any{"after": "c-2"}, page))
	require.NoError(t, err)
	require.NotNil(t, shown)

	require.Len(t, repo.upserts, 1)
	assert.Equal(t, repo.upserts[0].ID, shown.ID)
	assert.Equal(t, "c-2", repo.upserts[0].Payload[payloadNextCursor])
	assert.Equal(t, []string{"BQ-A", "BQ-B"}, rowNumbers(repo.upserts[0]))
}

func TestArtifactRecorder_AShortPagedTableStaysWhetherOrNotTheReplyLinkedIt(t *testing.T) {
	t.Parallel()

	repo := &stubArtifactRepo{}
	recorder := newPagingRecorder(t, repo)
	_, err := recorder.observe(pageRead("call_1", map[string]any{},
		queuePage(0, 2, map[string]any{"hasMore": true, "nextCursor": "c-2"})))
	require.NoError(t, err)

	recorder.keepLinked(map[string]bool{})

	assert.Len(t, recorder.recorded, 1)
}

func TestArtifactRecorder_ATableKeepsTheArgumentsItsListRanWith(t *testing.T) {
	t.Parallel()

	repo := &stubArtifactRepo{}
	recorder := newPagingRecorder(t, repo)
	observation := pageRead(
		"call_1",
		map[string]any{"status": "ready for review"},
		queuePage(0, 2, map[string]any{"hasMore": true, "nextCursor": "c-2"}),
	)
	observation.RanWith = map[string]any{"status": []any{"ReadyForReview"}}

	_, err := recorder.observe(observation)
	require.NoError(t, err)

	require.Len(t, repo.upserts, 1)
	assert.Equal(t,
		map[string]any{"status": []any{"ReadyForReview"}},
		repo.upserts[0].Payload[payloadReadWith],
	)
	observation.RanWith["status"] = []any{"Posted"}
	assert.Equal(t,
		map[string]any{"status": []any{"ReadyForReview"}},
		repo.upserts[0].Payload[payloadReadWith],
		"the table keeps its own copy",
	)
}

func newReaderRecorder(
	t *testing.T,
	repo *stubArtifactRepo,
	agents *delegateDefinitions,
	agentID pulid.ID,
) *artifactRecorder {
	t.Helper()

	svc := &Service{logger: zap.NewNop(), artifacts: repo, definitions: agents}

	return svc.newArtifactRecorder(
		t.Context(),
		&conversation.Thread{ID: pulid.MustNew("athr_"), AgentDefinitionID: agentID},
		pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		testActor(),
		nil,
	)
}

func TestArtifactRecorder_ATableNamesTheAgentThatReadIt(t *testing.T) {
	t.Parallel()

	agentID := pulid.MustNew("agdef_")
	agents := &delegateDefinitions{byID: map[pulid.ID]*agentdefinition.Definition{
		agentID: {ID: agentID, Name: "Dispatch Desk", Accent: "teal"},
	}}
	repo := &stubArtifactRepo{}
	recorder := newReaderRecorder(t, repo, agents, agentID)

	for at, call := range []string{"call_1", "call_2"} {
		_, err := recorder.observe(pageRead(
			call,
			map[string]any{"status": call},
			queuePage(at*2, 2, map[string]any{"hasMore": true, "nextCursor": call}),
		))
		require.NoError(t, err)
	}

	require.Len(t, repo.upserts, 2)
	want := artifactReader{ID: agentID.String(), Name: "Dispatch Desk", Accent: "teal"}
	assert.Equal(t, want, repo.upserts[0].Payload[payloadReadBy])
	assert.Equal(t, want, repo.upserts[1].Payload[payloadReadBy])
	assert.Equal(t, 1, agents.listed, "the agent is read once for the turn")
}

func TestArtifactRecorder_ATableDoesNotNameAnAgentItCannotRead(t *testing.T) {
	t.Parallel()

	agents := &delegateDefinitions{byID: map[pulid.ID]*agentdefinition.Definition{}}
	repo := &stubArtifactRepo{}
	recorder := newReaderRecorder(t, repo, agents, pulid.MustNew("agdef_"))

	_, err := recorder.observe(pageRead(
		"call_1",
		map[string]any{},
		queuePage(0, 2, map[string]any{"hasMore": true, "nextCursor": "c-2"}),
	))
	require.NoError(t, err)

	require.Len(t, repo.upserts, 1)
	assert.NotContains(t, repo.upserts[0].Payload, payloadReadBy)
}

func TestArtifactRecorder_AReportRunNamesTheAgentThatRanIt(t *testing.T) {
	t.Parallel()

	agentID := pulid.MustNew("agdef_")
	agents := &delegateDefinitions{byID: map[pulid.ID]*agentdefinition.Definition{
		agentID: {ID: agentID, Name: "Billing Desk", Accent: "violet"},
	}}
	repo := &stubArtifactRepo{}
	recorder := newReaderRecorder(t, repo, agents, agentID)

	_, err := recorder.observe(agentruntimeports.ToolObservation{
		Call: llmports.ToolCall{
			ID:        "call_run",
			Name:      toolRunReport,
			Arguments: map[string]any{"reportKey": "revenue-by-customer"},
		},
		Data: map[string]any{
			"runId":     pulid.MustNew("rrun_").String(),
			"reportKey": "revenue-by-customer",
			"status":    "queued",
			"finished":  false,
		},
	})
	require.NoError(t, err)

	require.Len(t, repo.upserts, 1)
	assert.Equal(t, assistantartifact.KindReportRun, repo.upserts[0].Kind)
	assert.Equal(
		t,
		artifactReader{ID: agentID.String(), Name: "Billing Desk", Accent: "violet"},
		repo.upserts[0].Payload[payloadReadBy],
	)
}
