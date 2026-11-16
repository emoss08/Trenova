package backup

import (
	"sort"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
)

const (
	originRequested = "requested"
	originFinal     = "final"
	scheduledKind   = "ScheduledBackup"

	maxListed      = 100
	maxRequestID   = 64
	failedListed   = 7 * 24 * time.Hour
	recoverySlack  = time.Minute
	listWarnPeriod = time.Minute
)

func (m *Manager) entries(
	items []unstructured.Unstructured,
	firstRecoverable *time.Time,
) []instanceapi.BackupEntry {
	out := make([]instanceapi.BackupEntry, 0, min(len(items), maxListed))
	for i := range items {
		if e, ok := m.entry(&items[i], firstRecoverable); ok {
			out = append(out, e)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].StartedAt.Equal(out[j].StartedAt) {
			return out[i].StartedAt.After(out[j].StartedAt)
		}
		return out[i].Id < out[j].Id
	})
	if len(out) > maxListed {
		out = out[:maxListed]
	}
	return out
}

func (m *Manager) entry(
	b *unstructured.Unstructured,
	firstRecoverable *time.Time,
) (instanceapi.BackupEntry, bool) {
	id, _, _ := unstructured.NestedString(b.Object, "status", "backupId")
	if id == "" {
		id = b.GetName()
	}
	started := timestamp(b, "status", "startedAt")
	if started == nil {
		created := b.GetCreationTimestamp().UTC()
		started = &created
	}
	stopped := timestamp(b, "status", "stoppedAt")
	ended := *started
	if stopped != nil {
		ended = *stopped
	}
	e := instanceapi.BackupEntry{
		Id:        clip(id, maxBackupID),
		Phase:     instanceapi.BackupEntryPhaseRunning,
		StartedAt: *started,
	}
	phase, _, _ := unstructured.NestedString(b.Object, "status", "phase")
	switch phase {
	case phaseCompleted:
		if firstRecoverable != nil && ended.Add(recoverySlack).Before(*firstRecoverable) {
			return e, false
		}
		e.Phase, e.CompletedAt = instanceapi.BackupEntryPhaseCompleted, stopped
	case phaseFailed:
		if m.now().Sub(ended) > failedListed {
			return e, false
		}
		e.Phase, e.CompletedAt = instanceapi.BackupEntryPhaseFailed, stopped
		reason, _, _ := unstructured.NestedString(b.Object, "status", "error")
		if why := message(reason); *why != "" {
			e.Message = why
		}
	}
	e.Origin, e.RequestId = origin(b)
	return e, true
}

func origin(b *unstructured.Unstructured) (*instanceapi.BackupEntryOrigin, *string) {
	annotations := b.GetAnnotations()
	switch annotations[annotationOrigin] {
	case originRequested:
		var request *string
		if id := annotations[annotationReqID]; id != "" {
			request = ptr(clip(id, maxRequestID))
		}
		return ptr(instanceapi.Requested), request
	case originFinal:
		return ptr(instanceapi.Final), nil
	}
	for _, owner := range b.GetOwnerReferences() {
		if owner.Kind == scheduledKind {
			return ptr(instanceapi.Scheduled), nil
		}
	}
	return nil, nil
}

func (m *Manager) warnList(err error) {
	at := m.now()
	if !m.listWarned.IsZero() && at.Sub(m.listWarned) < listWarnPeriod {
		return
	}
	m.listWarned = at
	m.logger.Warn("cannot list the base backups; the report leaves the list out", "error", err)
}
