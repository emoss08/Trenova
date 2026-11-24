package boardcardservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeLists struct {
	sizes map[pulid.ID]int
	asked []*assistantports.ListSizeRequest
}

func (f *fakeLists) ListSize(
	_ context.Context,
	req *assistantports.ListSizeRequest,
) (*assistantports.ListSize, error) {
	f.asked = append(f.asked, req)
	total, ok := f.sizes[req.ArtifactID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Artifact not found")
	}

	return &assistantports.ListSize{Title: "Shipments", Kind: "shipment", Total: total}, nil
}

func (f *fixture) keptTable(threadID pulid.ID) pulid.ID {
	id := pulid.MustNew("art_")
	f.artifacts.byID[id] = &assistantartifact.Artifact{
		ID: id, ThreadID: threadID, Kind: assistantartifact.KindTableView, Title: "Shipments",
		Payload: map[string]any{"tool": "shipment_list", "rows": []any{}},
	}

	return id
}

func wholeListBoard(t *testing.T, list pulid.ID) *board.Board {
	t.Helper()

	return parsed(t, map[string]any{
		"title": "Late loads", "place": "conversation", "verdict": "{all} loads are late.",
		"readings": []any{map[string]any{
			"key": "all", "label": "Shipments in the whole list", "list": list.String(),
		}},
		"blocks": []any{map[string]any{
			"type": "records", "title": "Furthest behind",
			"records": []any{pulid.MustNew("shp_").String()},
		}},
	})
}

func TestFillReadsAListsWholeSizeAgainEachTimeTheBoardIsRead(t *testing.T) {
	t.Parallel()

	f := newFixture()
	list := f.keptTable(f.thread)
	lists := &fakeLists{sizes: map[pulid.ID]int{list: 125}}
	ReadListsWith(f.svc, lists)
	request := f.request(wholeListBoard(t, list))
	request.Agent = &assistantports.BoardAgentRead{}

	filled, err := f.svc.Fill(t.Context(), request)
	require.NoError(t, err)
	assert.Equal(t, "125", board.FormatValue(filled.Reading("all").Value))

	lists.sizes[list] = 131
	filled, err = f.svc.Fill(t.Context(), request)
	require.NoError(t, err)
	assert.Equal(t, "131", board.FormatValue(filled.Reading("all").Value))
	require.Len(t, lists.asked, 2)
	assert.Equal(t, f.actor, lists.asked[0].Actor, "the list is read as whoever reads the board")
	assert.Equal(t, f.thread, lists.asked[0].ThreadID)
	assert.Same(t, request.Agent, lists.asked[0].Agent, "an agent's read keeps its data access")

	delete(lists.sizes, list)
	filled, err = f.svc.Fill(t.Context(), request)
	require.NoError(t, err)
	assert.Nil(t, filled.Reading("all").Value, "a list that cannot be counted is withheld")
}

func TestCheckRefusesAListThatIsNotATableThisConversationKept(t *testing.T) {
	t.Parallel()

	f := newFixture()
	ReadListsWith(f.svc, &fakeLists{})
	elsewhere := f.keptTable(pulid.MustNew("thr_"))

	err := f.svc.Check(t.Context(), f.request(wholeListBoard(t, elsewhere)))

	var mistake *agentruntimeports.PublishMistake
	require.ErrorAs(t, err, &mistake)
	assert.Contains(t, mistake.Reason, "which is not a table this conversation kept")

	require.NoError(t, f.svc.Check(t.Context(), f.request(wholeListBoard(t, f.keptTable(f.thread)))))
}
