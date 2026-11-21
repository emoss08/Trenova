package tenantagent_test

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/tms"
)

const (
	foreignBucket   = "another-servers-archives"
	endpointRefusal = "the archive endpoint is not an object store this server may reach"
)

var rotatedKeyMaterial = base64.StdEncoding.EncodeToString(
	[]byte("fixture-rotated-key-32-bytes-abc"),
)

func rotated() *instanceapi.SharedNightlyTarget {
	n := nightlyTarget()
	n.ArchiveKeys = []instanceapi.ArchiveKey{
		{Id: "server-key-1", Key: archiveKeyMaterial, Current: false},
		{Id: "server-key-2", Key: rotatedKeyMaterial, Current: true},
	}
	return n
}

func storageComponent(t *testing.T, r *rig) instanceapi.ComponentStatus {
	t.Helper()
	report, ok := r.cp.LastSharedReport()
	require.True(t, ok)
	for _, c := range report.Components {
		if c.Name == "archive-storage" {
			return c
		}
	}
	t.Fatal("the last report names no archive-storage component")
	return instanceapi.ComponentStatus{}
}

func TestTheNightlyStorageIsSentOnceAndOnceMoreWhenItsKeysRotate(t *testing.T) {
	r := newRig(t)
	r.cp.SetSharedStorage(storage())
	for range 5 {
		r.iterate(t)
	}
	grants := r.tms.NightlyGrants()
	require.Len(t, grants, 1, "an unchanged nightly target is sent once")
	assert.Equal(t, archiveBucket, grants[0].Bucket)
	assert.Equal(t, "tenants/", grants[0].Prefix)
	require.Len(t, grants[0].ArchiveKeys, 1)
	assert.Equal(t, 1, r.cp.SharedStorageAsks())
	assert.True(t, storageComponent(t, r).Ready)

	rotatedStorage := storage()
	rotatedStorage.Nightly = rotated()
	r.cp.SetSharedStorage(rotatedStorage)
	for range 5 {
		r.iterate(t)
	}
	grants = r.tms.NightlyGrants()
	require.Len(t, grants, 2, "a rotation is sent once more")
	assert.Equal(
		t,
		2,
		r.cp.SharedStorageAsks(),
		"the revision changed once, so storage is read once",
	)
	held, ok := r.tms.HeldNightlyGrant()
	require.True(t, ok)
	require.Len(t, held.ArchiveKeys, 2)
	for _, k := range held.ArchiveKeys {
		assert.Equal(t, k.ID == "server-key-2", k.Current, k.ID)
	}
}

func TestAnUnchangedRevisionReadsNoStorageAcrossManyPolls(t *testing.T) {
	r := newRig(t)
	id, cmd := uuid.New(), uuid.New()
	entry := tenant(id, 1, instanceapi.SharedTenantSuspended)
	entry.Commands = []instanceapi.SharedTenantCommand{
		archiveCommand(cmd, instanceapi.SharedArchiveFinal),
	}
	r.cp.SetSharedTenants([]instanceapi.SharedTenant{entry})
	r.cp.SetSharedStorage(storage(writeTarget(id, cmd, true)))
	for i := range 300 {
		r.iterate(t)
		if i%20 == 0 {
			r.clock.Advance(time.Hour)
		}
	}
	assert.Equal(t, 1, r.cp.SharedStorageAsks(),
		"a listed command and the passing hours cause no read while the revision holds")
	assert.Equal(t, 1, r.tms.CommandStarts(cmd.String()))
	assert.Len(t, r.tms.NightlyGrants(), 1)

	r.cp.BumpStorageRevision()
	r.iterate(t)
	r.iterate(t)
	assert.Equal(t, 2, r.cp.SharedStorageAsks(), "a new revision is read once")
	assert.Len(t, r.tms.NightlyGrants(), 1, "an unchanged nightly target is not sent again")
}

func TestNightlyDisappearingRemovesTheStoredGrantOnce(t *testing.T) {
	r := newRig(t)
	r.cp.SetSharedStorage(storage())
	r.iterate(t)
	_, held := r.tms.HeldNightlyGrant()
	require.True(t, held)
	assert.Zero(t, r.tms.NightlyGrantDeletes())

	r.cp.SetSharedStorage(instanceapi.SharedStorage{
		Archives: []instanceapi.SharedArchiveTarget{},
		Waiting:  []instanceapi.SharedArchiveWaiting{},
	})
	for range 4 {
		r.iterate(t)
	}
	assert.Equal(t, 1, r.tms.NightlyGrantDeletes(), "the stored grant is removed once")
	_, held = r.tms.HeldNightlyGrant()
	assert.False(t, held, "nightly archives stop")
	component := storageComponent(t, r)
	assert.True(t, component.Ready, "no nightly target is the control plane's choice")
	require.NotNil(t, component.Message)
	assert.Contains(t, *component.Message, "nightly archives are off")

	r.cp.SetSharedStorage(storage())
	r.iterate(t)
	_, held = r.tms.HeldNightlyGrant()
	assert.True(t, held, "the target back is sent again")
	assert.Len(t, r.tms.NightlyGrants(), 2)
}

func TestARefusedCredentialOrUnknownKeyReadsStorageOnce(t *testing.T) {
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
	require.Equal(t, 1, r.cp.SharedStorageAsks())

	r.tms.FailCommandWithCode(cmd.String(), "failed", "a part could not be written")
	for range 3 {
		r.iterate(t)
	}
	assert.Equal(t, 1, r.cp.SharedStorageAsks(), "a failure for another cause reads nothing")

	r.tms.FailNightly(id.String(), uuid.NewString(), tms.FailureStorageRefused,
		"the object store refused the archive credential or bucket (AccessDenied)")
	for range 3 {
		r.iterate(t)
	}
	assert.Equal(t, 2, r.cp.SharedStorageAsks(), "a refused credential reads storage once")

	r.tms.FailNightly(id.String(), uuid.NewString(), tms.FailureKeyUnknown,
		"the archive was sealed with archive key server-key-0, which the command does not carry")
	for range 3 {
		r.iterate(t)
	}
	assert.Equal(t, 3, r.cp.SharedStorageAsks(), "a key the storage does not carry reads it once")

	r.agent = r.build()
	for range 3 {
		r.iterate(t)
	}
	assert.Equal(t, 4, r.cp.SharedStorageAsks(),
		"a restarted agent reads storage once and does not count old failures again")
}

func TestATargetInAnotherBucketOrOverHTTPIsNotUsed(t *testing.T) {
	r := newRig(t)
	id, cmd := uuid.New(), uuid.New()
	entry := tenant(id, 1, instanceapi.SharedTenantSuspended)
	entry.Commands = []instanceapi.SharedTenantCommand{
		archiveCommand(cmd, instanceapi.SharedArchiveFinal),
	}
	r.cp.SetSharedTenants([]instanceapi.SharedTenant{entry})

	foreign := writeTarget(id, cmd, true)
	foreign.Bucket = foreignBucket
	otherEndpoint := writeTarget(id, cmd, true)
	otherEndpoint.Endpoint = "https://s3.elsewhere.test"
	plain := writeTarget(id, cmd, true)
	plain.Endpoint = "http://s3.test"
	twoCurrent := writeTarget(id, cmd, true)
	twoCurrent.ArchiveKeys = &[]instanceapi.ArchiveKey{
		{Id: "server-key-1", Key: archiveKeyMaterial, Current: true},
		{Id: "server-key-2", Key: rotatedKeyMaterial, Current: true},
	}
	for _, target := range []instanceapi.SharedArchiveTarget{
		foreign, otherEndpoint, plain, twoCurrent,
	} {
		r.cp.SetSharedStorage(storage(target))
		r.iterate(t)
		r.iterate(t)
		assert.Zero(t, r.tms.CommandPosts(cmd.String()), target.Endpoint+" "+target.Bucket)
	}
	logs := r.logs.String()
	assert.Contains(t, logs, "another bucket than the server's own")
	assert.Contains(t, logs, "not an https URL")
	assert.NotContains(t, logs, bucketSecret)
	assert.NotContains(t, logs, sessionSecret)

	r.cp.SetSharedStorage(storage(writeTarget(id, cmd, true)))
	r.iterate(t)
	assert.Equal(t, 1, r.tms.CommandPosts(cmd.String()), "the server's own bucket is used")
}

func TestARestartedAgentSendsTheNightlyStorageOnce(t *testing.T) {
	r := newRig(t)
	r.cp.SetSharedStorage(storage())
	r.iterate(t)
	r.iterate(t)
	require.Len(t, r.tms.NightlyGrants(), 1)

	r.agent = r.build()
	for range 4 {
		r.iterate(t)
	}
	assert.Len(t, r.tms.NightlyGrants(), 2, "the digest lives in memory, so a restart sends once")
	assert.Equal(t, 2, r.cp.SharedStorageAsks())
}

func TestAFailedStorageReadKeepsWhatWasHeldAndIsRetried(t *testing.T) {
	r := newRig(t)
	id, cmd := uuid.New(), uuid.New()
	r.cp.SetSharedStorage(storage())
	r.iterate(t)
	require.Equal(t, 1, r.cp.SharedStorageAsks())

	entry := tenant(id, 1, instanceapi.SharedTenantSuspended)
	entry.Commands = []instanceapi.SharedTenantCommand{
		archiveCommand(cmd, instanceapi.SharedArchiveFinal),
	}
	r.cp.SetSharedTenants([]instanceapi.SharedTenant{entry})
	r.cp.SetSharedStorage(storage(writeTarget(id, cmd, true)))
	r.cp.FailSharedStorage(1)
	r.iterate(t)
	assert.Equal(t, 2, r.cp.SharedStorageAsks())
	assert.Zero(t, r.tms.NightlyGrantDeletes(), "a failed read never removes what is held")
	_, held := r.tms.HeldNightlyGrant()
	assert.True(t, held)

	for range 4 {
		r.iterate(t)
	}
	assert.Equal(t, 3, r.cp.SharedStorageAsks(), "the read is retried, and once it works, no more")
	assert.Equal(t, 1, r.tms.CommandStarts(cmd.String()))
}

func TestARefusedNightlyStorageIsAServerProblemLoggedOncePerChange(t *testing.T) {
	r := newRig(t)
	r.tms.RefuseNightlyGrant(endpointRefusal)
	r.cp.SetSharedStorage(storage())
	for range 5 {
		r.iterate(t)
	}
	_, held := r.tms.HeldNightlyGrant()
	assert.False(t, held)
	refusals := func() int {
		return strings.Count(r.logs.String(), "level=ERROR msg=\"nightly archives cannot use")
	}
	assert.Equal(t, 1, refusals(), "logged once, not on every poll")
	report, ok := r.cp.LastSharedReport()
	require.True(t, ok)
	assert.False(t, report.Ready)
	component := storageComponent(t, r)
	assert.False(t, component.Ready)
	require.NotNil(t, component.Message)
	assert.Contains(t, *component.Message, endpointRefusal)
	assert.NotContains(t, *component.Message, bucketSecret)

	changed := storage()
	changed.Nightly = rotated()
	r.cp.SetSharedStorage(changed)
	for range 3 {
		r.iterate(t)
	}
	assert.Equal(t, 2, refusals(), "a changed target is tried and logged once more")

	r.tms.RefuseNightlyGrant("")
	r.cp.SetSharedStorage(storage())
	r.iterate(t)
	_, held = r.tms.HeldNightlyGrant()
	assert.True(t, held)
	assert.True(t, storageComponent(t, r).Ready)
	report, _ = r.cp.LastSharedReport()
	assert.True(t, report.Ready)
}

func TestANightlyTargetOutOfShapeIsNeverSent(t *testing.T) {
	for name, change := range map[string]func(*instanceapi.SharedNightlyTarget){
		"a prefix other than tenants/": func(n *instanceapi.SharedNightlyTarget) {
			n.Prefix = "tenants/elsewhere/"
		},
		"an http endpoint": func(n *instanceapi.SharedNightlyTarget) {
			n.Endpoint = "http://s3.test"
		},
		"no current key": func(n *instanceapi.SharedNightlyTarget) {
			n.ArchiveKeys[0].Current = false
		},
	} {
		t.Run(name, func(t *testing.T) {
			r := newRig(t)
			s := storage()
			change(s.Nightly)
			r.cp.SetSharedStorage(s)
			for range 3 {
				r.iterate(t)
			}
			assert.Empty(t, r.tms.NightlyGrants())
			assert.False(t, storageComponent(t, r).Ready)
			assert.Equal(t, 1, strings.Count(r.logs.String(), "level=ERROR"))
		})
	}
}
