package boardanswer_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardanswer"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	held     = []any{"shp_01JHELD00000000000000001", "shp_01JHELD00000000000000002"}
	dock     = []any{"shp_01JDOCK00000000000000001"}
	docks    = []any{"shp_01JDOCK00000000000000001", "shp_01JDOCK00000000000000002"}
	invoices = []any{"inv_01JINVO00000000000000001", "inv_01JINVO00000000000000002"}
)

func oneGroup() map[string]any {
	return map[string]any{
		"title":   "Late loads on the Dallas lanes",
		"answer":  "There are {count} late loads, the furthest {largest} behind.",
		"measure": "lateness",
		"groups": []any{map[string]any{
			"title":   "Waiting on one driver's hours",
			"records": held,
			"note":    "Marcus Bell is out of hours until 9:15 AM.",
		}},
	}
}

func twoGroups() map[string]any {
	args := oneGroup()
	args["measure"] = "revenue"
	args["answer"] = "{count_1} wait on one driver's hours and {count_2} are late at the dock, " +
		"worth {total} in all."
	args["groups"] = []any{
		map[string]any{"title": "Waiting on one driver's hours", "records": held},
		map[string]any{
			"title":   "Late at the dock",
			"records": docks,
			"note":    "The largest is {largest_2}.",
		},
	}

	return args
}

func translated(t *testing.T, args map[string]any) *boardanswer.Answer {
	t.Helper()
	answer, problems := boardanswer.Translate(args)
	require.Empty(t, problems)
	require.NotNil(t, answer)

	return answer
}

func keys(readings []board.Reading) []string {
	out := make([]string, 0, len(readings))
	for idx := range readings {
		out = append(out, readings[idx].Key)
	}

	return out
}

func TestTranslate_MakesOneGroupABoardInTheConversation(t *testing.T) {
	t.Parallel()

	answer := translated(t, oneGroup())
	layout := answer.Board

	assert.Equal(t, board.PlaceConversation, layout.Place)
	assert.Equal(t, "Late loads on the Dallas lanes", layout.Title)
	assert.Equal(t, "There are {count} late loads, the furthest {largest} behind.", layout.Verdict)
	require.Len(t, layout.Blocks, 1)
	block := layout.Blocks[0]
	assert.Equal(t, board.BlockRecords, block.Type)
	assert.False(t, block.Half)
	assert.Equal(t, board.Measure("lateness"), block.Measure)
	assert.Equal(t, board.OrderMeasure, block.Order)
	assert.Equal(t, "Marcus Bell is out of hours until 9:15 AM.", block.Caption)
	assert.Equal(t, recordkind.Kind(permission.ResourceShipment), block.Records.Kind)
	assert.Len(t, block.Records.IDs, 2)
	assert.Equal(t, []string{"count", "largest"}, keys(layout.Readings))
	assert.Equal(t, board.AggregateCount, layout.Readings[0].Aggregate)
	assert.Empty(t, layout.Readings[0].Measure, "count counts every record")
	assert.Equal(t, "Shipments", layout.Readings[0].Label)
	assert.Equal(t, board.AggregateMax, layout.Readings[1].Aggregate)
	assert.Equal(t, board.Measure("lateness"), layout.Readings[1].Measure)
	assert.Equal(t, "Furthest behind", layout.Readings[1].Label)
}

func TestTranslate_SetsTwoGroupsSideBySide(t *testing.T) {
	t.Parallel()

	args := twoGroups()
	args["actions"] = []any{
		map[string]any{"label": "Ask the Memphis dock", "ask": "Ask the dock for a time."},
	}
	layout := translated(t, args).Board

	require.Len(t, layout.Blocks, 3)
	assert.True(t, layout.Blocks[0].Half)
	assert.True(t, layout.Blocks[1].Half)
	assert.Equal(t, "The largest is {largest_2}.", layout.Blocks[1].Caption)
	assert.Equal(t, board.BlockActions, layout.Blocks[2].Type)
	assert.Equal(t, "Ask the Memphis dock", layout.Blocks[2].Actions[0].Label)
	assert.Equal(t, []string{"total", "count_1", "count_2", "largest_2"}, keys(layout.Readings))
	assert.Len(t, layout.Readings[0].Records.IDs, 4, "total adds up both groups")
	assert.Len(t, layout.Readings[1].Records.IDs, 2)
	assert.Len(t, layout.Readings[2].Records.IDs, 2)
	assert.Equal(t, "Shipments, Late at the dock", layout.Readings[2].Label)
	assert.Equal(t, board.AggregateMax, layout.Readings[3].Aggregate)
}

func TestTranslate_DeclaresEachNameOfTheVocabulary(t *testing.T) {
	t.Parallel()

	want := map[string]struct {
		aggregate board.Aggregate
		records   int
	}{
		"count": {board.AggregateCount, 4}, "all": {board.AggregateCount, 4},
		"total":   {board.AggregateSum, 4},
		"largest": {board.AggregateMax, 4}, "count_1": {board.AggregateCount, 2},
		"count_2": {board.AggregateCount, 2}, "total_1": {board.AggregateSum, 2},
		"total_2": {board.AggregateSum, 2}, "largest_1": {board.AggregateMax, 2},
		"largest_2": {board.AggregateMax, 2},
	}
	require.ElementsMatch(t, boardanswer.Names(2), keysOf(want))
	for _, name := range boardanswer.Names(2) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			args := twoGroups()
			args["answer"] = "It is {" + name + "}."
			args["groups"].([]any)[1].(map[string]any)["note"] = ""
			layout := translated(t, args).Board
			require.Len(t, layout.Readings, 1)
			assert.Equal(t, name, layout.Readings[0].Key)
			assert.Equal(t, want[name].aggregate, layout.Readings[0].Aggregate)
			assert.Len(t, layout.Readings[0].Records.IDs, want[name].records)
		})
	}
}

func keysOf[V any](values map[string]V) []string {
	out := make([]string, 0, len(values))
	for key := range values {
		out = append(out, key)
	}

	return out
}

func TestTranslate_RepairsWhatItCan(t *testing.T) {
	t.Parallel()

	t.Run("a measure the records do not have is dropped", func(t *testing.T) {
		t.Parallel()

		args := oneGroup()
		args["answer"] = "There are {count} late loads."
		args["measure"] = "amountDue"
		layout := translated(t, args).Board
		assert.Empty(t, layout.Blocks[0].Measure)
		assert.Equal(t, board.OrderGiven, layout.Blocks[0].Order)
	})
	t.Run("an unknown field is ignored", func(t *testing.T) {
		t.Parallel()

		args := oneGroup()
		args["verdict"] = "Not read."
		args["groups"].([]any)[0].(map[string]any)["tone"] = "bad"
		translated(t, args)
	})
	t.Run("an empty second group is dropped", func(t *testing.T) {
		t.Parallel()

		args := oneGroup()
		args["groups"] = append(args["groups"].([]any),
			map[string]any{"title": "Nothing here", "records": []any{}})
		layout := translated(t, args).Board
		require.Len(t, layout.Blocks, 1)
		assert.False(t, layout.Blocks[0].Half)
	})
	t.Run("count_1 is the one group", func(t *testing.T) {
		t.Parallel()

		args := oneGroup()
		args["answer"] = "There are {count_1} late loads."
		layout := translated(t, args).Board
		assert.Equal(t, []string{"count_1"}, keys(layout.Readings))
	})
	t.Run("actions past two and incomplete ones are left off", func(t *testing.T) {
		t.Parallel()

		args := oneGroup()
		args["actions"] = []any{
			map[string]any{"label": "No ask"},
			map[string]any{"label": "Call the dock", "ask": "Call the dock."},
			map[string]any{"label": "Call the driver", "ask": "Call the driver."},
			map[string]any{"label": "Call the customer", "ask": "Call the customer."},
		}
		actions := translated(t, args).Board.Blocks[1].Actions
		require.Len(t, actions, 2)
		assert.Equal(t, "Call the dock", actions[0].Label)
		assert.Equal(t, "Call the driver", actions[1].Label)
	})
}

func TestTranslate_RefusesWhatItCannotRepair(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		change func(map[string]any)
		want   string
	}{
		"no title, and no group's title to take": {
			change: func(args map[string]any) {
				delete(args, "title")
				delete(args["groups"].([]any)[0].(map[string]any), "title")
			},
			want: "title is required",
		},
		"no answer": {
			change: func(args map[string]any) { args["answer"] = "  " },
			want:   "answer is required",
		},
		"no groups": {
			change: func(args map[string]any) { delete(args, "groups") },
			want:   "groups needs 1 to 3 groups",
		},
		"no records": {
			change: func(args map[string]any) {
				args["groups"] = []any{map[string]any{"title": "Late", "records": []any{}}}
			},
			want: "groups[1].records needs 1 to 25 record ids",
		},
		"four groups": {
			change: func(args map[string]any) {
				group := map[string]any{"records": dock}
				args["groups"] = []any{group, group, group, group}
			},
			want: "groups holds at most 3 groups",
		},
		"a third group's name with two groups": {
			change: func(args map[string]any) {
				args["answer"] = "There are {count_3}."
				args["groups"] = []any{
					map[string]any{"records": held}, map[string]any{"records": dock},
				}
			},
			want: "answer places {count_3}",
		},
		"mixed kinds": {
			change: func(args map[string]any) {
				args["answer"] = "There are {count}."
				args["groups"] = []any{
					map[string]any{"records": held},
					map[string]any{"records": invoices},
				}
			},
			want: "groups[2].records mixes shipments and invoices",
		},
		"a kind a board does not take": {
			change: func(args map[string]any) {
				args["groups"] = []any{
					map[string]any{"records": []any{"doc_01JDOCS00000000000000001"}},
				}
			},
			want: "which is not a shipment",
		},
		"not a record id": {
			change: func(args map[string]any) {
				args["groups"] = []any{map[string]any{"records": []any{"SEED-SHP-008"}}}
			},
			want: `holds "SEED-SHP-008", which is not a record id`,
		},
		"an answer over its limit": {
			change: func(args map[string]any) { args["answer"] = strings.Repeat("late ", 121) },
			want:   "answer is longer than 600 characters",
		},
		"a note over its limit": {
			change: func(args map[string]any) {
				args["groups"].([]any)[0].(map[string]any)["note"] = strings.Repeat("held ", 61)
			},
			want: "groups[1].note is longer than 300 characters",
		},
		"an unknown name": {
			change: func(args map[string]any) { args["answer"] = "There are {late} late loads." },
			want:   "answer places {late}, which is not one of {count}, {all}, {total}, {largest}",
		},
		"a second group's name with one group": {
			change: func(args map[string]any) { args["answer"] = "There are {count_2}." },
			want:   "answer places {count_2}",
		},
		"a name in the title": {
			change: func(args map[string]any) { args["title"] = "The {count} late loads" },
			want:   "title holds {count}",
		},
		"a total with no measure": {
			change: func(args map[string]any) {
				delete(args, "measure")
				args["answer"] = "They are {total} behind."
			},
			want: "{total} adds up the measure, so set measure to one of lateness or revenue",
		},
		"a largest with a measure the records do not have": {
			change: func(args map[string]any) { args["measure"] = "amountDue" },
			want:   `a shipment has no measure "amountDue"`,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			args := oneGroup()
			tc.change(args)
			answer, problems := boardanswer.Translate(args)
			assert.Nil(t, answer)
			require.NotEmpty(t, problems)
			assert.Contains(t, strings.Join(problems, "; "), tc.want)
		})
	}
}

func TestTranslate_HandsBackItsWordsForTheFigureCheck(t *testing.T) {
	t.Parallel()

	args := oneGroup()
	args["answer"] = "Peak owes $6,175.00 on {count} invoices, 52% of it past 90 days."
	answer := translated(t, args)

	assert.Equal(t, "Late loads on the Dallas lanes\n\n"+args["answer"].(string)+
		"\n\nWaiting on one driver's hours\n\nMarcus Bell is out of hours until 9:15 AM.",
		answer.Words)
	assert.Equal(t, args["answer"], answer.Board.Verdict, "a typed figure stands as plain text")
}

func TestTranslate_ReadsAMeasureByTheFieldNameAToolResultGave(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		records []any
		name    string
		want    board.Measure
	}{
		"a shipment's totalCharge":  {held, "totalCharge", board.Measure("revenue")},
		"a shipment's lateMinutes":  {held, "lateMinutes", board.Measure("lateness")},
		"a shipment's lateBy":       {held, "lateBy", board.Measure("lateness")},
		"an invoice's openAmount":   {invoices, "openAmount", board.Measure("amountDue")},
		"an invoice's totalAmount":  {invoices, "totalAmount", board.Measure("total")},
		"a measure by its own name": {invoices, "age", board.Measure("age")},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			args := oneGroup()
			args["measure"] = tc.name
			args["groups"] = []any{map[string]any{"records": tc.records}}
			layout := translated(t, args).Board
			assert.Equal(t, tc.want, layout.Blocks[0].Measure)
			assert.Equal(t, tc.want, layout.Readings[1].Measure)
		})
	}
}

func TestTranslate_TakesRecordsAsIDs(t *testing.T) {
	t.Parallel()

	layout := translated(t, oneGroup()).Board
	assert.Equal(t, pulid.ID("shp_01JHELD00000000000000001"), layout.Blocks[0].Records.IDs[0])
}

func TestTranslate_StacksThreeGroups(t *testing.T) {
	t.Parallel()

	args := twoGroups()
	args["answer"] = "{count_1} on hours, {count_2} at the dock and {count_3} waiting on paperwork."
	args["groups"] = append(args["groups"].([]any), map[string]any{
		"title":   "Waiting on paperwork",
		"records": []any{"shp_01JPAPR00000000000000001"},
	})
	args["actions"] = []any{map[string]any{"label": "Chase the paperwork", "ask": "Chase it."}}
	layout := translated(t, args).Board

	require.Len(t, layout.Blocks, 4, "three groups and an actions block fit the conversation")
	for idx := range 3 {
		assert.False(t, layout.Blocks[idx].Half, "three groups are stacked full width")
	}
	assert.LessOrEqual(t, len(layout.Blocks), board.MaxConversationBlocks)
	assert.Equal(t, []string{"count_1", "count_2", "count_3", "largest_2"}, keys(layout.Readings))
	assert.Len(t, layout.Readings[2].Records.IDs, 1)
	assert.Equal(t, "Shipments, Waiting on paperwork", layout.Readings[2].Label)
	assert.Contains(t, boardanswer.Names(3), "total_3")
	assert.NotContains(t, boardanswer.Names(2), "total_3")
}

func comparison(answer string) map[string]any {
	return map[string]any{
		"title":   "SEED-SHP-010 and SEED-SHP-008 compared",
		"answer":  answer,
		"measure": "lateness",
		"groups": []any{
			map[string]any{"title": "SEED-SHP-010", "records": dock},
			map[string]any{"title": "SEED-SHP-008", "records": held[:1]},
		},
	}
}

func TestTranslate_ReadsEachRecordsOwnFigureByMeasure(t *testing.T) {
	t.Parallel()

	ten, eight := dock[0].(string), held[0].(string)
	answer := translated(t, comparison("SEED-SHP-010 is {lateness:"+ten+"} behind and brings "+
		"in {revenue:"+ten+"}, against {"+eight+":lateness} and {totalCharge:"+eight+"}."))

	assert.Equal(t, "SEED-SHP-010 is {lateness_of_1} behind and brings in {revenue_of_1}, "+
		"against {lateness_of_2} and {revenue_of_2}.", answer.Board.Verdict)
	require.Equal(t,
		[]string{"lateness_of_1", "lateness_of_2", "revenue_of_1", "revenue_of_2"},
		keys(answer.Board.Readings))
	byKey := map[string]board.Reading{}
	for _, reading := range answer.Board.Readings {
		byKey[reading.Key] = reading
	}
	assert.Equal(t, board.FigureSource{
		Label: "Revenue",
		Records: &board.RecordSet{
			Kind: recordkind.Kind(permission.ResourceShipment),
			IDs:  []pulid.ID{pulid.ID(ten)},
		},
		Measure:   board.Measure("revenue"),
		Aggregate: board.AggregateMax,
	}, byKey["revenue_of_1"].FigureSource)
	assert.Equal(t, "Time behind", byKey["lateness_of_2"].Label)
	assert.Equal(t, board.Measure("lateness"), answer.Board.Blocks[0].Measure,
		"the rows keep showing the answer's own measure")
}

func TestTranslate_RefusesAGroupsFigureForAGroupOfOneRecord(t *testing.T) {
	t.Parallel()

	_, problems := boardanswer.Translate(comparison(
		"SEED-SHP-008 brings in {total_1:revenue}, against {total_2:revenue}.",
	))

	require.Len(t, problems, 1)
	assert.Equal(t, "{total_1:revenue}, {total_2:revenue} are the figures of groups that each "+
		"hold one record, and a group's place is easy to mistake for the record a sentence names. "+
		"Name the record instead: its measure and its own number, as {lateness:A-1001}",
		problems[0])
}

func TestTranslate_RefusesARecordsFigureForARecordNoGroupShows(t *testing.T) {
	t.Parallel()

	_, problems := boardanswer.Translate(comparison("It brings in {revenue:" +
		invoices[0].(string) + "} and {revenue:SEED-SHP-099}."))

	require.Len(t, problems, 2)
	assert.Contains(t, problems[0], "a record no group shows")
	assert.Contains(t, problems[1], "names a record this answer shows, by the number a tool "+
		"result gave it")
}

func TestTranslate_TakesTheRowsMeasureFromANamedFigureWhenNoneIsSet(t *testing.T) {
	t.Parallel()

	args := comparison("They bring in {total:revenue} together.")
	delete(args, "measure")
	answer := translated(t, args)

	assert.Equal(t, "They bring in {total} together.", answer.Board.Verdict)
	assert.Equal(t, board.Measure("revenue"), answer.Board.Blocks[0].Measure)
}

func TestTranslate_NeverAddsTimeAcrossRecords(t *testing.T) {
	t.Parallel()

	_, problems := boardanswer.Translate(comparison(
		"SEED-SHP-010 brings in more: {largest} against {total} across the two loads.",
	))
	require.Len(t, problems, 1)
	assert.Equal(t, "{total} would add up lateness across 2 shipments, and time behind added "+
		"across records is not a figure anyone reads. For the most, write {largest}; for one "+
		"record's own, write its measure and its number, as {lateness:A-1001}; for an amount, "+
		"name its measure, as {total:revenue}", problems[0])

	_, problems = boardanswer.Translate(comparison("The two are {total_1:age} old."))
	assert.Equal(t, []string{`answer places {total_1:age}, and a shipment has no measure "age"; ` +
		"use one of lateness or revenue"}, problems)

	answer := translated(t, comparison("The furthest is {largest} behind, and together they "+
		"bring in {total:revenue}."))
	assert.Equal(t, []string{"total_revenue", "largest"}, keys(answer.Board.Readings))
}

func TestTranslate_RefusesATimeCalledAnAmountOrGivenATypedUnit(t *testing.T) {
	t.Parallel()

	billed := map[string]any{
		"title":  "Facilities with the most waiting",
		"answer": "Total detention billed across these facilities is {total:dwell}.",
		"groups": []any{map[string]any{
			"title":   "Most time past free",
			"records": []any{"loc_01HNW5TY00JXKJA33PTTNHR22D", "loc_01JM8ZK9Z8DMAEWD0ECY7YEMW7"},
			"measure": "dwell",
		}},
	}
	_, problems := boardanswer.Translate(billed)
	require.NotEmpty(t, problems)
	assert.Contains(t, strings.Join(problems, "; "), `call it "billed"`)

	minutes := map[string]any{
		"title":  "Facilities with the most waiting",
		"answer": "Riverside leads with {largest:dwell} minutes past free time.",
		"groups": billed["groups"],
	}
	answer, problems := boardanswer.Translate(minutes)
	require.Empty(t, problems, "a unit typed after a time figure is dropped, not refused")
	assert.Equal(t, "Riverside leads with {largest} past free time.", answer.Board.Verdict)

	lapsed := running("Jane Doe's two credentials lapsed {largest} days ago.", credentials)
	answer, problems = boardanswer.Translate(lapsed)
	require.Empty(t, problems)
	assert.NotContains(t, answer.Board.Verdict, "days")

	fine := map[string]any{
		"title": "Facilities with the most waiting",
		"answer": "Trucks waited {total:dwell} past free time across these facilities, and " +
			"none of it was billed yet.",
		"groups": billed["groups"],
	}
	_, problems = boardanswer.Translate(fine)
	assert.Empty(t, problems)
}
