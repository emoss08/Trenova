package tenantagent_test

import (
	"encoding/base64"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/tms"
)

const (
	bucketSecret  = "fixture-bucket-secret-not-real-0001"
	sessionSecret = "fixture-session-token-not-real-0001"
	archiveBucket = "trenova-archives-test"
)

var archiveKeyMaterial = base64.StdEncoding.EncodeToString(
	[]byte("fixture-archive-key-32-bytes-xyz"),
)

func archiveCommand(
	id uuid.UUID,
	purpose instanceapi.SharedArchiveCommandPurpose,
) instanceapi.SharedTenantCommand {
	var c instanceapi.SharedTenantCommand
	if err := c.FromSharedArchiveCommand(instanceapi.SharedArchiveCommand{
		Kind: instanceapi.SharedCommandArchive, Id: id, Purpose: purpose,
		RequestedAt: time.Date(2026, 11, 20, 14, 0, 0, 0, time.UTC),
	}); err != nil {
		panic(err)
	}
	return c
}

func restoreCommand(id uuid.UUID, location string) instanceapi.SharedTenantCommand {
	var c instanceapi.SharedTenantCommand
	if err := c.FromSharedRestoreCommand(instanceapi.SharedRestoreCommand{
		Kind: instanceapi.SharedCommandRestore, Id: id, Location: location,
		ExpectSha256: strings.Repeat("ab", 32),
		TakenAt:      time.Date(2026, 11, 19, 7, 0, 0, 0, time.UTC),
		RequestedAt:  time.Date(2026, 11, 20, 14, 0, 0, 0, time.UTC),
	}); err != nil {
		panic(err)
	}
	return c
}

func importCommand(id uuid.UUID, location string) instanceapi.SharedTenantCommand {
	var c instanceapi.SharedTenantCommand
	if err := c.FromSharedImportCommand(instanceapi.SharedImportCommand{
		Kind: instanceapi.SharedCommandImport, Id: id, Location: location,
		ExpectSha256: strings.Repeat("cd", 32),
		RequestedAt:  time.Date(2026, 11, 20, 14, 0, 0, 0, time.UTC),
	}); err != nil {
		panic(err)
	}
	return c
}

func keys() *[]instanceapi.ArchiveKey {
	return &[]instanceapi.ArchiveKey{{Id: "server-key-1", Key: archiveKeyMaterial, Current: true}}
}

func nightlyTarget() *instanceapi.SharedNightlyTarget {
	return &instanceapi.SharedNightlyTarget{
		Access: instanceapi.NightlyWrite, Endpoint: "https://s3.test", Region: "bhs",
		Bucket: archiveBucket, Prefix: "tenants/",
		AccessKeyId: "FIXTUREACCESSKEY", SecretAccessKey: bucketSecret,
		ArchiveKeys: *keys(),
	}
}

func writeTarget(instance, command uuid.UUID, withKeys bool) instanceapi.SharedArchiveTarget {
	session := sessionSecret
	t := instanceapi.SharedArchiveTarget{
		InstanceId: instance, CommandId: command, Kind: instanceapi.TargetArchive,
		Access: instanceapi.TargetWrite, Endpoint: "https://s3.test", Region: "bhs",
		Bucket: archiveBucket, Prefix: "tenants/" + instance.String() + "/archives/" + command.String() + "/",
		AccessKeyId: "FIXTUREACCESSKEY", SecretAccessKey: bucketSecret, SessionToken: &session,
	}
	if withKeys {
		t.ArchiveKeys = keys()
	}
	return t
}

func readTarget(
	instance, command uuid.UUID,
	kind instanceapi.SharedArchiveTargetKind,
	prefix string,
) instanceapi.SharedArchiveTarget {
	return instanceapi.SharedArchiveTarget{
		InstanceId: instance, CommandId: command, Kind: kind, Access: instanceapi.TargetRead,
		Endpoint: "https://s3.test", Region: "bhs", Bucket: archiveBucket, Prefix: prefix,
		AccessKeyId: "FIXTUREACCESSKEY", SecretAccessKey: bucketSecret, ArchiveKeys: keys(),
	}
}

func storage(targets ...instanceapi.SharedArchiveTarget) instanceapi.SharedStorage {
	return instanceapi.SharedStorage{
		Nightly:  nightlyTarget(),
		Archives: targets,
		Waiting:  []instanceapi.SharedArchiveWaiting{},
	}
}

func TestAnExportArchiveIsTakenOnceAndReported(t *testing.T) {
	r := newRig(t)
	id, cmd := uuid.New(), uuid.New()
	entry := tenant(id, 1, instanceapi.SharedTenantActive)
	entry.Commands = []instanceapi.SharedTenantCommand{
		archiveCommand(cmd, instanceapi.SharedArchiveExport),
	}
	r.cp.SetSharedTenants([]instanceapi.SharedTenant{entry})
	r.cp.SetSharedStorage(storage(writeTarget(id, cmd, false)))
	r.tms.SetBusy(id.String(), true)

	r.iterate(t)
	assert.Zero(
		t,
		r.tms.CommandPosts(cmd.String()),
		"nothing starts before the tenant holds its entry",
	)

	r.iterate(t)
	assert.Equal(t, 1, r.tms.CommandPosts(cmd.String()))
	assert.Empty(t, r.tms.CommandState(cmd.String()), "a busy tenant leaves the command waiting")
	assert.Nil(t, r.reported(t, id).CommandFailures, "waiting is not a failure")

	r.tms.SetBusy(id.String(), false)
	r.clock.Advance(time.Minute)
	r.iterate(t)
	require.Equal(t, "running", r.tms.CommandState(cmd.String()))
	target, ok := r.tms.CommandTarget(cmd.String())
	require.True(t, ok)
	assert.Equal(t, "write", target.Access)
	assert.Equal(t, "tenants/"+id.String()+"/archives/"+cmd.String()+"/", target.Prefix)
	assert.Empty(t, target.ArchiveKeys, "an export is written without the archive keys")

	r.iterate(t)
	r.iterate(t)
	assert.Equal(
		t,
		1,
		r.tms.CommandStarts(cmd.String()),
		"a running command is polled, not started again",
	)

	taken := time.Date(2026, 11, 20, 14, 12, 0, 0, time.UTC)
	r.tms.FinishArchive(cmd.String(), 4096, strings.Repeat("ef", 32), taken)
	r.iterate(t)
	status := r.reported(t, id)
	require.NotNil(t, status.LastArchive)
	assert.Equal(t, cmd, status.LastArchive.CommandId)
	assert.Equal(t, "s3://"+archiveBucket+"/"+target.Prefix, status.LastArchive.Location)
	assert.Equal(t, int64(4096), status.LastArchive.SizeBytes)
	grants := r.tms.NightlyGrants()
	require.Len(t, grants, 1, "the nightly storage comes from the nightly target alone, once")
	assert.Equal(t, "tenants/", grants[0].Prefix)
}

func TestAFinalArchiveThenTheTenantIsRemoved(t *testing.T) {
	r := newRig(t)
	id, cmd := uuid.New(), uuid.New()
	entry := tenant(id, 1, instanceapi.SharedTenantSuspended)
	entry.Commands = []instanceapi.SharedTenantCommand{
		archiveCommand(cmd, instanceapi.SharedArchiveFinal),
	}
	r.cp.SetSharedTenants([]instanceapi.SharedTenant{entry})
	r.cp.SetSharedStorage(storage(writeTarget(id, cmd, true)))

	r.iterate(t)
	r.iterate(t)
	require.Equal(t, 1, r.tms.CommandStarts(cmd.String()))
	target, _ := r.tms.CommandTarget(cmd.String())
	require.Len(t, target.ArchiveKeys, 1)
	assert.Equal(t, "server-key-1", target.ArchiveKeys[0].ID)
	assert.Equal(t, sessionSecret, target.SessionToken)
	grants := r.tms.NightlyGrants()
	require.Len(t, grants, 1, "the server's nightly storage reaches the TMS")
	assert.Equal(t, archiveBucket, grants[0].Bucket)
	assert.Empty(
		t,
		grants[0].SessionToken,
		"the nightly grant is the nightly target's, not a command's",
	)

	r.tms.FinishArchive(cmd.String(), 1<<20, strings.Repeat("01", 32), r.clock.Now())
	r.iterate(t)
	require.NotNil(t, r.reported(t, id).LastArchive)

	r.cp.SetSharedTenants(
		[]instanceapi.SharedTenant{tenant(id, 2, instanceapi.SharedTenantRemoved)},
	)
	r.iterate(t)
	status := r.reported(t, id)
	assert.Equal(t, instanceapi.TenantRemoved, status.State)
	assert.Equal(t, 1, r.tms.CommandStarts(cmd.String()))
}

func TestARestoreIsReportedDoneOrFailedWithWhetherTheTenantWasTouched(t *testing.T) {
	r := newRig(t)
	id := uuid.New()
	prefix := "tenants/" + id.String() + "/nightly/2026-11-19/"
	location := "s3://" + archiveBucket + "/" + prefix
	cases := []struct {
		name   string
		reason string
		want   string
	}{
		{name: "done"},
		{name: "untouched", reason: "Nothing was changed: the manifest's digest differs",
			want: "Nothing was changed: the manifest's digest differs"},
		{name: "incomplete", reason: "The tenant is incomplete: the load failed",
			want: "The tenant is incomplete: the load failed"},
	}
	for i, tc := range cases {
		cmd := uuid.New()
		entry := tenant(id, int64(i+1), instanceapi.SharedTenantSuspended)
		entry.Commands = []instanceapi.SharedTenantCommand{restoreCommand(cmd, location)}
		r.cp.SetSharedTenants([]instanceapi.SharedTenant{entry})
		r.cp.SetSharedStorage(storage(readTarget(id, cmd, instanceapi.TargetRestore, prefix)))
		r.iterate(t)
		r.iterate(t)
		require.Equal(t, 1, r.tms.CommandStarts(cmd.String()), tc.name)
		r.tms.FinishRestore(cmd.String(), r.clock.Now(), tc.reason)
		r.iterate(t)
		restore := r.reported(t, id).LastRestore
		require.NotNil(t, restore, tc.name)
		assert.Equal(t, cmd, restore.CommandId, tc.name)
		if tc.reason == "" {
			assert.Equal(t, instanceapi.RestoreDone, restore.State)
			assert.Nil(t, restore.Reason)
			continue
		}
		assert.Equal(t, instanceapi.RestoreFailed, restore.State, tc.name)
		require.NotNil(t, restore.Reason)
		assert.Equal(t, tc.want, *restore.Reason, tc.name)
	}

	refused := uuid.New()
	entry := tenant(id, 9, instanceapi.SharedTenantSuspended)
	entry.Commands = []instanceapi.SharedTenantCommand{restoreCommand(refused, location)}
	r.cp.SetSharedTenants([]instanceapi.SharedTenant{entry})
	r.cp.SetSharedStorage(storage(readTarget(id, refused, instanceapi.TargetRestore, prefix)))
	r.tms.RefuseCommand(refused.String(), "the tenant's state does not allow this command")
	r.iterate(t)
	r.iterate(t)
	restore := r.reported(t, id).LastRestore
	require.NotNil(t, restore)
	assert.Equal(t, refused, restore.CommandId)
	assert.Equal(t, instanceapi.RestoreFailed, restore.State)
	require.NotNil(t, restore.Reason)
	assert.True(t, strings.HasPrefix(*restore.Reason, "Nothing was changed: "), *restore.Reason)
	r.iterate(t)
	assert.Equal(t, 1, r.tms.CommandPosts(refused.String()), "a refused restore is not sent again")
}

func TestAFinishedRestoresNoteReachesStaffAsOnePrintableLineAndNeverAsAReason(t *testing.T) {
	r := newRig(t)
	id := uuid.New()
	prefix := "tenants/" + id.String() + "/nightly/2026-11-19/"
	location := "s3://" + archiveBucket + "/" + prefix
	long := strings.Repeat("x", 700)
	cases := []struct {
		name   string
		reason string
		note   string
		want   *string
	}{
		{name: "no note"},
		{
			name: "a note on a done restore",
			note: "Restored, but the audit log was marked.\nSecond\tline\x00.",
			want: ptr("Restored, but the audit log was marked. Second line ."),
		},
		{name: "a long note", note: long, want: ptr(strings.Repeat("x", 499) + "…")},
		{name: "a failed restore", reason: "The tenant is incomplete: the load failed", note: "kept"},
	}
	for i, tc := range cases {
		cmd := uuid.New()
		entry := tenant(id, int64(i+1), instanceapi.SharedTenantSuspended)
		entry.Commands = []instanceapi.SharedTenantCommand{restoreCommand(cmd, location)}
		r.cp.SetSharedTenants([]instanceapi.SharedTenant{entry})
		r.cp.SetSharedStorage(storage(readTarget(id, cmd, instanceapi.TargetRestore, prefix)))
		r.iterate(t)
		r.iterate(t)
		r.tms.FinishRestoreNoted(cmd.String(), r.clock.Now(), tc.reason, tc.note)
		r.iterate(t)
		restore := r.reported(t, id).LastRestore
		require.NotNil(t, restore, tc.name)
		if tc.reason != "" {
			assert.Equal(t, instanceapi.RestoreFailed, restore.State, tc.name)
			assert.Nil(t, restore.Note, "a failed restore carries no note: %s", tc.name)
			require.NotNil(t, restore.Reason)
			assert.NotContains(t, *restore.Reason, "kept")
			continue
		}
		assert.Equal(t, instanceapi.RestoreDone, restore.State, tc.name)
		assert.Nil(t, restore.Reason, "a note is never put in the reason: %s", tc.name)
		assert.Equal(t, tc.want, restore.Note, tc.name)
	}
}

func ptr(s string) *string {
	return &s
}

func TestATenantIsMadeFromAnArchiveOnItsFirstEntryOnly(t *testing.T) {
	r := newRig(t)
	id, cmd := uuid.New(), uuid.New()
	prefix := "tenants/" + id.String() + "/archives/" + uuid.NewString() + "/"
	location := "s3://" + archiveBucket + "/" + prefix
	entry := tenant(id, 1, instanceapi.SharedTenantSuspended)
	entry.Commands = []instanceapi.SharedTenantCommand{importCommand(cmd, location)}
	r.cp.SetSharedTenants([]instanceapi.SharedTenant{entry})
	r.cp.SetSharedStorage(instanceapi.SharedStorage{
		Archives: []instanceapi.SharedArchiveTarget{},
		Waiting: []instanceapi.SharedArchiveWaiting{{
			InstanceId: id, CommandId: cmd, Reason: instanceapi.SharedArchiveNoBucket,
		}},
	})

	r.iterate(t)
	assert.Zero(t, r.tms.TenantPuts(id.String()), "the tenant waits for its archive's location")

	r.cp.SetSharedStorage(storage(readTarget(id, cmd, instanceapi.TargetImport, prefix)))
	r.clock.Advance(time.Minute)
	r.iterate(t)
	require.Equal(t, 1, r.tms.CommandStarts(cmd.String()))
	assert.Contains(t, string(r.tms.TenantEntry(id.String())), "importTarget")
	assert.Equal(t, instanceapi.TenantApplying, r.reported(t, id).State)

	r.iterate(t)
	r.tms.FinishImport(cmd.String())
	r.iterate(t)
	status := r.reported(t, id)
	assert.Equal(t, instanceapi.TenantSuspended, status.State)
	require.NotNil(t, status.AppliedRevision)
	assert.Equal(t, int64(1), *status.AppliedRevision)

	later := tenant(id, 2, instanceapi.SharedTenantActive)
	later.Commands = []instanceapi.SharedTenantCommand{importCommand(cmd, location)}
	r.cp.SetSharedTenants([]instanceapi.SharedTenant{later})
	r.iterate(t)
	assert.Equal(t, int64(2), *r.reported(t, id).AppliedRevision)
	assert.NotContains(t, string(r.tms.TenantEntry(id.String())), "importTarget",
		"a later revision never carries the import again")
	assert.Equal(t, 1, r.tms.CommandStarts(cmd.String()))
}

func TestACommandTheTMSRefusesIsReportedAsAFailureOnce(t *testing.T) {
	r := newRig(t)
	id, cmd := uuid.New(), uuid.New()
	entry := tenant(id, 1, instanceapi.SharedTenantActive)
	entry.Commands = []instanceapi.SharedTenantCommand{
		archiveCommand(cmd, instanceapi.SharedArchiveFinal),
	}
	r.cp.SetSharedTenants([]instanceapi.SharedTenant{entry})
	r.cp.SetSharedStorage(storage(writeTarget(id, cmd, true)))
	r.tms.RefuseCommand(cmd.String(), "the tenant's state does not allow this command")

	r.iterate(t)
	r.iterate(t)
	r.iterate(t)
	failures := r.reported(t, id).CommandFailures
	require.NotNil(t, failures)
	require.Len(t, *failures, 1)
	assert.Equal(t, cmd, (*failures)[0].CommandId)
	assert.Equal(t, instanceapi.SharedFailedArchive, (*failures)[0].Kind)
	assert.Contains(t, (*failures)[0].Reason, "does not allow")
	assert.Equal(t, 1, r.tms.CommandPosts(cmd.String()), "a refused command is not sent again")

	r.cp.SetSharedTenants([]instanceapi.SharedTenant{tenant(id, 2, instanceapi.SharedTenantActive)})
	r.iterate(t)
	assert.Nil(
		t,
		r.reported(t, id).CommandFailures,
		"a failure stops once the command is not listed",
	)
}

func TestAFailedCommandOfTheTMSIsReported(t *testing.T) {
	r := newRig(t)
	id, cmd := uuid.New(), uuid.New()
	entry := tenant(id, 1, instanceapi.SharedTenantSuspended)
	entry.Commands = []instanceapi.SharedTenantCommand{
		archiveCommand(cmd, instanceapi.SharedArchiveMove),
	}
	r.cp.SetSharedTenants([]instanceapi.SharedTenant{entry})
	r.cp.SetSharedStorage(storage(writeTarget(id, cmd, true)))
	r.iterate(t)
	r.iterate(t)
	r.tms.FailCommand(cmd.String(), "a part could not be written")
	r.iterate(t)
	failures := r.reported(t, id).CommandFailures
	require.NotNil(t, failures)
	assert.Equal(t, "a part could not be written", (*failures)[0].Reason)
	r.iterate(t)
	assert.Equal(t, 1, r.tms.CommandStarts(cmd.String()))
}

func TestARestartedAgentDoesNotStartACommandTwice(t *testing.T) {
	r := newRig(t)
	id, archive, restore := uuid.New(), uuid.New(), uuid.New()
	prefix := "tenants/" + id.String() + "/nightly/2026-11-19/"
	entry := tenant(id, 1, instanceapi.SharedTenantSuspended)
	entry.Commands = []instanceapi.SharedTenantCommand{
		archiveCommand(archive, instanceapi.SharedArchiveFinal),
		restoreCommand(restore, "s3://"+archiveBucket+"/"+prefix),
	}
	r.cp.SetSharedTenants([]instanceapi.SharedTenant{entry})
	r.cp.SetSharedStorage(storage(
		writeTarget(id, archive, true),
		readTarget(id, restore, instanceapi.TargetRestore, prefix),
	))
	r.iterate(t)
	r.iterate(t)
	require.Equal(t, 1, r.tms.CommandStarts(archive.String()))
	assert.Zero(t, r.tms.CommandPosts(restore.String()), "commands run oldest first, one at a time")

	r.agent = r.build()
	r.iterate(t)
	r.iterate(t)
	assert.Equal(t, 1, r.tms.CommandStarts(archive.String()), "the TMS already holds it by id")
	assert.Equal(t, 1, r.tms.CommandPosts(archive.String()))
	assert.Zero(t, r.tms.CommandPosts(restore.String()))
}

func TestACommandWaitsWhileTheServerHasNoBucketOfItsOwn(t *testing.T) {
	r := newRig(t)
	id, archive, imported := uuid.New(), uuid.New(), uuid.New()
	newcomer := uuid.New()
	prefix := "tenants/" + newcomer.String() + "/archives/" + uuid.NewString() + "/"
	entry := tenant(id, 1, instanceapi.SharedTenantSuspended)
	entry.Commands = []instanceapi.SharedTenantCommand{
		archiveCommand(archive, instanceapi.SharedArchiveFinal),
	}
	made := tenant(newcomer, 1, instanceapi.SharedTenantSuspended)
	made.Commands = []instanceapi.SharedTenantCommand{
		importCommand(imported, "s3://"+archiveBucket+"/"+prefix),
	}
	r.cp.SetSharedTenants([]instanceapi.SharedTenant{entry, made})
	withoutNightly := storage(
		writeTarget(id, archive, true),
		readTarget(newcomer, imported, instanceapi.TargetImport, prefix),
	)
	withoutNightly.Nightly = nil
	r.cp.SetSharedStorage(withoutNightly)

	r.iterate(t)
	r.iterate(t)
	assert.Zero(t, r.tms.CommandPosts(archive.String()),
		"no command is started without the server's own bucket")
	assert.Zero(t, r.tms.TenantPuts(newcomer.String()),
		"a tenant made from an archive waits too")
	assert.Nil(t, r.reported(t, id).CommandFailures, "waiting is not a failure")

	r.cp.SetSharedStorage(storage(
		writeTarget(id, archive, true),
		readTarget(newcomer, imported, instanceapi.TargetImport, prefix),
	))
	r.cp.BumpStorageRevision()
	r.clock.Advance(time.Minute)
	r.iterate(t)
	r.iterate(t)
	r.iterate(t)
	assert.Equal(t, 1, r.tms.CommandStarts(archive.String()))
	assert.Equal(t, 1, r.tms.CommandStarts(imported.String()))
}

func TestATargetForAnotherCommandIsNeverUsed(t *testing.T) {
	r := newRig(t)
	id, cmd, other := uuid.New(), uuid.New(), uuid.New()
	entry := tenant(id, 1, instanceapi.SharedTenantSuspended)
	entry.Commands = []instanceapi.SharedTenantCommand{
		archiveCommand(cmd, instanceapi.SharedArchiveFinal),
	}
	r.cp.SetSharedTenants([]instanceapi.SharedTenant{entry})

	wrongKind := readTarget(id, cmd, instanceapi.TargetRestore,
		"tenants/"+id.String()+"/archives/"+cmd.String()+"/")
	wrongPrefix := writeTarget(id, cmd, true)
	wrongPrefix.Prefix = "tenants/" + id.String() + "/archives/" + other.String() + "/"
	otherTenant := writeTarget(uuid.New(), cmd, true)
	for _, target := range []instanceapi.SharedArchiveTarget{
		writeTarget(id, other, true), wrongKind, wrongPrefix, otherTenant,
	} {
		r.cp.SetSharedStorage(storage(target))
		r.clock.Advance(time.Minute)
		r.iterate(t)
		r.iterate(t)
		assert.Zero(t, r.tms.CommandPosts(cmd.String()))
		assert.Zero(t, r.tms.CommandPosts(other.String()))
	}
}

func TestSupportKeysAreDeliveredOnceAndAgainWhenTheyChange(t *testing.T) {
	r := newRig(t)
	support := &instanceapi.SharedSupport{
		StaffDomain: "trenova.app",
		Keys: []instanceapi.SupportKey{{
			Kid: "cp-1", Kty: "OKP", Crv: "Ed25519", Alg: "EdDSA", Use: "sig",
			X: "11qYAYKxCrfVS_7TyWQHOg7hcvPapiMlrwIaaPcHURo",
		}},
	}
	r.cp.SetSharedSupport(support)
	r.iterate(t)
	r.iterate(t)
	puts := r.tms.SupportKeyPuts()
	require.Len(t, puts, 1)
	assert.Empty(t, puts[0].InstanceID, "a shared server's keys name no instance")
	assert.Equal(t, "trenova.app", puts[0].StaffDomain)

	support.Keys = append(support.Keys, instanceapi.SupportKey{
		Kid: "cp-2", Kty: "OKP", Crv: "Ed25519", Alg: "EdDSA", Use: "sig",
		X: "pbcSVoRVgLNjqhYk7oC_UBN35UQTEvLIx4uYJhn8Ebo",
	})
	r.cp.SetSharedSupport(support)
	r.iterate(t)
	assert.Len(t, r.tms.SupportKeyPuts(), 2)

	r.cp.SetSharedSupport(nil)
	r.iterate(t)
	r.iterate(t)
	assert.Equal(t, 1, r.tms.SupportKeyDeletes(), "support turned off removes the keys once")
	assert.Nil(t, r.tms.SupportKeys())
}

func TestEachTenantsLiveGrantsAreReported(t *testing.T) {
	r := newRig(t)
	id, mixed := uuid.New(), uuid.New()
	r.cp.SetSharedTenants([]instanceapi.SharedTenant{
		tenant(id, 1, instanceapi.SharedTenantActive),
		tenant(mixed, 1, instanceapi.SharedTenantActive),
	})
	r.iterate(t)
	expires := time.Date(2026, 11, 21, 8, 0, 0, 0, time.UTC)
	grant := func(gid, org string) tms.SupportGrant {
		return tms.SupportGrant{
			ID: gid, OrganizationID: org, OrganizationName: "Acme Freight",
			StaffEmail: "Dana@Trenova.App", Scope: "read_only", ExpiresAt: expires,
		}
	}
	r.tms.SetTenantGrants(id.String(), []tms.SupportGrant{
		grant("sag_01", "org_01JAAAAAAAAAAAAAAAAAAAAAAA"),
	})
	r.tms.SetTenantGrants(mixed.String(), []tms.SupportGrant{
		grant("sag_02", "org_01JAAAAAAAAAAAAAAAAAAAAAAA"),
		grant("sag_03", "org_01JBBBBBBBBBBBBBBBBBBBBBBB"),
	})
	r.clock.Advance(time.Minute)
	r.iterate(t)
	grants := r.reported(t, id).SupportGrants
	require.NotNil(t, grants)
	require.Len(t, *grants, 1)
	assert.Equal(t, "dana@trenova.app", (*grants)[0].StaffEmail)
	assert.Equal(t, instanceapi.SupportGrantScope("read_only"), (*grants)[0].Scope)
	assert.Nil(t, r.reported(t, mixed).SupportGrants,
		"a list naming two organizations is never reported for one tenant")

	r.tms.SetTenantGrants(id.String(), []tms.SupportGrant{})
	r.clock.Advance(time.Minute)
	r.iterate(t)
	grants = r.reported(t, id).SupportGrants
	require.NotNil(t, grants, "an empty list says none is live")
	assert.Empty(t, *grants)
}

func TestNoSecretTheAgentHandlesReachesItsLogsOrItsStateDirectory(t *testing.T) {
	r := newRig(t)
	archived, exported, imported := uuid.New(), uuid.New(), uuid.New()
	finalCmd, exportCmd, importCmd := uuid.New(), uuid.New(), uuid.New()
	prefix := "tenants/" + imported.String() + "/archives/" + uuid.NewString() + "/"

	first := tenant(archived, 1, instanceapi.SharedTenantSuspended)
	first.Commands = []instanceapi.SharedTenantCommand{
		archiveCommand(finalCmd, instanceapi.SharedArchiveFinal),
	}
	second := tenant(exported, 1, instanceapi.SharedTenantActive)
	second.Commands = []instanceapi.SharedTenantCommand{
		archiveCommand(exportCmd, instanceapi.SharedArchiveExport),
	}
	third := tenant(imported, 1, instanceapi.SharedTenantSuspended)
	third.Commands = []instanceapi.SharedTenantCommand{
		importCommand(importCmd, "s3://"+archiveBucket+"/"+prefix),
	}
	r.cp.SetSharedTenants([]instanceapi.SharedTenant{first, second, third})
	r.cp.SetSharedStorage(storage(
		writeTarget(archived, finalCmd, true),
		writeTarget(exported, exportCmd, false),
		readTarget(imported, importCmd, instanceapi.TargetImport, prefix),
	))
	r.tms.RefuseCommand(exportCmd.String(), "refused for the test")
	for range 4 {
		r.iterate(t)
	}
	require.Equal(t, 1, r.tms.CommandStarts(finalCmd.String()))
	require.Equal(t, 1, r.tms.CommandStarts(importCmd.String()))
	require.NotEmpty(t, r.tms.NightlyGrants())

	secrets := []string{bucketSecret, sessionSecret, archiveKeyMaterial}
	logs := r.logs.String()
	require.NotEmpty(t, logs)
	for _, secret := range secrets {
		assert.NotContains(t, logs, secret, "a secret reached the agent's logs")
	}
	for _, report := range r.cp.SharedReports() {
		body, err := sonic.Marshal(report)
		require.NoError(t, err)
		for _, secret := range secrets {
			assert.NotContains(t, string(body), secret, "a secret reached a status report")
		}
	}
	require.NoError(t, filepath.WalkDir(r.state, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		body, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		for _, secret := range secrets {
			assert.NotContains(t, string(body), secret, "a secret reached %s", path)
		}
		return nil
	}))
}
