package supportaccessservice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/supportaccessports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func grantFor(
	t *testing.T,
	r *rig,
	admin pagination.TenantInfo,
	scope supportaccess.Scope,
	hours int,
) *supportaccess.Grant {
	t.Helper()
	grant, err := r.svc.Grant(t.Context(), &GrantRequest{
		TenantInfo:    admin,
		StaffEmail:    "Jane.Doe@Trenova.app",
		Reason:        "Customer ticket 4411: invoices not posting",
		DurationHours: hours,
		Scope:         scope,
	})
	require.NoError(t, err)
	return grant
}

func requireRefusal(t *testing.T, err error, reason supportaccess.RefusalReason) {
	t.Helper()
	var refusal *supportaccess.RefusalError
	require.ErrorAs(t, err, &refusal)
	assert.Equal(t, reason, refusal.Reason)
}

func permissionsOf(
	t *testing.T,
	r *rig,
	grant *supportaccess.Grant,
) map[string][]string {
	t.Helper()
	ctx := dbscope.WithTenant(t.Context(), dbscope.Tenant{
		OrganizationID: grant.OrganizationID,
		BusinessUnitID: grant.BusinessUnitID,
		UserID:         grant.PrincipalUserID,
	})
	perms, err := r.principals.Permissions(ctx, grant.PrincipalUserID, grant.OrganizationID)
	require.NoError(t, err)
	out := make(map[string][]string, len(perms.Resources))
	for resource, rp := range perms.Resources {
		out[resource] = rp.Operations
	}
	return out
}

func TestSupportAccess_LiveGrantsListsOnlyLiveGrantsAcrossOrganizationsSoonestFirst(t *testing.T) {
	t.Parallel()

	r := newRig(t, true)
	r.deliverKeys(t)
	acme := r.organization(newOrgID(), pulid.MustNew("bu_"))
	globex := r.organization(newOrgID(), pulid.MustNew("bu_"))
	initech := r.organization(newOrgID(), pulid.MustNew("bu_"))
	umbrella := r.organization(newOrgID(), pulid.MustNew("bu_"))
	long := grantFor(t, r, acme, supportaccess.ScopeFull, 72)
	short := grantFor(t, r, globex, supportaccess.ScopeReadOnly, 4)
	revoked := grantFor(t, r, initech, supportaccess.ScopeReadOnly, 24)
	_, err := r.svc.Revoke(t.Context(), &RevokeRequest{
		TenantInfo: initech,
		GrantID:    revoked.ID,
		Reason:     "Ticket closed",
	})
	require.NoError(t, err)
	grantFor(t, r, umbrella, supportaccess.ScopeReadOnly, 1)
	r.clock.Advance(2 * time.Hour)

	live, err := r.svc.LiveGrants(t.Context())
	require.NoError(t, err)
	require.Len(t, live, 2)
	assert.Equal(t, short.ID, live[0].Grant.ID, "the grant ending soonest comes first")
	assert.Equal(t, long.ID, live[1].Grant.ID)
	assert.Equal(t, globex.OrgID, live[0].Grant.OrganizationID)
	assert.Equal(t, acme.OrgID, live[1].Grant.OrganizationID)
}

func TestSupportAccess_LiveGrantsReadsNothingWhenSupportAccessIsOff(t *testing.T) {
	t.Parallel()

	r := newRig(t, false)
	_, err := r.svc.LiveGrants(t.Context())
	require.Error(t, err)
	assert.Zero(t, r.grants.calls)
}

func TestSupportAccess_StaffEnterUnderALiveGrantAsItsSupportPrincipal(t *testing.T) {
	t.Parallel()

	r := newRig(t, true)
	r.deliverKeys(t)
	admin := r.organization(newOrgID(), pulid.MustNew("bu_"))
	grant := grantFor(t, r, admin, supportaccess.ScopeFull, 4)

	assert.Equal(t, staffEmail, grant.StaffEmail, "the address is stored lower-cased")
	assert.True(t, supportaccess.IsPrincipalID(grant.PrincipalUserID))
	assert.Equal(t, r.clock.Now()+4*3600, grant.ExpiresAt)
	assert.Equal(t, admin.UserID, grant.GrantedByID)

	result, err := r.svc.Enter(t.Context(), &EnterRequest{
		Assertion: r.assertion(t, assertionOptions{org: admin.OrgID}),
	})
	require.NoError(t, err)
	require.NotEmpty(t, result.Token)

	sessions := r.sessions.live()
	require.Len(t, sessions, 1)
	sess := sessions[0]
	assert.Equal(t, grant.PrincipalUserID, sess.UserID, "staff act as the grant's support principal")
	assert.Equal(t, admin.OrgID, sess.OrganizationID)
	assert.False(t, sess.IssuedBySignIn)
	require.NotNil(t, sess.Support)
	assert.Equal(t, grant.ID, sess.Support.GrantID)
	assert.Equal(t, staffEmail, sess.Support.StaffEmail)
	assert.Equal(t, "Jane Doe", sess.Support.StaffName)
	assert.Equal(t, string(supportaccess.ScopeFull), sess.Support.Scope)
	assert.LessOrEqual(t, sess.ExpiresAt, grant.ExpiresAt, "a session never outlives its grant")

	assert.Equal(t, []string{ActionGranted, ActionSessionStarted}, r.actions())
	started := r.audit.Changes()[1]
	assert.Equal(t, permission.ResourceSupportAccess, started.Resource)
	assert.Equal(t, grant.PrincipalUserID, started.Actor.UserID)

	assert.Equal(t,
		[]supportaccessports.NoticeKind{supportaccessports.NoticeGranted, supportaccessports.NoticeFirstEntry},
		r.notices.kinds(), "administrators are told access was granted and first used")
	assert.Equal(t, grant.ID, r.notices.sent[1].Grant.ID)
	assert.Equal(t, "Jane Doe", r.notices.sent[1].StaffName)
	require.Len(t, r.events.records, 1)
	assert.Equal(t, iamports.AuthEventProviderSupport, r.events.records[0].Provider)

	require.NoError(t, r.principals.RequireLive(t.Context(), &supportaccessports.RequireLiveRequest{
		OrganizationID:  admin.OrgID,
		BusinessUnitID:  admin.BuID,
		GrantID:         grant.ID,
		PrincipalUserID: grant.PrincipalUserID,
		Now:             r.clock.Now(),
	}))
	perms := permissionsOf(t, r, grant)
	assert.Contains(t, perms[permission.ResourceShipment.String()], string(permission.OpUpdate))
	assert.NotContains(t, perms, permission.ResourceSupportAccess.String())
	assert.NotContains(t, perms[permission.ResourceUser.String()], string(permission.OpCreate))
}

func TestSupportAccess_AReadOnlyGrantReadsAndDoesNothingElse(t *testing.T) {
	t.Parallel()

	r := newRig(t, true)
	r.deliverKeys(t)
	admin := r.organization(newOrgID(), pulid.MustNew("bu_"))
	grant := grantFor(t, r, admin, supportaccess.ScopeReadOnly, 1)

	for resource, ops := range permissionsOf(t, r, grant) {
		assert.Equal(t, []string{string(permission.OpRead)}, ops, resource)
	}
}

func TestSupportAccess_RefusesAGrantToAnAddressOutsideTheStaffDomain(t *testing.T) {
	t.Parallel()

	r := newRig(t, true)
	r.deliverKeys(t)
	admin := r.organization(newOrgID(), pulid.MustNew("bu_"))

	for _, email := range []string{"jane@example.com", "jane@evil.trenova.app", "jаne@trenova.app"} {
		_, err := r.svc.Grant(t.Context(), &GrantRequest{
			TenantInfo:    admin,
			StaffEmail:    email,
			Reason:        "Ticket",
			DurationHours: 1,
			Scope:         supportaccess.ScopeReadOnly,
		})
		multiErr, ok := errortypes.RefusalOf(err)
		require.True(t, ok, email)
		assert.Contains(t, multiErr.Error(), "staffEmail", email)
	}
	assert.Empty(t, r.audit.Changes())
}

func TestSupportAccess_RefusesAGrantWithoutAReasonOrAFixedDuration(t *testing.T) {
	t.Parallel()

	r := newRig(t, true)
	r.deliverKeys(t)
	admin := r.organization(newOrgID(), pulid.MustNew("bu_"))

	for _, req := range []GrantRequest{
		{StaffEmail: staffEmail, Reason: "  ", DurationHours: 1, Scope: supportaccess.ScopeFull},
		{StaffEmail: staffEmail, Reason: "Ticket", DurationHours: 73, Scope: supportaccess.ScopeFull},
		{StaffEmail: staffEmail, Reason: "Ticket", DurationHours: 0, Scope: supportaccess.ScopeFull},
		{StaffEmail: staffEmail, Reason: "Ticket", DurationHours: 4, Scope: "Admin"},
	} {
		req.TenantInfo = admin
		_, err := r.svc.Grant(t.Context(), &req)
		_, ok := errortypes.RefusalOf(err)
		assert.True(t, ok, "%+v", req)
	}
}

func TestSupportAccess_ASupportPrincipalCannotGrantOrRevoke(t *testing.T) {
	t.Parallel()

	r := newRig(t, true)
	r.deliverKeys(t)
	admin := r.organization(newOrgID(), pulid.MustNew("bu_"))
	grant := grantFor(t, r, admin, supportaccess.ScopeFull, 1)

	asSupport := admin
	asSupport.UserID = grant.PrincipalUserID
	_, err := r.svc.Grant(t.Context(), &GrantRequest{
		TenantInfo: asSupport, StaffEmail: "other@trenova.app", Reason: "x", DurationHours: 1,
		Scope: supportaccess.ScopeFull,
	})
	assert.True(t, errortypes.IsAuthorizationError(err))
	_, err = r.svc.Revoke(t.Context(), &RevokeRequest{TenantInfo: asSupport, GrantID: grant.ID, Reason: "x"})
	assert.True(t, errortypes.IsAuthorizationError(err))
}

func TestSupportAccess_RefusesASecondLiveGrantForTheSameStaffMember(t *testing.T) {
	t.Parallel()

	r := newRig(t, true)
	r.deliverKeys(t)
	admin := r.organization(newOrgID(), pulid.MustNew("bu_"))
	grantFor(t, r, admin, supportaccess.ScopeReadOnly, 1)

	_, err := r.svc.Grant(t.Context(), &GrantRequest{
		TenantInfo: admin, StaffEmail: staffEmail, Reason: "again", DurationHours: 4,
		Scope: supportaccess.ScopeFull,
	})
	require.Error(t, err)
}

func TestSupportAccess_RefusesAReplayedAssertion(t *testing.T) {
	t.Parallel()

	r := newRig(t, true)
	r.deliverKeys(t)
	admin := r.organization(newOrgID(), pulid.MustNew("bu_"))
	grantFor(t, r, admin, supportaccess.ScopeFull, 1)
	token := r.assertion(t, assertionOptions{org: admin.OrgID})

	_, err := r.svc.Enter(t.Context(), &EnterRequest{Assertion: token})
	require.NoError(t, err)
	_, err = r.svc.Enter(t.Context(), &EnterRequest{Assertion: token})
	requireRefusal(t, err, supportaccess.RefusalReplayed)

	assert.Len(t, r.sessions.live(), 1)
	assert.Contains(t, r.actions(), ActionEntryRefused, "the organization's audit log shows the refused attempt")
}

func TestSupportAccess_RefusesAnAssertionForAnotherOrganizationOrInstance(t *testing.T) {
	t.Parallel()

	r := newRig(t, true)
	r.deliverKeys(t)
	mine := r.organization(newOrgID(), pulid.MustNew("bu_"))
	other := r.organization(newOrgID(), pulid.MustNew("bu_"))
	grantFor(t, r, mine, supportaccess.ScopeFull, 1)

	_, err := r.svc.Enter(t.Context(), &EnterRequest{
		Assertion: r.assertion(t, assertionOptions{org: other.OrgID}),
	})
	requireRefusal(t, err, supportaccess.RefusalNoGrant)

	_, err = r.svc.Enter(t.Context(), &EnterRequest{
		Assertion: r.assertion(t, assertionOptions{org: mine.OrgID, audience: "inst_another"}),
	})
	requireRefusal(t, err, supportaccess.RefusalAudience)

	_, err = r.svc.Enter(t.Context(), &EnterRequest{
		Assertion: r.assertion(t, assertionOptions{org: mine.OrgID, email: "john@trenova.app"}),
	})
	requireRefusal(t, err, supportaccess.RefusalNoGrant)

	assert.Empty(t, r.sessions.live())
	assert.Equal(t, []supportaccessports.NoticeKind{supportaccessports.NoticeGranted}, r.notices.kinds(),
		"a refused entry tells nobody it was used")
}

func TestSupportAccess_UsesTheGrantOfTheOrganizationTheAssertionNames(t *testing.T) {
	t.Parallel()

	r := newRig(t, true)
	r.deliverKeys(t)
	first := r.organization(newOrgID(), pulid.MustNew("bu_"))
	second := r.organization(newOrgID(), pulid.MustNew("bu_"))
	grantFor(t, r, first, supportaccess.ScopeFull, 1)
	secondGrant := grantFor(t, r, second, supportaccess.ScopeReadOnly, 1)

	_, err := r.svc.Enter(t.Context(), &EnterRequest{
		Assertion: r.assertion(t, assertionOptions{org: second.OrgID}),
	})
	require.NoError(t, err)

	sessions := r.sessions.live()
	require.Len(t, sessions, 1)
	assert.Equal(t, second.OrgID, sessions[0].OrganizationID)
	assert.Equal(t, secondGrant.ID, sessions[0].Support.GrantID)
	assert.Equal(t, string(supportaccess.ScopeReadOnly), sessions[0].Support.Scope)
}

func TestSupportAccess_ARevocationBetweenMintAndUseRefusesEntry(t *testing.T) {
	t.Parallel()

	r := newRig(t, true)
	r.deliverKeys(t)
	admin := r.organization(newOrgID(), pulid.MustNew("bu_"))
	grant := grantFor(t, r, admin, supportaccess.ScopeFull, 4)
	token := r.assertion(t, assertionOptions{org: admin.OrgID})

	_, err := r.svc.Revoke(t.Context(), &RevokeRequest{
		TenantInfo: admin, GrantID: grant.ID, Reason: "Resolved by phone",
	})
	require.NoError(t, err)

	_, err = r.svc.Enter(t.Context(), &EnterRequest{Assertion: token})
	requireRefusal(t, err, supportaccess.RefusalNoGrant)
	assert.Empty(t, r.sessions.live())
}

func TestSupportAccess_RevokingEndsLiveSessionsAtOnce(t *testing.T) {
	t.Parallel()

	r := newRig(t, true)
	r.deliverKeys(t)
	admin := r.organization(newOrgID(), pulid.MustNew("bu_"))
	grant := grantFor(t, r, admin, supportaccess.ScopeFull, 4)
	_, err := r.svc.Enter(t.Context(), &EnterRequest{
		Assertion: r.assertion(t, assertionOptions{org: admin.OrgID}),
	})
	require.NoError(t, err)
	require.Len(t, r.sessions.live(), 1)

	revoked, err := r.svc.Revoke(t.Context(), &RevokeRequest{
		TenantInfo: admin, GrantID: grant.ID, Reason: "Done",
	})
	require.NoError(t, err)
	assert.Equal(t, supportaccess.StatusRevoked, revoked.StatusAt(r.clock.Now()))

	assert.Empty(t, r.sessions.live(), "every session under the grant is deleted")
	assert.True(t, containsUser(r.sessions.endedForUser, grant.PrincipalUserID))

	err = r.principals.RequireLive(t.Context(), &supportaccessports.RequireLiveRequest{
		OrganizationID:  admin.OrgID,
		BusinessUnitID:  admin.BuID,
		GrantID:         grant.ID,
		PrincipalUserID: grant.PrincipalUserID,
		Now:             r.clock.Now(),
	})
	require.ErrorIs(t, err, supportaccess.ErrGrantNotFound, "the next request of any copy of the session ends it")
	assert.Empty(t, permissionsOf(t, r, grant))
	assert.Equal(t, ActionRevoked, r.audit.Changes()[len(r.audit.Changes())-1].Action)

	_, err = r.svc.Revoke(t.Context(), &RevokeRequest{TenantInfo: admin, GrantID: grant.ID, Reason: "Again"})
	require.Error(t, err, "a grant ends once")
}

func TestSupportAccess_AGrantThatExpiresMidSessionEndsIt(t *testing.T) {
	t.Parallel()

	r := newRig(t, true)
	r.deliverKeys(t)
	admin := r.organization(newOrgID(), pulid.MustNew("bu_"))
	grant := grantFor(t, r, admin, supportaccess.ScopeFull, 1)
	_, err := r.svc.Enter(t.Context(), &EnterRequest{
		Assertion: r.assertion(t, assertionOptions{org: admin.OrgID}),
	})
	require.NoError(t, err)

	r.clock.Advance(time.Hour + time.Second)

	err = r.principals.RequireLive(t.Context(), &supportaccessports.RequireLiveRequest{
		OrganizationID:  admin.OrgID,
		BusinessUnitID:  admin.BuID,
		GrantID:         grant.ID,
		PrincipalUserID: grant.PrincipalUserID,
		Now:             r.clock.Now(),
	})
	require.ErrorIs(t, err, supportaccess.ErrGrantNotFound)

	result, err := r.svc.RecordExpiries(t.Context(), r.clock.Now())
	require.NoError(t, err)
	assert.Equal(t, 1, result.Recorded)
	assert.Empty(t, r.sessions.live())
	assert.Equal(t, ActionExpired, r.audit.Changes()[len(r.audit.Changes())-1].Action)

	again, err := r.svc.RecordExpiries(t.Context(), r.clock.Now())
	require.NoError(t, err)
	assert.Zero(t, again.Recorded, "an expiry is recorded once")
}

func TestSupportAccess_RefusesEntryBeforeTheControlPlaneDeliversKeys(t *testing.T) {
	t.Parallel()

	r := newRig(t, true)
	admin := r.organization(newOrgID(), pulid.MustNew("bu_"))

	_, err := r.svc.Enter(t.Context(), &EnterRequest{
		Assertion: r.assertion(t, assertionOptions{org: admin.OrgID}),
	})
	requireRefusal(t, err, supportaccess.RefusalNotConfigured)

	settings, err := r.svc.Settings(t.Context())
	require.NoError(t, err)
	assert.False(t, settings.Available)
}

func TestSupportAccess_DoesNothingOnASelfHostedInstall(t *testing.T) {
	t.Parallel()

	r := newRig(t, false)
	admin := r.organization(newOrgID(), pulid.MustNew("bu_"))
	ctx := context.Background()

	_, err := r.svc.Grant(ctx, &GrantRequest{
		TenantInfo: admin, StaffEmail: staffEmail, Reason: "x", DurationHours: 1,
		Scope: supportaccess.ScopeFull,
	})
	require.ErrorIs(t, err, errNotAvailable)
	_, err = r.svc.Revoke(ctx, &RevokeRequest{TenantInfo: admin, GrantID: pulid.MustNew("sag_"), Reason: "x"})
	require.ErrorIs(t, err, errNotAvailable)
	_, err = r.svc.List(ctx, &supportaccessports.ListGrantsRequest{})
	require.ErrorIs(t, err, errNotAvailable)
	_, err = r.svc.Get(ctx, admin, pulid.MustNew("sag_"))
	require.ErrorIs(t, err, errNotAvailable)
	require.ErrorIs(t, r.svc.SetKeys(ctx, &supportaccess.KeySet{}), errNotAvailable)
	_, err = r.svc.Enter(ctx, &EnterRequest{Assertion: "a.b.c"})
	requireRefusal(t, err, supportaccess.RefusalNotConfigured)
	recorded, err := r.svc.RecordExpiries(ctx, r.clock.Now())
	require.NoError(t, err)
	assert.Zero(t, recorded.Recorded)

	principal := pulid.MustNew(supportaccess.PrincipalIDPrefix)
	perms, err := r.principals.Permissions(ctx, principal, admin.OrgID)
	require.NoError(t, err)
	assert.Empty(t, perms.Resources)
	require.ErrorIs(t, r.principals.RequireLive(ctx, &supportaccessports.RequireLiveRequest{
		OrganizationID: admin.OrgID, PrincipalUserID: principal,
	}), supportaccess.ErrGrantNotFound)

	settings, err := r.svc.Settings(ctx)
	require.NoError(t, err)
	assert.False(t, settings.Available)

	assert.Zero(t, r.grants.calls, "no support table is read or written")
	assert.Zero(t, r.keys.gets)
}

func TestSupportAccess_KeepsTheKeySetUntilItChanges(t *testing.T) {
	t.Parallel()

	r := newRig(t, true)
	r.deliverKeys(t)
	first := r.keys.held.ReceivedAt
	r.clock.Advance(time.Minute)
	r.deliverKeys(t)
	assert.Equal(t, first, r.keys.held.ReceivedAt, "an unchanged key set writes nothing")

	err := r.svc.SetKeys(t.Context(), &supportaccess.KeySet{InstanceID: testInstance, StaffDomain: testDomain})
	_, invalid := errortypes.RefusalOf(err)
	assert.True(t, invalid, "a key set with no key is refused rather than stored")
	assert.NotEmpty(t, r.keys.held.Keys)
}

func TestSupportAccess_ARefusedAssertionThatDidNotVerifyStaysOutOfTheCustomersLog(t *testing.T) {
	t.Parallel()

	r := newRig(t, true)
	r.deliverKeys(t)
	r.organization(newOrgID(), pulid.MustNew("bu_"))

	_, err := r.svc.Enter(t.Context(), &EnterRequest{Assertion: "eyJhbGciOiJub25lIn0.e30."})
	require.Error(t, err)
	assert.True(t, errors.As(err, new(*supportaccess.RefusalError)))
	assert.Empty(t, r.audit.Changes())
}
