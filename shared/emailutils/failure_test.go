package emailutils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProviderFailureText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "empty", raw: "  ", want: ""},
		{
			name: "postmark body",
			raw: `non-retryable email provider failure: postmark status 422: {"ErrorCode":406,` +
				`"Message":"You tried to send to recipient(s) that have been marked as inactive."}`,
			want: "You tried to send to recipient(s) that have been marked as inactive.",
		},
		{
			name: "resend body",
			raw:  `retryable: resend status 422: {"statusCode":422,"message":"Invalid to field.","name":"validation_error"}`,
			want: "Invalid to field.",
		},
		{
			name: "plain wrapped",
			raw:  "non-retryable email provider failure: 550 mailbox unavailable",
			want: "550 mailbox unavailable",
		},
		{name: "control characters", raw: "line one\nline\ttwo", want: "line one line two"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, ProviderFailureText(tt.raw))
		})
	}

	long := ProviderFailureText(strings.Repeat("a", 900))
	assert.Len(t, long, maxFailureTextRunes)
}

func TestCleanFailureText(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "recipient is suppressed: a@b.example", CleanFailureText(" recipient is suppressed: a@b.example\n"))
}
