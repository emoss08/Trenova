package stringutils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeEmailAddress(t *testing.T) {
	t.Parallel()

	require.Equal(t, "dispatch@example.com", NormalizeEmailAddress(" Dispatch@Example.COM "))
}

func TestNormalizeEmailAddresses(t *testing.T) {
	t.Parallel()

	require.Equal(
		t,
		[]string{"billing@example.com", "ops@example.com"},
		NormalizeEmailAddresses([]string{" Billing@Example.COM ", "", " ops@example.com "}),
	)
}

func TestFormatEmailAddress(t *testing.T) {
	t.Parallel()

	require.Equal(t, "Billing <billing@example.com>", FormatEmailAddress(" Billing ", " billing@example.com "))
	require.Equal(t, "billing@example.com", FormatEmailAddress("", " billing@example.com "))
}
