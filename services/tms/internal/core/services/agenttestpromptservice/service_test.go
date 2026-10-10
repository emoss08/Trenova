package agenttestpromptservice

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type definitions struct {
	agentdefinitionports.AgentDefinitionRepository
}

func (definitions) GetByID(
	context.Context,
	agentdefinitionports.GetAgentDefinitionByIDRequest,
) (*agentdefinition.Definition, error) {
	return &agentdefinition.Definition{}, nil
}

type prompts struct {
	count   int
	created []*agentdefinition.TestPrompt
}

func (p *prompts) List(
	context.Context,
	*agentdefinitionports.ListAgentTestPromptsRequest,
) ([]*agentdefinition.TestPrompt, error) {
	return p.created, nil
}

func (p *prompts) Count(context.Context, *agentdefinitionports.ListAgentTestPromptsRequest) (int, error) {
	return p.count, nil
}

func (p *prompts) Create(_ context.Context, prompt *agentdefinition.TestPrompt) error {
	p.created = append(p.created, prompt)
	return nil
}

func (p *prompts) Delete(context.Context, *agentdefinitionports.DeleteAgentTestPromptRequest) error {
	return nil
}

func TestKeepTrimsAndRecordsWhoKeptIt(t *testing.T) {
	t.Parallel()

	store := &prompts{}
	svc := New(Params{Definitions: definitions{}, Prompts: store})
	user := pulid.MustNew("usr_")

	kept, err := svc.Keep(t.Context(), &agentdefinitionports.KeepAgentTestPromptRequest{
		AgentDefinitionID: pulid.MustNew("agdef_"),
		Prompt:            "  Which invoices are on hold?  ",
		CreatedByID:       user,
	})

	require.NoError(t, err)
	assert.Equal(t, "Which invoices are on hold?", kept.Prompt)
	assert.Equal(t, user, *kept.CreatedByID)
	assert.Len(t, store.created, 1)
}

func TestKeepRefusesAnEmptyOrLongPrompt(t *testing.T) {
	t.Parallel()

	svc := New(Params{Definitions: definitions{}, Prompts: &prompts{}})
	for _, text := range []string{"   ", strings.Repeat("a", agentdefinition.MaxTestPromptRunes+1)} {
		_, err := svc.Keep(t.Context(), &agentdefinitionports.KeepAgentTestPromptRequest{Prompt: text})

		var multiErr *errortypes.MultiError
		require.ErrorAs(t, err, &multiErr)
	}
}

func TestKeepRefusesPastTheLimit(t *testing.T) {
	t.Parallel()

	store := &prompts{count: agentdefinition.MaxTestPrompts}
	svc := New(Params{Definitions: definitions{}, Prompts: store})

	_, err := svc.Keep(t.Context(), &agentdefinitionports.KeepAgentTestPromptRequest{Prompt: "One more"})

	require.Error(t, err)
	assert.Empty(t, store.created)
}
