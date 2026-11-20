package tender_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/stretchr/testify/require"
)

func TestMovePartnerKey(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		sequence int64
		moves    int
		want     string
	}{
		{
			name:     "a shipment with one move is its PRO number",
			sequence: 0,
			moves:    1,
			want:     "S2603001042",
		},
		{name: "the first of two moves", sequence: 0, moves: 2, want: "S2603001042-1"},
		{name: "the third of three moves", sequence: 2, moves: 3, want: "S2603001042-3"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			key := tender.MovePartnerKey(" S2603001042 ", tc.sequence, tc.moves)
			require.Equal(t, tc.want, key.String())
		})
	}
}

func TestPartnerKeyStringWithoutANumberIsEmpty(t *testing.T) {
	t.Parallel()

	require.Empty(t, tender.PartnerKey{Move: 2}.String())
}

func TestParsePartnerKeys(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		references []string
		want       []tender.PartnerKey
	}{
		{
			name:       "a PRO number reads as itself",
			references: []string{"S2603001042"},
			want:       []tender.PartnerKey{{ProNumber: "S2603001042"}},
		},
		{
			name:       "a move key reads as the whole number and as the PRO number and move",
			references: []string{"S2603001042-2"},
			want: []tender.PartnerKey{
				{ProNumber: "S2603001042-2"},
				{ProNumber: "S2603001042", Move: 2},
			},
		},
		{
			name:       "a trailing zero or a word is not a move",
			references: []string{"SEED-0", "SEED-CASE"},
			want:       []tender.PartnerKey{{ProNumber: "SEED-0"}, {ProNumber: "SEED-CASE"}},
		},
		{
			name:       "blanks and repeats are dropped",
			references: []string{"", "  ", "S1", " S1 "},
			want:       []tender.PartnerKey{{ProNumber: "S1"}},
		},
		{
			name:       "a leading separator is not a move",
			references: []string{"-3"},
			want:       []tender.PartnerKey{{ProNumber: "-3"}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.want, tender.ParsePartnerKeys(tc.references...))
		})
	}
}

func TestPartnerKeyRoundTrips(t *testing.T) {
	t.Parallel()

	key := tender.MovePartnerKey("S2603001042", 1, 2)
	parsed := tender.ParsePartnerKeys(key.String())
	require.Contains(t, parsed, key)
	require.Equal(t, int64(1), parsed[1].MoveSequence())
	require.Equal(t, []string{"S2603001042-2", "S2603001042"}, tender.ProNumbersOf(parsed))
}
