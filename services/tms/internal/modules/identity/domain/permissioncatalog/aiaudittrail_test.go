package permissioncatalog_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAIAuditTrail_IsAConfidentialAdministrationResource(t *testing.T) {
	t.Parallel()

	def, ok := permissioncatalog.NewRegistry().Get(permission.ResourceAIAuditTrail.String())
	require.True(t, ok, "ai_audit_trail must be registered")

	assert.Equal(t, "ai_audit_trail", permission.ResourceAIAuditTrail.String())
	assert.Equal(t, "Users & security", def.Category)
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

func TestAIAuditTrail_IsGrantedWithEveryUsersAndSecurityResource(t *testing.T) {
	t.Parallel()

	registry := permissioncatalog.NewRegistry()
	security := registry.GetByCategory("Users & security")

	var trail, auditLog bool
	for _, def := range security {
		switch def.Resource {
		case permission.ResourceAIAuditTrail.String():
			trail = true
		case permission.ResourceAuditLog.String():
			auditLog = true
		}
	}

	assert.True(t, auditLog)
	assert.True(t, trail, "an auditor granted every Users & security resource gets the trail")
}
