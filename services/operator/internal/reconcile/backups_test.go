package reconcile_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chartutil"
	kubefake "helm.sh/helm/v3/pkg/kube/fake"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/emoss08/trenova/operator/internal/backup"
	"github.com/emoss08/trenova/operator/internal/commands"
	"github.com/emoss08/trenova/operator/internal/controlplane"
	"github.com/emoss08/trenova/operator/internal/controlplanetest"
	"github.com/emoss08/trenova/operator/internal/credential"
	"github.com/emoss08/trenova/operator/internal/credential/secretstore"
	"github.com/emoss08/trenova/operator/internal/escrow/escrowsecret"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/reconcile"
	"github.com/emoss08/trenova/operator/internal/release"
	"github.com/emoss08/trenova/operator/internal/selfupgrade"
	"github.com/emoss08/trenova/operator/internal/values"
)

var bucketStorage = values.Storage{
	Endpoint: "https://s3.bhs.io.cloud.ovh.net", Region: "bhs", DocumentsBucket: "trenova-x-docs",
	BackupsPath: "s3://trenova-x-backups/pitr/",
}

type fakeStorage struct {
	mu        sync.Mutex
	hostnames [][]string
	revisions []int64
	err       error
}

func (f *fakeStorage) Ensure(
	_ context.Context,
	hostnames []string,
	revision int64,
) (values.Storage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.hostnames = append(f.hostnames, append([]string(nil), hostnames...))
	f.revisions = append(f.revisions, revision)
	if f.err != nil {
		return values.Storage{}, f.err
	}
	return bucketStorage, nil
}

type fakeBackups struct {
	mu     sync.Mutex
	inputs []backup.Input
	result backup.Result
}

func (f *fakeBackups) Run(_ context.Context, in *backup.Input) backup.Result {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.inputs = append(f.inputs, *in)
	return f.result
}

func (f *fakeBackups) last() backup.Input {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.inputs[len(f.inputs)-1]
}

type backupWorld struct {
	cp       *controlplanetest.Server
	loop     *reconcile.Loop
	health   *fakeHealth
	releases *countingReleases
	storage  *fakeStorage
	backups  *fakeBackups
	now      time.Time
}

func (w *backupWorld) set(result backup.Result) {
	w.backups.mu.Lock()
	defer w.backups.mu.Unlock()
	w.backups.result = result
}

func newBackupWorld(t *testing.T, spec instanceapi.InstanceSpec) *backupWorld {
	t.Helper()
	return newBackupWorldWith(t, spec, noDatabases{})
}

func newBackupWorldWith(
	t *testing.T,
	spec instanceapi.InstanceSpec,
	dbs reconcile.Databases,
) *backupWorld {
	t.Helper()
	cp := controlplanetest.New(t, spec)
	kube := fake.NewClientset()
	body, err := sonic.Marshal(cp.Bundle(7 * 24 * time.Hour))
	require.NoError(t, err)
	_, err = kube.CoreV1().Secrets("trenova-system").Create(t.Context(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "enrolment", Namespace: "trenova-system"},
		Data:       map[string][]byte{credential.BundleKey: body},
	}, metav1.CreateOptions{})
	require.NoError(t, err)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	transport := &credential.Transport{}
	client, err := controlplane.New(
		cp.URL,
		&http.Client{Transport: transport, Timeout: 10 * time.Second},
	)
	require.NoError(t, err)
	creds := credential.NewManager(&credential.ManagerParams{
		Store: secretstore.New(secretstore.Params{
			Client: kube, Namespace: "trenova-system", Enrolment: "enrolment", Identity: "identity",
		}),
		Transport: transport,
		Renewer:   client,
		Logger:    logger,
	})
	require.NoError(t, creds.Start(t.Context()))
	caps := chartutil.DefaultCapabilities.Copy()
	caps.KubeVersion = chartutil.KubeVersion{Version: "v1.31.5", Major: "1", Minor: "31"}
	w := &backupWorld{
		cp:      cp,
		health:  &fakeHealth{},
		storage: &fakeStorage{},
		backups: &fakeBackups{},
		now:     time.Now(),
		releases: &countingReleases{Releaser: release.NewReleaser(release.ReleaserParams{
			Config: &action.Configuration{
				Releases:     storage.Init(driver.NewMemory()),
				KubeClient:   &kubefake.PrintingKubeClient{Out: io.Discard},
				Capabilities: caps,
				Log:          func(string, ...any) {},
			},
			Kube: kube, Name: "trenova", Namespace: "trenova", Logger: logger,
		})},
	}
	kept := &fakeLogKeeping{}
	runner, err := commands.New(&commands.Params{
		ControlPlane: client, Kube: kube, Grants: noGrants{}, Logs: kept,
		Namespace: "trenova", Release: "trenova", Chart: "trenova", Logger: logger,
	})
	require.NoError(t, err)
	w.loop, err = reconcile.New(&reconcile.Params{
		ControlPlane: client,
		Credentials:  creds,
		Charts:       release.NewSource(release.SourceParams{Path: chartDir}),
		Releases:     w.releases,
		Health:       w.health,
		Escrow: escrowsecret.New(&escrowsecret.Params{
			Client: kube, ControlPlane: client, Namespace: "trenova", Name: values.EscrowSecret, Logger: logger,
		}),
		Usage:          &fakeUsage{now: time.Now},
		Administrators: &fakeAdmins{},
		Features:       &fakeFeatures{},
		SystemEmail:    &fakeSystemEmail{},
		SupportKeys:    &fakeSupportKeys{},
		SupportGrants:  &fakeSupportGrants{},
		Storage:        w.storage,
		Backups:        w.backups,
		Databases:      dbs,
		Commands:       runner,
		EmailEvents:    &fakeEmailEvents{},
		Logs:           kept,
		Telemetry:      &fakeTelemetry{},
		Operator: &fakeOperator{status: selfupgrade.Status{
			Version: "0.11.0", Phase: selfupgrade.PhaseCurrent,
		}},
		Base:      map[string]any{"clamav": map[string]any{"enabled": true}},
		Namespace: "trenova-system",
		Logger:    logger,
		Now:       func() time.Time { return w.now },
		Poll:      15 * time.Second,
		Report:    time.Minute,
	})
	require.NoError(t, err)
	return w
}

func TestEachRevisionAsksForTheInstancesStorage(t *testing.T) {
	t.Parallel()
	spec := controlplanetest.Spec(uuid.New())
	w := newBackupWorld(t, spec)
	w.health.set(true)
	w.loop.Tick(t.Context())

	spec.Plan.Vcpu, spec.Plan.MemoryGiB = 16, 64
	w.cp.SetSpec(spec)
	w.loop.Tick(t.Context())

	w.storage.mu.Lock()
	defer w.storage.mu.Unlock()
	require.Equal(t, []int64{1, 2}, w.storage.revisions)
}

func TestTheBucketAdmitsTheCustomHostnameOnlyWhenTheSpecCarriesAValidOne(t *testing.T) {
	t.Parallel()
	spec := controlplanetest.Spec(uuid.New())
	custom := "tms.acme-freight.com"
	spec.CustomHostname = &custom
	w := newBackupWorld(t, spec)
	w.health.set(true)
	w.loop.Tick(t.Context())

	invalid := "Not A Host"
	spec.CustomHostname = &invalid
	w.cp.SetSpec(spec)
	w.loop.Tick(t.Context())

	spec.CustomHostname = nil
	w.cp.SetSpec(spec)
	w.loop.Tick(t.Context())

	w.storage.mu.Lock()
	defer w.storage.mu.Unlock()
	require.Equal(t, [][]string{
		{spec.Hostname, custom},
		{spec.Hostname},
		{spec.Hostname},
	}, w.storage.hostnames)
}

func TestTheReportCarriesBackupHealthAndTheRestore(t *testing.T) {
	t.Parallel()
	spec := controlplanetest.Spec(uuid.New())
	restore := uuid.New()
	spec.Restore = &instanceapi.SpecRestore{
		Id:         restore,
		Generation: "r1a2b3c4d",
		TargetTime: time.Now().Add(-time.Hour),
	}
	w := newBackupWorld(t, spec)
	last := time.Now().Add(-time.Hour).UTC().Truncate(time.Second)
	w.backups.result = backup.Result{
		Health: &instanceapi.BackupHealth{
			WalArchiving: instanceapi.BackupHealthWalArchiving("healthy"), LastBaseBackupAt: &last,
		},
		Restore: &instanceapi.RestoreProgress{
			Id:    restore,
			Phase: instanceapi.RestoreProgressPhase("ready"),
		},
	}
	w.health.set(true)
	w.loop.Tick(t.Context())

	require.Equal(t, 1, w.releases.applies)
	require.Equal(t, [][]string{{spec.Hostname}}, w.storage.hostnames)
	in := w.backups.last()
	require.Equal(t, bucketStorage, in.Storage)
	require.True(t, in.Applied)
	r := w.cp.Reports()[len(w.cp.Reports())-1]
	require.NotNil(t, r.Backup)
	require.Equal(t, instanceapi.BackupHealthWalArchiving("healthy"), r.Backup.WalArchiving)
	require.NotNil(t, r.Restore)
	require.Equal(t, instanceapi.RestoreProgressPhase("ready"), r.Restore.Phase)
	require.Nil(t, r.FinalBackup)
}

func TestTheFinalBackupIsReportedWithTheDecommissionedRevision(t *testing.T) {
	t.Parallel()
	spec := controlplanetest.Spec(uuid.New())
	spec.Lifecycle = instanceapi.Decommissioned
	w := newBackupWorld(t, spec)
	w.backups.result = backup.Result{
		Health: &instanceapi.BackupHealth{
			WalArchiving: instanceapi.BackupHealthWalArchiving("healthy"),
		},
		Final: &instanceapi.FinalBackup{Id: "20261114T120500", CompletedAt: time.Now().UTC()},
	}
	w.health.set(true)
	w.loop.Tick(t.Context())
	r := w.cp.Reports()[len(w.cp.Reports())-1]
	require.NotNil(t, r.FinalBackup)
	require.Equal(t, "20261114T120500", r.FinalBackup.Id)
	require.Equal(t, w.backups.last().Revision, *r.AppliedRevision)
}

func TestNothingIsAppliedWithoutTheInstancesStorage(t *testing.T) {
	t.Parallel()
	w := newBackupWorld(t, controlplanetest.Spec(uuid.New()))
	w.storage.err = errors.New("the control plane is unreachable")
	w.loop.Tick(t.Context())
	require.Equal(t, 0, w.releases.applies)
	r := w.cp.Reports()[len(w.cp.Reports())-1]
	require.False(t, r.Ready)
	require.Contains(t, *r.Message, "nothing is applied without the instance's storage")
}

func listResult(list ...instanceapi.BackupEntry) backup.Result {
	if list == nil {
		list = []instanceapi.BackupEntry{}
	}
	return backup.Result{
		Health: &instanceapi.BackupHealth{
			WalArchiving: instanceapi.BackupHealthWalArchiving("healthy"),
		},
		Backups: &list,
	}
}

func dailyBackup(phase instanceapi.BackupEntryPhase, started time.Time) instanceapi.BackupEntry {
	e := instanceapi.BackupEntry{
		Id: "20261114T030000", Phase: phase, StartedAt: started.UTC().Truncate(time.Second),
		Origin: new(instanceapi.Scheduled),
	}
	if phase == instanceapi.BackupEntryPhaseCompleted {
		done := e.StartedAt.Add(4 * time.Minute)
		e.CompletedAt = &done
	}
	return e
}

func TestTheBackupListIsSentFirstAndThenOnlyWhenItChanges(t *testing.T) {
	t.Parallel()
	w := newBackupWorld(t, controlplanetest.Spec(uuid.New()))
	started := w.now.Add(-time.Hour)
	w.set(listResult(dailyBackup(instanceapi.BackupEntryPhaseRunning, started)))
	w.health.set(true)

	w.loop.Tick(t.Context())
	held, lists := w.cp.Backups()
	require.Equal(t, 1, lists)
	require.Equal(
		t,
		[]instanceapi.BackupEntry{dailyBackup(instanceapi.BackupEntryPhaseRunning, started)},
		held,
	)

	// Unchanged: the minute's report goes out without the list.
	w.now = w.now.Add(2 * time.Minute)
	reports := len(w.cp.Reports())
	w.loop.Tick(t.Context())
	require.Len(t, w.cp.Reports(), reports+1)
	r := w.cp.Reports()[len(w.cp.Reports())-1]
	require.NotNil(t, r.Backup)
	require.Nil(t, r.Backup.Backups)
	_, lists = w.cp.Backups()
	require.Equal(t, 1, lists)

	// It completes: sent at once, with no minute passing.
	w.set(listResult(dailyBackup(instanceapi.BackupEntryPhaseCompleted, started)))
	w.loop.Tick(t.Context())
	held, lists = w.cp.Backups()
	require.Equal(t, 2, lists)
	require.Equal(t, instanceapi.BackupEntryPhaseCompleted, held[0].Phase)

	// Retention removes it: the empty list is sent.
	w.set(listResult())
	w.loop.Tick(t.Context())
	held, lists = w.cp.Backups()
	require.Equal(t, 3, lists)
	require.Empty(t, held)
}

func TestABackupListOnARefusedReportIsSentAgain(t *testing.T) {
	t.Parallel()
	w := newBackupWorld(t, controlplanetest.Spec(uuid.New()))
	w.set(listResult(dailyBackup(instanceapi.BackupEntryPhaseCompleted, w.now.Add(-time.Hour))))
	w.health.set(true)
	w.cp.FailStatusNext(1)

	w.loop.Tick(t.Context())
	_, lists := w.cp.Backups()
	require.Equal(t, 0, lists)

	w.loop.Tick(t.Context())
	held, lists := w.cp.Backups()
	require.Equal(t, 1, lists)
	require.Len(t, held, 1)
}

func TestAFailedBackupListReadLeavesTheRestOfTheReport(t *testing.T) {
	t.Parallel()
	w := newBackupWorld(t, controlplanetest.Spec(uuid.New()))
	list := listResult(dailyBackup(instanceapi.BackupEntryPhaseCompleted, w.now.Add(-time.Hour)))
	w.set(list)
	w.health.set(true)
	w.loop.Tick(t.Context())

	last := w.now.Add(-time.Hour).UTC().Truncate(time.Second)
	w.set(backup.Result{Health: &instanceapi.BackupHealth{
		WalArchiving: instanceapi.BackupHealthWalArchiving("healthy"), LastBaseBackupAt: &last,
	}})
	w.now = w.now.Add(2 * time.Minute)
	w.loop.Tick(t.Context())
	r := w.cp.Reports()[len(w.cp.Reports())-1]
	require.True(t, r.Ready)
	require.NotNil(t, r.Backup)
	require.Equal(t, instanceapi.BackupHealthWalArchiving("healthy"), r.Backup.WalArchiving)
	require.Equal(t, last, *r.Backup.LastBaseBackupAt)
	require.Nil(t, r.Backup.Backups)
	_, lists := w.cp.Backups()
	require.Equal(t, 1, lists)

	// Read again and unchanged: the control plane already has it.
	w.set(list)
	w.now = w.now.Add(2 * time.Minute)
	w.loop.Tick(t.Context())
	_, lists = w.cp.Backups()
	require.Equal(t, 1, lists)
}
