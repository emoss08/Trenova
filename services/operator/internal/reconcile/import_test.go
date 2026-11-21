package reconcile_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/emoss08/trenova/operator/internal/controlplanetest"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/reconcile"
	"github.com/emoss08/trenova/operator/internal/values"
)

const importKeyFixture = "Zml4dHVyZS1hcmNoaXZlLWtleS0zMi1ieXRlcy14eXo="

var importStorage = values.Storage{
	Endpoint:        "https://s3.test",
	Region:          "bhs",
	DocumentsBucket: "trenova-test-docs",
	BackupsPath:     "s3://trenova-test-backups/pitr/",
}

type fixedStorage struct{}

func (fixedStorage) Ensure(context.Context, []string, int64) (values.Storage, error) {
	return importStorage, nil
}

func heldImportSpec() instanceapi.InstanceSpec {
	spec := controlplanetest.Spec(uuid.New())
	spec.Lifecycle = instanceapi.Suspended
	id := uuid.New()
	spec.Import = &instanceapi.SpecImport{
		Id:           id,
		Bucket:       "trenova-test-backups",
		Prefix:       "imports/" + id.String() + "/",
		ExpectSha256: strings.Repeat("ab", 32),
		ArchiveKeys: []instanceapi.ArchiveKey{
			{Id: "region-1", Key: importKeyFixture, Current: true},
		},
	}
	return spec
}

func importWorld(t *testing.T, spec instanceapi.InstanceSpec) *world {
	t.Helper()
	return newWorldWith(t, spec, nil, func(p *reconcile.Params) { p.Storage = fixedStorage{} })
}

func (w *world) releaseJob(t *testing.T, id string) *batchv1.Job {
	t.Helper()
	deployed, err := w.releases.Deployed()
	require.NoError(t, err)
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        deployed.ReleaseJob,
			Namespace:   "trenova",
			Annotations: map[string]string{"trenova.app/import": id},
		},
		Status: batchv1.JobStatus{StartTime: &metav1.Time{Time: time.Now().UTC()}},
	}
	created, err := w.kube.BatchV1().
		Jobs("trenova").
		Create(t.Context(), job, metav1.CreateOptions{})
	require.NoError(t, err)
	return created
}

func (w *world) finishJob(t *testing.T, job *batchv1.Job, exit int32, message string) {
	t.Helper()
	condition := batchv1.JobComplete
	if exit != 0 {
		condition = batchv1.JobFailed
	}
	job.Status.Conditions = []batchv1.JobCondition{{Type: condition, Status: corev1.ConditionTrue}}
	_, err := w.kube.BatchV1().
		Jobs("trenova").
		UpdateStatus(t.Context(), job, metav1.UpdateOptions{})
	require.NoError(t, err)
	_, err = w.kube.CoreV1().Pods("trenova").Create(t.Context(), &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: job.Name + "-x", Namespace: "trenova", Labels: map[string]string{"job-name": job.Name},
		},
		Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{
			Name: "import",
			State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{
				ExitCode: exit, Message: message, FinishedAt: metav1.Now(),
			}},
		}}},
	}, metav1.CreateOptions{})
	require.NoError(t, err)
}

func statusLine(t *testing.T, id, state, reason string) string {
	t.Helper()
	body, err := sonic.Marshal(map[string]string{"id": id, "state": state, "reason": reason})
	require.NoError(t, err)
	return `{"id":"` + id + `","state":"importing"}` + "\n" + string(body) + "\n"
}

func (w *world) keysHeld(t *testing.T) bool {
	t.Helper()
	_, err := w.kube.CoreV1().Secrets("trenova").
		Get(t.Context(), values.ImportKeysSecret("trenova"), metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return false
	}
	require.NoError(t, err)
	return true
}

func TestAHeldInstanceImportsThenStartsWhenTheSwitchArrives(t *testing.T) {
	t.Parallel()
	spec := heldImportSpec()
	w := importWorld(t, spec)
	id := spec.Import.Id.String()
	w.cp.UnavailableNext(1)

	w.loop.Tick(t.Context())
	assert.Zero(t, w.releases.applies, "a spec answered 503 installs nothing")
	w.now = w.now.Add(2 * time.Second)

	w.loop.Tick(t.Context())
	require.Equal(t, 1, w.releases.applies)
	deployed, err := w.releases.Deployed()
	require.NoError(t, err)
	assert.Equal(t, true, deployed.Config["stopped"], "no workload starts while held")
	pg, _ := deployed.Config["postgres"].(map[string]any)
	assert.Equal(t, false, pg["hibernate"], "the database runs for the import")
	imp, _ := deployed.Config["import"].(map[string]any)
	assert.Equal(t, id, imp["id"])
	stored, err := sonic.Marshal(deployed.Config)
	require.NoError(t, err)
	assert.NotContains(t, string(stored), importKeyFixture, "Helm stores no key")
	require.True(t, w.keysHeld(t), "the keys wait in their Secret for the Job")
	assert.Nil(t, w.lastReport(t).Import, "nothing is reported before the Job starts")

	job := w.releaseJob(t, id)
	w.now = w.now.Add(time.Minute)
	w.loop.Tick(t.Context())
	report := w.lastReport(t)
	require.NotNil(t, report.Import)
	assert.Equal(t, instanceapi.ImportImporting, report.Import.State)
	assert.Equal(t, spec.Import.Id, report.Import.Id)

	w.finishJob(t, job, 0, statusLine(t, id, "done", ""))
	w.now = w.now.Add(time.Minute)
	w.loop.Tick(t.Context())
	report = w.lastReport(t)
	require.NotNil(t, report.Import)
	assert.Equal(t, instanceapi.ImportDone, report.Import.State)
	assert.Equal(t, spec.Import.Id, report.Import.Id, "done names the import it finished")
	assert.False(t, w.keysHeld(t), "the keys are deleted once the import settles")

	w.now = w.now.Add(time.Minute)
	w.loop.Tick(t.Context())
	deployed, err = w.releases.Deployed()
	require.NoError(t, err)
	imp, _ = deployed.Config["import"].(map[string]any)
	assert.Equal(t, true, imp["settled"], "a finished import is never run again")
	assert.Equal(t, instanceapi.ImportDone, w.lastReport(t).Import.State)

	switched := spec
	switched.Lifecycle = instanceapi.Active
	switched.Import = nil
	switched.FirstAdministrator.Invite = &instanceapi.SpecAdministratorInvite{
		TokenSha256: strings.Repeat("cd", 32), ExpiresAt: time.Now().Add(time.Hour),
	}
	w.cp.SetSpec(switched)
	w.health.set(true)
	w.now = w.now.Add(time.Minute)
	w.loop.Tick(t.Context())
	w.now = w.now.Add(time.Minute)
	w.loop.Tick(t.Context())
	deployed, err = w.releases.Deployed()
	require.NoError(t, err)
	assert.Equal(t, false, deployed.Config["stopped"], "the workloads start with the switch")
	boot, _ := deployed.Config["bootstrap"].(map[string]any)
	assert.Empty(t, boot["adminEmail"], "an imported instance is never bootstrapped")
	assert.Nil(t, w.lastReport(t).Import, "the import is reported only while the spec carries it")
	assert.Empty(t, w.admins.invites, "no first administrator is invited to an imported instance")
}

func TestARefusedOrFailedImportIsReportedWithItsReasonAndNotRunAgain(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		exit   int32
		reason string
		prefix string
	}{
		{"refused", 2, "the archive's manifest does not match", "The archive was refused: "},
		{"failed", 1, "the bucket could not be read", "The import could not complete: "},
	} {
		spec := heldImportSpec()
		w := importWorld(t, spec)
		id := spec.Import.Id.String()
		w.loop.Tick(t.Context())
		job := w.releaseJob(t, id)
		w.finishJob(t, job, tc.exit, statusLine(t, id, "refused", tc.reason))
		w.now = w.now.Add(time.Minute)
		w.loop.Tick(t.Context())
		report := w.lastReport(t)
		require.NotNil(t, report.Import, tc.name)
		assert.Equal(t, instanceapi.ImportRefused, report.Import.State, tc.name)
		require.NotNil(t, report.Import.Reason, tc.name)
		assert.Equal(t, tc.prefix+tc.reason, *report.Import.Reason, tc.name)
		assert.False(t, w.keysHeld(t), tc.name)

		w.now = w.now.Add(time.Minute)
		w.loop.Tick(t.Context())
		deployed, err := w.releases.Deployed()
		require.NoError(t, err)
		imp, _ := deployed.Config["import"].(map[string]any)
		assert.Equal(t, true, imp["settled"], "%s: never run again under the same id", tc.name)

		retry := spec
		next := uuid.New()
		retry.Import = &instanceapi.SpecImport{
			Id: next, Bucket: spec.Import.Bucket, Prefix: "imports/" + next.String() + "/",
			ExpectSha256: spec.Import.ExpectSha256, ArchiveKeys: spec.Import.ArchiveKeys,
		}
		w.cp.SetSpec(retry)
		w.now = w.now.Add(time.Minute)
		w.loop.Tick(t.Context())
		deployed, err = w.releases.Deployed()
		require.NoError(t, err)
		imp, _ = deployed.Config["import"].(map[string]any)
		assert.Equal(t, next.String(), imp["id"], "%s: a new id runs again", tc.name)
		assert.Equal(t, false, imp["settled"], tc.name)
		assert.True(t, w.keysHeld(t), tc.name)
	}
}
