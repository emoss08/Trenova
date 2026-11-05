package scimservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/scim"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/scimproto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newUserResource(email string) *scimproto.User {
	return &scimproto.User{
		Schemas:  []string{scimproto.SchemaUser},
		UserName: email,
		Name:     &scimproto.Name{GivenName: "Grace", FamilyName: "Hopper"},
		Emails:   []scimproto.MultiValue{{Value: email, Type: "work", Primary: true}},
		Timezone: "America/Chicago",
		Active:   scimproto.BoolOf(true),
	}
}

func TestCreateUser_CreatesTheAccountInTheDirectorysOrganization(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	created, err := h.svc.CreateUser(t.Context(), h.dir, newUserResource("grace@example.com"))

	require.NoError(t, err)
	require.Len(t, h.accounts.created, 1)
	request := h.accounts.created[0]
	assert.Equal(t, h.dir.OrganizationID, request.Actor.OrganizationID)
	assert.Equal(t, h.dir.BusinessUnitID, request.Actor.BusinessUnitID)
	assert.Equal(t, h.dir.ID, request.Actor.DirectoryID)
	assert.Equal(t, "grace@example.com", request.Profile.EmailAddress)
	assert.Equal(t, "Grace Hopper", request.Profile.Name)
	require.Len(t, h.repo.links, 1)
	assert.True(t, h.repo.links[0].Provisioned)
	assert.Equal(t, h.dir.ID, h.repo.links[0].DirectoryID)
	assert.Equal(t, h.repo.links[0].UserID.String(), created.ID)
	assert.Contains(t, h.repo.statuses(), statusSucceeded)
}

func TestCreateUser_LinksAMemberTheOrganizationAlreadyHas(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	member := h.activeMember()
	h.repo.member = member

	created, err := h.svc.CreateUser(t.Context(), h.dir, newUserResource(member.EmailAddress))

	require.NoError(t, err)
	assert.Empty(t, h.accounts.created, "no second account is made for someone already here")
	require.Len(t, h.repo.links, 1)
	assert.False(t, h.repo.links[0].Provisioned)
	assert.Equal(t, member.ID.String(), created.ID)
	require.Len(t, h.accounts.restored, 1)
	assert.Equal(t, h.dir.OrganizationID, h.accounts.restored[0].Actor.OrganizationID)
	assert.Contains(t, h.repo.pending, member.ID)
	assert.Equal(t, []pulid.ID{h.dir.ID}, h.requester.requested)
}

func TestCreateUser_NeverLinksTheSystemUser(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.repo.member = &tenant.User{
		ID:           pulid.MustNew("usr_"),
		Username:     tenant.SystemUsername,
		EmailAddress: "system@example.com",
		Status:       domaintypes.StatusActive,
	}

	_, err := h.svc.CreateUser(t.Context(), h.dir, newUserResource("system@example.com"))

	scimErr, ok := scimproto.AsError(err)
	require.True(t, ok)
	assert.Equal(t, scimproto.ScimTypeUniqueness, scimErr.ScimType)
	assert.Empty(t, h.repo.links)
}

func TestCreateUser_NeedsAnEmailAddress(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	resource := newUserResource("grace")
	resource.Emails = nil

	_, err := h.svc.CreateUser(t.Context(), h.dir, resource)

	scimErr, ok := scimproto.AsError(err)
	require.True(t, ok)
	assert.Equal(t, scimproto.ScimTypeInvalidValue, scimErr.ScimType)
	assert.Empty(t, h.accounts.created)
}

func TestPatchUser_DeactivationRevokesAccessInTheDirectorysOrganization(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	member := h.activeMember()
	h.link(member, true)
	patch, err := scimproto.DecodePatch([]byte(`{
		"schemas": ["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
		"Operations": [{"op": "replace", "path": "active", "value": false}]
	}`), 10)
	require.NoError(t, err)

	resource, err := h.svc.PatchUser(t.Context(), h.dir, member.ID, patch.Operations)

	require.NoError(t, err)
	require.Len(t, h.accounts.revoked, 1)
	assert.Equal(t, member.ID, h.accounts.revoked[0].UserID)
	assert.Equal(t, h.dir.OrganizationID, h.accounts.revoked[0].Actor.OrganizationID)
	require.NotEmpty(t, h.repo.saved)
	assert.False(t, h.repo.saved[len(h.repo.saved)-1].Active)
	assert.False(t, resource.IsActive())
	assert.Contains(t, h.repo.pending, member.ID)
	assert.Equal(t, scimproto.ResourceTypeUser, resource.Meta.ResourceType)
}

func TestPatchUser_ActsOnlyOnPeopleThisDirectoryLinked(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	other := h.activeMember()
	foreign := h.link(other, true)
	foreign.DirectoryID = pulid.MustNew("scd_")
	patch, err := scimproto.DecodePatch([]byte(`{
		"schemas": ["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
		"Operations": [{"op": "replace", "path": "active", "value": false}]
	}`), 10)
	require.NoError(t, err)

	_, err = h.svc.PatchUser(t.Context(), h.dir, other.ID, patch.Operations)

	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))
	assert.Empty(t, h.accounts.revoked)
	assert.Empty(t, h.repo.saved)
}

func TestDeleteUser_EndsAccessAndDirectoryGrantsButKeepsTheAccount(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	member := h.activeMember()
	h.link(member, true)
	grant := &scim.RoleGrant{
		UserID:       member.ID,
		RoleID:       pulid.MustNew("rol_"),
		AssignmentID: pulid.MustNew("ura_"),
	}
	h.repo.grants = []*scim.RoleGrant{grant}

	require.NoError(t, h.svc.DeleteUser(t.Context(), h.dir, member.ID))

	require.Len(t, h.accounts.revoked, 1)
	require.Len(t, h.roles.unassigned, 1)
	assert.Equal(t, grant.AssignmentID, h.roles.unassigned[0].AssignmentID)
	assert.Equal(t, []pulid.ID{member.ID}, h.repo.deleted)
	assert.Empty(t, h.accounts.created)
}
