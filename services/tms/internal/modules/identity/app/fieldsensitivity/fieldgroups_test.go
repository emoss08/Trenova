package fieldsensitivity

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

type groupEngine struct {
	permissionports.PermissionEngine

	allowed map[string]bool
	failing map[string]bool
	asked   []*permissionports.PermissionCheckRequest
}

func (g *groupEngine) Check(
	_ context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	g.asked = append(g.asked, req)
	if g.failing[req.Resource] {
		return nil, assert.AnError
	}
	return &permissionports.PermissionCheckResult{Allowed: g.allowed[req.Resource]}, nil
}

func sessionActor() *permissionports.RequestActor {
	userID := pulid.MustNew("usr_")
	return &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
}

func TestFieldAccessGrantsTheGroupsTheActorCanRead(t *testing.T) {
	t.Parallel()

	engine := &groupEngine{allowed: map[string]bool{
		permission.ResourceProtectedMargins.String(): true,
	}}
	actor := sessionActor()

	visible := NewFieldAccess(engine).Visible(t.Context(), actor)

	assert.True(t, visible.Has(permission.FieldGroupMargins))
	assert.False(t, visible.Has(permission.FieldGroupPersonal))
	assert.Len(t, engine.asked, 2)
	for _, req := range engine.asked {
		assert.Equal(t, permission.OpRead, req.Operation)
		assert.Equal(t, actor.UserID, req.UserID)
		assert.Equal(t, actor.OrganizationID, req.OrganizationID)
	}
}

func TestFieldAccessWithholdsAGroupWhoseCheckFails(t *testing.T) {
	t.Parallel()

	engine := &groupEngine{
		allowed: map[string]bool{
			permission.ResourceProtectedPersonal.String(): true,
			permission.ResourceProtectedMargins.String():  true,
		},
		failing: map[string]bool{permission.ResourceProtectedPersonal.String(): true},
	}

	visible := NewFieldAccess(engine).Visible(t.Context(), sessionActor())

	assert.False(t, visible.Has(permission.FieldGroupPersonal))
	assert.True(t, visible.Has(permission.FieldGroupMargins))
}

func TestFieldAccessShowsEverythingToTheSystem(t *testing.T) {
	t.Parallel()

	engine := &groupEngine{}
	access := NewFieldAccess(engine)

	assert.Equal(t, permission.AllFieldGroups(), access.Visible(t.Context(), nil))
	assert.Equal(t, permission.AllFieldGroups(), access.Visible(t.Context(),
		&permissionports.RequestActor{PrincipalType: permissionports.PrincipalTypeSystem}))
	assert.Empty(t, engine.asked)
}

func TestVisibleFieldGroupsWithoutAccessShowsEverything(t *testing.T) {
	t.Parallel()

	assert.Equal(t, permission.AllFieldGroups(),
		permissionports.VisibleFieldGroups(t.Context(), nil, sessionActor()))
}
