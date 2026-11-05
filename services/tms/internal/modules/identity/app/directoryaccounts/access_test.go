package directoryaccounts

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/app/userservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/testutil/securityaudittest"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeUsers struct {
	tenantports.UserRepository

	user        *tenant.User
	memberships []*tenant.OrganizationMembership
	updated     []tenantports.UpdateUserColumnsRequest
	replaced    []tenantports.ReplaceOrganizationMembershipsRequest
}

func (f *fakeUsers) GetTenantMember(
	_ context.Context,
	req tenantports.GetTenantMemberRequest,
) (*tenant.User, error) {
	if f.user == nil || f.user.ID != req.UserID {
		return nil, errortypes.NewNotFoundError("User not found")
	}
	copied := *f.user
	return &copied, nil
}

func (f *fakeUsers) ListOrganizationMemberships(
	_ context.Context,
	_, _ pulid.ID,
) ([]*tenant.OrganizationMembership, error) {
	return f.memberships, nil
}

func (f *fakeUsers) UpdateColumns(
	_ context.Context,
	req tenantports.UpdateUserColumnsRequest,
) (*tenant.User, error) {
	f.updated = append(f.updated, req)
	return req.Entity, nil
}

func (f *fakeUsers) ReplaceOrganizationMemberships(
	_ context.Context,
	req tenantports.ReplaceOrganizationMembershipsRequest,
) ([]*tenant.OrganizationMembership, error) {
	f.replaced = append(f.replaced, req)
	memberships := make([]*tenant.OrganizationMembership, 0, len(req.OrganizationIDs))
	for _, orgID := range req.OrganizationIDs {
		memberships = append(memberships, &tenant.OrganizationMembership{
			UserID:         req.UserID,
			OrganizationID: orgID,
			BusinessUnitID: req.BusinessUnitID,
		})
	}
	return memberships, nil
}

type fakeLifecycle struct {
	exits []*userservice.OrganizationExit
	ended []pulid.ID
}

func (f *fakeLifecycle) ExitOrganization(
	_ context.Context,
	req *userservice.OrganizationExit,
) error {
	f.exits = append(f.exits, req)
	return nil
}

func (f *fakeLifecycle) EndUserAccess(
	_ context.Context,
	_ pagination.TenantInfo,
	userID pulid.ID,
) error {
	f.ended = append(f.ended, userID)
	return nil
}

func (f *fakeLifecycle) PublishUserChange(
	context.Context,
	pagination.TenantInfo,
	*tenant.User,
	string,
) {
}

type accessHarness struct {
	svc       *Service
	users     *fakeUsers
	lifecycle *fakeLifecycle
	auditor   *securityaudittest.Recorder
	actor     Actor
}

func newAccessHarness(t *testing.T, otherOrganizations int) *accessHarness {
	t.Helper()

	actor := Actor{
		DirectoryID:    pulid.MustNew("scd_"),
		TokenID:        pulid.MustNew("sct_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
	user := &tenant.User{
		ID:                    pulid.MustNew("usr_"),
		BusinessUnitID:        actor.BusinessUnitID,
		CurrentOrganizationID: actor.OrganizationID,
		Name:                  "Ada Lovelace",
		EmailAddress:          "ada@example.com",
		Status:                domaintypes.StatusActive,
	}
	memberships := []*tenant.OrganizationMembership{{
		UserID:         user.ID,
		OrganizationID: actor.OrganizationID,
		BusinessUnitID: actor.BusinessUnitID,
	}}
	for range otherOrganizations {
		memberships = append(memberships, &tenant.OrganizationMembership{
			UserID:         user.ID,
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: actor.BusinessUnitID,
		})
	}

	h := &accessHarness{
		users:     &fakeUsers{user: user, memberships: memberships},
		lifecycle: &fakeLifecycle{},
		auditor:   &securityaudittest.Recorder{},
		actor:     actor,
	}
	h.svc = &Service{users: h.users, lifecycle: h.lifecycle, auditor: h.auditor}

	return h
}

func TestRevoke_ForAPersonInOtherOrganizationsEndsOnlyThisOrganizationsAccess(t *testing.T) {
	t.Parallel()

	h := newAccessHarness(t, 1)

	require.NoError(
		t,
		h.svc.Revoke(t.Context(), &AccessRequest{Actor: h.actor, UserID: h.users.user.ID}),
	)

	require.Len(t, h.lifecycle.exits, 1)
	exit := h.lifecycle.exits[0]
	assert.Equal(t, h.actor.OrganizationID, exit.OrganizationID)
	assert.Equal(t, h.users.user.ID, exit.UserID)
	assert.Equal(t, h.actor.DirectoryID.String(), exit.Metadata[metadataDirectoryID])
	assert.Empty(t, h.users.updated, "the account itself stays active for the other organizations")
	assert.Empty(t, h.lifecycle.ended)
}

func TestRevoke_ForAPersonOnlyHereDeactivatesTheAccount(t *testing.T) {
	t.Parallel()

	h := newAccessHarness(t, 0)

	require.NoError(
		t,
		h.svc.Revoke(t.Context(), &AccessRequest{Actor: h.actor, UserID: h.users.user.ID}),
	)

	assert.Empty(t, h.lifecycle.exits)
	require.Len(t, h.users.updated, 1)
	assert.Equal(t, domaintypes.StatusInactive, h.users.updated[0].Entity.Status)
	assert.Equal(t, []pulid.ID{h.users.user.ID}, h.lifecycle.ended)
	change := h.auditor.Only(t)
	assert.Equal(t, "User sign-in access changed", change.Comment)
	assert.Equal(t, h.actor.TokenID.String(), change.Metadata[metadataTokenID])
}

func TestRevoke_LeavesSomeoneWhoIsNoLongerAMemberAlone(t *testing.T) {
	t.Parallel()

	h := newAccessHarness(t, 0)
	h.users.user.CurrentOrganizationID = pulid.MustNew("org_")
	h.users.memberships = []*tenant.OrganizationMembership{{
		UserID:         h.users.user.ID,
		OrganizationID: h.users.user.CurrentOrganizationID,
		BusinessUnitID: h.actor.BusinessUnitID,
	}}

	require.NoError(
		t,
		h.svc.Revoke(t.Context(), &AccessRequest{Actor: h.actor, UserID: h.users.user.ID}),
	)

	assert.Empty(t, h.lifecycle.exits)
	assert.Empty(t, h.users.updated)
}

func TestRestore_RejoinsTheOrganizationWithoutReactivatingASharedAccount(t *testing.T) {
	t.Parallel()

	h := newAccessHarness(t, 1)
	other := h.users.memberships[1]
	h.users.memberships = []*tenant.OrganizationMembership{other}
	h.users.user.Status = domaintypes.StatusInactive

	require.NoError(
		t,
		h.svc.Restore(t.Context(), &AccessRequest{Actor: h.actor, UserID: h.users.user.ID}),
	)

	require.Len(t, h.users.replaced, 1)
	assert.ElementsMatch(t,
		[]pulid.ID{other.OrganizationID, h.actor.OrganizationID},
		h.users.replaced[0].OrganizationIDs,
	)
	assert.Empty(
		t,
		h.users.updated,
		"a directory never reactivates an account other organizations share",
	)
}

func TestUpdate_WithholdsProfileChangesForASharedAccount(t *testing.T) {
	t.Parallel()

	h := newAccessHarness(t, 1)

	result, err := h.svc.Update(t.Context(), &UpdateRequest{
		Actor:  h.actor,
		UserID: h.users.user.ID,
		Profile: Profile{
			Name:         "Ada King",
			EmailAddress: "someone-else@example.com",
		},
	})

	require.NoError(t, err)
	assert.True(t, result.Withheld)
	assert.Equal(t, "ada@example.com", result.User.EmailAddress)
	assert.Empty(t, h.users.updated)
	assert.Empty(t, h.auditor.Changes())
}
