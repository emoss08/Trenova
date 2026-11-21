package importer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/values"
)

const (
	Container   = "import"
	keyID       = "id"
	keyState    = "state"
	keyAt       = "at"
	keyReason   = "reason"
	labelImport = "trenova.app/import"
	maxReason   = 500
	exitRefused = 2
)

type Importer struct {
	kube      kubernetes.Interface
	namespace string
	release   string
	logger    *slog.Logger
	now       func() time.Time
}

type Params struct {
	Kube      kubernetes.Interface
	Namespace string
	Release   string
	Logger    *slog.Logger
	Now       func() time.Time
}

func New(p *Params) (*Importer, error) {
	if p.Kube == nil || p.Namespace == "" || p.Release == "" || p.Logger == nil {
		return nil, errors.New("importer: every dependency is required")
	}
	now := p.Now
	if now == nil {
		now = time.Now
	}
	return &Importer{
		kube: p.Kube, namespace: p.Namespace, release: p.Release, logger: p.Logger, now: now,
	}, nil
}

type record struct {
	id     string
	state  instanceapi.ImportStatusState
	at     time.Time
	reason string
}

func (r *record) status() *instanceapi.ImportStatus {
	parsed, err := uuidOf(r.id)
	if err != nil {
		return nil
	}
	out := &instanceapi.ImportStatus{Id: parsed, State: r.state, At: r.at}
	if r.reason != "" {
		reason := r.reason
		out.Reason = &reason
	}
	return out
}

func (i *Importer) Prepare(
	ctx context.Context,
	spec *instanceapi.InstanceSpec,
	storage *values.Storage,
) (values.Import, error) {
	rec, found, err := i.record(ctx)
	if err != nil {
		return values.Import{}, err
	}
	imported := found && rec.state == instanceapi.ImportDone
	if spec.Import == nil {
		if err = i.dropKeys(ctx); err != nil {
			return values.Import{}, err
		}
		return values.Import{ImportedBefore: imported}, nil
	}
	if err = values.CheckImport(spec.Import, storage); err != nil {
		return values.Import{}, err
	}
	if found && rec.id == spec.Import.Id.String() {
		if err = i.dropKeys(ctx); err != nil {
			return values.Import{}, err
		}
		return values.Import{Spec: spec.Import, Settled: true, ImportedBefore: imported}, nil
	}
	if imported {
		return values.Import{}, fmt.Errorf(
			"%w: this instance was already made from import %s and is never imported again",
			values.ErrInvalidSpec, rec.id,
		)
	}
	if err = i.ensureKeys(ctx, spec.Import); err != nil {
		return values.Import{}, err
	}
	return values.Import{Spec: spec.Import, KeysSecret: values.ImportKeysSecret(i.release)}, nil
}

func (i *Importer) Imported(ctx context.Context) (bool, error) {
	rec, found, err := i.record(ctx)
	if err != nil {
		return false, err
	}
	return found && rec.state == instanceapi.ImportDone, nil
}

func (i *Importer) Observe(
	ctx context.Context,
	spec *instanceapi.InstanceSpec,
	releaseJob string,
) (status *instanceapi.ImportStatus, reported bool, err error) {
	if spec.Import == nil {
		return nil, false, nil
	}
	id := spec.Import.Id.String()
	rec, found, err := i.record(ctx)
	if err != nil {
		return nil, false, err
	}
	if found && rec.id == id {
		return rec.status(), true, nil
	}
	if releaseJob == "" {
		return nil, false, nil
	}
	job, err := i.kube.BatchV1().Jobs(i.namespace).
		Get(ctx, releaseJob, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("read the release Job: %w", err)
	}
	if job.Annotations[labelImport] != id {
		return nil, false, nil
	}
	outcome, settled, err := i.outcome(ctx, job, id)
	if err != nil {
		return nil, false, err
	}
	if !settled {
		return outcome, true, nil
	}
	if err = i.settle(ctx, outcome); err != nil {
		return nil, false, err
	}
	return outcome, true, nil
}

func (i *Importer) outcome(
	ctx context.Context,
	job *batchv1.Job,
	id string,
) (*instanceapi.ImportStatus, bool, error) {
	parsed, err := uuidOf(id)
	if err != nil {
		return nil, false, err
	}
	started := job.CreationTimestamp.UTC()
	if job.Status.StartTime != nil {
		started = job.Status.StartTime.UTC()
	}
	importing := &instanceapi.ImportStatus{
		Id:    parsed,
		State: instanceapi.ImportImporting,
		At:    started,
	}
	finished, failed, why := conditions(job)
	if !finished && !failed {
		return importing, false, nil
	}
	last, err := i.lastTermination(ctx, job)
	if err != nil {
		return nil, false, err
	}
	at := i.now().UTC()
	if last != nil && !last.FinishedAt.IsZero() {
		at = last.FinishedAt.UTC()
	}
	if finished {
		return &instanceapi.ImportStatus{
			Id:    parsed,
			State: instanceapi.ImportDone,
			At:    at,
		}, true, nil
	}
	reason := "The import could not complete: the release Job failed before the import ran: " + why
	if last != nil {
		reason = failureReason(last)
	}
	reason = clip(reason)
	return &instanceapi.ImportStatus{
		Id: parsed, State: instanceapi.ImportRefused, At: at, Reason: &reason,
	}, true, nil
}

func conditions(job *batchv1.Job) (finished, failed bool, why string) {
	for _, c := range job.Status.Conditions {
		if c.Status != corev1.ConditionTrue {
			continue
		}
		switch c.Type {
		case batchv1.JobComplete:
			finished = true
		case batchv1.JobFailed:
			failed, why = true, c.Message
		case batchv1.JobSuspended, batchv1.JobFailureTarget, batchv1.JobSuccessCriteriaMet:
		}
	}
	return finished, failed, why
}

func (i *Importer) lastTermination(
	ctx context.Context,
	job *batchv1.Job,
) (*corev1.ContainerStateTerminated, error) {
	pods, err := i.kube.CoreV1().Pods(i.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(labels.Set{"job-name": job.Name}).String(),
	})
	if err != nil {
		return nil, fmt.Errorf("read the release Job's pods: %w", err)
	}
	var last *corev1.ContainerStateTerminated
	for p := range pods.Items {
		for c := range pods.Items[p].Status.ContainerStatuses {
			st := &pods.Items[p].Status.ContainerStatuses[c]
			if st.Name != Container || st.State.Terminated == nil {
				continue
			}
			if last == nil || st.State.Terminated.FinishedAt.After(last.FinishedAt.Time) {
				last = st.State.Terminated
			}
		}
	}
	return last, nil
}

type statusLine struct {
	State  string `json:"state"`
	Reason string `json:"reason"`
}

func failureReason(t *corev1.ContainerStateTerminated) string {
	var line statusLine
	text := strings.TrimSpace(t.Message)
	if text != "" {
		_ = sonic.Unmarshal([]byte(lastLine(text)), &line)
	}
	reason := strings.TrimSpace(line.Reason)
	if t.ExitCode == exitRefused {
		if reason == "" {
			reason = "the archive or this instance was refused"
		}
		return "The archive was refused: " + reason
	}
	if reason == "" {
		reason = fmt.Sprintf("the import stopped with exit code %d", t.ExitCode)
	}
	return "The import could not complete: " + reason
}

func lastLine(s string) string {
	if idx := strings.LastIndexByte(s, '\n'); idx >= 0 {
		return s[idx+1:]
	}
	return s
}

func (i *Importer) settle(ctx context.Context, status *instanceapi.ImportStatus) error {
	data := map[string]string{
		keyID:    status.Id.String(),
		keyState: string(status.State),
		keyAt:    status.At.UTC().Format(time.RFC3339),
	}
	if status.Reason != nil {
		data[keyReason] = *status.Reason
	}
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      values.ImportRecord(i.release),
			Namespace: i.namespace,
			Labels:    map[string]string{"app.kubernetes.io/managed-by": "trenova-operator"},
		},
		Data: data,
	}
	_, err := i.kube.CoreV1().ConfigMaps(i.namespace).Create(ctx, cm, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		_, err = i.kube.CoreV1().ConfigMaps(i.namespace).Update(ctx, cm, metav1.UpdateOptions{})
	}
	if err != nil {
		return fmt.Errorf("record the import's outcome: %w", err)
	}
	i.logger.Info("the import settled", "import", status.Id, "state", status.State)
	return i.dropKeys(ctx)
}

func (i *Importer) record(ctx context.Context) (rec *record, found bool, err error) {
	cm, err := i.kube.CoreV1().ConfigMaps(i.namespace).
		Get(ctx, values.ImportRecord(i.release), metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("read the import record: %w", err)
	}
	at, _ := time.Parse(time.RFC3339, cm.Data[keyAt])
	return &record{
		id:     cm.Data[keyID],
		state:  instanceapi.ImportStatusState(cm.Data[keyState]),
		at:     at,
		reason: cm.Data[keyReason],
	}, true, nil
}

func (i *Importer) ensureKeys(ctx context.Context, spec *instanceapi.SpecImport) error {
	body, err := sonic.Marshal(spec.ArchiveKeys)
	if err != nil {
		return errors.New("encode the archive keys")
	}
	name := values.ImportKeysSecret(i.release)
	secrets := i.kube.CoreV1().Secrets(i.namespace)
	held, err := secrets.Get(ctx, name, metav1.GetOptions{})
	switch {
	case apierrors.IsNotFound(err):
		_, err = secrets.Create(ctx, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:        name,
				Namespace:   i.namespace,
				Labels:      map[string]string{"app.kubernetes.io/managed-by": "trenova-operator"},
				Annotations: map[string]string{labelImport: spec.Id.String()},
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{values.ImportKeysKey: body},
		}, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("keep the import's archive keys: %w", err)
		}
		i.logger.Info("the import's archive keys are held for its Job", "import", spec.Id)
		return nil
	case err != nil:
		return fmt.Errorf("read the import's archive keys: %w", err)
	case bytes.Equal(held.Data[values.ImportKeysKey], body) &&
		held.Annotations[labelImport] == spec.Id.String():
		return nil
	}
	held.Data = map[string][]byte{values.ImportKeysKey: body}
	if held.Annotations == nil {
		held.Annotations = map[string]string{}
	}
	held.Annotations[labelImport] = spec.Id.String()
	if _, err = secrets.Update(ctx, held, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("keep the import's archive keys: %w", err)
	}
	return nil
}

func (i *Importer) dropKeys(ctx context.Context) error {
	err := i.kube.CoreV1().Secrets(i.namespace).
		Delete(ctx, values.ImportKeysSecret(i.release), metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("delete the import's archive keys: %w", err)
	}
	if err == nil {
		i.logger.Info("the import's archive keys were deleted")
	}
	return nil
}

func clip(s string) string {
	if utf8.RuneCountInString(s) <= maxReason {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxReason-1]) + "…"
}

func uuidOf(id string) (uuid.UUID, error) {
	return uuid.Parse(id)
}
