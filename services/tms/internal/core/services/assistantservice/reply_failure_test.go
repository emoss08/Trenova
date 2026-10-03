package assistantservice

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplyFailure(t *testing.T) {
	t.Parallel()

	exhausted := &serviceports.ChatProvidersFailedError{
		Failures: []serviceports.ChatProviderFailure{{
			Name: "Anthropic", Model: "claude-sonnet", Vendor: "anthropic",
			Status: "Overloaded", Detail: "Anthropic returned 529 twice",
		}},
		Err: errors.New("overloaded"),
	}

	failure := replyFailure(fmt.Errorf("run: %w", exhausted), false)
	require.NotNil(t, failure)
	assert.Equal(t, conversation.ReplyFailureNoModel, failure.Kind)
	require.Len(t, failure.Providers, 1)
	assert.Equal(t, "Anthropic returned 529 twice", failure.Providers[0].Detail)
	assert.Equal(t, "anthropic", failure.Providers[0].Vendor)

	assert.Equal(t, conversation.ReplyFailureStopped, replyFailure(context.Canceled, true).Kind)
	assert.Equal(t, conversation.ReplyFailureInterrupted, replyFailure(errors.New("x"), true).Kind)
	assert.Equal(t, conversation.ReplyFailureBeforeStart, replyFailure(errors.New("x"), false).Kind)
}
