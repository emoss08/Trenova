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

func TestNormalizeEmailList(t *testing.T) {
	t.Parallel()

	require.Empty(t, NormalizeEmailList(nil))
	require.Equal(
		t,
		[]string{"ops@example.com", "billing@example.com"},
		NormalizeEmailList([]string{" Ops@Example.COM ", "", "billing@example.com", "ops@example.com"}),
		"a list keeps the order it was given in, having dropped blanks and repeats",
	)
}

func TestFormatEmailAddress(t *testing.T) {
	t.Parallel()

	require.Equal(
		t,
		"Billing <billing@example.com>",
		FormatEmailAddress(" Billing ", " billing@example.com "),
	)
	require.Equal(t, "billing@example.com", FormatEmailAddress("", " billing@example.com "))
}

func TestSplitEmailList(t *testing.T) {
	t.Parallel()

	require.Nil(t, SplitEmailList("   "))
	require.Equal(
		t,
		[]string{"billing@example.com", "ops@example.com", "ap@example.com"},
		SplitEmailList(
			"Billing@Example.COM, ops@example.com;\nap@example.com\tbilling@example.com",
		),
	)
}
