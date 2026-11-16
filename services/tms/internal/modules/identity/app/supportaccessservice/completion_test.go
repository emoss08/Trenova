package supportaccessservice

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/emoss08/trenova/internal/modules/identity/ports/supportaccessports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func grantWith(
	t *testing.T,
	r *rig,
	admin pagination.TenantInfo,
	scope supportaccess.Scope,
	assistant bool,
) *supportaccess.Grant {
	t.Helper()
	grant, err := r.svc.Grant(t.Context(), &GrantRequest{
		TenantInfo:      admin,
		StaffEmail:      staffEmail,
		Reason:          "Customer ticket 4411: invoices not posting",
		DurationHours:   4,
		Scope:           scope,
		AllowsAssistant: assistant,
	})
	require.NoError(t, err)
	return grant
}

func supportTenant(grant *supportaccess.Grant) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  grant.OrganizationID,
		BuID:   grant.BusinessUnitID,
		UserID: grant.PrincipalUserID,
	}
}

func TestSupportAccess_ExtendingALiveGrantCountsFromNowAndTellsAdministrators(t *testing.T) {
	t.Parallel()

	r := newRig(t, true)
	r.deliverKeys(t)
	admin := r.organization(newOrgID(), pulid.MustNew("bu_"))
	grant := grantFor(t, r, admin, supportaccess.ScopeReadOnly, 1)
	r.clock.Advance(30 * time.Minute)

	other := admin
	other.UserID = pulid.MustNew("usr_")
	extended, err := r.svc.Extend(t.Context(), &ExtendRequest{
		TenantInfo:    other,
		GrantID:       grant.ID,
		DurationHours: 24,
	})
	require.NoError(t, err)
	assert.Equal(t, r.clock.Now()+24*3600, extended.ExpiresAt)
	assert.Equal(t, other.UserID, extended.ExtendedByID, "any administrator may extend, not only who granted")
	assert.Equal(t, []string{ActionGranted, ActionExtended}, r.actions())
	assert.Equal(t,
		[]supportaccessports.NoticeKind{supportaccessports.NoticeGranted, supportaccessports.NoticeExtended},
		r.notices.kinds())
	assert.Equal(t, other.UserID, r.notices.sent[1].ActorID, "who extended it is not told about their own act")
}

func TestSupportAccess_AnEndedGrantIsGrantedAgainNotExtended(t *testing.T) {
	t.Parallel()

	r := newRig(t, true)
	r.deliverKeys(t)
	admin := r.organization(newOrgID(), pulid.MustNew("bu_"))
	grant := grantFor(t, r, admin, supportaccess.ScopeReadOnly, 1)
	r.clock.Advance(2 * time.Hour)

	_, err := r.svc.Extend(t.Context(), &ExtendRequest{TenantInfo: admin, GrantID: grant.ID, DurationHours: 4})
	require.Error(t, err)
	assert.NotContains(t, r.actions(), ActionExtended)
}

func TestSupportAccess_ASupportPrincipalCannotExtendItsOwnAccess(t *testing.T) {
	t.Parallel()

	r := newRig(t, true)
	r.deliverKeys(t)
	admin := r.organization(newOrgID(), pulid.MustNew("bu_"))
	grant := grantFor(t, r, admin, supportaccess.ScopeFull, 1)

	_, err := r.svc.Extend(t.Context(), &ExtendRequest{
		TenantInfo:    supportTenant(grant),
		GrantID:       grant.ID,
		DurationHours: 72,
	})
	var refusal *errortypes.AuthorizationError
	require.ErrorAs(t, err, &refusal)
}

func TestSupportAccess_AdministratorsAreToldOfTheFirstEntryAndTheEndOnly(t *testing.T) {
	t.Parallel()

	r := newRig(t, true)
	r.deliverKeys(t)
	admin := r.organization(newOrgID(), pulid.MustNew("bu_"))
	grant := grantFor(t, r, admin, supportaccess.ScopeReadOnly, 4)

	for range 2 {
		_, err := r.svc.Enter(t.Context(), &EnterRequest{
			Assertion: r.assertion(t, assertionOptions{org: admin.OrgID}),
		})
		require.NoError(t, err)
	}
	_, err := r.svc.Revoke(t.Context(), &RevokeRequest{TenantInfo: admin, GrantID: grant.ID, Reason: "Fixed"})
	require.NoError(t, err)

	assert.Equal(t, []supportaccessports.NoticeKind{
		supportaccessports.NoticeGranted,
		supportaccessports.NoticeFirstEntry,
		supportaccessports.NoticeRevoked,
	}, r.notices.kinds(), "a second entry under the same grant tells nobody again")
}

func TestSupportAccess_AGrantThatEndsOnItsOwnTellsAdministrators(t *testing.T) {
	t.Parallel()

	r := newRig(t, true)
	r.deliverKeys(t)
	admin := r.organization(newOrgID(), pulid.MustNew("bu_"))
	grantFor(t, r, admin, supportaccess.ScopeReadOnly, 1)
	r.clock.Advance(2 * time.Hour)

	_, err := r.svc.RecordExpiries(t.Context(), r.clock.Now())
	require.NoError(t, err)
	assert.Equal(t,
		[]supportaccessports.NoticeKind{supportaccessports.NoticeGranted, supportaccessports.NoticeExpired},
		r.notices.kinds())
}

func TestSupportAccess_TheAssistantIsHeldOnlyWhenTheGrantAllowsIt(t *testing.T) {
	t.Parallel()

	r := newRig(t, true)
	r.deliverKeys(t)
	first := r.organization(newOrgID(), pulid.MustNew("bu_"))
	without := grantWith(t, r, first, supportaccess.ScopeFull, false)
	assert.NotContains(t, permissionsOf(t, r, without), permission.ResourceAssistant.String(),
		"a full grant does not bring the assistant with it")

	second := r.organization(newOrgID(), pulid.MustNew("bu_"))
	with := grantWith(t, r, second, supportaccess.ScopeReadOnly, true)
	perms := permissionsOf(t, r, with)
	assert.ElementsMatch(t,
		[]string{string(permission.OpRead), string(permission.OpCreate)},
		perms[permission.ResourceAssistant.String()],
		"a read-only grant that allows the assistant may ask it questions")
	assert.Equal(t, []string{string(permission.OpRead)}, perms[permission.ResourceShipment.String()],
		"and still changes nothing")
}

func TestSupportAccess_AModelCallIsRefusedUnlessTheLiveGrantAllowsTheAssistant(t *testing.T) {
	t.Parallel()

	r := newRig(t, true)
	r.deliverKeys(t)
	first := r.organization(newOrgID(), pulid.MustNew("bu_"))
	without := grantWith(t, r, first, supportaccess.ScopeFull, false)

	err := r.principals.AdmitModelCall(t.Context(), supportTenant(without))
	var refusal *errortypes.BusinessError
	require.ErrorAs(t, err, &refusal, "nothing is spent on the organization's AI providers without its consent")

	scoped := dbscope.WithTenant(t.Context(), dbscope.Tenant{
		OrganizationID: without.OrganizationID,
		BusinessUnitID: without.BusinessUnitID,
		UserID:         without.PrincipalUserID,
	})
	require.Error(t, r.principals.AdmitModelCall(scoped, pagination.TenantInfo{
		OrgID: without.OrganizationID, BuID: without.BusinessUnitID,
	}), "a call made later in the session's scope is the session's call")

	second := r.organization(newOrgID(), pulid.MustNew("bu_"))
	with := grantWith(t, r, second, supportaccess.ScopeReadOnly, true)
	require.NoError(t, r.principals.AdmitModelCall(t.Context(), supportTenant(with)))

	_, err = r.svc.Revoke(t.Context(), &RevokeRequest{TenantInfo: second, GrantID: with.ID, Reason: "Done"})
	require.NoError(t, err)
	require.Error(t, r.principals.AdmitModelCall(t.Context(), supportTenant(with)),
		"a model call caused after the grant ended is refused")

	require.NoError(t, r.principals.AdmitModelCall(t.Context(), first),
		"a person's call is never the support policy's to decide")
}

func TestSupportAccess_ModelCallsAreRefusedForASupportPrincipalOnASelfHostedInstall(t *testing.T) {
	t.Parallel()

	r := newRig(t, false)
	err := r.principals.AdmitModelCall(t.Context(), pagination.TenantInfo{
		OrgID:  newOrgID(),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew(supportaccess.PrincipalIDPrefix),
	})
	require.Error(t, err)
	assert.Zero(t, r.grants.calls)
}

func TestSupportAccess_SettingsListWhatSupportCanNeverDo(t *testing.T) {
	t.Parallel()

	r := newRig(t, true)
	settings, err := r.svc.Settings(t.Context())
	require.NoError(t, err)
	assert.Equal(t, supportaccess.Limits(), settings.Limits)
	assert.Contains(t, settings.Limits, supportaccess.LimitPaymentDestinations)
}
