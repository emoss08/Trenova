package logs

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
)

func submitAndWait(t *testing.T, svc *Service, job *Job) Outcome {
	t.Helper()
	done := make(chan Outcome, 1)
	job.Done = func(o Outcome) { done <- o }
	require.Equal(t, Queued, svc.Submit(job))
	select {
	case o := <-done:
		return o
	case <-time.After(30 * time.Second):
		t.Fatal("the bundle never finished")
		return Outcome{}
	}
}

func dayJob() *Job {
	return &Job{BundleID: uuid.New(), Since: t0, Until: t0.Add(24 * time.Hour)}
}

func fill(src *fakeSource, n int) {
	for i := range n {
		src.add("api", t0.Add(time.Duration(i)*time.Second), "trenova-api-1", "api", fmt.Sprintf("request %d", i))
	}
}

func TestABundleIsStoredUnderItsKeyAndReported(t *testing.T) {
	t.Parallel()
	w := newTestWorld(t, DefaultCaps)
	fill(w.src, 10)
	w.start(t)
	job := dayJob()

	o := submitAndWait(t, w.svc, job)

	key := "support/logs/" + job.BundleID.String() + ".zip"
	require.Equal(t, instanceapi.LogBundleStored, o.Status, o.Message)
	assert.Equal(t, key, o.Key)
	obj, ok := w.objects.get(key)
	require.True(t, ok)
	assert.Equal(t, int64(len(obj.body)), o.Size)
	files := unzip(t, obj.body)
	assert.Len(t, lines(files["api.log"]), 10)
	assert.Contains(t, files, manifestName)
	assert.Equal(t, []instanceapi.LogBundleComponent{{Component: "api", Lines: 10}}, o.Components)
	assert.Equal(t,
		[]instanceapi.LogBundleUpdateStatus{instanceapi.LogBundleCollecting, instanceapi.LogBundleStored},
		w.cp.statuses(job.BundleID))
	last := w.cp.last(job.BundleID)
	assert.Equal(t, key, *last.ObjectKey)
	assert.Equal(t, o.Size, *last.SizeBytes)
	assert.Empty(t, scratchFiles(t, w.dir), "the scratch file is removed once uploaded")
}

func TestLokiFailingPartwayFailsTheBundleAndRemovesTheScratchFile(t *testing.T) {
	t.Parallel()
	w := newTestWorld(t, DefaultCaps)
	fill(w.src, 10)
	w.src.failAt = 1
	w.start(t)
	job := dayJob()

	o := submitAndWait(t, w.svc, job)

	assert.Equal(t, instanceapi.LogBundleFailed, o.Status)
	assert.Contains(t, o.Message, "could not collect the kept logs")
	assert.Equal(t, instanceapi.LogBundleFailed, w.cp.last(job.BundleID).Status)
	assert.Zero(t, w.objects.putCount())
	assert.Empty(t, scratchFiles(t, w.dir))
}

func TestStorageFailingFailsTheBundleAfterBoundedRetries(t *testing.T) {
	t.Parallel()
	w := newTestWorld(t, DefaultCaps)
	fill(w.src, 10)
	w.objects.failPut, w.objects.status = 10, http.StatusServiceUnavailable
	w.start(t)
	job := dayJob()

	o := submitAndWait(t, w.svc, job)

	assert.Equal(t, instanceapi.LogBundleFailed, o.Status)
	assert.Contains(t, o.Message, "could not store them")
	assert.Equal(t, uploadTries, w.objects.putCount())
	assert.Empty(t, scratchFiles(t, w.dir))
}

func TestAStorageRefusalIsNotRetried(t *testing.T) {
	t.Parallel()
	w := newTestWorld(t, DefaultCaps)
	fill(w.src, 3)
	w.objects.failPut, w.objects.status = 10, http.StatusForbidden
	w.start(t)

	o := submitAndWait(t, w.svc, dayJob())

	assert.Equal(t, instanceapi.LogBundleFailed, o.Status)
	assert.Equal(t, 1, w.objects.putCount())
}

func TestAnInstanceWithoutStorageSaysSoAndReadsNothing(t *testing.T) {
	t.Parallel()
	w := newTestWorld(t, DefaultCaps)
	fill(w.src, 3)
	w.objects.none = true
	w.start(t)

	o := submitAndWait(t, w.svc, dayJob())

	assert.Equal(t, instanceapi.LogBundleFailed, o.Status)
	assert.Equal(t, noStorageMessage, o.Message)
	assert.Zero(t, w.src.calls)
}

func TestABundleWithoutLogKeepingFails(t *testing.T) {
	t.Parallel()
	objects, cp, dir := newObjects(), newCP(), t.TempDir()
	svc := newService(t, nil, objects, cp, noTMS{}, DefaultCaps, dir)
	w := &world{objects: objects, cp: cp, dir: dir, svc: svc}
	w.start(t)
	job := dayJob()

	o := submitAndWait(t, svc, job)

	assert.False(t, svc.Kept())
	assert.Equal(t, instanceapi.LogBundleFailed, o.Status)
	assert.Equal(t, NotKeptMessage, o.Message)
	assert.Equal(t, instanceapi.LogBundleFailed, cp.last(job.BundleID).Status)
}

func TestABundleIsNeverUploadedTwice(t *testing.T) {
	t.Parallel()
	w := newTestWorld(t, DefaultCaps)
	fill(w.src, 10)
	w.start(t)
	job := dayJob()
	first := submitAndWait(t, w.svc, job)
	require.Equal(t, instanceapi.LogBundleStored, first.Status)
	calls := w.src.calls

	again := make(chan Outcome, 1)
	repeat := *job
	repeat.Done = func(o Outcome) { again <- o }
	assert.Equal(t, AlreadyRunning, w.svc.Submit(&repeat), "the same process answers from memory")
	assert.Equal(t, first, <-again)

	restarted := newService(t, w.src, w.objects, w.cp, noTMS{}, DefaultCaps, w.dir)
	rw := &world{svc: restarted}
	rw.start(t)
	afterRestart := submitAndWait(t, restarted, &Job{BundleID: job.BundleID, Since: job.Since, Until: job.Until})

	assert.Equal(t, instanceapi.LogBundleStored, afterRestart.Status)
	assert.Equal(t, first.Size, afterRestart.Size)
	assert.Equal(t, first.Components, afterRestart.Components, "read back from the object's metadata")
	assert.Equal(t, 1, w.objects.putCount())
	assert.Equal(t, calls, w.src.calls, "nothing was read again")
}

func TestTheSameBundleIsQueuedOnce(t *testing.T) {
	t.Parallel()
	w := newTestWorld(t, DefaultCaps)
	job := dayJob()

	assert.Equal(t, Queued, w.svc.Submit(job))
	assert.Equal(t, AlreadyRunning, w.svc.Submit(job))
	for range queueSize - 1 {
		require.Equal(t, Queued, w.svc.Submit(dayJob()))
	}
	assert.Equal(t, QueueFull, w.svc.Submit(dayJob()))
}

func TestTheLastLinesComeFromKeptLogsNewestLastWithinTheirCaps(t *testing.T) {
	t.Parallel()
	w := newTestWorld(t, DefaultCaps)
	now := t0.Add(2 * time.Hour)
	w.svc.now = func() time.Time { return now }
	for i := range 10 {
		w.src.add("api", t0.Add(time.Duration(i)*time.Minute), "trenova-api-1", "api", fmt.Sprintf("request %d", i))
	}

	out, err := w.svc.Last(t.Context(), "api", 3, func(s string) string { return s })

	require.NoError(t, err)
	assert.Equal(t,
		"trenova-api-1/api: 2026-11-14T00:07:00Z request 7\n"+
			"trenova-api-1/api: 2026-11-14T00:08:00Z request 8\n"+
			"trenova-api-1/api: 2026-11-14T00:09:00Z request 9\n",
		out)

	padding := strings.Repeat("x", 300)
	for i := range 2000 {
		w.src.add("worker", t0.Add(time.Duration(i)*time.Second), "trenova-worker-1", "worker",
			fmt.Sprintf("%04d %s", i, padding))
	}
	out, err = w.svc.Last(t.Context(), "worker", 5000, func(s string) string { return s })
	require.NoError(t, err)
	assert.LessOrEqual(t, len(out), maxViewBytes)
	assert.True(t, strings.HasSuffix(out, fmt.Sprintf("1999 %s\n", padding)), "the newest line is kept")
}
