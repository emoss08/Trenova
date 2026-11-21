package selfupgrade

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/emoss08/trenova/operator/internal/health"
)

const (
	AnnotationFailed  = "trenova.app/operator-upgrade-failed"
	AnnotationReason  = "trenova.app/operator-upgrade-reason"
	AnnotationStarted = "trenova.app/operator-upgrade-started"
	// AnnotationFrom is the release the operator ran when it asked for the
	// move: what it still runs while its Deployment rolls to the new one.
	AnnotationFrom = "trenova.app/operator-upgrade-from"
	// AnnotationVersion is on the operator's Deployment: the release its
	// chart was installed at. A release that did not rebuild the operator
	// changes it and nothing in the pod, so it is read, never baked into the
	// pod's environment.
	AnnotationVersion = "trenova.app/operator-version"

	addonKind          = "Addon"
	jobVersionEnv      = "VERSION"
	progressDeadline   = "ProgressDeadlineExceeded"
	maxJobFailures     = 3
	maxMessage         = 500
	DefaultDeadline    = 20 * time.Minute
	defaultChartsSpace = "kube-system"
)

var HelmCharts = schema.GroupVersionResource{
	Group:    "helm.cattle.io",
	Version:  "v1",
	Resource: "helmcharts",
}

type Phase string

const (
	PhaseCurrent   Phase = "current"
	PhaseUpgrading Phase = "upgrading"
	PhaseFailed    Phase = "failed"
	PhaseUnmanaged Phase = "unmanaged"
)

type Status struct {
	Version string
	Target  string
	Phase   Phase
	Message string
}

func (s Status) Holds() bool {
	return s.Phase == PhaseUpgrading || s.Phase == PhaseFailed
}

type Upgrader struct {
	dynamic    dynamic.Interface
	kube       kubernetes.Interface
	known      string
	chartSpace string
	chart      string
	namespace  string
	deployment string
	deadline   time.Duration
	logger     *slog.Logger
	now        func() time.Time
	said       string
}

type Params struct {
	Dynamic            dynamic.Interface
	Kube               kubernetes.Interface
	HelmChartNamespace string
	HelmChart          string
	Namespace          string
	Deployment         string
	Deadline           time.Duration
	Logger             *slog.Logger
	Now                func() time.Time
}

func New(p *Params) (*Upgrader, error) {
	var missing []string
	if p.Logger == nil {
		missing = append(missing, "a logger")
	}
	if p.Kube == nil || (p.HelmChart != "" && p.Dynamic == nil) {
		missing = append(missing, "Kubernetes clients")
	}
	if p.Namespace == "" || p.Deployment == "" {
		missing = append(missing, "the operator's own namespace and Deployment")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("selfupgrade: the upgrader needs %s", strings.Join(missing, ", "))
	}
	space := p.HelmChartNamespace
	if space == "" {
		space = defaultChartsSpace
	}
	deadline := p.Deadline
	if deadline <= 0 {
		deadline = DefaultDeadline
	}
	now := p.Now
	if now == nil {
		now = time.Now
	}
	return &Upgrader{
		dynamic:    p.Dynamic,
		kube:       p.Kube,
		chartSpace: space,
		chart:      p.HelmChart,
		namespace:  p.Namespace,
		deployment: p.Deployment,
		deadline:   deadline,
		logger:     p.Logger,
		now:        now,
	}, nil
}

// runningVersion is the release the operator runs: its Deployment's version once
// every pod of the Deployment is the current one, which a release that did
// not rebuild the operator reaches at once, without a new pod. While the
// Deployment rolls to a new pod, the operator still runs the release it
// moved from.
func runningVersion(d *appsv1.Deployment, notes map[string]string) string {
	version := d.Annotations[AnnotationVersion]
	if health.Deployment(d).Ready {
		return version
	}
	if from := notes[AnnotationFrom]; from != "" {
		return from
	}
	return version
}

func (u *Upgrader) Converge(ctx context.Context, want string) Status {
	st := Status{Version: u.known, Target: want, Phase: PhaseCurrent}
	d, derr := u.kube.AppsV1().Deployments(u.namespace).Get(ctx, u.deployment, metav1.GetOptions{})
	if u.chart == "" {
		return u.unmanaged(ctx, st, d, derr,
			"self-upgrade is off: the operator's version is left as it is")
	}
	hc, err := u.dynamic.Resource(HelmCharts).Namespace(u.chartSpace).
		Get(ctx, u.chart, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return u.unmanaged(ctx, st, d, derr, fmt.Sprintf(
			"no HelmChart %s/%s: k3s's helm-controller did not install this operator, so its version is left as it is",
			u.chartSpace,
			u.chart,
		))
	}
	if err != nil {
		if want == u.known {
			return st
		}
		return u.waiting(st, "cannot read the operator's HelmChart: "+err.Error())
	}
	notes := hc.GetAnnotations()
	if derr != nil {
		if want == u.known {
			return st
		}
		return u.waiting(st, "cannot read the operator's Deployment: "+derr.Error())
	}
	running := runningVersion(d, notes)
	if running == "" {
		return u.waiting(st, fmt.Sprintf(
			"the operator's Deployment %s/%s names no version (%s)",
			u.namespace, u.deployment, AnnotationVersion,
		))
	}
	u.known, st.Version = running, running
	if addon := pinnedBy(hc); addon != "" {
		st.Message = fmt.Sprintf(
			"the node's manifest %s pins the operator's version, and k3s puts it back at every start",
			addon,
		)
		u.sayOnce(ctx, slog.LevelWarn, st.Message)
	}
	asked, _, _ := unstructured.NestedString(hc.Object, "spec", "version")
	if want == running {
		u.settle(ctx, running, asked, notes)
		return st
	}
	if notes[AnnotationFailed] == want {
		reason := notes[AnnotationReason]
		if reason == "" {
			reason = "the upgrade failed"
		}
		return u.failed(st, reason)
	}
	if asked != want {
		started := u.now().UTC().Format(time.RFC3339)
		if err = u.patch(ctx, want, map[string]any{
			AnnotationStarted: started, AnnotationFrom: running,
			AnnotationFailed: nil, AnnotationReason: nil,
		}); err != nil {
			return u.waiting(st, "cannot ask helm-controller to upgrade the operator: "+err.Error())
		}
		u.logger.InfoContext(ctx, "upgrading the operator", "from", running, "to", want)
		return u.waiting(
			st,
			fmt.Sprintf("the operator is being upgraded from %s to %s", running, want),
		)
	}
	return u.watch(ctx, st, d, notes)
}

// watch follows a move helm-controller was asked for: it waits, or, once the
// move failed or took too long, puts the HelmChart back on the release that
// runs and records why.
func (u *Upgrader) watch(
	ctx context.Context,
	st Status,
	d *appsv1.Deployment,
	notes map[string]string,
) Status {
	want, running := st.Target, st.Version
	started, perr := time.Parse(time.RFC3339, notes[AnnotationStarted])
	if perr != nil {
		started = u.now()
		_ = u.patch(ctx, want, map[string]any{
			AnnotationStarted: started.UTC().Format(time.RFC3339), AnnotationFrom: running,
		})
	}
	reason := u.failure(ctx, want, d)
	if reason == "" && u.now().Sub(started) > u.deadline {
		reason = fmt.Sprintf("the operator %s did not take over within %s", want, u.deadline)
	}
	if reason == "" {
		return u.waiting(st, fmt.Sprintf(
			"the operator is being upgraded from %s to %s (since %s)",
			running, want, started.UTC().Format(time.RFC3339),
		))
	}
	reason = health.Clean(reason, maxMessage)
	if err := u.patch(ctx, running, map[string]any{
		AnnotationFailed: want, AnnotationReason: reason,
		AnnotationStarted: nil, AnnotationFrom: nil,
	}); err != nil {
		return u.waiting(
			st,
			"the upgrade to "+want+" failed and cannot be undone yet: "+err.Error(),
		)
	}
	u.logger.ErrorContext(ctx, "the operator's upgrade failed; it stays on its version",
		"version", running, "target", want, "reason", reason)
	return u.failed(st, reason)
}

// unmanaged reports an operator that does not move itself, at the version
// its Deployment names.
func (u *Upgrader) unmanaged(
	ctx context.Context,
	st Status,
	d *appsv1.Deployment,
	derr error,
	why string,
) Status {
	if derr == nil && d.Annotations[AnnotationVersion] != "" {
		u.known = d.Annotations[AnnotationVersion]
		st.Version = u.known
	}
	st.Phase = PhaseUnmanaged
	u.sayOnce(ctx, slog.LevelInfo, why)
	return st
}

func (u *Upgrader) settle(
	ctx context.Context,
	running, asked string,
	notes map[string]string,
) {
	stale := notes[AnnotationStarted] != "" || notes[AnnotationFailed] != "" ||
		notes[AnnotationReason] != "" || notes[AnnotationFrom] != ""
	if asked == running && !stale {
		return
	}
	if err := u.patch(ctx, running, map[string]any{
		AnnotationStarted: nil, AnnotationFailed: nil, AnnotationReason: nil, AnnotationFrom: nil,
	}); err != nil {
		u.logger.WarnContext(
			ctx,
			"cannot record the operator's version on its HelmChart",
			"error",
			err,
		)
		return
	}
	if asked != running {
		u.logger.InfoContext(ctx, "the operator's HelmChart names the version it runs",
			"was", asked, "version", running)
	}
}

func (u *Upgrader) failure(ctx context.Context, want string, d *appsv1.Deployment) string {
	job, err := u.kube.BatchV1().Jobs(u.chartSpace).
		Get(ctx, "helm-install-"+u.chart, metav1.GetOptions{})
	if err == nil && jobVersion(job) == want && job.Status.Succeeded == 0 {
		if jobFailed(job) || job.Status.Failed >= maxJobFailures {
			return fmt.Sprintf(
				"helm-controller could not install the operator's chart at %s (%d failed attempts of %s/%s); "+
					"the registry may not have that version",
				want,
				job.Status.Failed,
				u.chartSpace,
				job.Name,
			)
		}
	}
	if d.Annotations[AnnotationVersion] != want {
		return ""
	}
	for _, c := range d.Status.Conditions {
		if c.Type == appsv1.DeploymentProgressing && c.Status == corev1.ConditionFalse &&
			c.Reason == progressDeadline {
			return fmt.Sprintf("the operator %s never became ready: %s", want, c.Message)
		}
	}
	return ""
}

func (u *Upgrader) patch(ctx context.Context, version string, notes map[string]any) error {
	body, err := sonic.Marshal(map[string]any{
		"metadata": map[string]any{"annotations": notes},
		"spec":     map[string]any{"version": version},
	})
	if err != nil {
		return err
	}
	_, err = u.dynamic.Resource(HelmCharts).Namespace(u.chartSpace).
		Patch(ctx, u.chart, types.MergePatchType, body, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("patch the HelmChart %s/%s: %w", u.chartSpace, u.chart, err)
	}
	return nil
}

func (u *Upgrader) waiting(st Status, msg string) Status {
	st.Phase = PhaseUpgrading
	st.Message = join(msg, st.Message)
	return st
}

func (u *Upgrader) failed(st Status, msg string) Status {
	st.Phase = PhaseFailed
	st.Message = join(msg, st.Message)
	return st
}

func (u *Upgrader) sayOnce(ctx context.Context, level slog.Level, msg string) {
	if u.said == msg {
		return
	}
	u.said = msg
	u.logger.Log(ctx, level, msg)
}

func pinnedBy(hc *unstructured.Unstructured) string {
	for _, ref := range hc.GetOwnerReferences() {
		if ref.Kind == addonKind {
			return ref.Name
		}
	}
	return ""
}

func jobVersion(job *batchv1.Job) string {
	containers := job.Spec.Template.Spec.Containers
	for i := range containers {
		for _, e := range containers[i].Env {
			if e.Name == jobVersionEnv {
				return e.Value
			}
		}
	}
	return ""
}

func jobFailed(job *batchv1.Job) bool {
	for _, c := range job.Status.Conditions {
		if c.Type == batchv1.JobFailed && c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func join(first, second string) string {
	switch {
	case first == "":
		return health.Clean(second, maxMessage)
	case second == "":
		return health.Clean(first, maxMessage)
	}
	return health.Clean(first+"; "+second, maxMessage)
}
