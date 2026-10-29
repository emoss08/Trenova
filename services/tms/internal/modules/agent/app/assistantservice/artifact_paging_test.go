package assistantservice

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/artifactpayload"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/artifactrecorder"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/attribution"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/toolartifacts"
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

func pageRead(callID string, arguments, data map[string]any) agentruntimeports.ToolObservation {
	return agentruntimeports.ToolObservation{
		Call: llmports.ToolCall{ID: callID, Name: assistanttest.PagedTool, Arguments: arguments},
		Data: data,
	}
}

func newPagingRecorder(
	t *testing.T,
	repo *assistanttest.StubArtifactRepo,
) *artifactrecorder.Recorder {
	t.Helper()

	svc := &Service{logger: zap.NewNop(), artifacts: repo}

	return svc.newArtifactRecorder(
		t.Context(),
		&conversation.Thread{ID: pulid.MustNew("athr_")},
		pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		assistanttest.Actor(),
		nil,
	)
}

func rowNumbers(artifact *assistantartifact.Artifact) []string {
	rows := artifactpayload.RowsFrom(artifact.Payload[artifactpayload.PayloadRows])
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		record, _ := row.(map[string]any)
		number, _ := record["number"].(string)
		out = append(out, number)
	}

	return out
}

func TestArtifactRecorder_APageReadOnJoinsTheTable(t *testing.T) {
	t.Parallel()

	repo := &assistanttest.StubArtifactRepo{}
	recorder := newPagingRecorder(t, repo)
	filters := map[string]any{"status": "Ready", "limit": float64(2)}

	first, err := recorder.Observe(pageRead("call_1", filters,
		assistanttest.QueuePage(0, 2, map[string]any{"hasMore": true, "nextCursor": "c-2"})))
	require.NoError(t, err)
	require.NotNil(t, first)
	assert.True(t, first.Paged)

	second, err := recorder.Observe(pageRead("call_2",
		map[string]any{"status": "Ready", "limit": float64(2), "after": "c-2"},
		assistanttest.QueuePage(2, 2, map[string]any{"hasMore": true, "nextCursor": "c-4"})))
	require.NoError(t, err)
	require.NotNil(t, second)

	require.Len(t, repo.Upserts, 2)
	table := repo.Upserts[1]
	assert.Equal(t, first.ID, second.ID)
	assert.Equal(t, repo.Upserts[0].ID, table.ID)
	assert.Equal(t, "call_1", table.SourceToolCallID)
	assert.True(t, table.LineageID.IsNil())
	assert.Equal(t, repo.Upserts[0].LineageSeq, table.LineageSeq)
	assert.Equal(t, []string{"BQ-A", "BQ-B", "BQ-C", "BQ-D"}, rowNumbers(table))
	assert.Equal(t, 4, table.Payload[artifactpayload.PayloadRowCount])
	assert.Equal(t, 2, table.Payload[artifactpayload.PayloadPages])
	assert.Equal(t, "c-4", table.Payload[artifactpayload.PayloadNextCursor])
	assert.Equal(t, 4, second.Rows)
	assert.True(t, second.Paged)
	assert.True(t, second.MustBeLinked())
	require.Len(t, recorder.Recorded, 1)
	assert.Equal(t, []string{"BQ-A", "BQ-B", "BQ-C", "BQ-D"}, rowNumbers(recorder.Recorded[0]))
}

func TestArtifactRecorder_APageFromALaterActivityFindsItsTable(t *testing.T) {
	t.Parallel()

	repo := &assistanttest.StubArtifactRepo{}
	_, err := newPagingRecorder(t, repo).Observe(pageRead("call_1",
		map[string]any{"status": "Ready"},
		assistanttest.QueuePage(0, 2, map[string]any{"hasMore": true, "nextOffset": float64(2)})))
	require.NoError(t, err)

	later := newPagingRecorder(t, repo)
	shown, err := later.Observe(pageRead("call_2",
		map[string]any{"status": "Ready", "offset": float64(2)},
		assistanttest.QueuePage(2, 1, map[string]any{"hasMore": false})))
	require.NoError(t, err)
	require.NotNil(t, shown)

	require.Len(t, repo.Upserts, 2)
	table := repo.Upserts[1]
	assert.Equal(t, repo.Upserts[0].ID, shown.ID)
	assert.Equal(t, []string{"BQ-A", "BQ-B", "BQ-C"}, rowNumbers(table))
	assert.Equal(t, false, table.Payload[artifactpayload.PayloadHasMore])
	assert.NotContains(t, table.Payload, artifactpayload.PayloadNextOffset)
	assert.True(t, shown.Paged, "a table read in two pages is paged though nothing follows")
}

func TestArtifactRecorder_ALastPageOfOneRecordStaysInItsTable(t *testing.T) {
	t.Parallel()

	repo := &assistanttest.StubArtifactRepo{}
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

	_, err := recorder.Observe(read("call_1", map[string]any{}, map[string]any{
		"count": float64(2), "hasMore": true, "nextCursor": "c-2",
		"columns": []any{"id", "name", "status"},
		"items": []any{
			template("ft_01JPERMILE00000000000000", "Per mile"),
			template("ft_01JPERSTOP00000000000000", "Per stop"),
		},
	}))
	require.NoError(t, err)
	shown, err := recorder.Observe(read("call_2", map[string]any{"after": "c-2"}, map[string]any{
		"count": float64(1), "hasMore": false,
		"columns": []any{"id", "name", "status"},
		"items":   []any{template("ft_01JPERCWT000000000000000", "Per CWT")},
	}))
	require.NoError(t, err)

	require.Len(t, repo.Upserts, 2)
	assert.Equal(t, string(assistantartifact.KindTableView), shown.Kind)
	assert.Equal(t, repo.Upserts[0].ID, shown.ID)
	assert.Len(t, artifactpayload.RowsFrom(repo.Upserts[1].Payload[artifactpayload.PayloadRows]), 3)
}

func TestArtifactRecorder_ACursorTheTableDidNotHandOutIsKeptAsBefore(t *testing.T) {
	t.Parallel()

	repo := &assistanttest.StubArtifactRepo{}
	recorder := newPagingRecorder(t, repo)

	_, err := recorder.Observe(pageRead("call_1", map[string]any{"status": "Ready"},
		assistanttest.QueuePage(0, 2, map[string]any{"hasMore": true, "nextCursor": "c-2"})))
	require.NoError(t, err)
	_, err = recorder.Observe(pageRead("call_2",
		map[string]any{"status": "Ready", "after": "c-9"},
		assistanttest.QueuePage(8, 2, map[string]any{"hasMore": false})))
	require.NoError(t, err)

	require.Len(t, repo.Upserts, 2)
	assert.NotEqual(t, repo.Upserts[0].ID, repo.Upserts[1].ID)
	assert.Equal(t, repo.Upserts[0].ID, repo.Upserts[1].LineageID)
	assert.Equal(t, 2, repo.Upserts[1].LineageSeq)
}

func TestArtifactRecorder_APageThatWouldNotFitLeavesTheTableAsItIs(t *testing.T) {
	t.Parallel()

	repo := &assistanttest.StubArtifactRepo{}
	recorder := newPagingRecorder(t, repo)
	_, err := recorder.Observe(pageRead("call_1", map[string]any{},
		assistanttest.QueuePage(0, 2, map[string]any{"hasMore": true, "nextCursor": "c-2"})))
	require.NoError(t, err)

	long := strings.Repeat("x", assistantartifact.MaxPayloadBytes)
	page := assistanttest.QueuePage(2, 1, map[string]any{"hasMore": true, "nextCursor": "c-3"})
	page["items"].([]any)[0].(map[string]any)["status"] = long
	shown, err := recorder.Observe(pageRead("call_2", map[string]any{"after": "c-2"}, page))
	require.NoError(t, err)
	require.NotNil(t, shown)

	require.Len(t, repo.Upserts, 1)
	assert.Equal(t, repo.Upserts[0].ID, shown.ID)
	assert.Equal(t, "c-2", repo.Upserts[0].Payload[artifactpayload.PayloadNextCursor])
	assert.Equal(t, []string{"BQ-A", "BQ-B"}, rowNumbers(repo.Upserts[0]))
}

func TestArtifactRecorder_AShortPagedTableStaysWhetherOrNotTheReplyLinkedIt(t *testing.T) {
	t.Parallel()

	repo := &assistanttest.StubArtifactRepo{}
	recorder := newPagingRecorder(t, repo)
	_, err := recorder.Observe(pageRead("call_1", map[string]any{},
		assistanttest.QueuePage(0, 2, map[string]any{"hasMore": true, "nextCursor": "c-2"})))
	require.NoError(t, err)

	recorder.KeepLinked(map[string]bool{})

	assert.Len(t, recorder.Recorded, 1)
}

func TestArtifactRecorder_ATableKeepsTheArgumentsItsListRanWith(t *testing.T) {
	t.Parallel()

	repo := &assistanttest.StubArtifactRepo{}
	recorder := newPagingRecorder(t, repo)
	observation := pageRead(
		"call_1",
		map[string]any{"status": "ready for review"},
		assistanttest.QueuePage(0, 2, map[string]any{"hasMore": true, "nextCursor": "c-2"}),
	)
	observation.RanWith = map[string]any{"status": []any{"ReadyForReview"}}

	_, err := recorder.Observe(observation)
	require.NoError(t, err)

	require.Len(t, repo.Upserts, 1)
	assert.Equal(t,
		map[string]any{"status": []any{"ReadyForReview"}},
		repo.Upserts[0].Payload[artifactpayload.PayloadReadWith],
	)
	observation.RanWith["status"] = []any{"Posted"}
	assert.Equal(t,
		map[string]any{"status": []any{"ReadyForReview"}},
		repo.Upserts[0].Payload[artifactpayload.PayloadReadWith],
		"the table keeps its own copy",
	)
}

func newReaderRecorder(
	t *testing.T,
	repo *assistanttest.StubArtifactRepo,
	agents *assistanttest.DelegateDefinitions,
	agentID pulid.ID,
) *artifactrecorder.Recorder {
	t.Helper()

	svc := &Service{logger: zap.NewNop(), artifacts: repo, definitions: agents}

	return svc.newArtifactRecorder(
		t.Context(),
		&conversation.Thread{ID: pulid.MustNew("athr_"), AgentDefinitionID: agentID},
		pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		assistanttest.Actor(),
		nil,
	)
}

func TestArtifactRecorder_ATableNamesTheAgentThatReadIt(t *testing.T) {
	t.Parallel()

	agentID := pulid.MustNew("agdef_")
	agents := &assistanttest.DelegateDefinitions{ByID: map[pulid.ID]*agentdefinition.Definition{
		agentID: {ID: agentID, Name: "Dispatch Desk", Accent: "teal"},
	}}
	repo := &assistanttest.StubArtifactRepo{}
	recorder := newReaderRecorder(t, repo, agents, agentID)

	for at, call := range []string{"call_1", "call_2"} {
		_, err := recorder.Observe(pageRead(
			call,
			map[string]any{"status": call},
			assistanttest.QueuePage(at*2, 2, map[string]any{"hasMore": true, "nextCursor": call}),
		))
		require.NoError(t, err)
	}

	require.Len(t, repo.Upserts, 2)
	want := attribution.ArtifactReader{ID: agentID.String(), Name: "Dispatch Desk", Accent: "teal"}
	assert.Equal(t, want, repo.Upserts[0].Payload[toolartifacts.PayloadReadBy])
	assert.Equal(t, want, repo.Upserts[1].Payload[toolartifacts.PayloadReadBy])
	assert.Equal(t, 1, agents.Listed, "the agent is read once for the turn")
}

func TestArtifactRecorder_ATableDoesNotNameAnAgentItCannotRead(t *testing.T) {
	t.Parallel()

	agents := &assistanttest.DelegateDefinitions{ByID: map[pulid.ID]*agentdefinition.Definition{}}
	repo := &assistanttest.StubArtifactRepo{}
	recorder := newReaderRecorder(t, repo, agents, pulid.MustNew("agdef_"))

	_, err := recorder.Observe(pageRead(
		"call_1",
		map[string]any{},
		assistanttest.QueuePage(0, 2, map[string]any{"hasMore": true, "nextCursor": "c-2"}),
	))
	require.NoError(t, err)

	require.Len(t, repo.Upserts, 1)
	assert.NotContains(t, repo.Upserts[0].Payload, toolartifacts.PayloadReadBy)
}

func TestArtifactRecorder_AReportRunNamesTheAgentThatRanIt(t *testing.T) {
	t.Parallel()

	agentID := pulid.MustNew("agdef_")
	agents := &assistanttest.DelegateDefinitions{ByID: map[pulid.ID]*agentdefinition.Definition{
		agentID: {ID: agentID, Name: "Billing Desk", Accent: "violet"},
	}}
	repo := &assistanttest.StubArtifactRepo{}
	recorder := newReaderRecorder(t, repo, agents, agentID)

	_, err := recorder.Observe(agentruntimeports.ToolObservation{
		Call: llmports.ToolCall{
			ID:        "call_run",
			Name:      toolartifacts.ToolRunReport,
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

	require.Len(t, repo.Upserts, 1)
	assert.Equal(t, assistantartifact.KindReportRun, repo.Upserts[0].Kind)
	assert.Equal(
		t,
		attribution.ArtifactReader{ID: agentID.String(), Name: "Billing Desk", Accent: "violet"},
		repo.Upserts[0].Payload[toolartifacts.PayloadReadBy],
	)
}
