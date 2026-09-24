package agentsubjectrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/stretchr/testify/require"
)

// A subject type with no table is one raise_exception refuses outright, so a
// new kind of subject has to say where its records live.
func TestEverySubjectTypeHasATable(t *testing.T) {
	t.Parallel()

	for _, subject := range agent.AllSubjectTypes() {
		if subject == agent.SubjectOrganization {
			continue
		}
		table, ok := subjectTables[subject]
		require.Truef(t, ok, "subject type %q has no table to check", subject)
		require.NotNil(t, table.model)
		require.NotNil(t, table.tenant)
		require.NotEmpty(t, table.id.Name)
	}
}
