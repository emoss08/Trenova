package totputils

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecoveryCodeRejectsBytesThatWouldBiasTheAlphabet(t *testing.T) {
	t.Parallel()

	limit := byte(256 - 256%len(recoveryAlphabet))
	require.Equal(t, byte(248), limit)

	biased := bytes.Repeat([]byte{248, 249, 255}, 4)
	fair := []byte{0, 30, 31, 61, 247, 1, 2, 3, 4, 5}
	code, err := recoveryCodeFrom(bytes.NewReader(append(append(biased, fair...), bytes.Repeat([]byte{0}, 10)...)))
	require.NoError(t, err)

	want := []byte{
		recoveryAlphabet[0], recoveryAlphabet[30], recoveryAlphabet[0], recoveryAlphabet[30],
		recoveryAlphabet[247%31], '-', recoveryAlphabet[1], recoveryAlphabet[2],
		recoveryAlphabet[3], recoveryAlphabet[4], recoveryAlphabet[5],
	}
	assert.Equal(t, string(want), code)
}

func TestRecoveryCodeFailsWhenRandomnessRunsOut(t *testing.T) {
	t.Parallel()

	_, err := recoveryCodeFrom(bytes.NewReader(bytes.Repeat([]byte{255}, 10)))
	require.Error(t, err)
}

func TestRecoveryCodeCharactersAreUniform(t *testing.T) {
	t.Parallel()

	counts := make(map[rune]int, len(recoveryAlphabet))
	const samples = 4000
	for range samples {
		code, err := recoveryCode()
		require.NoError(t, err)
		for _, r := range strings.ReplaceAll(code, "-", "") {
			counts[r]++
		}
	}

	require.Len(t, counts, len(recoveryAlphabet))
	expected := float64(samples*recoveryCodeGroups*recoveryGroupSize) / float64(len(recoveryAlphabet))
	var chiSquared float64
	for _, count := range counts {
		diff := float64(count) - expected
		chiSquared += diff * diff / expected
	}
	assert.Less(t, chiSquared, 80.0)
}
