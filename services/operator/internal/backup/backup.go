package backup

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"

	"github.com/emoss08/trenova/operator/internal/health"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/values"
)

const (
	PluginName = "barman-cloud.cloudnative-pg.io"

	labelCluster     = "cnpg.io/cluster"
	labelInstance    = "app.kubernetes.io/instance"
	labelComponent   = "app.kubernetes.io/component"
	labelRequest     = "trenova.app/backup-request"
	annotationHibern = "cnpg.io/hibernation"
	healthyPhase     = "Cluster in healthy state"

	phaseCompleted = "completed"
	phaseFailed    = "failed"

	maxMessage   = 500
	maxBackupID  = 200
	retryFailed  = 10 * time.Minute
	restoreLimit = 6 * time.Hour
)

var (
	clusterResource = schema.GroupVersionResource{
		Group: "postgresql.cnpg.io", Version: "v1", Resource: "clusters",
	}
	backupResource = schema.GroupVersionResource{
		Group: "postgresql.cnpg.io", Version: "v1", Resource: "backups",
	}
	objectStoreResource = schema.GroupVersionResource{
		Group: "barmancloud.cnpg.io", Version: "v1", Resource: "objectstores",
	}
)

type Manager struct {
	dynamic   dynamic.Interface
	namespace string
	release   string
	logger    *slog.Logger
	now       func() time.Time
}

type Params struct {
	Dynamic   dynamic.Interface
	Namespace string
	Release   string
	Logger    *slog.Logger
	Now       func() time.Time
}

func New(p *Params) *Manager {
	now := p.Now
	if now == nil {
		now = time.Now
	}
	return &Manager{
		dynamic:   p.Dynamic,
		namespace: p.Namespace,
		release:   p.Release,
		logger:    p.Logger,
		now:       now,
	}
}

type Input struct {
	Spec     *instanceapi.InstanceSpec
	Revision int64
	Applied  bool
	Ready    bool
	Storage  values.Storage
}

type Result struct {
	Health  *instanceapi.BackupHealth
	Final   *instanceapi.FinalBackup
	Restore *instanceapi.RestoreProgress
}

func (m *Manager) Run(ctx context.Context, in Input) Result {
	if !in.Storage.Configured() {
		return Result{}
	}
	active := values.PostgresCluster(m.release, generation(in.Spec))
	h := m.health(ctx, active)
	out := Result{Health: &h}
	if req := in.Spec.Backup; req != nil {
		out.Health.Request = m.requested(ctx, in, active, req)
	}
	if in.Spec.Lifecycle == instanceapi.Decommissioned {
		out.Final = m.final(ctx, in, active, out.Health)
	}
	if r := in.Spec.Restore; r != nil {
		out.Restore = m.restore(ctx, in, active, r)
	}
	return out
}

func generation(spec *instanceapi.InstanceSpec) string {
	if spec.Database == nil {
		return ""
	}
	return spec.Database.Generation
}

func (m *Manager) health(ctx context.Context, active string) instanceapi.BackupHealth {
	h := instanceapi.BackupHealth{WalArchiving: instanceapi.BackupHealthWalArchiving("unknown")}
	cluster, err := m.dynamic.Resource(clusterResource).Namespace(m.namespace).Get(ctx, active, metav1.GetOptions{})
	if err != nil {
		h.Message = message("cluster " + active + ": " + err.Error())
		return h
	}
	if cond, ok := condition(cluster, "ContinuousArchiving"); ok {
		if cond["status"] == "True" {
			h.WalArchiving = instanceapi.BackupHealthWalArchiving("healthy")
		} else {
			h.WalArchiving = instanceapi.BackupHealthWalArchiving("failing")
			if msg, _ := cond["message"].(string); msg != "" {
				h.Message = message("WAL archiving: " + msg)
			}
		}
	}
	if at := timestamp(cluster, "status", "lastArchivedWALTime"); at != nil {
		h.LastArchivedWalAt = at
	}
	if store, err := m.dynamic.Resource(objectStoreResource).Namespace(m.namespace).
		Get(ctx, values.Fullname(m.release)+"-backups", metav1.GetOptions{}); err == nil {
		h.FirstRecoverabilityPoint = timestamp(store, "status", "serverRecoveryWindow", active, "firstRecoverabilityPoint")
		h.LastBaseBackupAt = timestamp(store, "status", "serverRecoveryWindow", active, "lastSuccessfulBackupTime")
	}
	latest, failed := m.latestBackups(ctx, active)
	if latest != nil {
		if id, _, _ := unstructured.NestedString(latest.Object, "status", "backupId"); id != "" {
			h.LastBaseBackupId = ptr(clip(id, maxBackupID))
		}
		if h.LastBaseBackupAt == nil {
			h.LastBaseBackupAt = timestamp(latest, "status", "stoppedAt")
		}
	}
	if h.Message == nil && failed != nil && (latest == nil || newer(failed, latest)) {
		reason, _, _ := unstructured.NestedString(failed.Object, "status", "error")
		h.Message = message("the last base backup failed: " + reason)
	}
	return h
}

func (m *Manager) latestBackups(ctx context.Context, cluster string) (completed, failed *unstructured.Unstructured) {
	list, err := m.dynamic.Resource(backupResource).Namespace(m.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, nil
	}
	items := list.Items
	sort.Slice(items, func(i, j int) bool { return newer(&items[i], &items[j]) })
	for i := range items {
		b := &items[i]
		name, _, _ := unstructured.NestedString(b.Object, "spec", "cluster", "name")
		if name != cluster {
			continue
		}
		phase, _, _ := unstructured.NestedString(b.Object, "status", "phase")
		switch {
		case phase == phaseCompleted && completed == nil:
			completed = b
		case phase == phaseFailed && failed == nil:
			failed = b
		}
	}
	return completed, failed
}

func newer(a, b *unstructured.Unstructured) bool {
	return a.GetCreationTimestamp().After(b.GetCreationTimestamp().Time)
}

func (m *Manager) requested(
	ctx context.Context,
	in Input,
	active string,
	req *instanceapi.SpecBackupRequest,
) *instanceapi.BackupRequestStatus {
	name := RequestName(req.Id)
	out := &instanceapi.BackupRequestStatus{Id: req.Id, Phase: instanceapi.BackupRequestStatusPhase("running")}
	b, err := m.backup(ctx, name)
	switch {
	case apierrors.IsNotFound(err):
		if !in.Applied || in.Spec.Lifecycle != instanceapi.Active {
			out.Message = ptr("waiting for the instance to run before the backup is taken")
			return out
		}
		if err = m.take(ctx, name, active, map[string]string{labelRequest: clip(req.Id, 63)}); err != nil {
			out.Message = message("cannot start the backup: " + err.Error())
		}
		return out
	case err != nil:
		out.Message = message(err.Error())
		return out
	}
	return progress(b, out)
}

func (m *Manager) final(
	ctx context.Context,
	in Input,
	active string,
	h *instanceapi.BackupHealth,
) *instanceapi.FinalBackup {
	name := FinalName(in.Revision)
	b, err := m.backup(ctx, name)
	switch {
	case apierrors.IsNotFound(err):
		if !in.Applied {
			return nil
		}
		if err = m.take(ctx, name, active, nil); err != nil {
			h.Message = message("cannot start the final backup: " + err.Error())
		}
		return nil
	case err != nil:
		h.Message = message("the final backup: " + err.Error())
		return nil
	}
	phase, _, _ := unstructured.NestedString(b.Object, "status", "phase")
	switch phase {
	case phaseCompleted:
		done := timestamp(b, "status", "stoppedAt")
		if done == nil {
			return nil
		}
		id, _, _ := unstructured.NestedString(b.Object, "status", "backupId")
		if id == "" {
			id = name
		}
		return &instanceapi.FinalBackup{Id: clip(id, maxBackupID), CompletedAt: *done}
	case phaseFailed:
		reason, _, _ := unstructured.NestedString(b.Object, "status", "error")
		h.Message = message("the final backup failed, and is taken again: " + reason)
		if stopped := timestamp(b, "status", "stoppedAt"); stopped == nil || m.now().Sub(*stopped) > retryFailed {
			if err := m.dynamic.Resource(backupResource).Namespace(m.namespace).
				Delete(ctx, name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
				m.logger.Warn("cannot clear a failed final backup", "backup", name, "error", err)
			}
		}
	}
	return nil
}

func (m *Manager) restore(
	ctx context.Context,
	in Input,
	active string,
	r *instanceapi.SpecRestore,
) *instanceapi.RestoreProgress {
	out := &instanceapi.RestoreProgress{Id: r.Id, Phase: instanceapi.RestoreProgressPhase("preparing")}
	candidate := values.PostgresCluster(m.release, r.Generation)
	if candidate == active {
		if !in.Applied || !in.Ready {
			return out
		}
		m.hibernateOthers(ctx, active)
		out.Phase = instanceapi.RestoreProgressPhase("switched")
		out.RecoveredTo = &r.TargetTime
		return out
	}
	cluster, err := m.dynamic.Resource(clusterResource).Namespace(m.namespace).Get(ctx, candidate, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			out.Message = message(err.Error())
		}
		return out
	}
	phase, _, _ := unstructured.NestedString(cluster.Object, "status", "phase")
	ready, _, _ := unstructured.NestedInt64(cluster.Object, "status", "readyInstances")
	switch {
	case phase == healthyPhase && ready >= 1:
		out.Phase = instanceapi.RestoreProgressPhase("ready")
		out.RecoveredTo = &r.TargetTime
	case m.now().Sub(cluster.GetCreationTimestamp().Time) > restoreLimit:
		out.Phase = instanceapi.RestoreProgressPhase("failed")
		out.Message = message(fmt.Sprintf("the restored database was not ready in six hours: %s", phase))
	case phase != "":
		out.Message = message(phase)
	}
	return out
}

func (m *Manager) hibernateOthers(ctx context.Context, active string) {
	clusters := m.dynamic.Resource(clusterResource).Namespace(m.namespace)
	list, err := clusters.List(ctx, metav1.ListOptions{
		LabelSelector: labelInstance + "=" + m.release + "," + labelComponent + "=postgres",
	})
	if err != nil {
		m.logger.Warn("cannot list the databases a restore replaced", "error", err)
		return
	}
	patch := []byte(`{"metadata":{"annotations":{"` + annotationHibern + `":"on"}}}`)
	for _, c := range list.Items {
		if c.GetName() == active || c.GetAnnotations()[annotationHibern] == "on" {
			continue
		}
		if _, err := clusters.Patch(ctx, c.GetName(), types.MergePatchType, patch, metav1.PatchOptions{}); err != nil {
			m.logger.Warn("cannot stop a database a restore replaced", "cluster", c.GetName(), "error", err)
			continue
		}
		m.logger.Info("a database a restore replaced is stopped and kept", "cluster", c.GetName())
	}
}

func (m *Manager) backup(ctx context.Context, name string) (*unstructured.Unstructured, error) {
	return m.dynamic.Resource(backupResource).Namespace(m.namespace).Get(ctx, name, metav1.GetOptions{})
}

func (m *Manager) take(ctx context.Context, name, cluster string, extra map[string]string) error {
	labels := map[string]any{labelCluster: cluster, labelInstance: m.release}
	for k, v := range extra {
		labels[k] = v
	}
	b := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "postgresql.cnpg.io/v1",
		"kind":       "Backup",
		"metadata": map[string]any{
			"name":      name,
			"namespace": m.namespace,
			"labels":    labels,
		},
		"spec": map[string]any{
			"cluster":             map[string]any{"name": cluster},
			"method":              "plugin",
			"pluginConfiguration": map[string]any{"name": PluginName},
		},
	}}
	_, err := m.dynamic.Resource(backupResource).Namespace(m.namespace).Create(ctx, b, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		return nil
	}
	if err == nil {
		m.logger.Info("base backup started", "backup", name, "cluster", cluster)
	}
	return err
}

func progress(b *unstructured.Unstructured, out *instanceapi.BackupRequestStatus) *instanceapi.BackupRequestStatus {
	phase, _, _ := unstructured.NestedString(b.Object, "status", "phase")
	switch phase {
	case phaseCompleted:
		out.Phase = instanceapi.BackupRequestStatusPhase("completed")
		out.CompletedAt = timestamp(b, "status", "stoppedAt")
		if id, _, _ := unstructured.NestedString(b.Object, "status", "backupId"); id != "" {
			out.BackupId = ptr(clip(id, maxBackupID))
		}
	case phaseFailed:
		out.Phase = instanceapi.BackupRequestStatusPhase("failed")
		reason, _, _ := unstructured.NestedString(b.Object, "status", "error")
		out.Message = message("the backup failed: " + reason)
	}
	return out
}

func RequestName(id string) string {
	hex := strings.ReplaceAll(strings.ToLower(id), "-", "")
	if parsed, err := uuid.Parse(id); err == nil {
		hex = strings.ReplaceAll(parsed.String(), "-", "")
	}
	if len(hex) > 16 {
		hex = hex[len(hex)-16:]
	}
	return "trenova-backup-" + hex
}

func FinalName(revision int64) string {
	return fmt.Sprintf("trenova-final-r%d", revision)
}

func condition(obj *unstructured.Unstructured, kind string) (map[string]any, bool) {
	conds, _, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")
	for _, raw := range conds {
		cond, ok := raw.(map[string]any)
		if ok && cond["type"] == kind {
			return cond, true
		}
	}
	return nil, false
}

func timestamp(obj *unstructured.Unstructured, path ...string) *time.Time {
	raw, ok, _ := unstructured.NestedString(obj.Object, path...)
	if !ok || raw == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05.999999-07", "2006-01-02 15:04:05-07"} {
		if t, err := time.Parse(layout, raw); err == nil {
			t = t.UTC()
			return &t
		}
	}
	return nil
}

func message(s string) *string {
	m := health.Clean(s, maxMessage)
	return &m
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func ptr[T any](v T) *T { return &v }
