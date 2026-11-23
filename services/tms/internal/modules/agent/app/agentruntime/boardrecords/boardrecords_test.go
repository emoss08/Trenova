package boardrecords_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/boardrecords"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolresult"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	shp008 = "shp_01M7V48EQGE9TBM60WEQJJGQTR"
	shp010 = "shp_01M7V48EQGE9TBM60WF4JKM3VQ"
	shp001 = "shp_01M7V48EQGE9TBM60WDH9D4TRN"
)

func incidentRead() boardrecords.Read {
	return boardrecords.Of([]string{
		toolresult.Fence("shipment_get", `{"count":2,"items":[`+
			`{"id":"`+shp008+`","proNumber":"SEED-SHP-008","customer":"Sunbelt Materials",`+
			`"totalCharge":"2400"},`+
			`{"id":"shp_01M7V48EQGE9TBM60WDXP5B6ZJ","proNumber":"SEED-SHP-004"}]}`),
		toolresult.Fence("shipment_get", `{"id":"`+shp001+`","proNumber":"SEED-SHP-001",`+
			`"customer":{"id":"cus_01M7V48ENWADFPH2DR2VN9TFYM","name":"Acme Manufacturing"}}`),
		toolresult.Fence("shipment_list", `{"count":1,"items":[`+
			`{"id":"`+shp010+`","proNumber":"SEED-SHP-010","totalCharge":"4925"}]}`),
	})
}

func shown(ids ...string) map[permission.Resource][]pulid.ID {
	out := make([]pulid.ID, 0, len(ids))
	for _, id := range ids {
		out = append(out, pulid.ID(id))
	}

	return map[permission.Resource][]pulid.ID{permission.ResourceShipment: out}
}

func TestSwapFor_PutsTheNamedRecordInPlaceOfTheOneNoWordNames(t *testing.T) {
	t.Parallel()

	read := incidentRead()
	words := "SEED-SHP-010 and SEED-SHP-008 compared\n\nSEED-SHP-010 is further behind."
	board := shown(shp010, shp001)

	unshown := read.Unshown(board, words)
	require.Len(t, unshown, 1)
	assert.Equal(t, shp008, unshown[0].ID)
	assert.Equal(t, "SEED-SHP-008", unshown[0].Label)

	swap, ok := read.SwapFor(board, words, unshown)
	require.True(t, ok)
	assert.Equal(t, shp001, swap.From.ID)
	assert.Equal(t, shp008, swap.To.ID)

	args := map[string]any{
		"title": "SEED-SHP-010 and SEED-SHP-008 compared",
		"groups": []any{
			map[string]any{"title": "Further behind", "records": []any{shp010}},
			map[string]any{"title": "Less behind", "records": []any{shp001}},
		},
		"actions": []any{map[string]any{"label": "Call", "records": []any{shp001, shp010}}},
	}
	assert.Equal(t, map[string]any{
		"title": "SEED-SHP-010 and SEED-SHP-008 compared",
		"groups": []any{
			map[string]any{"title": "Further behind", "records": []any{shp010}},
			map[string]any{"title": "Less behind", "records": []any{shp008}},
		},
		"actions": []any{map[string]any{"label": "Call", "records": []any{shp008, shp010}}},
	}, swap.Apply(args))
	assert.Equal(t, []any{shp001}, args["groups"].([]any)[1].(map[string]any)["records"],
		"the model's own arguments are left as it sent them")
}

func TestSwapFor_RefusesWhatIsNotOneForOne(t *testing.T) {
	t.Parallel()

	read := incidentRead()
	tests := []struct {
		name  string
		words string
		board map[permission.Resource][]pulid.ID
	}{
		{
			name:  "every shown record is named",
			words: "SEED-SHP-010 and SEED-SHP-001, unlike SEED-SHP-008",
			board: shown(shp010, shp001),
		},
		{
			name:  "two shown records go unnamed",
			words: "Late loads, not SEED-SHP-008",
			board: shown(shp010, shp001),
		},
		{
			name:  "two records named and not shown",
			words: "SEED-SHP-008 and SEED-SHP-004 against SEED-SHP-010",
			board: shown(shp010, shp001),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			unshown := read.Unshown(tt.board, tt.words)
			require.NotEmpty(t, unshown)
			_, ok := read.SwapFor(tt.board, tt.words, unshown)
			assert.False(t, ok)
		})
	}
}

func TestUnshown_NamesOnlyRecordsOfAKindTheBoardShows(t *testing.T) {
	t.Parallel()

	read := incidentRead()

	assert.Empty(t, read.Unshown(shown(shp010, shp008), "SEED-SHP-010 against SEED-SHP-008"))
	assert.Empty(t, read.Unshown(shown(shp010), "SEED-SHP-0100 is not SEED-SHP-01"),
		"a number inside a longer one is not that number")
	assert.Empty(t, read.Unshown(
		map[permission.Resource][]pulid.ID{
			permission.ResourceInvoice: {pulid.ID("inv_01M7V48JZXABMWM4E71Z778WBW")},
		},
		"The invoice for SEED-SHP-008",
	), "a shipment named beside invoices is not a shipment the board leaves out")
}

func TestRefusal_NamesTheNumberAndTheIdToSend(t *testing.T) {
	t.Parallel()

	read := incidentRead()
	unshown := read.Unshown(shown(shp010), "SEED-SHP-010 against SEED-SHP-008")

	assert.Equal(t, `Tool "show_answer" was not run: its words name SEED-SHP-008 (the shipment `+
		shp008+`), but no group holds it, so the answer would speak of a record it does not `+
		`show. Put each id in the group it belongs to, in place of any record you did not mean, `+
		`or take the number out of the words.`,
		boardrecords.Refusal("show_answer", "group", unshown))
}

func TestNumbers_ReadsARecordNumberAsTheRecordAResultGaveIt(t *testing.T) {
	t.Parallel()

	read := incidentRead()
	args := map[string]any{
		"groups": []any{
			map[string]any{"title": "Late", "records": []any{"SEED-SHP-008", shp010}},
			map[string]any{"title": "Unknown", "records": []any{"SEED-SHP-099"}},
		},
		"blocks": []any{map[string]any{"type": "timeline", "record": "seed-shp-001"}},
		"title":  "SEED-SHP-004",
	}

	got, changed := read.Numbers(args)

	require.True(t, changed)
	assert.Equal(t, map[string]any{
		"groups": []any{
			map[string]any{"title": "Late", "records": []any{shp008, shp010}},
			map[string]any{"title": "Unknown", "records": []any{"SEED-SHP-099"}},
		},
		"blocks": []any{map[string]any{"type": "timeline", "record": shp001}},
		"title":  "SEED-SHP-004",
	}, got)

	_, changed = read.Numbers(map[string]any{"groups": []any{
		map[string]any{"records": []any{shp010}},
	}})
	assert.False(t, changed)
}

func TestNumbers_ReadsANumberARowGaveBesideTheRecordsId(t *testing.T) {
	t.Parallel()

	read := boardrecords.Of([]string{
		toolresult.Fence("dispatch_get_board", `{"moves":[{"moveId":"sm_01M7WFY86G4P62W4BAWQ1RSFS6",`+
			`"proNumber":"S26110001005205","shipmentId":"shp_01M7WFY85YE86D9WBGRY4PYH10"}]}`),
		toolresult.Fence("receivable_list_open_items", `{"items":[{"invoiceId":`+
			`"inv_01JPEAK0000000000000000005","invoiceNumber":"SEED-INV-0005"}]}`),
	})

	got, changed := read.Numbers(map[string]any{"groups": []any{
		map[string]any{"records": []any{"S26110001005205"}},
		map[string]any{"records": []any{"SEED-INV-0005"}},
	}})

	require.True(t, changed)
	assert.Equal(t, map[string]any{"groups": []any{
		map[string]any{"records": []any{"shp_01M7WFY85YE86D9WBGRY4PYH10"}},
		map[string]any{"records": []any{"inv_01JPEAK0000000000000000005"}},
	}}, got)
}

func TestDecodeLists_ReadsAListSentAsJSONText(t *testing.T) {
	t.Parallel()

	args := map[string]any{
		"title":  "Peak",
		"groups": `[{"title": "Past due", "records": ["inv_01JPEAK0000000000000000005"]}]`,
	}

	got, changed := boardrecords.DecodeLists(args, "groups", "actions")

	require.True(t, changed)
	assert.Equal(t, map[string]any{
		"title": "Peak",
		"groups": []any{map[string]any{
			"title": "Past due", "records": []any{"inv_01JPEAK0000000000000000005"},
		}},
	}, got)
	assert.IsType(t, "", args["groups"], "the model's own arguments are left as it sent them")

	_, changed = boardrecords.DecodeLists(map[string]any{"groups": "not json"}, "groups")
	assert.False(t, changed)
}

func TestFigureNumbers_ReadsARecordsNumberInAFigure(t *testing.T) {
	t.Parallel()

	read := incidentRead()
	got, changed := read.FigureNumbers(map[string]any{
		"answer": "SEED-SHP-010 brings in {revenue:SEED-SHP-010}, against {SEED-SHP-008:revenue}; " +
			"{total} and {revenue:SEED-SHP-099} stay.",
		"groups": []any{map[string]any{"note": "Late by {lateness:seed-shp-001}."}},
	})

	require.True(t, changed)
	assert.Equal(t, map[string]any{
		"answer": "SEED-SHP-010 brings in {revenue:" + shp010 + "}, against {revenue:" + shp008 +
			"}; {total} and {revenue:SEED-SHP-099} stay.",
		"groups": []any{map[string]any{"note": "Late by {lateness:" + shp001 + "}."}},
	}, got)
}
