package backup_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"

	"github.com/emoss08/trenova/operator/internal/backup"
	"github.com/emoss08/trenova/operator/internal/fakecp"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/values"
)

const namespace = "trenova"

var (
	clusters     = schema.GroupVersionResource{Group: "postgresql.cnpg.io", Version: "v1", Resource: "clusters"}
	backups      = schema.GroupVersionResource{Group: "postgresql.cnpg.io", Version: "v1", Resource: "backups"}
	objectStores = schema.GroupVersionResource{Group: "barmancloud.cnpg.io", Version: "v1", Resource: "objectstores"}
	now          = time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	storage      = values.Storage{
		Endpoint: "https://s3.bhs.io.cloud.ovh.net", Region: "bhs", DocumentsBucket: "trenova-x-docs",
		BackupsPath: "s3://trenova-x-backups/pitr/",
	}
)

func cluster(name, phase string, archiving string, created time.Time) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "postgresql.cnpg.io/v1", "kind": "Cluster",
		"metadata": map[string]any{
			"name": name, "namespace": namespace, "creationTimestamp": created.Format(time.RFC3339),
			"labels": map[string]any{"app.kubernetes.io/instance": "trenova", "app.kubernetes.io/component": "postgres"},
		},
		"spec": map[string]any{"instances": int64(1)},
		"status": map[string]any{
			"phase": phase, "readyInstances": int64(1), "lastArchivedWALTime": now.Add(-time.Minute).Format(time.RFC3339),
			"conditions": []any{map[string]any{"type": "ContinuousArchiving", "status": archiving, "message": "archive failed"}},
		},
	}}
}

func store() *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "barmancloud.cnpg.io/v1", "kind": "ObjectStore",
		"metadata": map[string]any{"name": "trenova-backups", "namespace": namespace},
		"status": map[string]any{"serverRecoveryWindow": map[string]any{"trenova-postgres": map[string]any{
			"firstRecoverabilityPoint": now.Add(-29 * 24 * time.Hour).Format(time.RFC3339),
			"lastSuccessfulBackupTime": now.Add(-9 * time.Hour).Format(time.RFC3339),
		}}},
	}}
}

func done(name, cluster, phase string, stopped time.Time) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "postgresql.cnpg.io/v1", "kind": "Backup",
		"metadata": map[string]any{"name": name, "namespace": namespace, "creationTimestamp": stopped.Add(-time.Minute).Format(time.RFC3339)},
		"spec":     map[string]any{"cluster": map[string]any{"name": cluster}, "method": "plugin"},
		"status": map[string]any{
			"phase": phase, "backupId": "20261114T030000", "stoppedAt": stopped.Format(time.RFC3339), "error": "bucket unreachable",
		},
	}}
}

func manager(t *testing.T, objs ...runtime.Object) (*backup.Manager, *dynamicfake.FakeDynamicClient) {
	t.Helper()
	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		clusters: "ClusterList", backups: "BackupList", objectStores: "ObjectStoreList",
	}, objs...)
	return backup.New(&backup.Params{
		Dynamic: dyn, Namespace: namespace, Release: "trenova",
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Now: func() time.Time { return now },
	}), dyn
}

func spec(lc instanceapi.InstanceSpecLifecycle) *instanceapi.InstanceSpec {
	s := fakecp.Spec(uuid.New())
	s.Lifecycle = lc
	return &s
}

func TestTheStatusReportCarriesBackupHealth(t *testing.T) {
	t.Parallel()
	m, _ := manager(t,
		cluster("trenova-postgres", "Cluster in healthy state", "True", now.Add(-60*24*time.Hour)), store(),
		done("daily-1", "trenova-postgres", "completed", now.Add(-9*time.Hour)),
	)
	res := m.Run(context.Background(), backup.Input{Spec: spec(instanceapi.Active), Revision: 3, Applied: true, Ready: true, Storage: storage})
	h := res.Health
	require.NotNil(t, h)
	require.Equal(t, instanceapi.BackupHealthWalArchiving("healthy"), h.WalArchiving)
	require.Equal(t, now.Add(-9*time.Hour), *h.LastBaseBackupAt)
	require.Equal(t, now.Add(-29*24*time.Hour), *h.FirstRecoverabilityPoint)
	require.Equal(t, now.Add(-time.Minute), *h.LastArchivedWalAt)
	require.Equal(t, "20261114T030000", *h.LastBaseBackupId)
	require.Nil(t, h.Message)
	require.Nil(t, res.Final)

	// Failing archiving says why.
	m, _ = manager(t, cluster("trenova-postgres", "Cluster in healthy state", "False", now), store())
	h = m.Run(context.Background(), backup.Input{Spec: spec(instanceapi.Active), Applied: true, Storage: storage}).Health
	require.Equal(t, instanceapi.BackupHealthWalArchiving("failing"), h.WalArchiving)
	require.Contains(t, *h.Message, "archive failed")

	// Without storage, nothing is reported or taken.
	require.Equal(t, backup.Result{}, m.Run(context.Background(), backup.Input{Spec: spec(instanceapi.Active), Applied: true}))
}

func TestABackupBeforeAnUpdateIsTakenOnceAndReported(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	m, dyn := manager(t, cluster("trenova-postgres", "Cluster in healthy state", "True", now), store())
	s := spec(instanceapi.Active)
	id := uuid.NewString()
	s.Backup = &instanceapi.SpecBackupRequest{Id: id, RequestedAt: now}

	// Not applied yet: nothing is started.
	res := m.Run(ctx, backup.Input{Spec: s, Applied: false, Storage: storage})
	require.Equal(t, instanceapi.BackupRequestStatusPhase("running"), res.Health.Request.Phase)
	_, err := dyn.Resource(backups).Namespace(namespace).Get(ctx, backup.RequestName(id), metav1.GetOptions{})
	require.Error(t, err)

	res = m.Run(ctx, backup.Input{Spec: s, Applied: true, Storage: storage})
	require.Equal(t, id, res.Health.Request.Id)
	require.Equal(t, instanceapi.BackupRequestStatusPhase("running"), res.Health.Request.Phase)
	b, err := dyn.Resource(backups).Namespace(namespace).Get(ctx, backup.RequestName(id), metav1.GetOptions{})
	require.NoError(t, err)
	method, _, _ := unstructured.NestedString(b.Object, "spec", "method")
	plugin, _, _ := unstructured.NestedString(b.Object, "spec", "pluginConfiguration", "name")
	target, _, _ := unstructured.NestedString(b.Object, "spec", "cluster", "name")
	require.Equal(t, "plugin", method)
	require.Equal(t, backup.PluginName, plugin)
	require.Equal(t, "trenova-postgres", target)

	// Asked again: the same backup, not another.
	m.Run(ctx, backup.Input{Spec: s, Applied: true, Storage: storage})
	list, err := dyn.Resource(backups).Namespace(namespace).List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, list.Items, 1)

	// It fails: reported as failed, with the reason.
	require.NoError(t, unstructured.SetNestedMap(b.Object, map[string]any{
		"phase": "failed", "error": "bucket unreachable", "stoppedAt": now.Format(time.RFC3339),
	}, "status"))
	_, err = dyn.Resource(backups).Namespace(namespace).Update(ctx, b, metav1.UpdateOptions{})
	require.NoError(t, err)
	res = m.Run(ctx, backup.Input{Spec: s, Applied: true, Storage: storage})
	require.Equal(t, instanceapi.BackupRequestStatusPhase("failed"), res.Health.Request.Phase)
	require.Contains(t, *res.Health.Request.Message, "bucket unreachable")
}

func TestTheFinalBackupIsTakenWhenDecommissioned(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	m, dyn := manager(t, cluster("trenova-postgres", "Cluster in healthy state", "True", now), store())
	s := spec(instanceapi.Decommissioned)

	require.Nil(t, m.Run(ctx, backup.Input{Spec: s, Revision: 7, Applied: true, Storage: storage}).Final)
	b, err := dyn.Resource(backups).Namespace(namespace).Get(ctx, backup.FinalName(7), metav1.GetOptions{})
	require.NoError(t, err)

	require.NoError(t, unstructured.SetNestedMap(b.Object, map[string]any{
		"phase": "completed", "backupId": "20261114T120500", "stoppedAt": now.Add(5 * time.Minute).Format(time.RFC3339),
	}, "status"))
	_, err = dyn.Resource(backups).Namespace(namespace).Update(ctx, b, metav1.UpdateOptions{})
	require.NoError(t, err)
	final := m.Run(ctx, backup.Input{Spec: s, Revision: 7, Applied: true, Storage: storage}).Final
	require.NotNil(t, final)
	require.Equal(t, "20261114T120500", final.Id)
	require.Equal(t, now.Add(5*time.Minute), final.CompletedAt)

	// An active instance takes no final backup.
	m, dyn = manager(t, cluster("trenova-postgres", "Cluster in healthy state", "True", now), store())
	require.Nil(t, m.Run(ctx, backup.Input{Spec: spec(instanceapi.Active), Revision: 7, Applied: true, Storage: storage}).Final)
	_, err = dyn.Resource(backups).Namespace(namespace).Get(ctx, backup.FinalName(7), metav1.GetOptions{})
	require.Error(t, err)
}

func TestARestoreIsReportedAndOnlyItsSwitchStopsTheOldDatabase(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	target := now.Add(-3 * time.Hour)
	live := cluster("trenova-postgres", "Cluster in healthy state", "True", now.Add(-60*24*time.Hour))
	restored := cluster("trenova-postgres-r1a2b3c4d", "Setting up primary", "True", now.Add(-time.Hour))
	m, dyn := manager(t, live, restored, store())
	s := spec(instanceapi.Active)
	s.Restore = &instanceapi.SpecRestore{Id: uuid.New(), Generation: "r1a2b3c4d", TargetTime: target}

	res := m.Run(ctx, backup.Input{Spec: s, Applied: true, Ready: true, Storage: storage})
	require.Equal(t, instanceapi.RestoreProgressPhase("preparing"), res.Restore.Phase)

	require.NoError(t, unstructured.SetNestedField(restored.Object, "Cluster in healthy state", "status", "phase"))
	_, err := dyn.Resource(clusters).Namespace(namespace).Update(ctx, restored, metav1.UpdateOptions{})
	require.NoError(t, err)
	res = m.Run(ctx, backup.Input{Spec: s, Applied: true, Ready: true, Storage: storage})
	require.Equal(t, instanceapi.RestoreProgressPhase("ready"), res.Restore.Phase)
	require.Equal(t, target, *res.Restore.RecoveredTo)
	// Ready is not switched: the live database is untouched.
	got, err := dyn.Resource(clusters).Namespace(namespace).Get(ctx, "trenova-postgres", metav1.GetOptions{})
	require.NoError(t, err)
	require.NotEqual(t, "on", got.GetAnnotations()["cnpg.io/hibernation"])

	// Switched: the old database is stopped and kept.
	s.Database = &instanceapi.SpecDatabase{Generation: "r1a2b3c4d", TargetTime: target}
	res = m.Run(ctx, backup.Input{Spec: s, Applied: true, Ready: true, Storage: storage})
	require.Equal(t, instanceapi.RestoreProgressPhase("switched"), res.Restore.Phase)
	got, err = dyn.Resource(clusters).Namespace(namespace).Get(ctx, "trenova-postgres", metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, "on", got.GetAnnotations()["cnpg.io/hibernation"])
	kept, err := dyn.Resource(clusters).Namespace(namespace).Get(ctx, "trenova-postgres-r1a2b3c4d", metav1.GetOptions{})
	require.NoError(t, err)
	require.NotEqual(t, "on", kept.GetAnnotations()["cnpg.io/hibernation"])
}

func TestARestoreThatNeverBecomesReadyFails(t *testing.T) {
	t.Parallel()
	m, _ := manager(t,
		cluster("trenova-postgres", "Cluster in healthy state", "True", now.Add(-60*24*time.Hour)),
		cluster("trenova-postgres-r1a2b3c4d", "Setting up primary", "True", now.Add(-7*time.Hour)), store())
	s := spec(instanceapi.Active)
	s.Restore = &instanceapi.SpecRestore{Id: uuid.New(), Generation: "r1a2b3c4d", TargetTime: now.Add(-8 * time.Hour)}
	res := m.Run(context.Background(), backup.Input{Spec: s, Applied: true, Ready: true, Storage: storage})
	require.Equal(t, instanceapi.RestoreProgressPhase("failed"), res.Restore.Phase)
}
