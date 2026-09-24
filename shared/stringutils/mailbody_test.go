package stringutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMailBody(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "keeps line breaks and drops blank lines",
			body: "Hi team,\n\n  Please confirm pickup\r\nfor tomorrow.",
			want: "Hi team,\nPlease confirm pickup\nfor tomorrow.",
		},
		{
			name: "drops quoted lines anywhere",
			body: "Confirmed.\n> Can you confirm?\nSee you at 0800.",
			want: "Confirmed.\nSee you at 0800.",
		},
		{
			name: "stops at a quoted reply introduction",
			body: "Rate is fine.\nOn Tue, Sep 22, 2026 at 9:14 AM Dispatch <d@x.com> wrote:\nOld text",
			want: "Rate is fine.",
		},
		{
			name: "stops at a forwarded header",
			body: "FYI\n---------- Forwarded message ---------\nFrom: broker",
			want: "FYI",
		},
		{
			name: "stops at the signature separator",
			body: "Load delivered.\n--\nJane Doe",
			want: "Load delivered.",
		},
		{
			name: "a quote-only body is empty",
			body: "> earlier\n> more",
			want: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, MailBody(tc.body))
		})
	}
}
