package completionengine

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAStructuredCallCarriesItsFilesWithAnEntryForEach(t *testing.T) {
	t.Parallel()

	messages := structuredMessages(&RunRequest{
		UserContent: "Read this.",
		Files:       []llmports.MessageFile{{DocumentID: "doc_1", FileName: "pod.pdf"}},
	})

	require.Len(t, messages, 1)
	message := messages[0]
	require.Len(t, message.Files, 1)
	file := message.Files[0]
	assert.Equal(t, "[Attached file: pod.pdf]", message.Content[file.EntryStart:file.EntryEnd])
	assert.Equal(t, "[The file pod.pdf is attached]", file.NativeEntry)
	assert.Equal(t, "doc_1", file.DocumentID)
}

func TestAStructuredCallWithoutFilesIsOnePlainMessage(t *testing.T) {
	t.Parallel()

	messages := structuredMessages(&RunRequest{UserContent: "Hello"})

	require.Len(t, messages, 1)
	assert.Equal(t, "Hello", messages[0].Content)
	assert.Empty(t, messages[0].Files)
}
