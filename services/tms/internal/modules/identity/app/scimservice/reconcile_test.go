package scimservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/app/roleservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/scim"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (h *harness) reconcile(t *testing.T, link *scim.User) {
	t.Helper()
	require.NoError(t, h.svc.reconcileUser(t.Context(), h.dir, link, newAuthority(h.svc, h.dir)))
}

func TestReconcile_GrantsAMappedRoleUnderTheAuthorityOfWhoSavedTheMapping(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	person, saver := h.activeMember(), h.activeMember()
	link := h.link(person, true)
	roleID := pulid.MustNew("rol_")
	h.repo.sources = []*scim.RoleSource{
		{RoleID: roleID, MappingID: pulid.MustNew("sgr_"), SavedByID: saver.ID},
	}

	h.reconcile(t, link)

	require.Len(t, h.roles.assigned, 1)
	assigned := h.roles.assigned[0]
	assert.Equal(t, saver.ID, assigned.ActorID)
	assert.Equal(t, h.dir.OrganizationID, assigned.OrganizationID)
	assert.Equal(t, person.ID, assigned.Assignment.UserID)
	assert.Equal(t, roleID, assigned.Assignment.RoleID)
	require.Len(t, h.repo.created, 1)
	assert.Equal(t, assigned.Assignment.ID, h.repo.created[0].AssignmentID)
	assert.Equal(t, saver.ID, h.repo.created[0].GrantedByID)
	assert.Equal(t, []*scim.User{link}, h.repo.completed)
}

func TestReconcile_TakesAwayAGrantWhoseSaverCanNoLongerGrantTheRole(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	person, saver := h.activeMember(), h.activeMember()
	link := h.link(person, true)
	roleID := pulid.MustNew("rol_")
	h.repo.sources = []*scim.RoleSource{{RoleID: roleID, SavedByID: saver.ID}}
	h.roles.refuse[[2]pulid.ID{saver.ID, roleID}] = errortypes.NewValidationError(
		"permissions",
		errortypes.ErrForbidden,
		"cannot grant",
	)
	grant := &scim.RoleGrant{UserID: person.ID, RoleID: roleID, AssignmentID: pulid.MustNew("ura_")}
	h.repo.grants = []*scim.RoleGrant{grant}

	h.reconcile(t, link)

	assert.Empty(t, h.roles.assigned)
	require.Len(t, h.roles.unassigned, 1)
	assert.Equal(t, grant.AssignmentID, h.roles.unassigned[0].AssignmentID)
	assert.Equal(t, h.dir.OrganizationID, h.roles.unassigned[0].OrganizationID)
	assert.Contains(t, h.repo.statuses(), statusRefused)
}

func TestReconcile_GrantsNothingUnderASaverWhoLostTheirAccount(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	person, saver := h.activeMember(), h.activeMember()
	saver.Status = domaintypes.StatusInactive
	link := h.link(person, true)
	h.repo.sources = []*scim.RoleSource{{RoleID: pulid.MustNew("rol_"), SavedByID: saver.ID}}

	h.reconcile(t, link)

	assert.Empty(t, h.roles.assigned)
	assert.Empty(
		t,
		h.roles.checked,
		"a saver who cannot sign in is never asked whether they may grant",
	)
}

func TestReconcile_GrantsNothingUnderASaverOutsideTheOrganization(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	person := h.activeMember()
	link := h.link(person, true)
	h.repo.sources = []*scim.RoleSource{
		{RoleID: pulid.MustNew("rol_"), SavedByID: pulid.MustNew("usr_")},
	}

	h.reconcile(t, link)

	assert.Empty(t, h.roles.assigned)
	assert.Empty(t, h.roles.checked)
}

func TestReconcile_TakesEveryDirectoryGrantFromADeactivatedPerson(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	person := h.activeMember()
	link := h.link(person, false)
	h.repo.sources = []*scim.RoleSource{
		{RoleID: pulid.MustNew("rol_"), SavedByID: h.activeMember().ID},
	}
	h.repo.grants = []*scim.RoleGrant{
		{UserID: person.ID, RoleID: pulid.MustNew("rol_"), AssignmentID: pulid.MustNew("ura_")},
		{UserID: person.ID, RoleID: pulid.MustNew("rol_"), AssignmentID: pulid.MustNew("ura_")},
	}

	h.reconcile(t, link)

	assert.Empty(t, h.roles.assigned)
	assert.Len(t, h.roles.unassigned, 2)
	assert.Equal(t, []*scim.User{link}, h.repo.completed)
}

func TestReconcile_LeavesARoleThePersonWasGivenByHand(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	person, saver := h.activeMember(), h.activeMember()
	link := h.link(person, true)
	h.repo.sources = []*scim.RoleSource{{RoleID: pulid.MustNew("rol_"), SavedByID: saver.ID}}
	h.roles.assignErr = roleservice.ErrRoleAlreadyAssigned

	h.reconcile(t, link)

	assert.Empty(
		t,
		h.repo.created,
		"an assignment a person made is never recorded as the directory's",
	)
	assert.Equal(t, []*scim.User{link}, h.repo.completed)
}
