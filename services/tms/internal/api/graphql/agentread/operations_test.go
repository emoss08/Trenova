package agentread

import (
	"os"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// An operation the agent reads through is the persisted document the client
// sends, byte for byte. A client edit re-hashes it; until agentread/gen runs
// again the tool would read an operation no screen sends any more.
func TestEveryBoundOperationIsPersistedAsItIsRun(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("../persisted-documents.json")
	require.NoError(t, err)
	var manifest map[string]string
	require.NoError(t, sonic.Unmarshal(data, &manifest))

	require.NotEmpty(t, Operations())
	for _, op := range Operations() {
		document, persisted := manifest[op.Hash]
		if assert.Truef(t, persisted,
			"%s is not persisted under %s; run go run ./internal/api/graphql/agentread/gen",
			op.Name, op.Hash) {
			assert.Equalf(t, document, op.Document,
				"%s changed; run go run ./internal/api/graphql/agentread/gen", op.Name)
		}
	}
}
