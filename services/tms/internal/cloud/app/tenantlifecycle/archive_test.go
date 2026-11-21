package tenantlifecycle_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/cloud/app/tenantlifecycle"
	"github.com/emoss08/trenova/internal/cloud/domain/cloudtenant"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantarchive"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantcommand"
	"github.com/emoss08/trenova/internal/cloud/jobs/archivejobs/archivejobscontract"
	"github.com/emoss08/trenova/internal/cloud/ports/cloudtenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	archiveID  = "7d1a3c1e-8c1f-4a39-9a43-5b0d7c4e2f10"
	restoreID  = "8d1a3c1e-8c1f-4a39-9a43-5b0d7c4e2f10"
	importID   = "9d1a3c1e-8c1f-4a39-9a43-5b0d7c4e2f10"
	testSecret = "fake-secret-for-tests"
)

func target(prefix string, keys bool) *tenantarchive.Access {
	access := &tenantarchive.Access{
		Location: tenantarchive.Location{
			Endpoint: "https://objects.test", Bucket: "archives", Prefix: prefix,
		},
		Credentials: tenantarchive.Credentials{AccessKey: "test-access", SecretKey: testSecret},
	}
	if keys {
		access.Keys = []tenantarchive.Key{
			{
				ID:       "region-1",
				Material: bytes.Repeat([]byte{7}, tenantarchive.KeySize),
				Current:  true,
			},
		}
	}
	return access
}

func made(t *testing.T, r *rig, lifecycle cloudtenant.Lifecycle) {
	t.Helper()
	e := entry(instanceA, 1)
	e.Lifecycle = lifecycle
	_, err := r.svc.Apply(t.Context(), instanceA, e, nil)
	require.NoError(t, err)
}

func TestAFinalArchiveWaitsForTheTenantToBeSuspended(t *testing.T) {
	t.Parallel()
	r := newRig(t, true)
	made(t, r, cloudtenant.LifecycleActive)
	req := &tenantlifecycle.ArchiveRequest{
		CommandID: archiveID,
		Purpose:   tenantarchive.PurposeFinal,
		Access:    target(tenantarchive.CommandPrefix(instanceA, archiveID), true),
	}
	_, err := r.svc.StartArchive(t.Context(), instanceA, req)
	require.ErrorIs(t, err, tenantlifecycle.ErrTenantState)

	exportReq := &tenantlifecycle.ArchiveRequest{
		CommandID: archiveID,
		Purpose:   tenantarchive.PurposeExport,
		Access:    target(tenantarchive.CommandPrefix(instanceA, archiveID), false),
	}
	view, err := r.svc.StartArchive(t.Context(), instanceA, exportReq)
	require.NoError(t, err, "an export is taken while the tenant keeps working")
	assert.Equal(t, tenantcommand.StateQueued, view.State)
	again, err := r.svc.StartArchive(t.Context(), instanceA, exportReq)
	require.NoError(t, err)
	assert.Equal(t, view.CommandID, again.CommandID, "the same command id answers the same command")
	assert.Equal(t, []string{
		archivejobscontract.WorkflowID(tenantcommand.KindArchive, archiveID),
		archivejobscontract.WorkflowID(tenantcommand.KindArchive, archiveID),
	}, r.workflows.started[len(r.workflows.started)-2:])

	held := r.commands.rows[archiveID]
	assert.NotContains(t, held.SealedAccess, testSecret, "the access is kept sealed")
	_, access, err := r.svc.CommandAccess(t.Context(), archiveID)
	require.NoError(t, err)
	assert.Equal(t, testSecret, access.Credentials.SecretKey)

	_, err = r.svc.StartArchive(t.Context(), instanceA, &tenantlifecycle.ArchiveRequest{
		CommandID: archiveID,
		Purpose:   tenantarchive.PurposeMove,
		Access:    target(tenantarchive.CommandPrefix(instanceA, archiveID), true),
	})
	require.ErrorIs(t, err, tenantlifecycle.ErrCommandConflict)
}

func TestAnArchiveIsRecordedForTheStatusAndItsSecretsAreErased(t *testing.T) {
	t.Parallel()
	r := newRig(t, true)
	made(t, r, cloudtenant.LifecycleSuspended)
	_, err := r.svc.StartArchive(t.Context(), instanceA, &tenantlifecycle.ArchiveRequest{
		CommandID: archiveID,
		Purpose:   tenantarchive.PurposeFinal,
		Access:    target(tenantarchive.CommandPrefix(instanceA, archiveID), true),
	})
	require.NoError(t, err)
	require.NoError(t, r.svc.BeginCommand(t.Context(), archiveID))
	taken := time.Unix(now, 0).UTC()
	require.NoError(t, r.svc.FinishArchive(t.Context(), &tenantlifecycle.ArchiveResult{
		CommandID: archiveID,
		Location:  "s3://archives/" + tenantarchive.CommandPrefix(instanceA, archiveID),
		SHA256:    strings.Repeat("a", 64),
		SizeBytes: 4096,
		TakenAt:   taken,
	}))
	assert.Empty(t, r.commands.rows[archiveID].SealedAccess)
	held := r.tenants.rows[instanceA]
	assert.True(t, held.HasFinalArchive())
	page, err := r.svc.List(t.Context(), "", 10)
	require.NoError(t, err)
	require.NotNil(t, page.Tenants[0].LastArchive)
	assert.Equal(t, archiveID, page.Tenants[0].LastArchive.CommandID)
	body, err := sonic.Marshal(held)
	require.NoError(t, err)
	assert.NotContains(t, string(body), testSecret)
}

func TestARestoreHoldsTheTenantSuspendedAndReportsHowItEnded(t *testing.T) {
	t.Parallel()
	r := newRig(t, true)
	made(t, r, cloudtenant.LifecycleActive)
	req := &tenantlifecycle.RestoreRequest{
		CommandID:    restoreID,
		ExpectSHA256: strings.Repeat("b", 64),
		Access:       target("tenants/"+instanceA+"/nightly/2026-11-19/", true),
	}
	_, err := r.svc.StartRestore(t.Context(), instanceA, req)
	require.ErrorIs(
		t,
		err,
		tenantlifecycle.ErrTenantState,
		"a restore is listed only while suspended",
	)

	suspended := entry(instanceA, 2)
	suspended.Lifecycle = cloudtenant.LifecycleSuspended
	_, err = r.svc.Apply(t.Context(), instanceA, suspended, nil)
	require.NoError(t, err)
	_, err = r.svc.StartRestore(t.Context(), instanceA, req)
	require.NoError(t, err)
	assert.Equal(t, cloudtenant.StateRestoring, r.tenants.rows[instanceA].State)

	status, err := r.svc.Apply(t.Context(), instanceA, entry(instanceA, 3), nil)
	require.NoError(t, err)
	assert.Equal(t, cloudtenant.StateSuspended, status.State, "a revision waits for the restore")
	assert.Nil(t, status.LastRestore, "a running restore is not reported")

	require.NoError(t, r.svc.FailRestore(t.Context(), restoreID,
		tenantarchive.Refuse(tenantarchive.RefusalChecksum, "a part does not match"), false))
	held := r.tenants.rows[instanceA]
	assert.Equal(t, cloudtenant.StateSuspended, held.State)
	assert.True(t, strings.HasPrefix(held.LastRestoreReason, "Nothing was changed: "))
	scope := held.OrganizationID.String()
	assert.Equal(t, managedfeatures.LifecycleSuspended, r.features.lifecycles[scope])

	second := "ad1a3c1e-8c1f-4a39-9a43-5b0d7c4e2f10"
	req.CommandID = second
	_, err = r.svc.StartRestore(t.Context(), instanceA, req)
	require.NoError(t, err)
	require.NoError(t, r.svc.FinishRestore(t.Context(), &tenantlifecycle.RestoreResult{
		CommandID: second, SHA256: strings.Repeat("b", 64),
	}))
	held = r.tenants.rows[instanceA]
	assert.Equal(
		t,
		cloudtenant.StateSuspended,
		held.State,
		"the control plane resumes it, not the restore",
	)
	assert.Equal(t, cloudtenant.RestoreDone, held.LastRestoreState)
}

func TestAnImportMakesTheTenantFromTheArchiveInsteadOfBootstrapping(t *testing.T) {
	t.Parallel()
	r := newRig(t, true)
	prefix := tenantarchive.CommandPrefix(instanceA, archiveID)
	e := entry(instanceA, 1)
	e.Lifecycle = cloudtenant.LifecycleSuspended
	e.FirstAdministrator.Invite = nil
	e.Commands = []cloudtenant.Command{{
		Kind: cloudtenant.CommandImport, ID: importID, Location: "s3://archives/" + prefix,
		ExpectSHA256: strings.Repeat("c", 64), RequestedAt: time.Unix(now, 0).UTC(),
	}}
	status, err := r.svc.Apply(t.Context(), instanceA, e, nil)
	require.NoError(t, err)
	require.NotNil(t, status.Refused, "an import entry needs its target")
	assert.Zero(t, r.bootstrap.calls)

	e.Revision = 2
	status, err = r.svc.Apply(t.Context(), instanceA, e, target(prefix, true))
	require.NoError(t, err)
	assert.Nil(t, status.Refused)
	assert.Equal(t, cloudtenant.StateApplying, status.State)
	assert.Zero(t, r.bootstrap.calls, "the tenant is not bootstrapped")
	assert.Contains(
		t,
		r.workflows.started,
		archivejobscontract.WorkflowID(tenantcommand.KindImport, importID),
	)
	desired, err := sonic.Marshal(r.tenants.rows[instanceA].Desired)
	require.NoError(t, err)
	assert.NotContains(t, string(desired), testSecret)

	org, bu := pulid.MustNew("org_"), pulid.MustNew("bu_")
	require.NoError(t, r.svc.BeginCommand(t.Context(), importID))
	require.NoError(t, r.svc.FinishImport(t.Context(), &tenantlifecycle.ImportResult{
		CommandID: importID,
		Tenant:    dbscope.Tenant{OrganizationID: org, BusinessUnitID: bu},
		Owner:     pulid.MustNew("usr_"),
		SHA256:    strings.Repeat("c", 64),
	}))
	held := r.tenants.rows[instanceA]
	assert.Equal(t, org, held.OrganizationID)
	assert.Equal(t, cloudtenant.StateSuspended, held.State)
	assert.Equal(t, int64(2), held.AppliedRevision)
	assert.Positive(t, held.ImportedAt)
	assert.Empty(t, r.invites.installed, "an imported owner gets no invitation")
}

func TestAFailedImportIsReportedOnceAsACommandFailure(t *testing.T) {
	t.Parallel()
	r := newRig(t, true)
	prefix := tenantarchive.CommandPrefix(instanceA, archiveID)
	e := entry(instanceA, 1)
	e.Commands = []cloudtenant.Command{{
		Kind: cloudtenant.CommandImport, ID: importID, Location: "s3://archives/" + prefix,
		ExpectSHA256: strings.Repeat("c", 64), RequestedAt: time.Unix(now, 0).UTC(),
	}}
	_, err := r.svc.Apply(t.Context(), instanceA, e, target(prefix, true))
	require.NoError(t, err)
	require.NoError(t, r.svc.FailImport(
		t.Context(),
		importID,
		tenantarchive.Refuse(
			tenantarchive.RefusalTaken,
			"an address is already used on this server",
		),
	))
	page, err := r.svc.List(t.Context(), "", 10)
	require.NoError(t, err)
	require.Len(t, page.Tenants[0].CommandFailures, 1)
	assert.Equal(t, cloudtenant.CommandFailure{
		CommandID: importID, Kind: "import", Code: "taken",
		Reason: "an address is already used on this server",
	}, page.Tenants[0].CommandFailures[0])
	assert.Nil(t, page.Tenants[0].Refused)
}

const (
	endpointRefusal = "the archive endpoint's host is not in this server's managed.archive.allowedEndpoints"
	refusedEndpoint = "https://objects.elsewhere.test"
)

func nightlyStorage() *tenantarchive.Access {
	access := target(tenantarchive.TenantsRoot, true)
	return access
}

func TestTheNightlyStorageIsHeldAndNightlyArchivesGoUnderItsPrefix(t *testing.T) {
	t.Parallel()
	r := newRig(t, true)
	made(t, r, cloudtenant.LifecycleActive)
	require.NoError(t, r.svc.SetNightlyGrant(t.Context(), nightlyStorage(), time.Time{}))
	require.NotNil(t, r.commands.grant)
	assert.Equal(t, "s3://archives/tenants/", r.commands.grant.Location)
	assert.NotContains(t, r.commands.grant.SealedAccess, testSecret)

	page, err := r.svc.NightlyTargets(t.Context(), "", 10)
	require.NoError(t, err)
	require.Len(t, page.Targets, 1)
	day := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	cmd, access, err := r.svc.BeginNightly(t.Context(), &page.Targets[0], day)
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.NotNil(t, access)
	assert.Equal(t, "tenants/"+instanceA+"/nightly/2026-11-20/", access.Location.Prefix)
	assert.Equal(t, "archives", access.Location.Bucket)
	current := access.CurrentKey()
	require.NotNil(t, current, "a nightly archive is written under the current key")
	assert.Equal(t, "region-1", current.ID)
}

func TestANightlyStorageOutsideTenantsOrAtARefusedEndpointIsNotHeld(t *testing.T) {
	t.Parallel()
	r := newRig(t, true)
	for _, prefix := range []string{"", "tenants/" + instanceA + "/", "archives/", "tenants"} {
		access := nightlyStorage()
		access.Location.Prefix = prefix
		err := r.svc.SetNightlyGrant(t.Context(), access, time.Time{})
		require.ErrorIs(t, err, tenantarchive.ErrNightlyPrefix, prefix)
	}

	r.buckets.refused = refusedEndpoint
	access := nightlyStorage()
	access.Location.Endpoint = refusedEndpoint
	err := r.svc.SetNightlyGrant(t.Context(), access, time.Time{})
	refusal, ok := tenantarchive.RefusalOf(err)
	require.True(t, ok, "%v", err)
	assert.Equal(t, tenantarchive.RefusalLocation, refusal.Code)
	assert.NotContains(t, err.Error(), testSecret)

	plain := nightlyStorage()
	plain.Location.Endpoint = "http://objects.test"
	require.ErrorIs(t, r.svc.SetNightlyGrant(t.Context(), plain, time.Time{}),
		tenantarchive.ErrLocationInvalid)
	assert.Nil(t, r.commands.grant, "a refused storage is never held")
}

func TestRemovingTheNightlyStorageStopsNightlyArchives(t *testing.T) {
	t.Parallel()
	r := newRig(t, true)
	made(t, r, cloudtenant.LifecycleActive)
	_, err := r.svc.NightlyTargets(t.Context(), "", 10)
	require.ErrorIs(t, err, cloudtenantports.ErrGrantMissing, "no storage is held yet")

	require.NoError(t, r.svc.SetNightlyGrant(t.Context(), nightlyStorage(), time.Time{}))
	page, err := r.svc.NightlyTargets(t.Context(), "", 10)
	require.NoError(t, err)
	require.Len(t, page.Targets, 1)

	require.NoError(t, r.svc.RemoveNightlyGrant(t.Context()))
	assert.Nil(t, r.commands.grant)
	_, err = r.svc.NightlyTargets(t.Context(), "", 10)
	require.ErrorIs(t, err, cloudtenantports.ErrGrantMissing)
	_, _, err = r.svc.BeginNightly(t.Context(), &page.Targets[0], time.Unix(now, 0))
	require.ErrorIs(t, err, cloudtenantports.ErrGrantMissing)

	dedicated := newRig(t, false)
	require.ErrorIs(t, dedicated.svc.RemoveNightlyGrant(t.Context()), tenantlifecycle.ErrNotShared)
	require.ErrorIs(t, dedicated.svc.SetNightlyGrant(t.Context(), nightlyStorage(), time.Time{}),
		tenantlifecycle.ErrNotShared)
}

func TestAStorageOrKeyFailureIsReportedWithItsCode(t *testing.T) {
	t.Parallel()
	r := newRig(t, true)
	made(t, r, cloudtenant.LifecycleSuspended)
	req := &tenantlifecycle.RestoreRequest{
		CommandID:    restoreID,
		ExpectSHA256: strings.Repeat("b", 64),
		Access:       target("tenants/"+instanceA+"/nightly/2026-11-19/", true),
	}
	_, err := r.svc.StartRestore(t.Context(), instanceA, req)
	require.NoError(t, err)
	require.NoError(t, r.svc.FailRestore(t.Context(), restoreID, tenantarchive.Refuse(
		tenantarchive.RefusalKeyAbsent,
		"the archive was sealed with archive key region-0, which the command does not carry",
	), false))
	page, err := r.svc.List(t.Context(), "", 10)
	require.NoError(t, err)
	require.Len(t, page.Tenants[0].CommandFailures, 1)
	failure := page.Tenants[0].CommandFailures[0]
	assert.Equal(t, restoreID, failure.CommandID)
	assert.Equal(t, "restore", failure.Kind)
	assert.Equal(t, string(tenantarchive.RefusalKeyAbsent), failure.Code)
}
