package benchresult

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDataProfileOf_NamesTheDataARunReadsWithoutRefusingVolume(t *testing.T) {
	t.Parallel()

	_, err := DataProfileOf(0, 0, 0)
	require.ErrorIs(t, err, ErrNoFixtureData)

	fixture, err := DataProfileOf(40, 0, 0)
	require.NoError(t, err)
	assert.Equal(t, DataProfile{Shape: DataFixture}, fixture)

	volume, err := DataProfileOf(40, 408, 0)
	require.NoError(t, err)
	assert.Equal(t, DataProfile{Shape: DataVolume}, volume)

	drifted, err := DataProfileOf(40, 408, 37)
	require.NoError(t, err)
	assert.Equal(t, DataProfile{
		Shape: DataVolume,
		Drift: "37 changes earlier bench runs approved are still in the data",
	}, drifted)
	assert.Equal(t,
		"fixture and volume seeds; 37 changes earlier bench runs approved are still in the data",
		drifted.String())
}

func TestComparedDataNote_SaysWhenTwoRunsReadDifferentData(t *testing.T) {
	t.Parallel()

	assert.Empty(t, ComparedDataNote(DataProfile{}, DataProfile{Shape: DataVolume}))
	assert.Empty(
		t,
		ComparedDataNote(DataProfile{Shape: DataVolume}, DataProfile{Shape: DataVolume}),
	)
	assert.Equal(t,
		"the earlier run ran on the fixture seed and this one on the fixture and volume seeds, "+
			"so a moved pass rate may be the data rather than the change",
		ComparedDataNote(DataProfile{Shape: DataFixture}, DataProfile{Shape: DataVolume}))
}
