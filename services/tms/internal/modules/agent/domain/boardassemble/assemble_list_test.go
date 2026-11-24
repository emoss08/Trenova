package boardassemble_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardassemble"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardparse"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const keptList = pulid.ID("art_01M861HFM3CMMC4RFVB9KTAFMT")

func listBoard(t *testing.T) *board.Board {
	t.Helper()

	parsed, problems := boardparse.Parse(map[string]any{
		"title":   "Late loads right now",
		"place":   "conversation",
		"verdict": "{all} loads are late.",
		"readings": []any{map[string]any{
			"key": "all", "label": "Shipments in the whole list", "list": keptList.String(),
		}},
		"blocks": []any{map[string]any{
			"type": "records", "title": "The furthest behind", "records": ids("shp_", 2),
		}},
	})
	require.Empty(t, problems)
	assert.Equal(t, []pulid.ID{keptList}, parsed.Lists())

	return parsed
}

func listReads(read *boardassemble.ListRead) *boardassemble.Reads {
	return &boardassemble.Reads{
		Lists: map[pulid.ID]*boardassemble.ListRead{keptList: read},
	}
}

func TestAssembleReadsAListsWholeSizeFromTheTableItWasKeptAs(t *testing.T) {
	t.Parallel()

	total := 125
	filled := boardassemble.Assemble(listBoard(t), listReads(&boardassemble.ListRead{
		Title: "Shipments", Kind: "shipment", Total: &total,
	}))

	all := filled.Reading("all")
	require.NotNil(t, all)
	assert.Equal(t, "125", board.FormatValue(all.Value))
	assert.Equal(t, board.AggregateCount, all.Aggregate)
	assert.Equal(t, board.Source{Kind: board.SourceTable, Title: "Shipments",
		RecordKind: "shipment"}, all.Source)
	assert.Equal(t, "125", filled.Verdict[0].Text)
}

func TestAssembleWithholdsAListItCouldNotCount(t *testing.T) {
	t.Parallel()

	filled := boardassemble.Assemble(listBoard(t), listReads(&boardassemble.ListRead{}))

	all := filled.Reading("all")
	require.NotNil(t, all)
	assert.Nil(t, all.Value, "a list not counted is withheld, never zero")
}
