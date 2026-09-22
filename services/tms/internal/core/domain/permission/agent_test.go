package permission_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		{permission.ResourceDocument, permission.OpUpdate},
		{permission.ResourceAgentException, permission.OpCreate},
		{permission.ResourceWorkerCredential, permission.OpRead},
		{permission.ResourceDriverMessage, permission.OpCreate},
		{permission.ResourceCustomerCommunication, permission.OpCreate},
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
		{permission.ResourceShipment, permission.OpUpdate},
		{permission.ResourceDriverMessage, permission.OpDelete},
		{permission.ResourceCustomerCommunication, permission.OpUpdate},
		{permission.ResourceUser, permission.OpRead},
		{permission.ResourceRole, permission.OpRead},
		{permission.ResourceAPIKey, permission.OpRead},
	}
	for _, entry := range denied {
		assert.False(t, permission.IsAgentAllowed(entry.resource, entry.operation),
			"%s %s must stay closed to an agent", entry.resource, entry.operation)
	}
}

// Sending something outside the organization is its own permission, held
// apart from editing the record it is about: a dispatcher who may change a
// shipment does not thereby get to email its customer.
func TestCommunicationResourcesAreRegisteredAsSendOnly(t *testing.T) {
	t.Parallel()

	registry := permission.NewRegistry()
	for _, resource := range []permission.Resource{
		permission.ResourceDriverMessage,
		permission.ResourceCustomerCommunication,
	} {
		def, ok := registry.Get(resource.String())
		require.True(t, ok, "%s must be registered", resource)
		assert.Equal(t, "Communications", def.Category)

		operations := make([]permission.Operation, 0, len(def.Operations))
		for _, op := range def.Operations {
			operations = append(operations, op.Operation)
		}
		assert.ElementsMatch(t,
			[]permission.Operation{permission.OpRead, permission.OpCreate}, operations,
			"%s offers exactly read and send", resource)
	}
}
