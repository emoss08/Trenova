package commands

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/logs"
	"github.com/emoss08/trenova/operator/internal/tms"
)

type pull struct {
	email string
	kind  tms.LogPullKind
}

type fakeGrants struct {
	mu    sync.Mutex
	live  map[string]bool
	err   error
	pulls []pull
}

func (f *fakeGrants) AuthorizeLogPull(_ context.Context, email string, kind tms.LogPullKind) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pulls = append(f.pulls, pull{email: email, kind: kind})
	if f.err != nil {
		return f.err
	}
	if !f.live[email] {
		return tms.ErrNoGrant
	}
	return nil
}

type fakeLogs struct {
	mu        sync.Mutex
	kept      bool
	full      bool
	lastCalls int
	redacted  string
	jobs      []*logs.Job
	finished  map[uuid.UUID]logs.Outcome
}

func (f *fakeLogs) Kept() bool { return f.kept }

func (f *fakeLogs) Longest() time.Duration { return logs.DefaultCaps.Period }

func (f *fakeLogs) Last(_ context.Context, component string, lines int, redact func(string) string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastCalls++
	f.redacted = redact("password=hunter2hunter2")
	return component + " kept lines\n", nil
}

func (f *fakeLogs) Submit(job *logs.Job) logs.Submitted {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.full {
		return logs.QueueFull
	}
	for _, j := range f.jobs {
		if j.BundleID == job.BundleID {
			return logs.AlreadyRunning
		}
	}
	f.jobs = append(f.jobs, job)
	return logs.Queued
}

func (f *fakeLogs) Finish(_ context.Context, id uuid.UUID, outcome logs.Outcome) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.finished == nil {
		f.finished = map[uuid.UUID]logs.Outcome{}
	}
	f.finished[id] = outcome
}

func apiPod() *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "trenova-api-1", Namespace: ns, Labels: labelsFor("api")},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "api"}}},
	}
}

func bundleCommand(by *instanceapi.RequestedBy) instanceapi.Command {
	id := uuid.New()
	since := time.Date(2026, 11, 11, 9, 0, 0, 0, time.UTC)
	until := since.Add(24 * time.Hour)
	return instanceapi.Command{
		Id: uuid.New(), Kind: instanceapi.CollectLogBundle, BundleId: &id, Since: &since, Until: &until,
		RequestedBy: by, IssuedAt: time.Now(),
	}
}

func TestStaffLinesWithoutAGrantAreRefusedAndNothingIsRead(t *testing.T) {
	t.Parallel()
	cmd := collect("api")
	cmd.RequestedBy = staff("sam@trenova.app")
	cp := &fakeControlPlane{commands: []instanceapi.Command{cmd}}
	grants, kept := &fakeGrants{}, &fakeLogs{kept: true}
	r, _ := runnerWith(t, cp, grants, kept, apiPod())

	require.NoError(t, r.Run(t.Context()))

	got := cp.answers[cmd.Id][0]
	assert.Equal(t, instanceapi.CommandResultStatusRefused, got.Status)
	assert.Contains(t, *got.Message, "support access")
	assert.Nil(t, got.Output)
	assert.Zero(t, kept.lastCalls)
	assert.Equal(t, []pull{{email: "sam@trenova.app", kind: tms.LogPullLines}}, grants.pulls)
}

func TestStaffLinesWithAGrantAreCollected(t *testing.T) {
	t.Parallel()
	cmd := collect("api")
	cmd.RequestedBy = staff("sam@trenova.app")
	cp := &fakeControlPlane{commands: []instanceapi.Command{cmd}}
	grants := &fakeGrants{live: map[string]bool{"sam@trenova.app": true}}
	r, _ := runnerWith(t, cp, grants, &fakeLogs{}, apiPod())

	require.NoError(t, r.Run(t.Context()))

	got := cp.answers[cmd.Id][0]
	require.Equal(t, instanceapi.CommandResultStatusSucceeded, got.Status)
	assert.Equal(t, "trenova-api-1/api: fake logs\n", *got.Output)
	assert.Len(t, grants.pulls, 1)
}

func TestNothingIsCollectedWhenTheGrantCannotBeChecked(t *testing.T) {
	t.Parallel()
	cmd := collect("api")
	cmd.RequestedBy = staff("sam@trenova.app")
	cp := &fakeControlPlane{commands: []instanceapi.Command{cmd}}
	kept := &fakeLogs{kept: true}
	r, _ := runnerWith(t, cp, &fakeGrants{err: errors.New("dial tcp: connection refused")}, kept, apiPod())

	require.NoError(t, r.Run(t.Context()))

	got := cp.answers[cmd.Id][0]
	assert.Equal(t, instanceapi.CommandResultStatusFailed, got.Status)
	assert.Contains(t, *got.Message, "no logs were collected")
	assert.Nil(t, got.Output)
	assert.Zero(t, kept.lastCalls)
}

func TestACustomersRequestNeedsNoGrantAndReadsKeptLines(t *testing.T) {
	t.Parallel()
	cmd := collect("api")
	cp := &fakeControlPlane{commands: []instanceapi.Command{cmd}}
	grants, kept := &fakeGrants{}, &fakeLogs{kept: true}
	r, _ := runnerWith(t, cp, grants, kept, apiPod())

	require.NoError(t, r.Run(t.Context()))

	got := cp.answers[cmd.Id][0]
	require.Equal(t, instanceapi.CommandResultStatusSucceeded, got.Status)
	assert.Equal(t, "api kept lines\n", *got.Output)
	assert.Empty(t, grants.pulls)
	assert.Equal(t, "password=[redacted]", kept.redacted, "the kept lines are redacted too")
}

func TestLinesNeedSomeoneWhoAsked(t *testing.T) {
	t.Parallel()
	cmd := collect("api")
	cmd.RequestedBy = nil
	cp := &fakeControlPlane{commands: []instanceapi.Command{cmd}}
	r, _ := runner(t, cp, apiPod())

	require.NoError(t, r.Run(t.Context()))

	assert.Equal(t, instanceapi.CommandResultStatusFailed, cp.answers[cmd.Id][0].Status)
}

func TestAStaffBundleWithoutAGrantIsRefusedOnTheCommandAndTheBundle(t *testing.T) {
	t.Parallel()
	cmd := bundleCommand(staff("sam@trenova.app"))
	cp := &fakeControlPlane{commands: []instanceapi.Command{cmd}}
	grants, kept := &fakeGrants{}, &fakeLogs{kept: true}
	r, _ := runnerWith(t, cp, grants, kept)

	require.NoError(t, r.Run(t.Context()))

	assert.Equal(t, instanceapi.CommandResultStatusRefused, cp.answers[cmd.Id][0].Status)
	assert.Equal(t, instanceapi.LogBundleRefused, kept.finished[*cmd.BundleId].Status)
	assert.Empty(t, kept.jobs, "nothing is collected")
	assert.Equal(t, []pull{{email: "sam@trenova.app", kind: tms.LogPullBundle}}, grants.pulls)
}

func TestABundleFailsWhenLogKeepingIsOff(t *testing.T) {
	t.Parallel()
	cmd := bundleCommand(customer())
	cp := &fakeControlPlane{commands: []instanceapi.Command{cmd}}
	kept := &fakeLogs{}
	r, _ := runnerWith(t, cp, &fakeGrants{}, kept)

	require.NoError(t, r.Run(t.Context()))

	got := cp.answers[cmd.Id][0]
	assert.Equal(t, instanceapi.CommandResultStatusFailed, got.Status)
	assert.Equal(t, logs.NotKeptMessage, *got.Message)
	assert.Equal(t, instanceapi.LogBundleFailed, kept.finished[*cmd.BundleId].Status)
}

func TestABundleLongerThanAWeekFails(t *testing.T) {
	t.Parallel()
	cmd := bundleCommand(customer())
	until := cmd.Since.Add(8 * 24 * time.Hour)
	cmd.Until = &until
	cp := &fakeControlPlane{commands: []instanceapi.Command{cmd}}
	kept := &fakeLogs{kept: true}
	r, _ := runnerWith(t, cp, &fakeGrants{}, kept)

	require.NoError(t, r.Run(t.Context()))

	assert.Equal(t, instanceapi.CommandResultStatusFailed, cp.answers[cmd.Id][0].Status)
	assert.Empty(t, kept.jobs)
}

func TestABundleIsCollectedOnceAndAnsweredWhenItIsStored(t *testing.T) {
	t.Parallel()
	cmd := bundleCommand(customer())
	cp := &fakeControlPlane{commands: []instanceapi.Command{cmd}}
	kept := &fakeLogs{kept: true}
	r, _ := runnerWith(t, cp, &fakeGrants{}, kept)

	require.NoError(t, r.Run(t.Context()))
	require.NoError(t, r.Run(t.Context()))
	require.Len(t, kept.jobs, 1, "a command listed again is not collected again")
	assert.Empty(t, cp.answers[cmd.Id], "the command waits for its bundle")

	kept.jobs[0].Done(logs.Outcome{
		Status: instanceapi.LogBundleStored, Key: "support/logs/" + cmd.BundleId.String() + ".zip", Size: 4096,
	})
	require.NoError(t, r.Run(t.Context()))
	require.NoError(t, r.Run(t.Context()))

	require.Len(t, kept.jobs, 1)
	require.Len(t, cp.answers[cmd.Id], 2, "answered from memory, not collected again")
	assert.Equal(t, instanceapi.CommandResultStatusSucceeded, cp.answers[cmd.Id][0].Status)
	assert.Contains(t, *cp.answers[cmd.Id][0].Message, "4096 bytes")
}

func TestABundleWaitsWhenTheQueueIsFull(t *testing.T) {
	t.Parallel()
	cmd := bundleCommand(customer())
	cp := &fakeControlPlane{commands: []instanceapi.Command{cmd}}
	kept := &fakeLogs{kept: true, full: true}
	r, _ := runnerWith(t, cp, &fakeGrants{}, kept)

	require.NoError(t, r.Run(t.Context()))
	assert.Empty(t, cp.answers[cmd.Id])

	kept.full = false
	require.NoError(t, r.Run(t.Context()))
	assert.Len(t, kept.jobs, 1, "taken on the next tick")
}
