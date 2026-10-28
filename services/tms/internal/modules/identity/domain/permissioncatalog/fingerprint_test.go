package permissioncatalog

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGrantFingerprintIsStableAcrossRegistries(t *testing.T) {
	t.Parallel()

	assert.Equal(t, NewRegistry().GrantFingerprint(), NewRegistry().GrantFingerprint())
	assert.Len(t, NewRegistry().GrantFingerprint(), 64)
}

func TestGrantFingerprintChangesWithAnOperation(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	before := registry.GrantFingerprint()

	require.NoError(t, registry.Register(&ResourceDefinition{
		Resource:   "fingerprint_probe",
		Operations: []OperationDefinition{{Operation: permission.OpRead}},
	}))
	added := registry.GrantFingerprint()
	assert.NotEqual(t, before, added)

	def, ok := registry.Get("fingerprint_probe")
	require.True(t, ok)
	def.Operations = append(def.Operations, OperationDefinition{Operation: permission.OpUpdate})
	assert.NotEqual(t, added, registry.GrantFingerprint())
}
