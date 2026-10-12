package inboundmessage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestBeforeAppendModelStripsNULFromMailFields(t *testing.T) {
	t.Parallel()

	msg := &InboundMessage{
		MessageID:   "<a\x00b@example.com>",
		InReplyTo:   "<c\x00d@example.com>",
		FromName:    "Dock\x00 Office",
		Subject:     "POD\x00 attached",
		TextBody:    "See\x00 attached.",
		FailureText: "bad\x00 part",
	}
	require.NoError(t, msg.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil)))
	assert.Equal(t, "<ab@example.com>", msg.MessageID)
	assert.Equal(t, "<cd@example.com>", msg.InReplyTo)
	assert.Equal(t, "Dock Office", msg.FromName)
	assert.Equal(t, "POD attached", msg.Subject)
	assert.Equal(t, "See attached.", msg.TextBody)
	assert.Equal(t, "bad part", msg.FailureText)
}
