package permission_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/stretchr/testify/assert"
)

// The agent principal is bounded by this list before any definition's tool
// list applies: a tool an organization enables still cannot reach a resource
// the platform never lets an agent touch.
func TestIsAgentAllowedCoversWhatTheRegisteredToolsNeed(t *testing.T) {
	t.Parallel()

	allowed := []struct {
		resource  permission.Resource
		operation permission.Operation
	}{
		{permission.ResourceShipment, permission.OpRead},
		{permission.ResourceShipmentMove, permission.OpRead},
		{permission.ResourceShipmentMove, permission.OpUpdate},
		{permission.ResourceWorker, permission.OpRead},
		{permission.ResourceTractor, permission.OpRead},
		{permission.ResourceTrailer, permission.OpRead},
		{permission.ResourceCustomer, permission.OpRead},
		{permission.ResourceLocation, permission.OpRead},
		{permission.ResourceBillingQueue, permission.OpRead},
		{permission.ResourceBillingQueue, permission.OpUpdate},
		{permission.ResourceDocument, permission.OpRead},
		{permission.ResourceDocument, permission.OpCreate},
		{permission.ResourceAgentException, permission.OpCreate},
	}
	for _, entry := range allowed {
		assert.True(t, permission.IsAgentAllowed(entry.resource, entry.operation),
			"%s %s should be allowed for an agent", entry.resource, entry.operation)
	}
}

func TestIsAgentAllowedNeverGrantsApprovalOrDeletion(t *testing.T) {
	t.Parallel()

	denied := []struct {
		resource  permission.Resource
		operation permission.Operation
	}{
		{permission.ResourceBillingQueue, permission.OpApprove},
		{permission.ResourceShipment, permission.OpDelete},
		{permission.ResourceShipmentMove, permission.OpDelete},
		{permission.ResourceWorker, permission.OpUpdate},
		{permission.ResourceUser, permission.OpRead},
		{permission.ResourceRole, permission.OpRead},
		{permission.ResourceAPIKey, permission.OpRead},
	}
	for _, entry := range denied {
		assert.False(t, permission.IsAgentAllowed(entry.resource, entry.operation),
			"%s %s must stay closed to an agent", entry.resource, entry.operation)
	}
}
