package referenceutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCandidates(t *testing.T) {
	t.Parallel()

	got := Candidates(10, "Re: PRO 88213 delivered", "BOL-2026-0001 and 88213 again, 48 pallets")
	assert.Equal(t, []string{"88213", "BOL-2026-0001"}, got)
}

func TestCandidatesRespectsLimitAndCase(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{"SHP-0001"}, Candidates(5, "SHP-0001 shp-0001"))
	assert.Len(t, Candidates(2, "A1234 B1234 C1234"), 2)
	assert.Nil(t, Candidates(0, "A1234"))
}

func TestPlausible(t *testing.T) {
	t.Parallel()

	assert.True(t, Plausible("88213"))
	assert.False(t, Plausible("48"))
	assert.False(t, Plausible("Thursday"))
}
