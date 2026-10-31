package triagenova_test

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/documents/app/triagenova"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type nova struct {
	err  error
	task *llmports.NovaTask
}

func (n *nova) Available(context.Context, aitask.Task, pagination.TenantInfo) (bool, error) {
	return n.err == nil, nil
}

func (n *nova) Run(
	_ context.Context,
	task *llmports.NovaTask,
	_ any,
) (*llmports.StructuredCompletionResult, error) {
	n.task = task
	if n.err != nil {
		return nil, n.err
	}

	return &llmports.StructuredCompletionResult{Text: "{}"}, nil
}

func TestRun_FilesTheInboxsTasksUnderInboundMail(t *testing.T) {
	t.Parallel()

	runner := &nova{}
	caller := triagenova.New(triagenova.Params{Nova: runner})

	_, err := caller.Run(t.Context(), &llmports.NovaTask{Instructions: "Your task: draft a reply."}, &map[string]any{})
	require.NoError(t, err)

	assert.Equal(t, aitask.TaskInboundClassification, runner.task.Name)
	assert.Equal(t, aiusage.FeatureInboundMail, runner.task.Feature)
}

func TestRun_SaysWhenNovaCannotDraftOrCannotBeReached(t *testing.T) {
	t.Parallel()

	_, err := triagenova.New(triagenova.Params{}).Run(t.Context(), &llmports.NovaTask{}, nil)
	require.ErrorIs(t, err, triagenova.ErrUnavailable)

	unconfigured := triagenova.New(triagenova.Params{Nova: &nova{err: llmports.ErrNoProviderConfigured}})
	_, err = unconfigured.Run(t.Context(), &llmports.NovaTask{}, nil)
	require.ErrorIs(t, err, triagenova.ErrUnavailable)

	down := triagenova.New(triagenova.Params{Nova: &nova{
		err: &llmports.ChatProvidersFailedError{Err: errors.New("overloaded")},
	}})
	_, err = down.Run(t.Context(), &llmports.NovaTask{}, nil)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Contains(t, err.Error(), "could not reach an AI provider")

	available, err := unconfigured.Available(t.Context(), pagination.TenantInfo{})
	require.NoError(t, err)
	assert.False(t, available)
}
