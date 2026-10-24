package agentsubjectrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/stretchr/testify/require"
)

// A subject type with no table is one agent_raise_exception refuses outright, so a
// new kind of subject has to say where its records live.
func TestEverySubjectTypeHasATable(t *testing.T) {
	t.Parallel()

	for _, subject := range agent.AllSubjectTypes() {
		if subject == agent.SubjectOrganization {
			continue
		}
		exists, ok := subjectTables[subject]
		require.Truef(t, ok, "subject type %q has no table to check", subject)
		require.NotNil(t, exists)
	}
}
