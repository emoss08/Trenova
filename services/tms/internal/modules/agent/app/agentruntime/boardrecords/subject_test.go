package boardrecords_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func pair(measure, answer string) map[string]any {
	args := map[string]any{
		"answer": answer,
		"groups": []any{
			map[string]any{"title": "Further behind", "records": []any{shp010}},
			map[string]any{"title": "Less behind", "records": []any{shp008}},
		},
	}
	if measure != "" {
		args["measure"] = measure
	}

	return args
}

func TestGroupFigures_GivesEachFigureTheRecordItsClauseNames(t *testing.T) {
	t.Parallel()

	read := incidentRead()
	tests := []struct {
		name    string
		measure string
		answer  string
		want    string
	}{
		{
			name:    "the groups taken the wrong way round",
			measure: "revenue",
			answer:  "SEED-SHP-008 brings in {total_1}, against {total_2} for SEED-SHP-010.",
			want: "SEED-SHP-008 brings in {revenue:" + shp008 + "}, against {revenue:" + shp010 +
				"} for SEED-SHP-010.",
		},
		{
			name:    "a duration where money was meant, and the groups swapped",
			measure: "lateness",
			answer:  "SEED-SHP-010 brings in {total_2:revenue} and SEED-SHP-008 brings in {total_1:revenue}.",
			want: "SEED-SHP-010 brings in {revenue:" + shp010 + "} and SEED-SHP-008 brings in " +
				"{revenue:" + shp008 + "}.",
		},
		{
			name:    "a clause with no number takes the sentence's record",
			measure: "lateness",
			answer: "SEED-SHP-010 is further behind, its window closed {total_1} ago, and it " +
				"brings in {total_1:revenue}. SEED-SHP-008 is {total_2} behind.",
			want: "SEED-SHP-010 is further behind, its window closed {lateness:" + shp010 +
				"} ago, and it brings in {revenue:" + shp010 + "}. SEED-SHP-008 is {lateness:" +
				shp008 + "} behind.",
		},
		{
			name:    "no record named keeps the group's own",
			measure: "revenue",
			answer:  "The first brings in {total_1}.",
			want:    "The first brings in {revenue:" + shp010 + "}.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, changed := read.GroupFigures(pair(tt.measure, tt.answer))

			require.True(t, changed)
			assert.Equal(t, tt.want, got["answer"])
		})
	}
}

func TestGroupFigures_LeavesWhatIsNotOneRecordsFigure(t *testing.T) {
	t.Parallel()

	read := incidentRead()
	args := pair("revenue", "{count_1} and {largest} and {total}.")
	_, changed := read.GroupFigures(args)
	assert.False(t, changed)

	several := pair("revenue", "SEED-SHP-010 brings in {total_1}.")
	several["groups"].([]any)[0].(map[string]any)["records"] = []any{shp010, shp001}
	_, changed = read.GroupFigures(several)
	assert.False(t, changed, "a group of several records keeps its own figure")

	_, changed = read.GroupFigures(pair("", "SEED-SHP-010 brings in {total_1}."))
	assert.False(t, changed, "with no measure there is nothing to read")
}
