package validationframework

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeelTableReferenceCheckRefusesAnInvalidTableName(t *testing.T) {
	t.Parallel()

	for _, table := range []string{"", "GL_ACCOUNTS", `gl"accounts`, "gl_accounts; drop", "1table"} {
		assert.Panics(t, func() { KeelTableReferenceCheck(nil, table) }, table)
	}
}

func TestKeelTableReferenceCheckAcceptsAnEmptyReference(t *testing.T) {
	t.Parallel()

	check := KeelTableReferenceCheck(nil, "gl_accounts")

	exists, err := check(t.Context(), pulid.ID("org_1"), pulid.ID("bu_1"), "")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestKeelTableReferenceCheckNeedsAConnection(t *testing.T) {
	t.Parallel()

	check := KeelTableReferenceCheck(nil, "gl_accounts")

	_, err := check(t.Context(), pulid.ID("org_1"), pulid.ID("bu_1"), pulid.ID("gla_1"))
	require.ErrorIs(t, err, errDBNotInitialized)
}
