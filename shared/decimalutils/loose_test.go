package decimalutils_test

import (
	"testing"

	"github.com/emoss08/trenova/shared/decimalutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLooseReadsAmountsAsReportsPrintThem(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"1,250.50":       "1250.5",
		"$1,250.50":      "1250.5",
		"(312.00)":       "-312",
		"-45":            "-45",
		"1 000.25":       "1000.25",
		"12.5 gal":       "12.5",
		"USD 99.99":      "99.99",
		" 7.10":     "7.1",
	}
	for raw, want := range cases {
		got, err := decimalutils.ParseLoose(raw)
		require.NoError(t, err, raw)
		assert.Equal(t, want, got.String(), raw)
	}
}

func TestParseLooseRefusesWhatIsNotANumber(t *testing.T) {
	t.Parallel()

	_, err := decimalutils.ParseLoose("")
	require.ErrorIs(t, err, decimalutils.ErrNumberMissing)

	for _, raw := range []string{"abc", ".", "1#2"} {
		_, err = decimalutils.ParseLoose(raw)
		require.ErrorIs(t, err, decimalutils.ErrNotANumber, raw)
	}
}
