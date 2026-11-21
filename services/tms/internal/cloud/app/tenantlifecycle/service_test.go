package tenantlifecycle_test

import (
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/cloud/app/tenantlifecycle"
	"github.com/emoss08/trenova/internal/cloud/domain/cloudtenant"
	"github.com/emoss08/trenova/internal/cloud/jobs/tenantjobs/tenantjobscontract"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/emoss08/trenova/internal/modules/identity/ports/supportaccessports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedusage"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedusageports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	instanceA = "5d1a3c1e-8c1f-4a39-9a43-5b0d7c4e2f10"
	instanceB = "6e2b4d2f-9d2a-4b4a-8b54-6c1e8d5f3a21"
	tokenA    = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	tokenB    = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

func entry(id string, revision int64) *cloudtenant.Desired {
	return &cloudtenant.Desired{
		InstanceID:       id,
		Revision:         revision,
		Lifecycle:        cloudtenant.LifecycleActive,
		OrganizationName: "Acme Freight",
		Plan: cloudtenant.Plan{
			Key: "shared", Name: "Shared", BackupRetentionDays: 14, Support: "email",
		},
		Addons:     []string{},
		Features:   []string{string(platformcatalog.FeatureAccounting)},
		AIProvider: cloudtenant.AIProviderCustomer,
		Allowances: []cloudtenant.Allowance{},
		FirstAdministrator: cloudtenant.FirstAdministrator{
			Name:             "Dana Ortiz",
			Email:            "dana-" + id[:4] + "@acme.test",
			OrganizationName: "Acme Freight",
			Invite: &cloudtenant.Invite{
				TokenSHA256: tokenA,
				ExpiresAt:   time.Unix(now+86400, 0).UTC(),
			},
		},
		Account: cloudtenant.Account{
			PlanURL:         "https://console.test/instances/x/plan",
			FinalBackupDays: 30,
		},
		Commands: []cloudtenant.Command{},
	}
}

func TestApplyMakesATenantOnceAndAnswersItsState(t *testing.T) {
	t.Parallel()
	r := newRig(t, true)

	status, err := r.svc.Apply(t.Context(), instanceA, entry(instanceA, 1), nil)
	require.NoError(t, err)
	assert.Equal(t, cloudtenant.StateActive, status.State)
	require.NotNil(t, status.AppliedRevision)
	assert.Equal(t, int64(1), *status.AppliedRevision)
	require.NotNil(t, status.Administrator)
	assert.Equal(t, tokenA, status.Administrator.InviteTokenSHA256)
	assert.Nil(t, status.Refused)

	held, err := r.tenants.Get(t.Context(), instanceA)
	require.NoError(t, err)
	assert.True(t, held.Made())
	scope := held.OrganizationID.String()
	require.Contains(t, r.features.lists, scope)
	assert.Equal(
		t,
		[]platformcatalog.FeatureKey{platformcatalog.FeatureAccounting},
		r.features.lists[scope].Features,
	)
	assert.Equal(t, managedfeatures.LifecycleActive, r.features.lifecycles[scope])
	assert.Equal(t, []string{tokenA}, r.invites.installed)
	assert.Equal(
		t,
		[]cloudtenant.EventKind{cloudtenant.EventCreated, cloudtenant.EventApplied},
		r.tenants.kinds(),
	)
	assert.Len(t, r.audit.Changes(), 4, "the administrator, the role, its grant and the system actor are audited")
	for _, change := range r.audit.Changes() {
		assert.Equal(t, instanceA, change.Metadata["controlPlaneInstanceId"])
	}

	_, err = r.svc.Apply(t.Context(), instanceA, entry(instanceA, 1), nil)
	require.NoError(t, err)
	assert.Equal(t, 1, r.bootstrap.calls, "the same revision applies nothing again")
	assert.Len(t, r.tenants.kinds(), 2)
}

func TestApplyRefusesOutsideASharedServer(t *testing.T) {
	t.Parallel()
	r := newRig(t, false)

	_, err := r.svc.Apply(t.Context(), instanceA, entry(instanceA, 1), nil)
	require.ErrorIs(t, err, tenantlifecycle.ErrNotShared)
}

func TestApplyRefusesAnEntryAddressedToAnotherInstance(t *testing.T) {
	t.Parallel()
	r := newRig(t, true)

	_, err := r.svc.Apply(t.Context(), instanceB, entry(instanceA, 1), nil)
	require.ErrorIs(t, err, cloudtenant.ErrInstanceIDMismatch)
}

func TestATakenAddressRefusesTheEntryWithoutNamingTheHolder(t *testing.T) {
	t.Parallel()
	r := newRig(t, true)
	e := entry(instanceA, 1)
	r.admins.taken[e.FirstAdministrator.Email] = true

	status, err := r.svc.Apply(t.Context(), instanceA, e, nil)
	require.NoError(t, err)
	assert.Equal(t, cloudtenant.StateApplying, status.State)
	require.NotNil(t, status.Refused)
	assert.Equal(t, int64(1), status.Refused.Revision)
	assert.Equal(t, cloudtenant.RefusalInvalid, status.Refused.Code)
	assert.NotContains(t, status.Refused.Reason, e.FirstAdministrator.Email)
	assert.Equal(t, 0, r.bootstrap.calls)

	delete(r.admins.taken, e.FirstAdministrator.Email)
	status, err = r.svc.Apply(t.Context(), instanceA, entry(instanceA, 2), nil)
	require.NoError(t, err)
	assert.Equal(t, cloudtenant.StateActive, status.State)
	assert.Nil(t, status.Refused, "a later revision that can be applied clears the refusal")
}

func TestTheIdentityOfAMadeTenantCannotChange(t *testing.T) {
	t.Parallel()
	r := newRig(t, true)
	_, err := r.svc.Apply(t.Context(), instanceA, entry(instanceA, 1), nil)
	require.NoError(t, err)

	renamed := entry(instanceA, 2)
	renamed.OrganizationName = "Acme Logistics"
	renamed.FirstAdministrator.OrganizationName = "Acme Logistics"
	status, err := r.svc.Apply(t.Context(), instanceA, renamed, nil)
	require.NoError(t, err)
	require.NotNil(t, status.Refused)
	assert.Equal(t, int64(2), status.Refused.Revision)
	assert.Equal(t, int64(1), *status.AppliedRevision)
}

func TestSuspendAndResume(t *testing.T) {
	t.Parallel()
	r := newRig(t, true)
	_, err := r.svc.Apply(t.Context(), instanceA, entry(instanceA, 1), nil)
	require.NoError(t, err)
	held, _ := r.tenants.Get(t.Context(), instanceA)
	scope := held.OrganizationID.String()
	before := len(r.audit.Changes())

	suspended := entry(instanceA, 2)
	suspended.Lifecycle = cloudtenant.LifecycleSuspended
	status, err := r.svc.Apply(t.Context(), instanceA, suspended, nil)
	require.NoError(t, err)
	assert.Equal(t, cloudtenant.StateSuspended, status.State)
	assert.Equal(t, managedfeatures.LifecycleSuspended, r.features.lifecycles[scope])
	assert.Contains(t, r.workflows.started, tenantjobscontract.SessionsWorkflowID(instanceA, 2))
	changes := r.audit.Changes()
	require.Len(t, changes, before+1)
	assert.Equal(t, "organization.suspended", changes[len(changes)-1].Action)

	resumed := entry(instanceA, 3)
	status, err = r.svc.Apply(t.Context(), instanceA, resumed, nil)
	require.NoError(t, err)
	assert.Equal(t, cloudtenant.StateActive, status.State)
	assert.Equal(t, managedfeatures.LifecycleActive, r.features.lifecycles[scope])
	changes = r.audit.Changes()
	assert.Equal(t, "organization.resumed", changes[len(changes)-1].Action)
	assert.Equal(t, []cloudtenant.EventKind{
		cloudtenant.EventCreated, cloudtenant.EventApplied,
		cloudtenant.EventSuspended, cloudtenant.EventResumed,
	}, r.tenants.kinds())
}

func TestAStaleRevisionChangesNothing(t *testing.T) {
	t.Parallel()
	r := newRig(t, true)
	_, err := r.svc.Apply(t.Context(), instanceA, entry(instanceA, 5), nil)
	require.NoError(t, err)

	old := entry(instanceA, 4)
	old.Lifecycle = cloudtenant.LifecycleSuspended
	status, err := r.svc.Apply(t.Context(), instanceA, old, nil)
	require.NoError(t, err)
	assert.Equal(t, cloudtenant.StateActive, status.State)
	assert.Equal(t, int64(5), *status.AppliedRevision)
}

func TestTheSameRevisionWithAMovedPlanEndIsApplied(t *testing.T) {
	t.Parallel()
	r := newRig(t, true)
	_, err := r.svc.Apply(t.Context(), instanceA, entry(instanceA, 1), nil)
	require.NoError(t, err)

	ending := entry(instanceA, 1)
	ends := time.Unix(now+30*86400, 0).UTC()
	ending.Account.EndsAt = &ends
	_, err = r.svc.Apply(t.Context(), instanceA, ending, nil)
	require.NoError(t, err)
	held, _ := r.tenants.Get(t.Context(), instanceA)
	assert.Equal(t, ends.Unix(), r.features.lists[held.OrganizationID.String()].PlanEndsAt)
}

func TestANewInviteIsInstalledAndAnExpiredOneIsNot(t *testing.T) {
	t.Parallel()
	r := newRig(t, true)
	_, err := r.svc.Apply(t.Context(), instanceA, entry(instanceA, 1), nil)
	require.NoError(t, err)

	reissued := entry(instanceA, 2)
	reissued.FirstAdministrator.Invite.TokenSHA256 = tokenB
	_, err = r.svc.Apply(t.Context(), instanceA, reissued, nil)
	require.NoError(t, err)

	expired := entry(instanceA, 3)
	expired.FirstAdministrator.Invite = &cloudtenant.Invite{
		TokenSHA256: strings.Repeat("c", 64),
		ExpiresAt:   time.Unix(now-60, 0).UTC(),
	}
	_, err = r.svc.Apply(t.Context(), instanceA, expired, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{tokenA, tokenB}, r.invites.installed)
}

func TestAnArchiveCommandInTheEntryIsAppliedWithoutARefusal(t *testing.T) {
	t.Parallel()
	r := newRig(t, true)
	e := entry(instanceA, 1)
	e.Lifecycle = cloudtenant.LifecycleSuspended
	e.Commands = []cloudtenant.Command{{
		Kind: cloudtenant.CommandArchive, ID: instanceB, Purpose: cloudtenant.ArchivePurposeFinal,
		RequestedAt: time.Unix(now, 0).UTC(),
	}}

	status, err := r.svc.Apply(t.Context(), instanceA, e, nil)
	require.NoError(t, err)
	assert.Equal(t, cloudtenant.StateSuspended, status.State)
	assert.Nil(t, status.Refused)
}

func TestRemovalWaitsForAFinalArchive(t *testing.T) {
	t.Parallel()
	r := newRig(t, true)
	_, err := r.svc.Apply(t.Context(), instanceA, entry(instanceA, 1), nil)
	require.NoError(t, err)

	removed := entry(instanceA, 2)
	removed.Lifecycle = cloudtenant.LifecycleRemoved
	status, err := r.svc.Apply(t.Context(), instanceA, removed, nil)
	require.NoError(t, err)
	assert.Equal(t, cloudtenant.StateActive, status.State)
	require.NotNil(t, status.Refused)
	assert.Contains(t, status.Refused.Reason, "final archive")
	assert.Empty(t, r.workflows.started)

	_, err = r.svc.Remove(t.Context(), instanceA)
	require.ErrorIs(t, err, tenantlifecycle.ErrNotRemovable)
}

func TestRemovalStartsOnceTheFinalArchiveIsRecorded(t *testing.T) {
	t.Parallel()
	r := newRig(t, true)
	_, err := r.svc.Apply(t.Context(), instanceA, entry(instanceA, 1), nil)
	require.NoError(t, err)
	held, _ := r.tenants.Get(t.Context(), instanceA)
	held.LastArchiveCommandID, held.LastArchivePurpose = instanceB, cloudtenant.ArchivePurposeFinal
	require.NoError(t, r.tenants.Save(t.Context(), held))

	removed := entry(instanceA, 2)
	removed.Lifecycle = cloudtenant.LifecycleRemoved
	status, err := r.svc.Apply(t.Context(), instanceA, removed, nil)
	require.NoError(t, err)
	assert.Equal(t, cloudtenant.StateApplying, status.State, "removing is reported as applying")
	assert.Equal(t, []string{tenantjobscontract.RemovalWorkflowID(instanceA)}, r.workflows.started)

	_, err = r.svc.Apply(t.Context(), instanceA, removed, nil)
	require.NoError(t, err)
	assert.Len(t, r.workflows.started, 2, "a repeated entry makes sure the removal runs")

	require.NoError(t, r.svc.FinishRemoval(t.Context(), instanceA, map[string]any{"rows": 10}))
	status, err = r.svc.Apply(t.Context(), instanceA, removed, nil)
	require.NoError(t, err)
	assert.Equal(t, cloudtenant.StateRemoved, status.State)
	assert.Equal(t, int64(2), *status.AppliedRevision)
	assert.Nil(t, status.Administrator)
}

func TestListPagesEveryTenant(t *testing.T) {
	t.Parallel()
	r := newRig(t, true)
	_, err := r.svc.Apply(t.Context(), instanceA, entry(instanceA, 1), nil)
	require.NoError(t, err)
	_, err = r.svc.Apply(t.Context(), instanceB, entry(instanceB, 1), nil)
	require.NoError(t, err)

	first, err := r.svc.List(t.Context(), "", 1)
	require.NoError(t, err)
	require.Len(t, first.Tenants, 1)
	assert.Equal(t, instanceA, first.Tenants[0].InstanceID)
	assert.Equal(t, instanceA, first.Next)

	second, err := r.svc.List(t.Context(), first.Next, 1)
	require.NoError(t, err)
	require.Len(t, second.Tenants, 1)
	assert.Equal(t, instanceB, second.Tenants[0].InstanceID)
	assert.Empty(t, second.Next)

	_, err = r.svc.List(t.Context(), "not-a-uuid", 1)
	require.ErrorIs(t, err, tenantlifecycle.ErrCursorInvalid)
}

func TestEachTenantListsItsOwnLiveSupportGrantsInOneRead(t *testing.T) {
	t.Parallel()
	r := newRig(t, true)
	_, err := r.svc.Apply(t.Context(), instanceA, entry(instanceA, 1), nil)
	require.NoError(t, err)
	_, err = r.svc.Apply(t.Context(), instanceB, entry(instanceB, 1), nil)
	require.NoError(t, err)
	orgA := r.tenants.rows[instanceA].OrganizationID
	r.grants.live = []*supportaccessports.LiveGrant{{
		Grant: &supportaccess.Grant{
			ID: pulid.MustNew("sag_"), OrganizationID: orgA, StaffEmail: "dana@trenova.app",
			Scope: supportaccess.ScopeFull, ExpiresAt: now + 3600,
		},
		OrganizationName: "Acme Freight",
	}}

	r.grants.asked = nil
	page, err := r.svc.List(t.Context(), "", 10)
	require.NoError(t, err)
	require.Len(t, page.Tenants, 2)
	require.Len(t, r.grants.asked, 1, "one read for the whole page")
	assert.Len(t, r.grants.asked[0].OrganizationIDs, 2)
	a, b := page.Tenants[0], page.Tenants[1]
	require.NotNil(t, a.SupportGrants)
	require.Len(t, *a.SupportGrants, 1)
	assert.Equal(t, orgA.String(), (*a.SupportGrants)[0].OrganizationID)
	assert.Equal(t, "full", (*a.SupportGrants)[0].Scope)
	require.NotNil(t, b.SupportGrants, "a tenant with none says so with an empty list")
	assert.Empty(t, *b.SupportGrants)
}

func TestReadingsAreAskedOncePerPageForEveryMeteredTenant(t *testing.T) {
	t.Parallel()
	r := newRig(t, true)
	_, err := r.svc.Apply(t.Context(), instanceA, entry(instanceA, 1), nil)
	require.NoError(t, err)
	_, err = r.svc.Apply(t.Context(), instanceB, entry(instanceB, 1), nil)
	require.NoError(t, err)
	a, _ := r.tenants.Get(t.Context(), instanceA)
	hour := now - now%3600
	r.usage.readings[a.OrganizationID] = []managedusageports.MeterReading{
		{Meter: platformcatalog.MeterShipmentsTotal, Hour: hour - 3600, Quantity: 4},
		{Meter: platformcatalog.MeterShipmentsTotal, Hour: hour, Quantity: 2},
	}

	page, err := r.svc.Readings(t.Context(), &tenantlifecycle.ReadingsRequest{
		From: hour - 3600, To: hour - 3600,
	})
	require.NoError(t, err)
	require.Len(t, r.usage.asked, 1, "one question for the whole page")
	assert.Len(t, r.usage.asked[0], 2)
	require.Len(t, page.Tenants, 2)
	assert.Equal(t, []tenantlifecycle.TenantReading{{
		Meter:    string(platformcatalog.MeterShipmentsTotal),
		Hour:     time.Unix(hour-3600, 0).UTC().Format(time.RFC3339),
		Quantity: 4,
	}}, page.Tenants[0].Readings)

	_, err = r.svc.Readings(
		t.Context(),
		&tenantlifecycle.ReadingsRequest{From: hour + 1, To: hour + 1},
	)
	require.ErrorIs(t, err, tenantlifecycle.ErrWindowInvalid)
}

func TestUsageAnswersReachEachTenantsScopeAndNameStrangers(t *testing.T) {
	t.Parallel()
	r := newRig(t, true)
	_, err := r.svc.Apply(t.Context(), instanceA, entry(instanceA, 1), nil)
	require.NoError(t, err)
	held, _ := r.tenants.Get(t.Context(), instanceA)

	answer, err := r.svc.SetUsage(t.Context(), []tenantlifecycle.TenantUsage{
		{
			InstanceID: instanceA,
			Overage:    true,
			Window: &tenantlifecycle.UsageWindow{
				Start: time.Unix(now-86400, 0).UTC(),
				End:   time.Unix(now+86400, 0).UTC(),
			},
			Meters: []managedusage.Meter{},
		},
		{InstanceID: instanceB, Meters: []managedusage.Meter{}},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, answer.Applied)
	require.Len(t, answer.Refused, 1)
	assert.Equal(t, instanceB, answer.Refused[0].InstanceID)
	state := r.usage.states[held.OrganizationID.String()]
	require.NotNil(t, state)
	assert.True(t, state.Overage)
	assert.Equal(t, now-86400, state.WindowStart)
}

func TestSessionsOfASuspendedTenantAreEnded(t *testing.T) {
	t.Parallel()
	r := newRig(t, true)
	_, err := r.svc.Apply(t.Context(), instanceA, entry(instanceA, 1), nil)
	require.NoError(t, err)
	held, _ := r.tenants.Get(t.Context(), instanceA)
	r.members.ids = []pulid.ID{held.AdminUserID}

	result, err := r.svc.RevokeSessions(t.Context(), &tenantlifecycle.RevocationRequest{
		Tenant: dbscope.Tenant{
			OrganizationID: held.OrganizationID,
			BusinessUnitID: held.BusinessUnitID,
		},
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Members)
	assert.Equal(t, 1, result.Sessions)
	assert.Equal(t, []pulid.ID{held.AdminUserID}, r.sessions.ended)
}

func TestTheInstanceTagIsTheTenantsInstance(t *testing.T) {
	t.Parallel()
	r := newRig(t, true)
	_, err := r.svc.Apply(t.Context(), instanceA, entry(instanceA, 1), nil)
	require.NoError(t, err)
	held, _ := r.tenants.Get(t.Context(), instanceA)

	tags := tenantlifecycle.NewInstanceTags(r.tenants)
	tag, err := tags.InstanceTag(t.Context(), pagination.TenantInfo{
		OrgID: held.OrganizationID, BuID: held.BusinessUnitID,
	})
	require.NoError(t, err)
	assert.Equal(t, instanceA, tag)

	tag, err = tags.InstanceTag(t.Context(), pagination.TenantInfo{OrgID: pulid.MustNew("org_")})
	require.NoError(t, err)
	assert.Empty(t, tag)
}

func TestSupportEntryFindsOnlyAMadeTenantAndWhetherItIsActive(t *testing.T) {
	t.Parallel()
	r := newRig(t, true)
	_, err := r.svc.Apply(t.Context(), instanceA, entry(instanceA, 1), nil)
	require.NoError(t, err)
	tenants := tenantlifecycle.NewSupportTenants(r.tenants)

	held, err := tenants.Tenant(t.Context(), instanceA)
	require.NoError(t, err)
	assert.Equal(t, r.tenants.rows[instanceA].OrganizationID, held.OrganizationID)
	assert.True(t, held.Active)

	suspended := entry(instanceA, 2)
	suspended.Lifecycle = cloudtenant.LifecycleSuspended
	_, err = r.svc.Apply(t.Context(), instanceA, suspended, nil)
	require.NoError(t, err)
	held, err = tenants.Tenant(t.Context(), instanceA)
	require.NoError(t, err)
	assert.False(t, held.Active)

	for _, id := range []string{instanceB, "not-an-instance"} {
		_, err = tenants.Tenant(t.Context(), id)
		require.ErrorIs(t, err, supportaccessports.ErrTenantNotHeld, id)
	}
}
