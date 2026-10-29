package ratematrixcell_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrixcell"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostalRangeMatchesByLeadingCharacters(t *testing.T) {
	t.Parallel()

	zip5 := ratematrixcell.NewPostalRange(" 30000", "31999 ")
	zip3 := ratematrixcell.NewPostalRange("300", "319")
	canada := ratematrixcell.NewPostalRange("k1a", "K1Z")

	cases := []struct {
		name   string
		postal ratematrixcell.PostalRange
		key    string
		want   bool
	}{
		{"the first code", zip5, "30000", true},
		{"the last code", zip5, "31999", true},
		{"inside", zip5, " 30303 ", true},
		{"zip plus four by its first five", zip5, "30303-1234", true},
		{"below", zip5, "29999", false},
		{"above", zip5, "32000", false},
		{"a key shorter than the range", zip5, "303", false},
		{"a five-digit key on a three-digit range", zip3, "31555", true},
		{"a three-digit range above", zip3, "32000", false},
		{"upper-cased and trimmed", canada, " k1p 1j1", true},
		{"outside the letters", canada, "K2A 0B1", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, tc.postal.Contains(tc.key))
		})
	}
}

func TestPostalRangeShape(t *testing.T) {
	t.Parallel()

	assert.True(t, ratematrixcell.NewPostalRange("30000", "31999").IsWellFormed())
	assert.True(t, ratematrixcell.NewPostalRange("30000", "30000").IsWellFormed())
	assert.False(t, ratematrixcell.NewPostalRange("300", "31999").IsWellFormed())
	assert.False(t, ratematrixcell.NewPostalRange("31999", "30000").IsWellFormed())
	assert.False(t, ratematrixcell.NewPostalRange("", "").IsWellFormed())

	a := ratematrixcell.NewPostalRange("30000", "31999")
	assert.True(t, a.Overlaps(ratematrixcell.NewPostalRange("31999", "32999")))
	assert.False(t, a.Overlaps(ratematrixcell.NewPostalRange("32000", "32999")))
	assert.False(t, a.Overlaps(ratematrixcell.NewPostalRange("300", "319")),
		"ranges of different lengths are refused elsewhere, not compared")
}

func TestPostalIndexFindsTheRangeHoldingAKey(t *testing.T) {
	t.Parallel()

	index := ratematrixcell.NewPostalIndex([]ratematrixcell.PostalRange{
		{From: "32000", To: "34999"},
		{From: "30000", To: "31999"},
		{From: "40000", To: "41999"},
		{From: "bad", To: "toolong"},
	})

	found := index.Find("30303-1234")
	require.Len(t, found, 1)
	assert.Equal(t, "30000", found[0].From)

	found = index.Find("34999")
	require.Len(t, found, 1)
	assert.Equal(t, "32000", found[0].From)

	assert.Empty(t, index.Find("35000"), "a gap between ranges holds nothing")
	assert.Empty(t, index.Find("29999"))
	assert.Empty(t, index.Find("3"))
	assert.Empty(t, (*ratematrixcell.PostalIndex)(nil).Find("30000"))
}

func postalDimension(position int16) *ratematrix.RateMatrixDimension {
	return &ratematrix.RateMatrixDimension{
		Position:  position,
		Kind:      ratematrix.DimensionKindZipRange,
		MatchMode: ratematrix.MatchModeRange,
		Label:     "Origin",
	}
}

func TestAPostalRangeAxisIsMatchedByRangeWithoutOverflow(t *testing.T) {
	t.Parallel()

	valid := postalDimension(0)
	multiErr := errortypes.NewMultiError()
	valid.Validate(multiErr)
	assert.False(t, multiErr.HasErrors(), multiErr.Error())

	exact := postalDimension(0)
	exact.MatchMode = ratematrix.MatchModeExact
	multiErr = errortypes.NewMultiError()
	exact.Validate(multiErr)
	assert.True(t, multiErr.HasErrors())

	overflow := postalDimension(0)
	overflow.RangeOverflow = ratematrix.RangeOverflowNearest
	multiErr = errortypes.NewMultiError()
	overflow.Validate(multiErr)
	assert.True(t, multiErr.HasErrors())

	zone := &ratematrix.RateMatrixDimension{
		Kind:      ratematrix.DimensionKindZone,
		MatchMode: ratematrix.MatchModeRange,
	}
	multiErr = errortypes.NewMultiError()
	zone.Validate(multiErr)
	assert.True(t, multiErr.HasErrors(), "a zone still cannot be banded")
}

func TestAPostalCellIsNormalisedAndChecked(t *testing.T) {
	t.Parallel()

	dimensions := []*ratematrix.RateMatrixDimension{postalDimension(0)}

	cell := &ratematrixcell.RateMatrixCell{D0Key: " k1a ", D0KeyTo: "k1z"}
	multiErr := errortypes.NewMultiError()
	cell.ValidateAgainst(dimensions, multiErr)
	assert.False(t, multiErr.HasErrors(), multiErr.Error())
	assert.Equal(t, "K1A", cell.D0Key)
	assert.Equal(t, "K1Z", cell.D0KeyTo)

	for name, bad := range map[string]*ratematrixcell.RateMatrixCell{
		"no last code":          {D0Key: "30000"},
		"different lengths":     {D0Key: "300", D0KeyTo: "31999"},
		"ends before it starts": {D0Key: "31999", D0KeyTo: "30000"},
	} {
		multiErr = errortypes.NewMultiError()
		bad.ValidateAgainst(dimensions, multiErr)
		assert.True(t, multiErr.HasErrors(), name)
	}

	exact := []*ratematrix.RateMatrixDimension{{
		Position:  0,
		Kind:      ratematrix.DimensionKindZone,
		MatchMode: ratematrix.MatchModeExact,
	}}
	stray := &ratematrixcell.RateMatrixCell{D0Key: "SE", D0KeyTo: "NE"}
	multiErr = errortypes.NewMultiError()
	stray.ValidateAgainst(exact, multiErr)
	assert.False(t, multiErr.HasErrors())
	assert.Empty(t, stray.D0KeyTo, "an exact axis keeps no range end")
}

func TestPostalAxesRefuseOverlapsAndMixedLengths(t *testing.T) {
	t.Parallel()

	dimensions := []*ratematrix.RateMatrixDimension{
		postalDimension(0),
		{
			Position:  1,
			Kind:      ratematrix.DimensionKindWeightBreak,
			MatchMode: ratematrix.MatchModeRange,
		},
	}

	clean := []*ratematrixcell.RateMatrixCell{
		{D0Key: "30000", D0KeyTo: "31999"},
		{D0Key: "30000", D0KeyTo: "31999"},
		{D0Key: "32000", D0KeyTo: "34999"},
	}
	multiErr := errortypes.NewMultiError()
	ratematrixcell.ValidatePostalAxes(dimensions, clean, multiErr)
	assert.False(t, multiErr.HasErrors(), "the same range on many cells is one range")

	overlapping := []*ratematrixcell.RateMatrixCell{
		{D0Key: "30000", D0KeyTo: "39999"},
		{D0Key: "31000", D0KeyTo: "31999"},
		{D0Key: "35000", D0KeyTo: "36999"},
	}
	multiErr = errortypes.NewMultiError()
	ratematrixcell.ValidatePostalAxes(dimensions, overlapping, multiErr)
	require.True(t, multiErr.HasErrors())
	assert.Len(t, multiErr.Errors, 2, "both ranges inside the wide one overlap it")

	mixed := []*ratematrixcell.RateMatrixCell{
		{D0Key: "30000", D0KeyTo: "31999"},
		{D0Key: "320", D0KeyTo: "349"},
	}
	multiErr = errortypes.NewMultiError()
	ratematrixcell.ValidatePostalAxes(dimensions, mixed, multiErr)
	assert.True(t, multiErr.HasErrors())
}
