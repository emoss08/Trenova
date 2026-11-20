package ediconnectionfield

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/pkg/x12"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTheAckWindowKeepsSecondPrecision(t *testing.T) {
	t.Parallel()

	spec := Catalog[SetField(
		edidocument.TransactionSet990,
		edidocument.DocumentDirectionOutbound,
		SetAttrAckWindow,
	)]
	require.NotNil(t, spec)

	cases := map[string]string{
		"45 seconds":          "45 seconds",
		"within 45 secs":      "45 seconds",
		"90 seconds":          "90 seconds",
		"1 minute":            "1 minute",
		"120 seconds":         "2 minutes",
		"1.5 hours":           "90 minutes",
		"4 hours":             "4 hours",
		"2 business days":     "2 days",
		"within 1 second":     "1 second",
		"7 days":              "7 days",
		"a response is fast.": "",
		"45":                  "",
		"0 seconds":           "",
		"8 days":              "",
		"4 hours or 1 day":    "",
	}
	for input, want := range cases {
		got, err := spec.Normalize(input)
		if want == "" {
			require.Error(t, err, input)
			continue
		}
		require.NoError(t, err, input)
		assert.Equal(t, want, got, input)

		again, err := spec.Normalize(got)
		require.NoError(t, err, got)
		assert.Equal(t, got, again, "a normalized window reads back as itself")
	}

	window, ok := ParseDuration("45 seconds")
	require.True(t, ok)
	assert.Equal(t, 45*time.Second, window)
	assert.True(t, valueInQuote(spec, "45 seconds",
		"Kestrel acknowledges within 45 seconds and rejects (AK5 R) any transaction"))
	assert.False(t, valueInQuote(spec, "45 seconds", "Kestrel acknowledges within 45 minutes"))
}

func TestTheLineEndingAfterEachSegmentIsNamed(t *testing.T) {
	t.Parallel()

	spec := Catalog[LineEnding]
	require.NotNil(t, spec)

	cases := map[string]string{
		"LF":                          LineEndingLF,
		"line feed":                   LineEndingLF,
		"newline":                     LineEndingLF,
		"CRLF":                        LineEndingCRLF,
		"CR/LF":                       LineEndingCRLF,
		"carriage return":             LineEndingCR,
		"none":                        LineEndingNone,
		"each segment ends in a CRLF": LineEndingCRLF,
	}
	for input, want := range cases {
		got, err := spec.Normalize(input)
		require.NoError(t, err, input)
		assert.Equal(t, want, got, input)
	}
	_, err := spec.Normalize("tilde")
	require.Error(t, err)

	assert.True(t, valueInQuote(spec, LineEndingLF,
		"Kestrel writes a line feed after every segment terminator."))
	assert.Equal(t, x12.LineEndingLF, WireLineEnding(LineEndingLF))
	assert.Equal(t, x12.LineEndingCRLF, WireLineEnding(LineEndingCRLF))
	assert.Equal(t, LineEndingCRLF, lineEndingValue(x12.LineEndingCRLF))
	assert.Equal(t, LineEndingNone, lineEndingValue(""))
}
