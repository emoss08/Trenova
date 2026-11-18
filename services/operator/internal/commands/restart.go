package commands

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/emoss08/trenova/operator/internal/health"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
)

const (
	tierWait = 10 * time.Minute
	tierPoll = 5 * time.Second

	kindDeployment  = "deployment"
	kindStatefulSet = "statefulset"
)

var restartOrder = [][]string{
	{"redis", "meilisearch", "temporal", "minio", "clamav", "docling", "embedding", "loki"},
	{"gtc", "alloy"},
	{"api", "worker", "imaging", "gotenberg"},
	{"client"},
}

type workload struct {
	kind      string
	name      string
	component string
}

func (r *Runner) restart(ctx context.Context, component string) (int, error) {
	if component == componentPostgres {
		return 0, errors.New("the database is not restarted this way")
	}
	list, err := r.workloads(ctx, component)
	if err != nil {
		return 0, err
	}
	if len(list) == 0 {
		return 0, fmt.Errorf("found no workload for the component %q", component)
	}
	body := r.restartPatch()
	for i, w := range list {
		if perr := r.patch(ctx, w, body); perr != nil {
			return i, perr
		}
	}
	return len(list), nil
}

func (r *Runner) restartAll(ctx context.Context, id uuid.UUID) (instanceapi.CommandResult, state) {
	r.mu.Lock()
	busy := r.rolling
	r.mu.Unlock()
	if busy {
		r.logger.InfoContext(ctx,
			"another restart of every part is still going; the command waits for the next tick",
			"command", id,
		)
		return instanceapi.CommandResult{}, retry
	}
	list, err := r.workloads(ctx, "")
	if err != nil {
		return failed(err), answered
	}
	plan := tiers(list)
	if len(plan) == 0 {
		return failed(errors.New("found no Trenova workload to restart")), answered
	}
	r.mu.Lock()
	r.rolling = true
	r.pending[id] = struct{}{}
	r.mu.Unlock()
	bound := time.Duration(len(plan)) * (r.tierWait + time.Minute)
	run, cancel := context.WithTimeout(context.WithoutCancel(ctx), bound)
	go func() {
		defer cancel()
		res := r.roll(run, plan)
		r.mu.Lock()
		r.rolling = false
		r.mu.Unlock()
		r.settle(id, res)
		r.logger.Info("a restart of every part is ready to answer",
			"command", id, "status", string(res.Status), "message", deref(res.Message))
	}()
	return instanceapi.CommandResult{}, later
}

func (r *Runner) roll(ctx context.Context, plan [][]workload) instanceapi.CommandResult {
	body := r.restartPatch()
	restarted := 0
	var late []string
	for _, tier := range plan {
		for _, w := range tier {
			if err := r.patch(ctx, w, body); err != nil {
				return failed(fmt.Errorf(
					"restarted %d workload(s), then could not go on: %w",
					restarted,
					err,
				))
			}
			restarted++
		}
		late = append(late, r.await(ctx, tier)...)
	}
	if len(late) > 0 {
		return failedMessage(fmt.Sprintf(
			"restarted %d workload(s) in order, but %s did not become ready within %s of its restart",
			restarted,
			strings.Join(late, ", "),
			r.tierWait,
		))
	}
	return succeeded(fmt.Sprintf("restarted %d workload(s)", restarted), nil)
}

func (r *Runner) await(ctx context.Context, tier []workload) []string {
	waiting := slices.Clone(tier)
	deadline := time.Now().Add(r.tierWait)
	for {
		waiting = slices.DeleteFunc(waiting, func(w workload) bool { return r.ready(ctx, w) })
		if len(waiting) == 0 {
			return nil
		}
		left := time.Until(deadline)
		if left <= 0 || ctx.Err() != nil {
			return components(waiting)
		}
		timer := time.NewTimer(min(r.tierPoll, left))
		select {
		case <-ctx.Done():
			timer.Stop()
			return components(waiting)
		case <-timer.C:
		}
	}
}

func (r *Runner) ready(ctx context.Context, w workload) bool {
	switch w.kind {
	case kindDeployment:
		d, err := r.kube.AppsV1().Deployments(r.namespace).Get(ctx, w.name, metav1.GetOptions{})
		return err == nil && health.Deployment(d).Ready
	default:
		s, err := r.kube.AppsV1().StatefulSets(r.namespace).Get(ctx, w.name, metav1.GetOptions{})
		return err == nil && health.StatefulSet(s).Ready
	}
}

func (r *Runner) workloads(ctx context.Context, component string) ([]workload, error) {
	list := metav1.ListOptions{LabelSelector: r.selector(component)}
	deployments, err := r.kube.AppsV1().Deployments(r.namespace).List(ctx, list)
	if err != nil {
		return nil, fmt.Errorf("list deployments: %w", err)
	}
	sets, err := r.kube.AppsV1().StatefulSets(r.namespace).List(ctx, list)
	if err != nil {
		return nil, fmt.Errorf("list statefulsets: %w", err)
	}
	out := make([]workload, 0, len(deployments.Items)+len(sets.Items))
	for i := range deployments.Items {
		out = append(out, deploymentWorkload(&deployments.Items[i]))
	}
	for i := range sets.Items {
		out = append(out, statefulSetWorkload(&sets.Items[i]))
	}
	return out, nil
}

func deploymentWorkload(d *appsv1.Deployment) workload {
	return workload{kind: kindDeployment, name: d.Name, component: d.Labels[labelComponent]}
}

func statefulSetWorkload(s *appsv1.StatefulSet) workload {
	return workload{kind: kindStatefulSet, name: s.Name, component: s.Labels[labelComponent]}
}

func (r *Runner) restartPatch() []byte {
	return fmt.Appendf(nil,
		`{"spec":{"template":{"metadata":{"annotations":{%q:%q}}}}}`,
		restartedAt, r.now().UTC().Format(time.RFC3339),
	)
}

func (r *Runner) patch(ctx context.Context, w workload, body []byte) error {
	var err error
	switch w.kind {
	case kindDeployment:
		_, err = r.kube.AppsV1().Deployments(r.namespace).
			Patch(ctx, w.name, types.StrategicMergePatchType, body, metav1.PatchOptions{})
	default:
		_, err = r.kube.AppsV1().StatefulSets(r.namespace).
			Patch(ctx, w.name, types.StrategicMergePatchType, body, metav1.PatchOptions{})
	}
	if err != nil {
		return fmt.Errorf("restart %s %s: %w", w.kind, w.name, err)
	}
	return nil
}

func tierOf(component string) int {
	for i, names := range restartOrder {
		if slices.Contains(names, component) {
			return i
		}
	}
	return len(restartOrder)
}

func tiers(list []workload) [][]workload {
	byTier := make([][]workload, len(restartOrder)+1)
	for _, w := range list {
		if w.component == componentPostgres {
			continue
		}
		t := tierOf(w.component)
		byTier[t] = append(byTier[t], w)
	}
	out := make([][]workload, 0, len(byTier))
	for _, tier := range byTier {
		if len(tier) == 0 {
			continue
		}
		slices.SortFunc(tier, func(a, b workload) int {
			return cmp.Or(cmp.Compare(a.component, b.component), cmp.Compare(a.name, b.name))
		})
		out = append(out, tier)
	}
	return out
}

func components(list []workload) []string {
	out := make([]string, 0, len(list))
	for _, w := range list {
		if !slices.Contains(out, w.component) {
			out = append(out, w.component)
		}
	}
	return out
}
