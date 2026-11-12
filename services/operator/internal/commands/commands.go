// Package commands carries out what the control plane asks of the instance
// besides converging to its spec: restarting Trenova or one of its
// components, and collecting a component's recent log lines for staff.
//
// The instance takes no inbound connections, so the operator asks for its
// commands, carries each out once, and answers. Log lines leave the
// instance only after every secret the namespace holds has been removed
// from them.
package commands

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
)

const (
	labelName      = "app.kubernetes.io/name"
	labelInstance  = "app.kubernetes.io/instance"
	labelComponent = "app.kubernetes.io/component"

	// restartedAt is the annotation `kubectl rollout restart` sets: a new
	// value on the pod template makes the workload roll its pods.
	restartedAt = "kubectl.kubernetes.io/restartedAt"

	componentPostgres = "postgres"

	maxMessage   = 500
	maxOutput    = 262144
	defaultLines = 200
	maxLines     = 500
	// maxPods bounds the pods one collection reads, so a component scaled
	// wide cannot make it unbounded.
	maxPods = 5
	// remembered is how long an answered command's ID is kept, to carry
	// each out once even if the answer was lost and it is listed again.
	remembered = time.Hour
)

// ControlPlane is the part of the instance API commands use.
type ControlPlane interface {
	Commands(ctx context.Context) ([]instanceapi.Command, error)
	CompleteCommand(ctx context.Context, id uuid.UUID, result instanceapi.CommandResult) error
}

// Runner carries out the instance's commands.
type Runner struct {
	kube      kubernetes.Interface
	namespace string
	release   string
	chart     string
	logger    *slog.Logger
	now       func() time.Time

	// done holds the result of each command already carried out, by ID,
	// until it is answered and for a while after.
	done map[uuid.UUID]finished
}

type finished struct {
	result instanceapi.CommandResult
	at     time.Time
}

// Params configures a Runner for one Helm release in one namespace.
type Params struct {
	Kube      kubernetes.Interface
	Namespace string
	Release   string
	Chart     string
	Logger    *slog.Logger
	Now       func() time.Time
}

// New makes a Runner.
func New(p *Params) *Runner {
	now := p.Now
	if now == nil {
		now = time.Now
	}
	return &Runner{
		kube: p.Kube, namespace: p.Namespace, release: p.Release, chart: p.Chart,
		logger: p.Logger, now: now, done: map[uuid.UUID]finished{},
	}
}

// Run asks for the instance's commands, carries out those not yet carried
// out, and answers each. A command is carried out once: if its answer does
// not reach the control plane it is listed again, and answered again with
// the result it already has.
func (r *Runner) Run(ctx context.Context, cp ControlPlane) error {
	list, err := cp.Commands(ctx)
	if err != nil {
		return err
	}
	r.forget()
	var errs []error
	for i := range list {
		cmd := &list[i]
		f, seen := r.done[cmd.Id]
		if !seen {
			f = finished{result: r.execute(ctx, cmd), at: r.now()}
			r.done[cmd.Id] = f
			r.logger.InfoContext(ctx, "carried out a command",
				"command", cmd.Id, "kind", string(cmd.Kind), "component", deref(cmd.Component),
				"status", string(f.result.Status))
		}
		if err := cp.CompleteCommand(ctx, cmd.Id, f.result); err != nil {
			errs = append(errs, fmt.Errorf("answer command %s: %w", cmd.Id, err))
		}
	}
	return errors.Join(errs...)
}

func (r *Runner) forget() {
	cutoff := r.now().Add(-remembered)
	for id, f := range r.done {
		if f.at.Before(cutoff) {
			delete(r.done, id)
		}
	}
}

func (r *Runner) execute(ctx context.Context, cmd *instanceapi.Command) instanceapi.CommandResult {
	switch cmd.Kind {
	case instanceapi.Restart:
		n, err := r.restart(ctx, deref(cmd.Component))
		if err != nil {
			return failed(err)
		}
		return succeeded(fmt.Sprintf("restarted %d workload(s)", n), nil)
	case instanceapi.CollectLogs:
		out, err := r.logs(ctx, deref(cmd.Component), cmd.Lines)
		if err != nil {
			return failed(err)
		}
		return succeeded("", &out)
	default:
		return failed(fmt.Errorf("this operator does not know the command %q", cmd.Kind))
	}
}

func (r *Runner) selector(component string) string {
	set := labels.Set{labelName: r.chart, labelInstance: r.release}
	if component != "" {
		set[labelComponent] = component
	}
	return labels.SelectorFromSet(set).String()
}

// restart rolls the pods of a component's workloads, or of every Trenova
// workload when no component is named, the way `kubectl rollout restart`
// does. The database is CloudNativePG's to restart and is left alone.
func (r *Runner) restart(ctx context.Context, component string) (int, error) {
	if component == componentPostgres {
		return 0, errors.New("the database is not restarted this way")
	}
	list := metav1.ListOptions{LabelSelector: r.selector(component)}
	patch := []byte(fmt.Sprintf(
		`{"spec":{"template":{"metadata":{"annotations":{%q:%q}}}}}`,
		restartedAt, r.now().UTC().Format(time.RFC3339),
	))
	restarted := 0
	deployments, err := r.kube.AppsV1().Deployments(r.namespace).List(ctx, list)
	if err != nil {
		return 0, fmt.Errorf("list deployments: %w", err)
	}
	for i := range deployments.Items {
		name := deployments.Items[i].Name
		if _, err := r.kube.AppsV1().Deployments(r.namespace).
			Patch(ctx, name, types.StrategicMergePatchType, patch, metav1.PatchOptions{}); err != nil {
			return restarted, fmt.Errorf("restart deployment %s: %w", name, err)
		}
		restarted++
	}
	sets, err := r.kube.AppsV1().StatefulSets(r.namespace).List(ctx, list)
	if err != nil {
		return restarted, fmt.Errorf("list statefulsets: %w", err)
	}
	for i := range sets.Items {
		name := sets.Items[i].Name
		if _, err := r.kube.AppsV1().StatefulSets(r.namespace).
			Patch(ctx, name, types.StrategicMergePatchType, patch, metav1.PatchOptions{}); err != nil {
			return restarted, fmt.Errorf("restart statefulset %s: %w", name, err)
		}
		restarted++
	}
	if restarted == 0 {
		if component == "" {
			return 0, errors.New("found no Trenova workload to restart")
		}
		return 0, fmt.Errorf("found no workload for the component %q", component)
	}
	return restarted, nil
}

// logs returns the most recent lines of a component's pods, each line
// prefixed with its pod and container, with every secret in the namespace
// removed.
func (r *Runner) logs(ctx context.Context, component string, lines *int) (string, error) {
	if component == "" {
		return "", errors.New("name the component whose logs to collect")
	}
	want := int64(defaultLines)
	if lines != nil {
		want = int64(min(max(*lines, 1), maxLines))
	}
	pods, err := r.kube.CoreV1().Pods(r.namespace).List(ctx, metav1.ListOptions{LabelSelector: r.selector(component)})
	if err != nil {
		return "", fmt.Errorf("list the component's pods: %w", err)
	}
	if len(pods.Items) == 0 {
		return "", fmt.Errorf("the component %q has no pods", component)
	}
	slices.SortFunc(pods.Items, func(a, b corev1.Pod) int { return strings.Compare(a.Name, b.Name) })
	if len(pods.Items) > maxPods {
		pods.Items = pods.Items[:maxPods]
	}
	redact, err := r.redactor(ctx)
	if err != nil {
		return "", err
	}
	var out strings.Builder
	for i := range pods.Items {
		pod := &pods.Items[i]
		for c := range pod.Spec.Containers {
			container := pod.Spec.Containers[c].Name
			tail := want
			stream, err := r.kube.CoreV1().Pods(r.namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
				Container: container, TailLines: &tail, Timestamps: true,
			}).Stream(ctx)
			if err != nil {
				fmt.Fprintf(&out, "%s/%s: could not read its log: %s\n", pod.Name, container, oneLine(err.Error()))
				continue
			}
			scan := bufio.NewScanner(io.LimitReader(stream, maxOutput))
			scan.Buffer(make([]byte, 0, 64*1024), 256*1024)
			for scan.Scan() {
				fmt.Fprintf(&out, "%s/%s: %s\n", pod.Name, container, redact(scan.Text()))
			}
			_ = stream.Close()
		}
	}
	text := out.String()
	if len(text) > maxOutput {
		// Keep the newest lines, cut at a line's start.
		text = text[len(text)-maxOutput:]
		if nl := strings.IndexByte(text, '\n'); nl >= 0 {
			text = text[nl+1:]
		}
	}
	return text, nil
}

func succeeded(message string, output *string) instanceapi.CommandResult {
	res := instanceapi.CommandResult{Status: instanceapi.Succeeded, Output: output}
	if message != "" {
		res.Message = &message
	}
	return res
}

func failed(err error) instanceapi.CommandResult {
	msg := oneLine(err.Error())
	if len(msg) > maxMessage {
		msg = msg[:maxMessage]
	}
	return instanceapi.CommandResult{Status: instanceapi.Failed, Message: &msg}
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
