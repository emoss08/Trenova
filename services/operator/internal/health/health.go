package health

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

const (
	componentRelease  = "release"
	componentPostgres = "postgres"
	labelName         = "app.kubernetes.io/name"
	labelInstance     = "app.kubernetes.io/instance"
	labelComponent    = "app.kubernetes.io/component"
	maxMessage        = 200
	healthyPhase      = "Cluster in healthy state"
)

var clusterResource = schema.GroupVersionResource{
	Group:    "postgresql.cnpg.io",
	Version:  "v1",
	Resource: "clusters",
}

type Component struct {
	Name    string
	Ready   bool
	Message string
}

type Checker struct {
	kube      kubernetes.Interface
	dynamic   dynamic.Interface
	namespace string
	release   string
	chart     string
}

type CheckerParams struct {
	Kube      kubernetes.Interface
	Dynamic   dynamic.Interface
	Namespace string
	Release   string
	Chart     string
}

func NewChecker(p *CheckerParams) *Checker {
	return &Checker{
		kube:      p.Kube,
		dynamic:   p.Dynamic,
		namespace: p.Namespace,
		release:   p.Release,
		chart:     p.Chart,
	}
}

type CheckParams struct {
	HelmRevision int
	Stopped      bool
	Hibernated   bool
}

func (c *Checker) Check(ctx context.Context, p CheckParams) ([]Component, error) {
	selector := fmt.Sprintf("%s=%s,%s=%s", labelName, c.chart, labelInstance, c.release)
	list := metav1.ListOptions{LabelSelector: selector}
	found := []checked{}

	deployments, err := c.kube.AppsV1().Deployments(c.namespace).List(ctx, list)
	if err != nil {
		return nil, fmt.Errorf("list deployments: %w", err)
	}
	for i := range deployments.Items {
		d := &deployments.Items[i]
		found = append(found, checked{deployment(d), podSelector(d.Spec.Selector)})
	}
	sets, err := c.kube.AppsV1().StatefulSets(c.namespace).List(ctx, list)
	if err != nil {
		return nil, fmt.Errorf("list statefulsets: %w", err)
	}
	for i := range sets.Items {
		s := &sets.Items[i]
		found = append(found, checked{statefulSet(s), podSelector(s.Spec.Selector)})
	}
	pg, cluster := c.postgres(ctx, p.Hibernated)
	found = append(found, checked{
		comp: pg,
		pods: labels.SelectorFromSet(labels.Set{labelCNPGCluster: cluster}),
	})
	if !p.Stopped {
		job, jobName, jerr := c.releaseJob(ctx, &list, p.HelmRevision)
		if jerr != nil {
			return nil, jerr
		}
		var sel labels.Selector
		if jobName != "" {
			sel = labels.SelectorFromSet(labels.Set{labelJobName: jobName})
		}
		found = append(found, checked{job, sel})
	}

	out := make([]Component, 0, len(found))
	var seen *observed
	for _, f := range found {
		if !f.comp.Ready && f.pods != nil {
			if seen == nil {
				o := c.observe(ctx)
				seen = &o
			}
			f.comp = explain(f.comp, seen.cause(f.pods))
		}
		out = append(out, f.comp)
	}
	slices.SortFunc(out, func(a, b Component) int { return strings.Compare(a.Name, b.Name) })
	return out, nil
}

// checked is a component and the selector of the pods that make it up.
type checked struct {
	comp Component
	pods labels.Selector
}

func podSelector(s *metav1.LabelSelector) labels.Selector {
	sel, err := metav1.LabelSelectorAsSelector(s)
	if err != nil {
		return nil
	}
	return sel
}

func component(m *metav1.ObjectMeta) string {
	if n := m.Labels[labelComponent]; n != "" {
		return n
	}
	return m.Name
}

func deployment(d *appsv1.Deployment) Component {
	want := int32(1)
	if d.Spec.Replicas != nil {
		want = *d.Spec.Replicas
	}
	c := Component{Name: component(&d.ObjectMeta)}
	switch {
	case want == 0 && d.Status.Replicas == 0:
		c.Ready, c.Message = true, "stopped"
	case d.Status.ObservedGeneration < d.Generation:
		c.Message = "rolling out"
	case d.Status.UpdatedReplicas < want:
		c.Message = fmt.Sprintf("%d of %d replicas updated", d.Status.UpdatedReplicas, want)
	case d.Status.AvailableReplicas < want:
		c.Message = fmt.Sprintf("%d of %d replicas available", d.Status.AvailableReplicas, want)
	case d.Status.Replicas > want:
		c.Message = "old replicas terminating"
	default:
		c.Ready = true
	}
	return c
}

func statefulSet(s *appsv1.StatefulSet) Component {
	want := int32(1)
	if s.Spec.Replicas != nil {
		want = *s.Spec.Replicas
	}
	c := Component{Name: component(&s.ObjectMeta)}
	switch {
	case want == 0 && s.Status.Replicas == 0:
		c.Ready, c.Message = true, "stopped"
	case s.Status.ObservedGeneration < s.Generation:
		c.Message = "rolling out"
	case s.Status.UpdateRevision != "" && s.Status.CurrentRevision != s.Status.UpdateRevision:
		c.Message = "updating"
	case s.Status.ReadyReplicas < want:
		c.Message = fmt.Sprintf("%d of %d replicas ready", s.Status.ReadyReplicas, want)
	default:
		c.Ready = true
	}
	return c
}

func (c *Checker) postgres(
	ctx context.Context,
	hibernated bool,
) (comp Component, clusterName string) {
	name := c.release + "-postgres"
	if c.release != c.chart && !strings.Contains(c.release, c.chart) {
		name = c.release + "-" + c.chart + "-postgres"
	}
	obj, err := c.dynamic.Resource(clusterResource).
		Namespace(c.namespace).
		Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return Component{
			Name:    componentPostgres,
			Message: trim("cluster " + name + ": " + err.Error()),
		}, name
	}
	return cluster(obj, hibernated), name
}

func cluster(obj *unstructured.Unstructured, hibernated bool) Component {
	c := Component{Name: componentPostgres}
	phase, _, _ := unstructured.NestedString(obj.Object, "status", "phase")
	want, _, _ := unstructured.NestedInt64(obj.Object, "spec", "instances")
	ready, _, _ := unstructured.NestedInt64(obj.Object, "status", "readyInstances")
	if hibernated {
		if hibernationApplied(obj) {
			c.Ready, c.Message = true, "hibernated"
		} else {
			c.Message = "hibernating"
		}
		return c
	}
	switch {
	case phase == healthyPhase && ready >= want:
		c.Ready = true
	case phase != "":
		c.Message = trim(fmt.Sprintf("%s (%d of %d instances ready)", phase, ready, want))
	default:
		c.Message = "starting"
	}
	return c
}

func hibernationApplied(obj *unstructured.Unstructured) bool {
	conds, _, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")
	for _, raw := range conds {
		cond, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if cond["type"] == "cnpg.io/hibernation" && cond["status"] == "True" {
			return true
		}
	}
	return false
}

func (c *Checker) releaseJob(
	ctx context.Context,
	list *metav1.ListOptions,
	revision int,
) (Component, string, error) {
	comp := Component{Name: componentRelease}
	jobs, err := c.kube.BatchV1().Jobs(c.namespace).List(ctx, *list)
	if err != nil {
		return comp, "", fmt.Errorf("list jobs: %w", err)
	}
	suffix := "-" + strconv.Itoa(revision)
	for i := range jobs.Items {
		j := &jobs.Items[i]
		if j.Labels[labelComponent] != componentRelease || !strings.HasSuffix(j.Name, suffix) {
			continue
		}
		return job(j), j.Name, nil
	}
	comp.Message = "waiting for the release Job of revision " + strconv.Itoa(revision)
	return comp, "", nil
}

func job(j *batchv1.Job) Component {
	c := Component{Name: componentRelease}
	for _, cond := range j.Status.Conditions {
		if cond.Status != corev1.ConditionTrue {
			continue
		}
		if cond.Type == batchv1.JobComplete {
			c.Ready = true
			return c
		}
		if cond.Type == batchv1.JobFailed {
			c.Message = trim("failed: " + cond.Message)
			return c
		}
	}
	c.Message = "running"
	return c
}

// trim keeps a component message within the instance API's limit.
func trim(s string) string { return Clean(s, maxMessage) }

// Clean makes s acceptable to the instance API, which refuses control
// characters in a message: every run of control characters and spaces
// becomes one space (Helm's and Kubernetes' errors are often several lines),
// and the result is cut between characters to at most maxLen.
func Clean(s string, maxLen int) string {
	var b strings.Builder
	space := false
	for _, r := range s {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			space = true
			continue
		}
		if space && b.Len() > 0 {
			b.WriteByte(' ')
		}
		space = false
		b.WriteRune(r)
	}
	out := []rune(b.String())
	if len(out) <= maxLen {
		return string(out)
	}
	return string(out[:maxLen-1]) + "…"
}
