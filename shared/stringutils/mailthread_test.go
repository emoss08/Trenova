package stringutils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMessageIDToken(t *testing.T) {
	t.Parallel()

	cases := []struct {
		raw  string
		want string
		ok   bool
	}{
		{raw: "<abc@bigshipper.com>", want: "<abc@bigshipper.com>", ok: true},
		{raw: "  abc.123@mail.example  ", want: "<abc.123@mail.example>", ok: true},
		{raw: "", ok: false},
		{raw: "<>", ok: false},
		{raw: "no-at-sign", ok: false},
		{raw: "@example.com", ok: false},
		{raw: "a@b@c", ok: false},
		{raw: "abc@example.com>\r\nBcc: victim@example.com", ok: false},
		{raw: "abc@exa mple.com", ok: false},
		{raw: "abc@exämple.com", ok: false},
		{raw: strings.Repeat("a", 1000) + "@example.com", ok: false},
	}

	for _, tc := range cases {
		got, ok := MessageIDToken(tc.raw)
		assert.Equalf(t, tc.ok, ok, "MessageIDToken(%q)", tc.raw)
		assert.Equalf(t, tc.want, got, "MessageIDToken(%q)", tc.raw)
	}
}

func TestReplyReferences_AppendsTheOriginalDropsJunkAndKeepsTheNewest(t *testing.T) {
	t.Parallel()

	got := ReplyReferences(
		[]string{"<one@a.example>", "garbage", "<two@a.example>", "<one@a.example>"},
		"three@a.example",
		10,
	)
	assert.Equal(t, "<one@a.example> <two@a.example> <three@a.example>", got)

	assert.Equal(t,
		"<two@a.example> <three@a.example>",
		ReplyReferences([]string{"<one@a.example>", "<two@a.example>"}, "<three@a.example>", 2),
	)
	assert.Empty(t, ReplyReferences(nil, "not an id", 5))
}

func TestReplySubject(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Re: Where is 88213?", ReplySubject("Where is 88213?"))
	assert.Equal(t, "Re: Where is 88213?", ReplySubject("Re: Where is 88213?"))
	assert.Equal(t, "Re: load", ReplySubject("RE: load"))
	assert.Equal(t, "Re: two words", ReplySubject("  two \n words "))
	assert.Equal(t, "Re: your message", ReplySubject("   "))
}

func TestIsAutomatedSender(t *testing.T) {
	t.Parallel()

	for _, address := range []string{
		"noreply@shipper.example",
		"no-reply@shipper.example",
		"DoNotReply@shipper.example",
		"MAILER-DAEMON@mx.example",
		"postmaster@mx.example",
		"bounces+123@mail.example",
		"orders-noreply@shipper.example",
	} {
		assert.Truef(t, IsAutomatedSender(address), "%s is automated", address)
	}

	for _, address := range []string{
		"dana@shipper.example",
		"replies@shipper.example",
		"bouncer@club.example",
		"",
	} {
		assert.Falsef(t, IsAutomatedSender(address), "%s is a person", address)
	}
}
