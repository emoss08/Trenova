package backup_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"

	"github.com/emoss08/trenova/operator/internal/backup"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
)

type kept struct {
	name, cluster, phase, id, reason string
	started, stopped                 time.Time
	annotations                      map[string]any
	scheduled                        bool
}

func (k kept) object() *unstructured.Unstructured {
	status := map[string]any{"phase": k.phase}
	if k.id != "" {
		status["backupId"] = k.id
	}
	if !k.started.IsZero() {
		status["startedAt"] = k.started.Format(time.RFC3339)
	}
	if !k.stopped.IsZero() {
		status["stoppedAt"] = k.stopped.Format(time.RFC3339)
	}
	if k.reason != "" {
		status["error"] = k.reason
	}
	created := k.started
	if created.IsZero() {
		created = now.Add(-time.Minute)
	}
	meta := map[string]any{
		"name": k.name, "namespace": namespace, "creationTimestamp": created.Format(time.RFC3339),
	}
	if k.annotations != nil {
		meta["annotations"] = k.annotations
	}
	if k.scheduled {
		meta["ownerReferences"] = []any{map[string]any{
			"apiVersion": "postgresql.cnpg.io/v1", "kind": "ScheduledBackup",
			"name": "trenova-postgres-daily", "uid": "5c1f0a52-8f1e-4d7e-9a43-1f0f3b2e6a10",
		}}
	}
	cluster := k.cluster
	if cluster == "" {
		cluster = "trenova-postgres"
	}
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "postgresql.cnpg.io/v1", "kind": "Backup",
		"metadata": meta,
		"spec":     map[string]any{"cluster": map[string]any{"name": cluster}, "method": "plugin"},
		"status":   status,
	}}
}

func healthy() []runtime.Object {
	return []runtime.Object{
		cluster("trenova-postgres", "Cluster in healthy state", "True", now.Add(-60*24*time.Hour)), store(),
	}
}

func listed(t *testing.T, objs ...runtime.Object) []instanceapi.BackupEntry {
	t.Helper()
	m, _ := manager(t, append(healthy(), objs...)...)
	res := m.Run(context.Background(), &backup.Input{Spec: spec(instanceapi.Active), Applied: true, Storage: storage})
	require.NotNil(t, res.Backups)
	return *res.Backups
}

func TestTheBackupListMapsEachPhaseAndOrigin(t *testing.T) {
	t.Parallel()
	request := uuid.NewString()
	got := listed(t,
		kept{
			name: "trenova-postgres-daily-20261114030000", phase: "completed", id: "20261114T030000",
			started: now.Add(-9*time.Hour - 10*time.Minute), stopped: now.Add(-9 * time.Hour), scheduled: true,
		}.object(),
		kept{
			name: backup.RequestName(request), phase: "completed", id: "20261112T101500",
			started: now.Add(-2 * 24 * time.Hour), stopped: now.Add(-2*24*time.Hour + 4*time.Minute),
			annotations: map[string]any{
				"trenova.app/backup-origin": "requested", "trenova.app/backup-request-id": request,
			},
		}.object(),
		kept{
			name: backup.FinalName(9), phase: "running", started: now.Add(-5 * time.Minute),
			annotations: map[string]any{"trenova.app/backup-origin": "final"},
		}.object(),
		kept{
			name: "trenova-postgres-daily-20261111030000", phase: "failed", reason: "bucket\nunreachable",
			started: now.Add(-3 * 24 * time.Hour), stopped: now.Add(-3*24*time.Hour + time.Minute), scheduled: true,
		}.object(),
		kept{name: "by-hand", phase: ""}.object(),
		kept{
			name: "another-cluster", cluster: "trenova-postgres-r1a2b3c4d", phase: "completed", id: "20261114T040000",
			started: now.Add(-8 * time.Hour), stopped: now.Add(-8 * time.Hour),
		}.object(),
	)
	require.Len(t, got, 5)

	pending := got[0]
	require.Equal(t, "by-hand", pending.Id)
	require.Equal(t, instanceapi.BackupEntryPhaseRunning, pending.Phase)
	require.Equal(t, now.Add(-time.Minute), pending.StartedAt)
	require.Nil(t, pending.Origin)
	require.Nil(t, pending.CompletedAt)

	final := got[1]
	require.Equal(t, backup.FinalName(9), final.Id)
	require.Equal(t, instanceapi.BackupEntryPhaseRunning, final.Phase)
	require.Equal(t, instanceapi.Final, *final.Origin)
	require.Nil(t, final.CompletedAt)
	require.Nil(t, final.RequestId)

	daily := got[2]
	require.Equal(t, "20261114T030000", daily.Id)
	require.Equal(t, instanceapi.BackupEntryPhaseCompleted, daily.Phase)
	require.Equal(t, instanceapi.Scheduled, *daily.Origin)
	require.Equal(t, now.Add(-9*time.Hour-10*time.Minute), daily.StartedAt)
	require.Equal(t, now.Add(-9*time.Hour), *daily.CompletedAt)
	require.Nil(t, daily.SizeBytes)
	require.Nil(t, daily.Message)

	requested := got[3]
	require.Equal(t, "20261112T101500", requested.Id)
	require.Equal(t, instanceapi.Requested, *requested.Origin)
	require.Equal(t, request, *requested.RequestId)

	failed := got[4]
	require.Equal(t, instanceapi.BackupEntryPhaseFailed, failed.Phase)
	require.Equal(t, "trenova-postgres-daily-20261111030000", failed.Id)
	require.Equal(t, instanceapi.Scheduled, *failed.Origin)
	require.Equal(t, now.Add(-3*24*time.Hour+time.Minute), *failed.CompletedAt)
	require.Equal(t, "bucket unreachable", *failed.Message)
}

func TestTheBackupListLeavesOutWhatCannotBeRestoredFromAndOldFailures(t *testing.T) {
	t.Parallel()
	frp := now.Add(-29 * 24 * time.Hour)
	got := listed(t,
		kept{
			name: "oldest-kept", phase: "completed", id: "oldest-kept",
			started: frp.Add(-10 * time.Minute), stopped: frp, scheduled: true,
		}.object(),
		kept{
			name: "removed-by-retention", phase: "completed", id: "removed-by-retention",
			started: frp.Add(-24*time.Hour - 10*time.Minute), stopped: frp.Add(-24 * time.Hour), scheduled: true,
		}.object(),
		kept{
			name: "failed-last-week", phase: "failed", reason: "bucket unreachable",
			started: now.Add(-8 * 24 * time.Hour), stopped: now.Add(-8*24*time.Hour + time.Minute),
		}.object(),
		kept{
			name: "failed-this-week", phase: "failed", reason: "bucket unreachable",
			started: now.Add(-6 * 24 * time.Hour), stopped: now.Add(-6*24*time.Hour + time.Minute),
		}.object(),
		kept{name: "stuck", phase: "running", started: now.Add(-40 * 24 * time.Hour)}.object(),
	)
	ids := make([]string, 0, len(got))
	for _, e := range got {
		ids = append(ids, e.Id)
	}
	require.Equal(t, []string{"failed-this-week", "oldest-kept", "stuck"}, ids)
}

func TestTheBackupListKeepsTheHundredNewest(t *testing.T) {
	t.Parallel()
	objs := make([]runtime.Object, 0, 130)
	for i := range 130 {
		started := now.Add(-time.Duration(i+1) * time.Hour)
		objs = append(objs, kept{
			name: fmt.Sprintf("b-%03d", i), id: fmt.Sprintf("b-%03d", i), phase: "completed",
			started: started, stopped: started.Add(5 * time.Minute), scheduled: true,
		}.object())
	}
	got := listed(t, objs...)
	require.Len(t, got, 100)
	require.Equal(t, "b-000", got[0].Id)
	require.Equal(t, "b-099", got[99].Id)
	for i := 1; i < len(got); i++ {
		require.True(t, got[i-1].StartedAt.After(got[i].StartedAt))
	}
}

func TestBackupsTheOperatorTakesAreListedWithTheirOrigin(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	m, dyn := manager(t, healthy()...)
	id := uuid.NewString()
	s := spec(instanceapi.Active)
	s.Backup = &instanceapi.SpecBackupRequest{Id: id, RequestedAt: now}
	m.Run(ctx, &backup.Input{Spec: s, Applied: true, Storage: storage})
	m.Run(ctx, &backup.Input{Spec: spec(instanceapi.Decommissioned), Revision: 4, Applied: true, Storage: storage})
	for _, name := range []string{backup.RequestName(id), backup.FinalName(4)} {
		b, err := dyn.Resource(backups).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
		require.NoError(t, err)
		require.NoError(t, unstructured.SetNestedField(b.Object, now.Add(-time.Hour).Format(time.RFC3339), "status", "startedAt"))
		_, err = dyn.Resource(backups).Namespace(namespace).Update(ctx, b, metav1.UpdateOptions{})
		require.NoError(t, err)
	}

	res := m.Run(ctx, &backup.Input{Spec: spec(instanceapi.Active), Applied: true, Storage: storage})
	require.NotNil(t, res.Backups)
	origins := map[string]instanceapi.BackupEntry{}
	for _, e := range *res.Backups {
		origins[e.Id] = e
	}
	requested := origins[backup.RequestName(id)]
	require.Equal(t, instanceapi.Requested, *requested.Origin)
	require.Equal(t, id, *requested.RequestId)
	require.Equal(t, instanceapi.Final, *origins[backup.FinalName(4)].Origin)
}

func TestAFailedListLeavesTheListOutAndTheHealthAsItWas(t *testing.T) {
	t.Parallel()
	m, dyn := manager(t, healthy()...)
	dyn.PrependReactor("list", "backups", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("the API server is restarting")
	})
	res := m.Run(context.Background(), &backup.Input{Spec: spec(instanceapi.Active), Applied: true, Storage: storage})
	require.Nil(t, res.Backups)
	require.NotNil(t, res.Health)
	require.Equal(t, instanceapi.BackupHealthWalArchiving("healthy"), res.Health.WalArchiving)
	require.Equal(t, now.Add(-9*time.Hour), *res.Health.LastBaseBackupAt)
	require.Equal(t, now.Add(-29*24*time.Hour), *res.Health.FirstRecoverabilityPoint)
	require.Nil(t, res.Health.Message)
	require.Nil(t, res.Health.Backups)
}
