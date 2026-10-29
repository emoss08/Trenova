package completionrouter

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUsableFor_TheScopeGuardFallsBackToTheChatModel(t *testing.T) {
	t.Parallel()

	chat := openAIChatProvider("hosted", "http://127.0.0.1:1", 5)
	chat.Tasks = []aitask.Task{aitask.TaskAssistantChat}

	usable, err := newTestService(t, chat).
		usableFor(t.Context(), aitask.TaskScopeClassification, availabilityTenant())
	require.NoError(t, err)
	require.Len(t, usable, 1)
	assert.Equal(t, chat.ID, usable[0].ID, "no provider assigned to the guard still runs it")
}

func TestUsableFor_TheScopeGuardTriesItsOwnProvidersFirst(t *testing.T) {
	t.Parallel()

	local := openAIChatProvider("local", "http://127.0.0.1:1", 10)
	local.Tasks = []aitask.Task{aitask.TaskScopeClassification}
	chat := openAIChatProvider("hosted", "http://127.0.0.1:2", 5)
	chat.Tasks = []aitask.Task{aitask.TaskAssistantChat, aitask.TaskScopeClassification}

	usable, err := newTestService(t, local, chat).
		usableFor(t.Context(), aitask.TaskScopeClassification, availabilityTenant())
	require.NoError(t, err)
	require.Len(t, usable, 2, "a provider assigned to both is listed once")
	assert.Equal(t, local.ID, usable[0].ID)
	assert.Equal(t, chat.ID, usable[1].ID)
}

func TestUsableFor_OtherTasksBorrowNothing(t *testing.T) {
	t.Parallel()

	chat := openAIChatProvider("hosted", "http://127.0.0.1:1", 5)
	chat.Tasks = []aitask.Task{aitask.TaskAssistantChat}

	_, err := newTestService(t, chat).
		usableFor(t.Context(), aitask.TaskDocumentExtraction, availabilityTenant())
	require.Error(t, err)
}
