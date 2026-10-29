package tenantbootstrap

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/sequenceconfig"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBootstrapRequiresEveryPart(t *testing.T) {
	t.Parallel()

	_, err := Bootstrap(t.Context(), nil, BootstrapParams{})
	require.ErrorIs(t, err, ErrBusinessUnitRequired)

	_, err = Bootstrap(t.Context(), nil, BootstrapParams{BusinessUnit: &tenant.BusinessUnit{}})
	require.ErrorIs(t, err, ErrOrganizationRequired)

	_, err = Bootstrap(t.Context(), nil, BootstrapParams{
		BusinessUnit: &tenant.BusinessUnit{},
		Organization: &tenant.Organization{},
	})
	require.ErrorIs(t, err, ErrOwnerRequired)
}

func TestCandidateFor(t *testing.T) {
	t.Parallel()

	first, err := candidateFor("acme-freight", 20, 0)
	require.NoError(t, err)
	assert.Equal(t, "acme-freight", first)

	second, err := candidateFor("acme-freight", 20, 1)
	require.NoError(t, err)
	assert.Equal(t, "acme-freight-2", second)

	truncated, err := candidateFor("abcdefghijklmnopqrst", 20, 3)
	require.NoError(t, err)
	assert.Equal(t, "abcdefghijklmnopqr-4", truncated)
	assert.LessOrEqual(t, len(truncated), 20)

	random, err := candidateFor("abcdefghijklmnopqrst", 20, 12)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(random), 20)
	assert.True(t, strings.HasPrefix(random, "abcdefghijk"))
}

func TestFirstAvailableSkipsTakenCandidates(t *testing.T) {
	t.Parallel()

	taken := map[string]bool{"acme": true, "acme-2": true}
	got, err := firstAvailable(t.Context(), "acme", 20, func(candidate string) (bool, error) {
		return taken[candidate], nil
	})
	require.NoError(t, err)
	assert.Equal(t, "acme-3", got)
}

func TestFirstAvailableGivesUp(t *testing.T) {
	t.Parallel()

	_, err := firstAvailable(t.Context(), "acme", 20, func(string) (bool, error) {
		return true, nil
	})
	require.ErrorIs(t, err, ErrNoCandidate)
}

func TestBucketNameFor(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "acme-freight", BucketNameFor("acme-freight"))
	assert.Equal(t, "tenant-ab", BucketNameFor("ab"))
	assert.Len(t, BucketNameFor(strings.Repeat("a", 80)), maxBucketNameLength)
}

func TestDefaultSequenceTypesCoverRequiredSequences(t *testing.T) {
	t.Parallel()

	types := DefaultSequenceTypes()
	assert.Len(t, types, 7)
	assert.Contains(t, types, sequenceconfig.SequenceTypeProNumber)
	types[0] = "mutated"
	assert.Equal(t, sequenceconfig.SequenceTypeProNumber, DefaultSequenceTypes()[0])
}
