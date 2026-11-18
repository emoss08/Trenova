package supportaccessservice

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportlog"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func pullsRecorded(r *rig) []string {
	var out []string
	for _, change := range r.audit.Changes() {
		if change.Action == ActionLogsPulled {
			out = append(out, change.OrganizationID.String())
		}
	}
	return out
}

func TestLogPull_UnderALiveGrantIsRecordedWhereTheGrantingOrganizationSeesIt(t *testing.T) {
	t.Parallel()

	r := newRig(t, true)
	r.deliverKeys(t)
	admin := r.organization(newOrgID(), pulid.MustNew("bu_"))
	grant := grantFor(t, r, admin, supportaccess.ScopeReadOnly, 4)

	require.NoError(t, r.svc.RecordLogPull(t.Context(), &LogPullRequest{
		StaffEmail: "Jane.Doe@trenova.app",
		Kind:       supportlog.PullBundle,
	}))

	var pulled bool
	for _, change := range r.audit.Changes() {
		if change.Action != ActionLogsPulled {
			continue
		}
		pulled = true
		assert.Equal(t, admin.OrgID, change.OrganizationID)
		assert.Equal(t, admin.BuID, change.BusinessUnitID)
		assert.Equal(t, grant.PrincipalUserID, change.Actor.UserID, "the pull is credited to the grant, never a person")
		assert.Contains(t, change.Comment, "pulled a log bundle")
		assert.Contains(t, change.Comment, staffEmail)
	}
	assert.True(t, pulled)
}

func TestLogPull_LinesAreNamedAsLines(t *testing.T) {
	t.Parallel()

	r := newRig(t, true)
	r.deliverKeys(t)
	admin := r.organization(newOrgID(), pulid.MustNew("bu_"))
	grantFor(t, r, admin, supportaccess.ScopeReadOnly, 1)

	require.NoError(t, r.svc.RecordLogPull(t.Context(), &LogPullRequest{
		StaffEmail: staffEmail,
		Kind:       supportlog.PullLines,
	}))
	for _, change := range r.audit.Changes() {
		if change.Action == ActionLogsPulled {
			assert.Contains(t, change.Comment, "pulled recent log lines")
		}
	}
}

func TestLogPull_IsRecordedInEveryOrganizationThatGrantedTheAddress(t *testing.T) {
	t.Parallel()

	r := newRig(t, true)
	r.deliverKeys(t)
	first := r.organization(newOrgID(), pulid.MustNew("bu_"))
	second := r.organization(newOrgID(), pulid.MustNew("bu_"))
	grantFor(t, r, first, supportaccess.ScopeReadOnly, 4)
	grantFor(t, r, second, supportaccess.ScopeFull, 24)

	require.NoError(t, r.svc.RecordLogPull(t.Context(), &LogPullRequest{
		StaffEmail: staffEmail,
		Kind:       supportlog.PullLines,
	}))
	assert.ElementsMatch(t, []string{first.OrgID.String(), second.OrgID.String()}, pullsRecorded(r))
}

func TestLogPull_IsRefusedTheSameWayWhateverTheReason(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		setup  func(t *testing.T, r *rig) string
		reason supportaccess.RefusalReason
	}{
		{
			name: "no grant at all",
			setup: func(t *testing.T, r *rig) string {
				t.Helper()
				r.deliverKeys(t)
				return staffEmail
			},
			reason: supportaccess.RefusalNoGrant,
		},
		{
			name: "an expired grant",
			setup: func(t *testing.T, r *rig) string {
				t.Helper()
				r.deliverKeys(t)
				admin := r.organization(newOrgID(), pulid.MustNew("bu_"))
				grantFor(t, r, admin, supportaccess.ScopeReadOnly, 1)
				r.clock.Advance(time.Hour)
				return staffEmail
			},
			reason: supportaccess.RefusalNoGrant,
		},
		{
			name: "a revoked grant",
			setup: func(t *testing.T, r *rig) string {
				t.Helper()
				r.deliverKeys(t)
				admin := r.organization(newOrgID(), pulid.MustNew("bu_"))
				grant := grantFor(t, r, admin, supportaccess.ScopeReadOnly, 24)
				_, err := r.svc.Revoke(t.Context(), &RevokeRequest{
					TenantInfo: admin,
					GrantID:    grant.ID,
					Reason:     "Ticket closed",
				})
				require.NoError(t, err)
				return staffEmail
			},
			reason: supportaccess.RefusalNoGrant,
		},
		{
			name: "another staff address",
			setup: func(t *testing.T, r *rig) string {
				t.Helper()
				r.deliverKeys(t)
				admin := r.organization(newOrgID(), pulid.MustNew("bu_"))
				grantFor(t, r, admin, supportaccess.ScopeReadOnly, 24)
				return "john.roe@trenova.app"
			},
			reason: supportaccess.RefusalNoGrant,
		},
		{
			name: "an address outside the staff domain",
			setup: func(t *testing.T, r *rig) string {
				t.Helper()
				r.deliverKeys(t)
				admin := r.organization(newOrgID(), pulid.MustNew("bu_"))
				grantFor(t, r, admin, supportaccess.ScopeReadOnly, 24)
				return "jane.doe@trenova.app.example.com"
			},
			reason: supportaccess.RefusalStaffEmail,
		},
		{
			name: "no keys delivered",
			setup: func(t *testing.T, _ *rig) string {
				t.Helper()
				return staffEmail
			},
			reason: supportaccess.RefusalNotConfigured,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := newRig(t, true)
			email := tc.setup(t, r)
			err := r.svc.RecordLogPull(t.Context(), &LogPullRequest{StaffEmail: email, Kind: supportlog.PullBundle})
			requireRefusal(t, err, tc.reason)
			assert.Empty(t, pullsRecorded(r), "a refused pull writes no pull entry")
		})
	}
}

func TestLogPull_IsRefusedOnASelfHostedInstance(t *testing.T) {
	t.Parallel()

	r := newRig(t, false)
	err := r.svc.RecordLogPull(t.Context(), &LogPullRequest{StaffEmail: staffEmail, Kind: supportlog.PullLines})
	requireRefusal(t, err, supportaccess.RefusalNotConfigured)
}
