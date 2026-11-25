package runtimetools

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolresult"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	driverOne    = "wrk_01M7V48DVVQGDF0KG409BJH1FJ"
	driverTwo    = "wrk_01M7V48DVVQGDF0KG409BJH1FK"
	workerTable  = "art_01M861HFM3CMMC4RFVB9KTAFWK"
	newShipments = "art_01M861HFM3CMMC4RFVB9KTASHP"
)

func keptAs(title, table string) string {
	return "\n\n[This result is kept as a table titled \"" + title + "\" beside the " +
		"conversation. To point to it, write [" + title + "](artifact:" + table + ") as part " +
		"of the sentence.]"
}

func newShipmentsRead() llmports.Message {
	return llmports.Message{
		Role: llmports.RoleTool, ToolName: "records_query",
		Content: toolresult.Fence("records_query", `{"count":2,"hasMore":true,"total":159,`+
			`"items":[{"id":"`+loadEight+`","proNumber":"SEED-SHP-008"},`+
			`{"id":"`+loadTen+`","proNumber":"SEED-SHP-010"}]}`) +
			keptAs("New shipments", newShipments),
	}
}

func activeWorkersRead(total string) llmports.Message {
	return llmports.Message{
		Role: llmports.RoleTool, ToolName: "records_query",
		Content: toolresult.Fence("records_query", `{"count":2,`+total+`"items":[`+
			`{"id":"`+driverOne+`","name":"Ana Ruiz"},`+
			`{"id":"`+driverTwo+`","name":"Ben Ode"}]}`) + keptAs("Active workers", workerTable),
	}
}

func driversAnswer(answer string) map[string]any {
	return map[string]any{
		"title":  "Drivers for the new shipments",
		"answer": answer,
		"groups": []any{
			map[string]any{"title": "Ranked", "records": []any{driverOne, driverTwo}},
		},
	}
}

func TestShowAnswer_ReadsAllFromTheListOfTheKindItsWordsCount(t *testing.T) {
	t.Parallel()

	paged := `"hasMore":true,"total":372,`
	whole := `"hasMore":false,"total":2,`
	for _, tc := range []struct {
		name     string
		reads    []llmports.Message
		args     map[string]any
		list     pulid.ID
		records  int
		refusal  string
		mentions string
	}{
		{
			name:  "shipments counted over a board of workers read the shipment list",
			reads: []llmports.Message{newShipmentsRead(), activeWorkersRead(paged)},
			args: driversAnswer("There are {all} new shipments waiting for a driver; " +
				"these {count} drivers fit best."),
			list: newShipments,
		},
		{
			name:  "drivers counted over a board of workers read the worker list",
			reads: []llmports.Message{newShipmentsRead(), activeWorkersRead(paged)},
			args:  driversAnswer("{all} drivers are active; these {count} fit best."),
			list:  workerTable,
		},
		{
			name:  "shipments counted with no shipment list read are refused",
			reads: []llmports.Message{activeWorkersRead(paged)},
			args: driversAnswer("There are {all} new shipments waiting for a driver; " +
				"these {count} drivers fit best."),
			refusal: `Tool "show_answer" was not run: {all} reads the whole list of the ` +
				`records your groups hold, which are workers (372 read), but the words count ` +
				`shipments. This conversation read no list of shipments that reported its ` +
				`total: read it first, or name the shipments in a group.`,
		},
		{
			name:  "shipments counted over the workers shown are never the shown count",
			reads: []llmports.Message{activeWorkersRead(whole)},
			args: driversAnswer("The four planned batches cover {all} new shipments; " +
				"these {count} drivers take them."),
			mentions: "which are workers (2 shown), but the words count shipments",
		},
		{
			name: "two kinds counted with one {all} are refused",
			reads: []llmports.Message{
				newShipmentsRead(), activeWorkersRead(paged),
			},
			args: driversAnswer("{all} drivers are active and {all} loads are new; " +
				"these {count} fit best."),
			mentions: "the words count both drivers and loads with it",
		},
		{
			name:    "no noun keeps the records shown when the list was read whole",
			reads:   []llmports.Message{activeWorkersRead(whole)},
			args:    driversAnswer("All {all} of them fit."),
			records: 2,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			outcome := showAnswer(&answerTurn{messages: tc.reads}, tc.args)

			switch {
			case tc.refusal != "":
				require.True(t, outcome.Failed)
				assert.Equal(t, tc.refusal, outcome.Content)
				return
			case tc.mentions != "":
				require.True(t, outcome.Failed)
				assert.Contains(t, outcome.Content, tc.mentions)
				return
			}
			require.False(t, outcome.Failed, outcome.Content)
			request, ok := outcome.Data.(agentruntimeports.BoardRequest)
			require.True(t, ok)
			all := request.Board.Reading("all")
			require.NotNil(t, all)
			if tc.records > 0 {
				require.NotNil(t, all.Records)
				assert.Len(t, all.Records.IDs, tc.records)
				return
			}
			assert.Equal(t, tc.list, all.List)
			assert.Nil(t, all.Records)
			assert.Contains(t, outcome.RanWith["answer"], "{all:"+tc.list.String()+"}")
		})
	}
}

func TestShowAnswer_ReadsAllFromTheBoardsOwnListWhenItsWordsCountItsRecords(t *testing.T) {
	t.Parallel()

	turn := pagedTurnKept(`"total":125,`, keptNote())
	turn.messages = append(turn.messages, activeWorkersRead(`"hasMore":true,"total":372,`))

	outcome := showAnswer(turn, lateAnswer("{all} loads are late, and these two are the "+
		"furthest behind."))

	require.False(t, outcome.Failed, outcome.Content)
	request, ok := outcome.Data.(agentruntimeports.BoardRequest)
	require.True(t, ok)
	assert.Equal(t, pulid.ID(lateList), request.Board.Reading("all").List)
}
