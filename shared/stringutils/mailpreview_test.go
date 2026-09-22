package stringutils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMailPreview(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		body string
		max  int
		want string
	}{
		{
			name: "folds line breaks and runs of space into one space",
			body: "Hi team,\n\n  Please confirm   pickup\r\nfor tomorrow.",
			max:  200,
			want: "Hi team, Please confirm pickup for tomorrow.",
		},
		{
			name: "drops quoted lines",
			body: "Confirmed.\n> Can you confirm the 0800 appointment?\n> Thanks",
			max:  200,
			want: "Confirmed.",
		},
		{
			name: "stops at the line that introduces a quoted reply",
			body: "Rate is fine.\nOn Tue, Sep 22, 2026 at 9:14 AM Dispatch <d@x.com> wrote:\nOld thread text",
			max:  200,
			want: "Rate is fine.",
		},
		{
			name: "stops at an Outlook original message header",
			body: "See attached POD.\n-----Original Message-----\nFrom: someone",
			max:  200,
			want: "See attached POD.",
		},
		{
			name: "stops at an Outlook underscore separator",
			body: "Invoice attached.\n________________________________\nFrom: Billing",
			max:  200,
			want: "Invoice attached.",
		},
		{
			name: "stops at the signature separator",
			body: "Load 4471 delivered.\n-- \nJane Doe\nACME Logistics",
			max:  200,
			want: "Load 4471 delivered.",
		},
		{
			name: "cuts on a rune boundary with an ellipsis",
			body: "Détention à facturer pour le chargement numéro 12345",
			max:  12,
			want: "Détention à…",
		},
		{
			name: "an empty body has no preview",
			body: " \n\t\n",
			max:  200,
			want: "",
		},
		{
			name: "a body that is only a quote has no preview",
			body: "> earlier text\n> more",
			max:  200,
			want: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, MailPreview(tc.body, tc.max))
		})
	}
}

func TestMailPreviewReadsOnlyAsFarAsItNeeds(t *testing.T) {
	t.Parallel()

	body := "Short opening line.\n" + strings.Repeat("filler text ", 4000)
	got := MailPreview(body, 40)

	assert.LessOrEqual(t, len([]rune(got)), 41)
	assert.True(t, strings.HasPrefix(got, "Short opening line. filler"))
}
