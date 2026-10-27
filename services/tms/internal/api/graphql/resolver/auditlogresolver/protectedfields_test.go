package auditlogresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func entryResolver(t *testing.T, denied ...permission.Resource) (*AuditEntryResolver, context.Context) {
	t.Helper()

	names := make([]string, 0, len(denied))
	for _, resource := range denied {
		names = append(names, resource.String())
	}
	deps := &Deps{Core: &base.Core{
		L:                zap.NewNop(),
		PermissionEngine: &mocks.DenyResourcesPermissionEngine{Denied: names},
	}}
	ctx := gqlctx.WithAuthContext(t.Context(), &authctx.AuthContext{
		PrincipalType:  string(permissionports.PrincipalTypeUser),
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})
	return &AuditEntryResolver{deps}, ctx
}

func workerEntry() *audit.Entry {
	return &audit.Entry{
		Resource: permission.ResourceWorker,
		PreviousState: map[string]any{
			"firstName": "Rosa",
			"profile":   map[string]any{"licenseNumber": "D1"},
		},
		CurrentState: map[string]any{
			"firstName": "Rosa",
			"profile":   map[string]any{"licenseNumber": "D2"},
		},
		Changes: map[string]any{
			"profile.licenseNumber": map[string]any{"from": "D1", "to": "D2", "type": "updated"},
		},
	}
}

func TestAnAuditEntryWithholdsPersonalDetailsFromAReaderWithoutTheGroup(t *testing.T) {
	t.Parallel()

	resolver, ctx := entryResolver(t, permission.ResourceProtectedPersonal)
	entry := workerEntry()

	current, err := resolver.CurrentState(ctx, entry)
	require.NoError(t, err)
	assert.Nil(t, current["profile"].(map[string]any)["licenseNumber"])
	assert.Equal(t, "Rosa", current["firstName"])

	previous, err := resolver.PreviousState(ctx, entry)
	require.NoError(t, err)
	assert.Nil(t, previous["profile"].(map[string]any)["licenseNumber"])

	changes, err := resolver.Changes(ctx, entry)
	require.NoError(t, err)
	change := changes["profile.licenseNumber"].(map[string]any)
	assert.Nil(t, change["from"])
	assert.Nil(t, change["to"])
	assert.Equal(t, "updated", change["type"])

	assert.Equal(t, "D2", entry.CurrentState["profile"].(map[string]any)["licenseNumber"],
		"the stored entry keeps the whole diff")
}

func TestAnAuditEntryWithholdsMarginsFromAReaderWithoutTheGroup(t *testing.T) {
	t.Parallel()

	entry := &audit.Entry{
		Resource:     permission.ResourceRateAgreement,
		CurrentState: map[string]any{"name": "Acme", "marginFloorPercent": "12"},
	}

	denied, ctx := entryResolver(t, permission.ResourceProtectedMargins)
	current, err := denied.CurrentState(ctx, entry)
	require.NoError(t, err)
	assert.Nil(t, current["marginFloorPercent"])
	assert.Equal(t, "Acme", current["name"])

	granted, ctx := entryResolver(t)
	current, err = granted.CurrentState(ctx, entry)
	require.NoError(t, err)
	assert.Equal(t, "12", current["marginFloorPercent"])
}

func TestAnAuditEntryShowsTheWholeDiffToAReaderWithEveryGroup(t *testing.T) {
	t.Parallel()

	resolver, ctx := entryResolver(t)
	entry := workerEntry()

	changes, err := resolver.Changes(ctx, entry)
	require.NoError(t, err)
	assert.Equal(t, "D2", changes["profile.licenseNumber"].(map[string]any)["to"])
}
