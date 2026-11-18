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
	"sync"
	"time"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"

	"github.com/emoss08/trenova/operator/internal/health"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/logs"
	"github.com/emoss08/trenova/operator/internal/tms"
)

const (
	labelName      = "app.kubernetes.io/name"
	labelInstance  = "app.kubernetes.io/instance"
	labelComponent = "app.kubernetes.io/component"

	restartedAt = "kubectl.kubernetes.io/restartedAt"

	componentPostgres = "postgres"

	maxMessage   = 500
	maxOutput    = 262144
	defaultLines = 200
	maxLines     = 500
	maxKeptLines = 5000
	maxPods      = 5
	remembered   = time.Hour
)

type ControlPlane interface {
	Commands(ctx context.Context) (instanceapi.CommandList, error)
	CompleteCommand(ctx context.Context, id uuid.UUID, result instanceapi.CommandResult) error
}

type Grants interface {
	AuthorizeLogPull(ctx context.Context, email string, kind tms.LogPullKind) error
}

type Logs interface {
	Kept() bool
	Longest() time.Duration
	Last(
		ctx context.Context,
		component string,
		lines int,
		redact func(string) string,
	) (string, error)
	Submit(job *logs.Job) logs.Submitted
	Finish(ctx context.Context, bundle uuid.UUID, outcome *logs.Outcome)
}

type Runner struct {
	cp        ControlPlane
	kube      kubernetes.Interface
	grants    Grants
	kept      Logs
	namespace string
	release   string
	chart     string
	logger    *slog.Logger
	now       func() time.Time
	tierWait  time.Duration
	tierPoll  time.Duration

	mu          sync.Mutex
	done        map[uuid.UUID]finished
	pending     map[uuid.UUID]struct{}
	emailEvents bool
	rolling     bool
}

type finished struct {
	result instanceapi.CommandResult
	at     time.Time
}

type Params struct {
	ControlPlane ControlPlane
	Kube         kubernetes.Interface
	Grants       Grants
	Logs         Logs
	Namespace    string
	Release      string
	Chart        string
	Logger       *slog.Logger
	Now          func() time.Time
}

func New(p *Params) (*Runner, error) {
	var missing []string
	if p.ControlPlane == nil {
		missing = append(missing, "the control plane")
	}
	if p.Kube == nil {
		missing = append(missing, "a Kubernetes client")
	}
	if p.Grants == nil {
		missing = append(missing, "the TMS's support access grants")
	}
	if p.Logs == nil {
		missing = append(missing, "the kept logs")
	}
	if p.Logger == nil {
		missing = append(missing, "a logger")
	}
	if p.Namespace == "" || p.Release == "" || p.Chart == "" {
		missing = append(missing, "the release's namespace, name and chart")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("commands: the runner needs %s", strings.Join(missing, ", "))
	}
	now := p.Now
	if now == nil {
		now = time.Now
	}
	return &Runner{
		cp:        p.ControlPlane,
		kube:      p.Kube,
		grants:    p.Grants,
		kept:      p.Logs,
		namespace: p.Namespace,
		release:   p.Release,
		chart:     p.Chart,
		logger:    p.Logger,
		now:       now,
		tierWait:  tierWait,
		tierPoll:  tierPoll,
		done:      map[uuid.UUID]finished{},
		pending:   map[uuid.UUID]struct{}{},
	}, nil
}

type state int

const (
	answered state = iota
	later
	retry
)

func (r *Runner) EmailEventsWaiting() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.emailEvents
}

func (r *Runner) Run(ctx context.Context) error {
	listed, err := r.cp.Commands(ctx)
	r.mu.Lock()
	r.emailEvents = err == nil && listed.EmailEvents
	r.mu.Unlock()
	if err != nil {
		return err
	}
	r.forget()
	list := listed.Commands
	var errs []error
	for i := range list {
		cmd := &list[i]
		r.mu.Lock()
		f, seen := r.done[cmd.Id]
		_, waiting := r.pending[cmd.Id]
		r.mu.Unlock()
		if waiting {
			continue
		}
		if !seen {
			r.logger.InfoContext(ctx, "carrying out a command",
				"command", cmd.Id, "kind", string(cmd.Kind), "component", deref(cmd.Component))
			result, st := r.execute(ctx, cmd)
			if st != answered {
				continue
			}
			f = r.settle(cmd.Id, result)
			r.logger.InfoContext(ctx, "carried out a command",
				"command", cmd.Id, "kind", string(cmd.Kind), "component", deref(cmd.Component),
				"status", string(f.result.Status), "message", deref(f.result.Message))
		}
		if err = r.cp.CompleteCommand(ctx, cmd.Id, f.result); err != nil {
			errs = append(errs, fmt.Errorf("answer command %s: %w", cmd.Id, err))
		}
	}
	return errors.Join(errs...)
}

func (r *Runner) settle(id uuid.UUID, result instanceapi.CommandResult) finished {
	r.mu.Lock()
	defer r.mu.Unlock()
	f := finished{result: result, at: r.now()}
	delete(r.pending, id)
	r.done[id] = f
	return f
}

func (r *Runner) forget() {
	r.mu.Lock()
	defer r.mu.Unlock()
	cutoff := r.now().Add(-remembered)
	for id, f := range r.done {
		if f.at.Before(cutoff) {
			delete(r.done, id)
		}
	}
}

func (r *Runner) execute(
	ctx context.Context,
	cmd *instanceapi.Command,
) (instanceapi.CommandResult, state) {
	switch cmd.Kind {
	case instanceapi.Restart:
		component := deref(cmd.Component)
		if component == "" {
			return r.restartAll(ctx, cmd.Id)
		}
		n, err := r.restart(ctx, component)
		if err != nil {
			return failed(err), answered
		}
		return succeeded(fmt.Sprintf("restarted %d workload(s)", n), nil), answered
	case instanceapi.CollectLogs:
		return r.collectLines(ctx, cmd), answered
	case instanceapi.CollectLogBundle:
		return r.collectBundle(ctx, cmd)
	default:
		return failed(fmt.Errorf("this operator does not know the command %q", cmd.Kind)), answered
	}
}

func (r *Runner) selector(component string) string {
	set := labels.Set{labelName: r.chart, labelInstance: r.release}
	if component != "" {
		set[labelComponent] = component
	}
	return labels.SelectorFromSet(set).String()
}

type logRequest struct {
	pod       *corev1.Pod
	container string
	tail      int64
	previous  bool
	redact    func(string) string
}

func (r *Runner) kubeletLines(
	ctx context.Context,
	component string,
	lines int,
	redact func(string) string,
) (string, error) {
	want := int64(min(lines, maxLines))
	pods, err := r.kube.CoreV1().Pods(r.namespace).
		List(ctx, metav1.ListOptions{LabelSelector: r.selector(component)})
	if err != nil {
		return "", fmt.Errorf("list the component's pods: %w", err)
	}
	if len(pods.Items) == 0 {
		return "", fmt.Errorf("the component %q has no pods", component)
	}
	slices.SortFunc(pods.Items, newestFirst)
	if len(pods.Items) > maxPods {
		pods.Items = pods.Items[:maxPods]
	}
	var out strings.Builder
	for i := range pods.Items {
		pod := &pods.Items[i]
		for _, container := range containers(pod) {
			req := logRequest{pod: pod, container: container, tail: want, redact: redact}
			if crashed(pod, container) {
				previous := req
				previous.previous = true
				r.readLog(ctx, &out, previous)
			}
			r.readLog(ctx, &out, req)
		}
	}
	text := out.String()
	if len(text) > maxOutput {
		text = text[len(text)-maxOutput:]
		if nl := strings.IndexByte(text, '\n'); nl >= 0 {
			text = text[nl+1:]
		}
	}
	return text, nil
}

func (r *Runner) readLog(ctx context.Context, out *strings.Builder, req logRequest) {
	label := req.pod.Name + "/" + req.container
	if req.previous {
		label += " (previous)"
	}
	tail := req.tail
	stream, err := r.kube.CoreV1().Pods(r.namespace).GetLogs(req.pod.Name, &corev1.PodLogOptions{
		Container: req.container, TailLines: &tail, Timestamps: true, Previous: req.previous,
	}).Stream(ctx)
	if err != nil {
		fmt.Fprintf(
			out,
			"%s: could not read its log: %s\n",
			label,
			req.redact(oneLine(err.Error())),
		)
		return
	}
	defer func() { _ = stream.Close() }()
	scan := bufio.NewScanner(io.LimitReader(stream, maxOutput))
	scan.Buffer(make([]byte, 0, 64*1024), 256*1024)
	for scan.Scan() {
		fmt.Fprintf(out, "%s: %s\n", label, req.redact(scan.Text()))
	}
}

func newestFirst(a, b corev1.Pod) int { //nolint:gocritic // hugeParam: fixed by slices.SortFunc
	if c := b.CreationTimestamp.Compare(a.CreationTimestamp.Time); c != 0 {
		return c
	}
	return strings.Compare(a.Name, b.Name)
}

func containers(pod *corev1.Pod) []string {
	out := make([]string, 0, len(pod.Spec.InitContainers)+len(pod.Spec.Containers))
	for i := range pod.Spec.InitContainers {
		out = append(out, pod.Spec.InitContainers[i].Name)
	}
	for i := range pod.Spec.Containers {
		out = append(out, pod.Spec.Containers[i].Name)
	}
	return out
}

func crashed(pod *corev1.Pod, container string) bool {
	statuses := slices.Concat(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses)
	for i := range statuses {
		s := &statuses[i]
		if s.Name == container {
			return s.RestartCount > 0 && s.State.Running == nil &&
				s.LastTerminationState.Terminated != nil
		}
	}
	return false
}

func succeeded(message string, output *string) instanceapi.CommandResult {
	res := instanceapi.CommandResult{
		Status: instanceapi.CommandResultStatusSucceeded,
		Output: output,
	}
	if message != "" {
		res.Message = &message
	}
	return res
}

func failed(err error) instanceapi.CommandResult {
	msg := health.Clean(err.Error(), maxMessage)
	return instanceapi.CommandResult{Status: instanceapi.CommandResultStatusFailed, Message: &msg}
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
