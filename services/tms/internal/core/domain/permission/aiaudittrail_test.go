package permission_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAIAuditTrail_IsAConfidentialAdministrationResource(t *testing.T) {
	t.Parallel()

	def, ok := permission.NewRegistry().Get(permission.ResourceAIAuditTrail.String())
	require.True(t, ok, "ai_audit_trail must be registered")

	assert.Equal(t, "ai_audit_trail", permission.ResourceAIAuditTrail.String())
	assert.Equal(t, "Administration", def.Category)
	assert.Equal(t, permission.SensitivityConfidential, def.DefaultSensitivity)

	operations := make([]permission.Operation, 0, len(def.Operations))
	for _, op := range def.Operations {
		operations = append(operations, op.Operation)
	}
	assert.Equal(t, []permission.Operation{permission.OpRead, permission.OpExport}, operations)
}

func TestAIAuditTrail_IsNeverAnAgentsToHold(t *testing.T) {
	t.Parallel()

	for _, op := range []permission.Operation{permission.OpRead, permission.OpExport} {
		assert.False(t, permission.IsAgentAllowed(permission.ResourceAIAuditTrail, op),
			"an agent must not read or export the record of what agents did")
	}
}

func TestAIAuditTrail_IsGrantedWithEveryAdministrationResource(t *testing.T) {
	t.Parallel()

	registry := permission.NewRegistry()
	administration := registry.GetByCategory("Administration")

	var trail, auditLog bool
	for _, def := range administration {
		switch def.Resource {
		case permission.ResourceAIAuditTrail.String():
			trail = true
		case permission.ResourceAuditLog.String():
			auditLog = true
		}
	}

	assert.True(t, auditLog)
	assert.True(t, trail, "an administrator granted every Administration resource gets the trail")
}
