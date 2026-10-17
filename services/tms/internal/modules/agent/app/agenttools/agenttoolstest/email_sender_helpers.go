package agenttoolstest

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type FakeSenders struct {
	Suppressed []string
	Refused    bool
}

func (f FakeSenders) ResolveSender(
	context.Context,
	*emailports.SendEmailRequest,
) (*emailports.EmailSender, error) {
	return &emailports.EmailSender{
		Email:      "ops@carrier.example",
		Name:       "Carrier Operations",
		Suppressed: f.Suppressed,
		Refused:    f.Refused,
	}, nil
}

func AssertMessageIsTheSend(
	t *testing.T,
	message *agent.MessagePreview,
	sent *emailports.SendEmailRequest,
) {
	t.Helper()

	require.NotNil(t, message)
	require.NotNil(t, sent)
	assert.Equal(t, agent.MessageChannelEmail, message.Channel)
	assert.Equal(t, sent.To, message.To)
	assert.Equal(t, sent.Subject, message.Subject)
	assert.Equal(t, sent.Text, message.Body)
	assert.Equal(t, "Carrier Operations <ops@carrier.example>", message.From)
}
