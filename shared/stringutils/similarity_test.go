package stringutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeNameFoldsCasePunctuationAndSpacing(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "acme freight", NormalizeName("  ACME   Freight, "))
	assert.Equal(t, "fuel surcharge revenue", NormalizeName("Fuel-Surcharge/Revenue"))
	assert.Equal(t, "o brien trucking", NormalizeName("O'Brien Trucking"))
	assert.Equal(t, "", NormalizeName(" -/., "))
}

func TestNormalizeNameKeepsNonLatinLetters(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "müller logistik", NormalizeName("Müller Logistik"))
	assert.Equal(t, "運輸 公司", NormalizeName("運輸 公司"))
}

func TestNormalizeCompanyNameDropsLegalSuffixes(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "acme freight", NormalizeCompanyName("Acme Freight, Inc."))
	assert.Equal(t, "acme freight", NormalizeCompanyName("ACME FREIGHT LLC"))
	assert.Equal(t, "peak distributing", NormalizeCompanyName("Peak Distributing Co"))
	assert.Equal(t, "the company store", NormalizeCompanyName("The Company Store"))
	assert.Equal(t, "llc", NormalizeCompanyName("LLC"))
}

func TestJaroWinklerMatchesKnownValues(t *testing.T) {
	t.Parallel()

	assert.InDelta(t, 1.0, JaroWinkler("freight", "freight"), 1e-9)
	assert.InDelta(t, 0.0, JaroWinkler("", "freight"), 1e-9)
	assert.InDelta(t, 1.0, JaroWinkler("", ""), 1e-9)
	assert.InDelta(t, 0.9611, JaroWinkler("MARTHA", "MARHTA"), 1e-4)
	assert.InDelta(t, 0.8400, JaroWinkler("DWAYNE", "DUANE"), 1e-4)
	assert.InDelta(t, 0.8133, JaroWinkler("DIXON", "DICKSONX"), 1e-4)
	assert.InDelta(t, 0.0, JaroWinkler("abc", "xyz"), 1e-9)
}

func TestJaroWinklerIsSymmetric(t *testing.T) {
	t.Parallel()

	assert.InDelta(t, JaroWinkler("detention", "detension"), JaroWinkler("detension", "detention"), 1e-12)
}

func TestTokenSetRatioIgnoresOrderAndExtraWords(t *testing.T) {
	t.Parallel()

	assert.InDelta(t, 1.0, TokenSetRatio("fuel surcharge", "surcharge fuel"), 1e-9)
	assert.InDelta(t, 1.0, TokenSetRatio("fuel surcharge", "fuel surcharge income"), 1e-9)
	assert.InDelta(t, 0.0, TokenSetRatio("", "fuel"), 1e-9)
	assert.Less(t, TokenSetRatio("detention", "lumper fee"), TokenSetRatio("detention", "detention fee"))
}

func TestNameSimilarityRanksTheIntendedMatchFirst(t *testing.T) {
	t.Parallel()

	target := "Freight Revenue"
	candidates := []string{"Freight Income", "Fuel Surcharge Revenue", "Freight Revenue", "Accounts Receivable"}

	best, bestScore := "", -1.0
	for _, candidate := range candidates {
		if score := NameSimilarity(target, candidate); score > bestScore {
			best, bestScore = candidate, score
		}
	}

	assert.Equal(t, "Freight Revenue", best)
	assert.InDelta(t, 1.0, bestScore, 1e-9)
	assert.Greater(t, NameSimilarity(target, "Freight Income"), NameSimilarity(target, "Accounts Receivable"))
}

func TestNameSimilarityScoresNormalizedEqualNamesAsExact(t *testing.T) {
	t.Parallel()

	assert.InDelta(t, 1.0, NameSimilarity("DETENTION", " detention "), 1e-9)
	assert.InDelta(t, 0.0, NameSimilarity("", "detention"), 1e-9)
}

func TestCompanyNameSimilarityIgnoresLegalSuffixes(t *testing.T) {
	t.Parallel()

	assert.InDelta(t, 1.0, CompanyNameSimilarity("Acme Freight, Inc.", "ACME FREIGHT LLC"), 1e-9)
	assert.Less(t, CompanyNameSimilarity("Acme Freight", "Apex Logistics"), 0.7)
}
