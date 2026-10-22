package refview_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/app/refview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/stretchr/testify/assert"
)

func role(max permission.FieldSensitivity, grants ...*permission.ResourcePermission) *permission.Role {
	return &permission.Role{MaxSensitivity: max, Permissions: grants}
}

func grant(resource permission.Resource, ops ...permission.Operation) *permission.ResourcePermission {
	return &permission.ResourcePermission{Resource: resource.String(), Operations: ops}
}

func TestRoleAccess_ReadsWhatItsGrantsImply(t *testing.T) {
	t.Parallel()

	registry := permission.NewRegistry()
	access := refview.NewRoleAccess(registry, permission.GrantsOf(
		role(permission.SensitivityInternal, grant(permission.ResourceCustomer, permission.OpUpdate)),
	))

	assert.True(t, access.Reads(t.Context(), permission.ResourceCustomer), "update brings read")
	assert.False(t, access.Reads(t.Context(), permission.ResourceWorker))
	assert.True(t, access.Allows(permission.ResourceCustomer.String(), permission.OpUpdate))
	assert.False(t, access.Allows(permission.ResourceCustomer.String(), permission.OpExport))
}

func TestRoleAccess_HoldsALinkedRecordsViewToTheRolesCeiling(t *testing.T) {
	t.Parallel()

	views := refview.New(permission.NewRegistry())
	public := refview.NewRoleAccess(nil, permission.GrantsOf(role(permission.SensitivityPublic)))
	internal := refview.NewRoleAccess(nil, permission.GrantsOf(
		role(permission.SensitivityPublic),
		role(permission.SensitivityInternal),
	))

	assert.Equal(t, refview.WithheldSensitivity,
		views.Decide(t.Context(), public, permission.ResourceWorker, "wholeName", refview.Nested))
	assert.Equal(t, refview.Visible,
		views.Decide(t.Context(), internal, permission.ResourceWorker, "wholeName", refview.Nested),
		"the widest ceiling among the roles applies")
	assert.Equal(t, refview.Visible,
		views.Decide(t.Context(), public, permission.ResourceWorker, "wholeName", refview.Root),
		"a root record is the screen's own")
}

func TestDecidePart_APartOfALinkedRecordNeedsItsKindsRead(t *testing.T) {
	t.Parallel()

	views := refview.New(permission.NewRegistry())
	none := refview.NewRoleAccess(nil, permission.GrantsOf(role(permission.SensitivityInternal)))
	reader := refview.NewRoleAccess(nil, permission.GrantsOf(
		role(permission.SensitivityInternal, grant(permission.ResourceShipment, permission.OpRead)),
	))

	assert.Equal(t, refview.WithheldKind,
		views.DecidePart(t.Context(), none, permission.ResourceShipment, "proNumber", refview.Nested),
		"a part has no view, even for a field named like one in its carrier's")
	assert.Equal(t, refview.Visible,
		views.DecidePart(t.Context(), reader, permission.ResourceShipment, "status", refview.Nested))
	assert.Equal(t, refview.Visible,
		views.DecidePart(t.Context(), none, permission.ResourceShipment, "status", refview.Root))
}
