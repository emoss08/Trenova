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

func TestSame(t *testing.T) {
	t.Parallel()

	assert.True(t, Same("ACH-1001", " ach-1001 "))
	assert.False(t, Same("ACH-1001", "ACH-10011"))
	assert.False(t, Same("", ""), "two blank references name nothing")
	assert.False(t, Same("  ", "  "))
	assert.True(t, Same("ACH.1001", "ach-1001"), "a dot joins segments as a hyphen does")
	assert.True(t, Same("ACH_1001", "ACH-1001"))
	assert.False(t, Same("ACH1001", "ACH-1001"), "a separator is not dropped")
}

func TestMentions(t *testing.T) {
	t.Parallel()

	assert.True(t, Mentions("INV-1001", "Paying inv-1001 and INV-1002"))
	assert.True(t, Mentions("INV-1002", "ACH 88213", "remit: INV-1001, INV-1002"))
	assert.False(t, Mentions("INV-100", "Paying INV-1001"), "a prefix of a reference is not that reference")
	assert.False(t, Mentions("INV-1001", "Paying INV-10012"))
	assert.False(t, Mentions(" ", "anything"))
}

func TestMentions_SplitsReferencesOnASlash(t *testing.T) {
	t.Parallel()

	assert.True(t, Mentions("INV-1001", "INV-1001/INV-1002"))
	assert.True(t, Mentions("INV-1002", "INV-1001/INV-1002"))
	assert.False(t, Mentions("INV-1001/INV-1002", "INV-1001/INV-1002"),
		"a slash separates two references rather than naming one")
}

func TestMentions_ReadsADotAsASeparator(t *testing.T) {
	t.Parallel()

	assert.True(t, Mentions("INV-1001", "Paying INV.1001 in full"))
	assert.True(t, Mentions("INV.1001", "Paying INV-1001."), "a sentence's full stop is not part of it")
	assert.False(t, Mentions("INV-1001", "Paying INV.10012"))
	assert.False(t, Mentions("INV-1001", "Paying INV1001"))
}
