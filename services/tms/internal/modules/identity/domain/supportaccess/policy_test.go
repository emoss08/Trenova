package supportaccess_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/agentceiling"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A resource of a sensitive kind is one a new feature adds without thinking about Trenova
// staff. This fails until the support policy names it, denied or reviewed.
func TestEverySensitiveResourceIsClassifiedForSupport(t *testing.T) {
	t.Parallel()

	registry := permissioncatalog.NewRegistry()
	var unclassified []string
	for _, def := range registry.All() {
		if supportaccess.SensitiveResource(def.Resource, def.Category) &&
			!supportaccess.Classified(def.Resource) {
			unclassified = append(unclassified, def.Resource+" ("+def.Category+")")
		}
	}
	require.Empty(t, unclassified,
		"classify each in supportaccess/policy.go: denied (with what it guards) or reviewed (with why support may use it)")
}

func TestReadOnlySupportRefusesEveryOperationButReadAcrossTheRegistry(t *testing.T) {
	t.Parallel()

	for _, def := range permissioncatalog.NewRegistry().All() {
		for _, op := range def.Operations {
			allowed := supportaccess.Allows(supportaccess.ScopeReadOnly, def.Resource, op.Operation)
			if op.Operation != permission.OpRead {
				assert.False(t, allowed, "%s:%s", def.Resource, op.Operation)
			}
		}
	}
}

func TestNoSupportSessionTakesWhatTheDenyListGuards(t *testing.T) {
	t.Parallel()

	never := []struct {
		resource permission.Resource
		ops      []permission.Operation
	}{
		{permission.ResourceUser, []permission.Operation{permission.OpCreate, permission.OpUpdate, permission.OpAssign, permission.OpUnassign}},
		{permission.ResourceRole, []permission.Operation{permission.OpCreate, permission.OpUpdate, permission.OpDelete}},
		{permission.ResourceAccessPolicy, []permission.Operation{permission.OpCreate, permission.OpUpdate, permission.OpDelete}},
		{permission.ResourceIdentityProvider, []permission.Operation{permission.OpRead, permission.OpCreate, permission.OpUpdate}},
		{permission.ResourceSCIMDirectory, []permission.Operation{permission.OpRead, permission.OpCreate}},
		{permission.ResourceAPIKey, []permission.Operation{permission.OpRead, permission.OpCreate}},
		{permission.ResourceSupportAccess, []permission.Operation{permission.OpRead, permission.OpCreate, permission.OpDelete}},
		{permission.ResourceIntegration, []permission.Operation{permission.OpRead, permission.OpUpdate}},
		{permission.ResourceAccountingIntegration, []permission.Operation{permission.OpRead, permission.OpUpdate}},
		{permission.ResourceAIProvider, []permission.Operation{permission.OpRead, permission.OpCreate}},
		{permission.ResourceEmailProfile, []permission.Operation{permission.OpRead, permission.OpUpdate}},
		{permission.ResourceFactoringCompany, []permission.Operation{permission.OpUpdate, permission.OpCreate}},
		{permission.ResourceDocumentTemplate, []permission.Operation{permission.OpUpdate, permission.OpActivate}},
		{permission.ResourceDriverPortal, []permission.Operation{permission.OpCreate}},
		{permission.ResourceCaptureDevice, []permission.Operation{permission.OpCreate, permission.OpUpdate}},
	}
	for _, scope := range []supportaccess.Scope{supportaccess.ScopeReadOnly, supportaccess.ScopeFull} {
		for _, n := range never {
			for _, op := range n.ops {
				assert.False(t, supportaccess.Allows(scope, n.resource.String(), op),
					"%s support may not %s %s", scope, op, n.resource)
			}
		}
	}
}

func TestNoSupportSessionExportsInBulk(t *testing.T) {
	t.Parallel()

	for _, def := range permissioncatalog.NewRegistry().All() {
		assert.False(t, supportaccess.Allows(supportaccess.ScopeFull, def.Resource, permission.OpExport), def.Resource)
	}
}

func TestFullSupportHasAnAdministratorsReachOutsideTheDenyList(t *testing.T) {
	t.Parallel()

	assert.True(t, supportaccess.Allows(supportaccess.ScopeFull, permission.ResourceShipment.String(), permission.OpUpdate))
	assert.True(t, supportaccess.Allows(supportaccess.ScopeFull, permission.ResourceInvoice.String(), permission.OpUpdate))
	assert.True(t, supportaccess.Allows(supportaccess.ScopeFull, permission.ResourceUser.String(), permission.OpRead))
	assert.True(t, supportaccess.Allows(supportaccess.ScopeReadOnly, permission.ResourceShipment.String(), permission.OpRead))
	assert.False(t, supportaccess.Allows(supportaccess.Scope("Admin"), permission.ResourceShipment.String(), permission.OpRead),
		"an unknown scope allows nothing")
}

func TestSupportAccessIsNeverAnAgentsToHold(t *testing.T) {
	t.Parallel()

	for _, op := range []permission.Operation{permission.OpRead, permission.OpCreate, permission.OpDelete} {
		assert.False(t, agentceiling.Allows(permission.ResourceSupportAccess, op))
	}
}

func TestSupportAccessIsAPersonOnlyAdministrationResource(t *testing.T) {
	t.Parallel()

	def, ok := permissioncatalog.NewRegistry().Get(permission.ResourceSupportAccess.String())
	require.True(t, ok)
	assert.Equal(t, "Users & security", def.Category)
	assert.Equal(t, permission.SensitivityConfidential, def.DefaultSensitivity)
	assert.Empty(t, def.ParentResource, "a parent's grant would reach support access")
	var ops []permission.Operation
	for _, op := range def.Operations {
		ops = append(ops, op.Operation)
	}
	assert.ElementsMatch(t, []permission.Operation{permission.OpRead, permission.OpCreate, permission.OpDelete}, ops)
}

func TestEffectiveSupportNeverReachesADeniedChildThroughItsParent(t *testing.T) {
	t.Parallel()

	registry := permissioncatalog.NewRegistry()
	for _, scope := range []supportaccess.Scope{supportaccess.ScopeReadOnly, supportaccess.ScopeFull} {
		effective := supportaccess.Effective(scope, registry.All())
		for _, def := range registry.All() {
			for _, op := range def.Operations {
				if supportaccess.Allows(scope, def.Resource, op.Operation) {
					continue
				}
				assert.NotContains(t, effective[def.Resource], op.Operation, "%s:%s", def.Resource, op.Operation)
				if def.ParentResource != "" {
					assert.NotContains(t, effective[def.ParentResource], op.Operation,
						"%s's parent %s would carry %s to it", def.Resource, def.ParentResource, op.Operation)
				}
			}
		}
	}
}
